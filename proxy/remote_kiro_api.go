package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"runtime"
	"strings"
)

const remoteKiroUserAgent = "kiro-go-remote/1.0"
const remoteAnthropicVersion = "2023-06-01"

// remoteChatURL builds the OpenAI chat completions URL for a normalized base.
func remoteChatURL(base string) string {
	return strings.TrimRight(base, "/") + "/v1/chat/completions"
}

// remoteMessagesURL builds the Anthropic messages URL for a normalized base.
func remoteMessagesURL(base string) string {
	return strings.TrimRight(base, "/") + "/v1/messages"
}

// remoteModelsURL builds the OpenAI models list URL for a normalized base.
func remoteModelsURL(base string) string {
	return strings.TrimRight(base, "/") + "/v1/models"
}

// CallRemoteKiroAPI proxies generation to another Kiro-Go peer.
//
// Protocol is preserved end-to-end:
//   - Claude clients (SourceClaude) → POST {base}/v1/messages with the original
//     Anthropic body. We deliberately do NOT convert Claude→OpenAI: peers often
//     only fully support Claude models on the Anthropic path, and the conversion
//     produced empty/odd streams that local still logged as success (client then
//     retried → two near-identical Remote OK rows, then "stuck").
//   - OpenAI clients (SourceOpenAI) → POST {base}/v1/chat/completions with the
//     original OpenAI body (no Grok schema sanitization / model aliases).
//
// ctx is the caller's request context; the upstream request is derived from it so
// a client disconnect cancels the generation on the peer too.
func CallRemoteKiroAPI(ctx context.Context, account *config.Account, payload *KiroPayload, callback *KiroStreamCallback) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if callback == nil {
		callback = &KiroStreamCallback{}
	}
	if account == nil {
		return fmt.Errorf("remotekiro: account is nil")
	}
	if payload == nil {
		return fmt.Errorf("remotekiro: payload is nil")
	}

	base, err := validateRemoteBaseURL(account.RemoteBaseURL)
	if err != nil {
		return fmt.Errorf("remotekiro: %w", err)
	}
	bearer := strings.TrimSpace(account.AccessToken)
	if bearer == "" {
		return fmt.Errorf("remotekiro: no API key configured (AccessToken)")
	}

	model := resolvePayloadModelForGrok(payload)
	stream := isStreamRequested(payload)

	var (
		bodyBytes []byte
		url       string
		claude    bool
	)
	switch {
	case payload.SourceClaude != nil:
		// Pass the original Claude request through. Override model/stream from the
		// resolved payload so ModelFallback rewrites (if any) still apply.
		// Force stream onto a map so stream:false is not dropped by omitempty —
		// peers that default stream differently would otherwise hang the client.
		reqCopy := *payload.SourceClaude
		if model != "" {
			reqCopy.Model = model
		}
		reqCopy.Stream = stream
		raw, mErr := json.Marshal(&reqCopy)
		if mErr != nil {
			return fmt.Errorf("remotekiro: marshal claude request: %w", mErr)
		}
		var asMap map[string]interface{}
		if err := json.Unmarshal(raw, &asMap); err != nil {
			return fmt.Errorf("remotekiro: remap claude request: %w", err)
		}
		if model != "" {
			asMap["model"] = model
		}
		asMap["stream"] = stream
		bodyBytes, err = json.Marshal(asMap)
		if err != nil {
			return fmt.Errorf("remotekiro: marshal claude request: %w", err)
		}
		url = remoteMessagesURL(base)
		claude = true
	case payload.SourceOpenAI != nil:
		// Marshal the original OpenAI request (not OpenAIToOpenAI — that applies
		// Grok-only schema sanitization and model defaults). Force model/stream
		// onto a map so stream:false is not dropped by omitempty.
		reqCopy := *payload.SourceOpenAI
		if model != "" {
			reqCopy.Model = model
		}
		reqCopy.Stream = stream
		raw, mErr := json.Marshal(&reqCopy)
		if mErr != nil {
			return fmt.Errorf("remotekiro: marshal openai request: %w", mErr)
		}
		var asMap map[string]interface{}
		if err := json.Unmarshal(raw, &asMap); err != nil {
			return fmt.Errorf("remotekiro: remap openai request: %w", err)
		}
		if model != "" {
			asMap["model"] = model
		}
		asMap["stream"] = stream
		bodyBytes, err = json.Marshal(asMap)
		if err != nil {
			return fmt.Errorf("remotekiro: marshal openai request: %w", err)
		}
		url = remoteChatURL(base)
	default:
		return fmt.Errorf("remotekiro: no source request on payload (need SourceClaude or SourceOpenAI)")
	}

	if logger.GetLevel() == logger.LevelDebug {
		logger.Debugf("[RemoteKiro] Request to %s (model=%s, stream=%v, claude=%v)", url, model, stream, claude)
	}

	client := GetClientForProxy(ResolveAccountProxyURL(account))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("remotekiro: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", remoteKiroUserAgent, runtime.GOOS, runtime.GOARCH))
	if claude {
		// Anthropic clients expect x-api-key; Kiro-Go accepts either. Send both so
		// stock peers and pure-Anthropic forks both authenticate.
		req.Header.Set("x-api-key", bearer)
		req.Header.Set("anthropic-version", remoteAnthropicVersion)
	}
	req.Header.Set("Accept", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("remotekiro: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return newUpstreamError("remotekiro", resp.StatusCode, string(errBody), "")
	}

	// A peer that accepts the request then stalls mid-stream would otherwise block
	// forever: this client has Timeout: 0 and ResponseHeaderTimeout only covers the
	// wait for the first header. Peers are the likeliest upstream to hang, since a
	// peer that is itself stuck holds the connection open without sending bytes.
	idleReader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
	defer idleReader.Stop()

	if claude {
		if stream {
			return parseRemoteClaudeSSE(idleReader, callback, model)
		}
		return parseRemoteClaudeResponse(idleReader, callback, model)
	}
	if stream {
		return parseGrokOpenAISSE(idleReader, callback, model)
	}
	return parseGrokOpenAIResponse(idleReader, callback, model)
}

// parseRemoteClaudeSSE reads Anthropic SSE (event: + data: lines) from a peer and
// drives KiroStreamCallback. Empty content with no tool_use is treated as failure
// so the account loop can rotate instead of logging a silent "success".
func parseRemoteClaudeSSE(body io.Reader, callback *KiroStreamCallback, model string) error {
	if callback == nil {
		callback = &KiroStreamCallback{}
	}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		eventName                  string
		inputTokens, outputTokens  int
		gotText, gotThinking, gotTool bool
		// tool_use blocks stream as content_block_start (id/name) then
		// input_json_delta fragments, then content_block_stop.
		toolID, toolName string
		toolArgs         strings.Builder
		inTool           bool
	)

	flushTool := func() {
		if !inTool || callback.OnToolUse == nil {
			toolID, toolName = "", ""
			toolArgs.Reset()
			inTool = false
			return
		}
		if toolName != "" {
			input := map[string]interface{}{}
			if toolArgs.Len() > 0 {
				_ = json.Unmarshal([]byte(toolArgs.String()), &input)
			}
			id := toolID
			if id == "" {
				id = "toolu_remote"
			}
			callback.OnToolUse(KiroToolUse{ToolUseID: id, Name: toolName, Input: input})
			gotTool = true
		}
		toolID, toolName = "", ""
		toolArgs.Reset()
		inTool = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			eventName = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			continue
		}
		// Prefer the event: line; fall back to the type field inside data.
		typ := eventName
		if typ == "" {
			if t, ok := raw["type"].(string); ok {
				typ = t
			}
		}

		switch typ {
		case "content_block_start":
			block, _ := raw["content_block"].(map[string]interface{})
			if block == nil {
				continue
			}
			bType, _ := block["type"].(string)
			switch bType {
			case "tool_use":
				flushTool()
				inTool = true
				toolID, _ = block["id"].(string)
				toolName, _ = block["name"].(string)
				toolArgs.Reset()
			case "text":
				if t, ok := block["text"].(string); ok && t != "" && callback.OnText != nil {
					callback.OnText(t, false)
					gotText = true
				}
			case "thinking":
				if t, ok := block["thinking"].(string); ok && t != "" && callback.OnText != nil {
					callback.OnText(t, true)
					gotThinking = true
				}
			}
		case "content_block_delta":
			delta, _ := raw["delta"].(map[string]interface{})
			if delta == nil {
				continue
			}
			dType, _ := delta["type"].(string)
			switch dType {
			case "text_delta":
				if t, ok := delta["text"].(string); ok && t != "" && callback.OnText != nil {
					callback.OnText(t, false)
					gotText = true
				}
			case "thinking_delta":
				if t, ok := delta["thinking"].(string); ok && t != "" && callback.OnText != nil {
					callback.OnText(t, true)
					gotThinking = true
				}
			case "input_json_delta":
				if partial, ok := delta["partial_json"].(string); ok {
					toolArgs.WriteString(partial)
				}
			}
		case "content_block_stop":
			flushTool()
		case "message_delta":
			if usage, ok := raw["usage"].(map[string]interface{}); ok {
				if v, ok := usage["output_tokens"].(float64); ok {
					outputTokens = int(v)
				}
				// Some peers also put input_tokens on message_delta.
				if v, ok := usage["input_tokens"].(float64); ok && v > 0 {
					inputTokens = int(v)
				}
			}
		case "message_start":
			if msg, ok := raw["message"].(map[string]interface{}); ok {
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if v, ok := usage["input_tokens"].(float64); ok {
						inputTokens = int(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok && v > 0 {
						outputTokens = int(v)
					}
				}
			}
		case "error":
			msg := "remote claude stream error"
			if errObj, ok := raw["error"].(map[string]interface{}); ok {
				if m, ok := errObj["message"].(string); ok && m != "" {
					msg = m
				}
			}
			err := fmt.Errorf("remotekiro: %s", msg)
			if callback.OnError != nil {
				callback.OnError(err)
			}
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if callback.OnError != nil {
			callback.OnError(err)
		}
		return err
	}
	flushTool()

	if !gotText && !gotThinking && !gotTool {
		err := fmt.Errorf("remotekiro: empty claude stream response (model=%s)", model)
		if callback.OnError != nil {
			callback.OnError(err)
		}
		return err
	}

	if callback.OnComplete != nil {
		callback.OnComplete(inputTokens, outputTokens)
	}
	return nil
}

// parseRemoteClaudeResponse handles a non-streaming Anthropic messages JSON body.
func parseRemoteClaudeResponse(body io.Reader, callback *KiroStreamCallback, model string) error {
	if callback == nil {
		callback = &KiroStreamCallback{}
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("remotekiro: read claude response: %w", err)
	}

	var resp ClaudeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("remotekiro: decode claude response: %w", err)
	}

	var gotText, gotThinking, gotTool bool
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != "" && callback.OnText != nil {
				callback.OnText(block.Text, false)
				gotText = true
			}
		case "thinking":
			if block.Thinking != "" && callback.OnText != nil {
				callback.OnText(block.Thinking, true)
				gotThinking = true
			}
		case "tool_use":
			if callback.OnToolUse != nil && block.Name != "" {
				input := map[string]interface{}{}
				switch v := block.Input.(type) {
				case map[string]interface{}:
					input = v
				case string:
					_ = json.Unmarshal([]byte(v), &input)
				default:
					if b, err := json.Marshal(v); err == nil {
						_ = json.Unmarshal(b, &input)
					}
				}
				id := block.ID
				if id == "" {
					id = "toolu_remote"
				}
				callback.OnToolUse(KiroToolUse{ToolUseID: id, Name: block.Name, Input: input})
				gotTool = true
			}
		}
	}

	if !gotText && !gotThinking && !gotTool {
		return fmt.Errorf("remotekiro: empty claude response (model=%s)", model)
	}

	if callback.OnComplete != nil {
		callback.OnComplete(resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
	return nil
}

// FetchRemoteKiroModels lists model IDs from a remote peer's GET /v1/models.
func FetchRemoteKiroModels(account *config.Account) ([]string, error) {
	if account == nil {
		return nil, fmt.Errorf("remotekiro: account is nil")
	}
	base, err := validateRemoteBaseURL(account.RemoteBaseURL)
	if err != nil {
		return nil, err
	}
	bearer := strings.TrimSpace(account.AccessToken)
	if bearer == "" {
		return nil, fmt.Errorf("remotekiro: no API key configured")
	}

	req, err := http.NewRequest(http.MethodGet, remoteModelsURL(base), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", remoteKiroUserAgent)

	resp, err := GetRestClientForProxy(ResolveAccountProxyURL(account)).Do(req)
	if err != nil {
		return nil, fmt.Errorf("remotekiro: models request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, newUpstreamError("remotekiro", resp.StatusCode, string(body), fmt.Sprintf("models HTTP %d", resp.StatusCode))
	}

	ids, err := parseOpenAIModelIDs(body)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ValidateRemoteKiro validates base URL + sk by probing GET /v1/models.
// Returns the canonical base URL and non-empty model id list.
func ValidateRemoteKiro(baseURL, apiKey, proxyURL string) (canonical string, modelIDs []string, err error) {
	canonical, err = validateRemoteBaseURL(baseURL)
	if err != nil {
		return "", nil, err
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return "", nil, fmt.Errorf("API key is required")
	}

	probe := &config.Account{
		RemoteBaseURL: canonical,
		AccessToken:   key,
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
		ProxyURL:      strings.TrimSpace(proxyURL),
	}
	modelIDs, err = FetchRemoteKiroModels(probe)
	if err != nil {
		return "", nil, err
	}
	if len(modelIDs) == 0 {
		return "", nil, fmt.Errorf("remote /v1/models returned no models")
	}
	return canonical, modelIDs, nil
}

// parseOpenAIModelIDs extracts data[].id from an OpenAI-compatible models list body.
func parseOpenAIModelIDs(body []byte) ([]string, error) {
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("remotekiro: parse models: %w", err)
	}
	ids := make([]string, 0, len(parsed.Data))
	seen := make(map[string]bool, len(parsed.Data))
	for _, m := range parsed.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

// remoteCheckKeyResponse is the subset of a Kiro-Go-style check-key payload we
// consume. Both /check/api/lookup (stock) and /checkkey/info (forks) return
// these fields. creditLimit <= 0 means the remote key is unlimited.
type remoteCheckKeyResponse struct {
	Name             string  `json:"name"`
	Enabled          bool    `json:"enabled"`
	CreditLimit      float64 `json:"creditLimit"`
	CreditsUsed      float64 `json:"creditsUsed"`
	CreditsRemaining float64 `json:"creditsRemaining"`
	ExpiresAt        int64   `json:"expiresAt"`
}

// FetchRemoteKiroKeyCredit calls the peer's check-key endpoint with the account's
// sk and returns the parsed credit view. Requires account.RemoteCheckKeyURL to be
// set; the URL is SSRF-validated (host reuse of the base URL rules) before use.
func FetchRemoteKiroKeyCredit(account *config.Account) (*remoteCheckKeyResponse, error) {
	if account == nil {
		return nil, fmt.Errorf("remotekiro: account is nil")
	}
	checkURL := strings.TrimSpace(account.RemoteCheckKeyURL)
	if checkURL == "" {
		return nil, fmt.Errorf("remotekiro: no check-key URL configured")
	}
	if err := validateRemoteCheckKeyURL(checkURL); err != nil {
		return nil, err
	}
	bearer := strings.TrimSpace(account.AccessToken)
	if bearer == "" {
		return nil, fmt.Errorf("remotekiro: no API key configured")
	}

	reqBody, _ := json.Marshal(map[string]string{"key": bearer})
	req, err := http.NewRequest(http.MethodPost, checkURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", remoteKiroUserAgent)

	resp, err := GetRestClientForProxy(ResolveAccountProxyURL(account)).Do(req)
	if err != nil {
		return nil, fmt.Errorf("remotekiro: check-key request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, newUpstreamError("remotekiro", resp.StatusCode, string(body), fmt.Sprintf("check-key HTTP %d", resp.StatusCode))
	}

	var parsed remoteCheckKeyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("remotekiro: parse check-key: %w", err)
	}
	return &parsed, nil
}

// refreshRemoteKiroInfo mirrors the remote key's credit balance into AccountInfo.
// When RemoteCheckKeyURL is unset, it returns an empty info (no credit sync) so
// the account still refreshes without error. Credits map onto Usage* fields so
// the pool's over-quota skip (UsageCurrent >= UsageLimit) applies automatically
// once the remote key runs out.
func refreshRemoteKiroInfo(account *config.Account, info *config.AccountInfo) (*config.AccountInfo, error) {
	if strings.TrimSpace(account.RemoteCheckKeyURL) == "" {
		return info, nil
	}
	cred, err := FetchRemoteKiroKeyCredit(account)
	if err != nil {
		return nil, err
	}
	if cred.CreditLimit > 0 {
		info.UsageLimit = cred.CreditLimit
		info.UsageCurrent = cred.CreditsUsed
		info.UsagePercent = cred.CreditsUsed / cred.CreditLimit
	} else {
		// Unlimited remote key: clear any prior limit so the account is never
		// treated as over-quota.
		info.UsageLimit = 0
		info.UsageCurrent = cred.CreditsUsed
		info.UsagePercent = 0
	}
	return info, nil
}

// remoteModelInfos builds ModelInfo entries from bare model ids for admin UI / cache merge.
func remoteModelInfos(ids []string) []ModelInfo {
	out := make([]ModelInfo, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, ModelInfo{
			ModelId:   id,
			ModelName: id,
		})
	}
	return out
}
