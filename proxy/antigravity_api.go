package proxy

// antigravity_api.go performs the upstream call to the Antigravity (Google Cloud
// Code / Gemini) endpoint and parses its SSE response, translating events onto the
// provider-neutral KiroStreamCallback so all client-facing SSE emission in the
// handlers is reused unchanged.
//
// Wire flow (port of 9router open-sse/executors/antigravity.js):
//   POST https://daily-cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse
//   body: {project, model, userAgent, requestType, requestId, request:{...}}
//   resp: SSE lines `data: {"response":{"candidates":[...],"usageMetadata":{...}}}`

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/auth"
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// agBaseURLs are the Antigravity Cloud Code hosts, tried in order. This must match
// the production host used by auth/antigravity.go (fetchAvailableModels) and 9router's
// antigravity provider. The internal "daily-" / ".sandbox" hosts return HTTP 404
// "Requested entity was not found" for normal accounts and must not be used.
var agBaseURLs = []string{
	"https://cloudcode-pa.googleapis.com",
}

// agUserAgentVersion mirrors the real Antigravity binary version string.
const agUserAgentVersion = "1.107.0"


// CallAntigravityAPI translates the stashed source request into a Gemini envelope
// and streams the Antigravity response back through the callback.
// ctx is the caller's request context; each host attempt derives its cancelable
// upstream context from it so a client disconnect tears down the generation.
func CallAntigravityAPI(ctx context.Context, account *config.Account, payload *KiroPayload, callback *KiroStreamCallback) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if callback == nil {
		callback = &KiroStreamCallback{}
	}
	if account == nil {
		return fmt.Errorf("antigravity: account is nil")
	}
	if payload == nil {
		return fmt.Errorf("antigravity: payload is nil")
	}

	model := resolvePayloadModelID(payload)

	// Build the Gemini inner request from the pristine source request. The Kiro
	// wire fields on the payload are ignored on this path.
	var inner *GeminiInner
	var err error
	switch {
	case payload.SourceClaude != nil:
		inner, err = ClaudeToGemini(payload.SourceClaude, payload.SourceThinking)
	case payload.SourceOpenAI != nil:
		inner, err = OpenAIToGemini(payload.SourceOpenAI, payload.SourceThinking)
	default:
		return fmt.Errorf("antigravity: no source request on payload")
	}
	if err != nil {
		return fmt.Errorf("antigravity: translate request: %w", err)
	}

	// Anti-ban cloaking + Gemini 3+ thoughtSignature backfill.
	nameMap := cloakGeminiTools(inner)
	backfillThoughtSignatures(inner)

	sessionID := strings.TrimSpace(account.AGSessionID)
	if sessionID == "" {
		sessionID = uuid.New().String() + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	inner.SessionID = sessionID

	projectID := strings.TrimSpace(account.AGProjectID)
	if projectID == "" {
		projectID = generateAntigravityProjectID()
	}

	envelope := &GeminiEnvelope{
		Project:     projectID,
		Model:       model,
		UserAgent:   "antigravity",
		RequestType: "agent",
		RequestID:   "agent-" + uuid.New().String(),
		Request:     inner,
	}

	// Restore original (un-suffixed) tool names before handing tool_use to the
	// client, mirroring CallKiroAPI's OnToolUse wrapping.
	if callback.OnToolUse != nil && len(nameMap) > 0 {
		original := callback.OnToolUse
		wrapped := *callback
		wrapped.OnToolUse = func(tu KiroToolUse) {
			if orig, ok := nameMap[tu.Name]; ok {
				tu.Name = orig
			}
			original(tu)
		}
		callback = &wrapped
	}

	reqBody, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("antigravity: marshal request: %w", err)
	}
	if logger.GetLevel() == logger.LevelDebug {
		logger.Debugf("[Antigravity] Request payload: %s", string(reqBody))
	}

	client := GetClientForProxy(ResolveAccountProxyURL(account))
	userAgent := fmt.Sprintf("antigravity/%s %s/%s", agUserAgentVersion, runtime.GOOS, runtime.GOARCH)

	var lastErr error
	for _, base := range agBaseURLs {
		url := base + "/v1internal:streamGenerateContent?alt=sse"

		// done reports whether this host produced a final outcome (success or a
		// terminal error). Each host gets its own cancelable context so a stalled
		// stream on one host cannot outlive the attempt; cancel must not be
		// deferred to function exit inside a loop.
		streamErr, done := func() (error, bool) {
			ctx, cancel := context.WithCancel(ctx)
			cancelled := false
			defer func() {
				if !cancelled {
					cancel()
				}
			}()

			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqBody)))
			if reqErr != nil {
				lastErr = reqErr
				return nil, false
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+account.AccessToken)
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("x-request-source", "local")
			req.Header.Set("X-Machine-Session-Id", sessionID)
			req.Header.Set("Accept", "text/event-stream")

			resp, doErr := client.Do(req)
			if doErr != nil {
				lastErr = doErr
				logger.Warnf("[Antigravity] host %s failed: %v", base, doErr)
				return nil, false
			}

			if resp.StatusCode != 200 {
				errBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = newUpstreamError("antigravity", resp.StatusCode, string(errBody), fmt.Sprintf("HTTP %d from %s", resp.StatusCode, base))
				// Auth / payment errors are terminal — do not try the fallback host.
				if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 402 {
					return lastErr, true
				}
				logger.Warnf("[Antigravity] host %s error: %v", base, lastErr)
				return nil, false
			}

			// A stalled stream must not block this goroutine forever: the client
			// used here has Timeout: 0, and ResponseHeaderTimeout only bounds the
			// wait for the first header. The idle reader cancels the request when
			// no bytes arrive for streamIdleTimeout.
			idleReader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
			parseErr := parseGeminiSSE(idleReader, callback)
			idleReader.Stop()
			resp.Body.Close()
			cancel()
			cancelled = true
			return parseErr, true
		}()
		if done {
			return streamErr
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("antigravity: all hosts failed")
}

// resolvePayloadModelID returns the model id the request targets.
func resolvePayloadModelID(payload *KiroPayload) string {
	return payload.ConversationState.CurrentMessage.UserInputMessage.ModelID
}

// generateAntigravityProjectID delegates to auth so login + chat share one shape.
func generateAntigravityProjectID() string {
	return auth.GenerateAntigravityProjectID()
}

// ==================== Image generation ====================

// agImageAspectRatios maps common OpenAI image sizes to Gemini aspect ratios.
// Mirrors 9router's sizeToAspectRatio (imageProviders/_base.js).
var agImageAspectRatios = map[string]string{
	"1024x1024": "1:1",
	"1024x1792": "9:16",
	"1792x1024": "16:9",
	"1024x1536": "2:3",
	"1536x1024": "3:2",
}

// sizeToAspectRatio converts an OpenAI size string to a Gemini aspect ratio,
// defaulting to 1:1 for unknown/empty sizes.
func sizeToAspectRatio(size string) string {
	if ar, ok := agImageAspectRatios[strings.TrimSpace(size)]; ok {
		return ar
	}
	return "1:1"
}

// parseImageInputToInline converts a data URI or raw base64 string into a Gemini
// inlineData part for image editing. Returns nil for unusable input.
func parseImageInputToInline(input string) *GeminiInlineData {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	// data:image/png;base64,....
	if strings.HasPrefix(input, "data:") {
		semi := strings.Index(input, ";base64,")
		if semi > 5 {
			mime := input[5:semi]
			data := input[semi+len(";base64,"):]
			if mime != "" && data != "" {
				return &GeminiInlineData{MimeType: mime, Data: data}
			}
		}
		return nil
	}
	// Raw base64 (assume PNG); guard against obvious URLs.
	if !strings.HasPrefix(input, "http") && len(input) > 100 {
		return &GeminiInlineData{MimeType: "image/png", Data: input}
	}
	return nil
}

// CallAntigravityImageAPI generates an image via the Antigravity (Gemini) Cloud
// Code endpoint and returns the first result as base64. Unlike the chat path this
// uses the NON-streaming generateContent endpoint with requestType "image_gen"
// (mirrors 9router executors/antigravity.js image branch). Returns (b64, mime, err).
func CallAntigravityImageAPI(ctx context.Context, account *config.Account, req *CodexImageRequest) (b64 string, mimeType string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if account == nil {
		return "", "", fmt.Errorf("antigravity image: account is nil")
	}
	if req == nil {
		return "", "", fmt.Errorf("antigravity image: request is nil")
	}

	// Build parts: text prompt + optional reference image (for edits).
	parts := []GeminiPart{}
	imageInput := strings.TrimSpace(req.Image)
	if imageInput == "" && len(req.Images) > 0 {
		imageInput = strings.TrimSpace(req.Images[0])
	}
	if inline := parseImageInputToInline(imageInput); inline != nil {
		parts = append(parts, GeminiPart{InlineData: inline})
	}
	parts = append(parts, GeminiPart{Text: req.Prompt})

	sessionID := strings.TrimSpace(account.AGSessionID)
	if sessionID == "" {
		sessionID = uuid.New().String() + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	projectID := strings.TrimSpace(account.AGProjectID)
	if projectID == "" {
		projectID = generateAntigravityProjectID()
	}

	// image_gen needs an imageConfig on generationConfig, which the shared
	// GeminiGenConfig struct doesn't carry — build the request as a raw map.
	inner := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"role": "user", "parts": parts},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     1.0,
			"topP":            0.95,
			"topK":            40,
			"maxOutputTokens": 8192,
			"imageConfig":     map[string]interface{}{"aspectRatio": sizeToAspectRatio(req.Size)},
		},
		"sessionId": sessionID,
	}
	envelope := map[string]interface{}{
		"project":     projectID,
		"model":       req.Model,
		"userAgent":   "antigravity",
		"requestType": "image_gen",
		"requestId":   "image-" + uuid.New().String(),
		"request":     inner,
	}

	reqBody, err := json.Marshal(envelope)
	if err != nil {
		return "", "", fmt.Errorf("antigravity image: marshal request: %w", err)
	}
	if logger.GetLevel() == logger.LevelDebug {
		logger.Debugf("[AntigravityImage] Request payload: %s", string(reqBody))
	}

	client := GetClientForProxy(ResolveAccountProxyURL(account))
	userAgent := fmt.Sprintf("antigravity/%s %s/%s", agUserAgentVersion, runtime.GOOS, runtime.GOARCH)

	var lastErr error
	for _, base := range agBaseURLs {
		// Image generation uses the non-streaming generateContent action. The body
		// is still read off a Timeout: 0 client, so it gets the same idle watchdog
		// as the streaming path — io.ReadAll on a stalled connection would
		// otherwise block forever.
		imageURL := base + "/v1internal:generateContent"
		respBody, status, readErr := func() ([]byte, int, error) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, imageURL, strings.NewReader(string(reqBody)))
			if reqErr != nil {
				return nil, 0, reqErr
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", "Bearer "+account.AccessToken)
			httpReq.Header.Set("User-Agent", userAgent)
			httpReq.Header.Set("x-request-source", "local")
			httpReq.Header.Set("X-Machine-Session-Id", sessionID)
			httpReq.Header.Set("Accept", "application/json")

			resp, doErr := client.Do(httpReq)
			if doErr != nil {
				return nil, 0, doErr
			}
			idleReader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
			body, _ := io.ReadAll(idleReader)
			idleReader.Stop()
			resp.Body.Close()
			return body, resp.StatusCode, nil
		}()
		if readErr != nil {
			lastErr = readErr
			logger.Warnf("[AntigravityImage] host %s failed: %v", base, readErr)
			continue
		}

		if status != 200 {
			lastErr = newUpstreamError("antigravity image", status, string(respBody), fmt.Sprintf("HTTP %d from %s", status, base))
			if status == 401 || status == 403 || status == 402 {
				return "", "", lastErr
			}
			logger.Warnf("[AntigravityImage] host %s error: %v", base, lastErr)
			continue
		}

		var parsed geminiSSEResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", "", fmt.Errorf("antigravity image: decode response: %w", err)
		}
		candidates := parsed.Candidates
		if parsed.Response != nil {
			candidates = parsed.Response.Candidates
		}
		for _, c := range candidates {
			for _, part := range c.Content.Parts {
				if part.InlineData != nil && part.InlineData.Data != "" {
					mime := part.InlineData.MimeType
					if mime == "" {
						mime = "image/png"
					}
					return part.InlineData.Data, mime, nil
				}
			}
		}
		return "", "", fmt.Errorf("antigravity image: no image in response")
	}

	if lastErr != nil {
		return "", "", lastErr
	}
	return "", "", fmt.Errorf("antigravity image: all hosts failed")
}

// ==================== SSE parsing ====================

// geminiSSEResponse is the shape of each Antigravity SSE data line.
type geminiSSEResponse struct {
	Response *geminiCandidatesResponse `json:"response"`
	// Some responses are unwrapped (no "response" key); the fields below let the
	// same struct decode either shape.
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata *geminiUsageMeta    `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
	ResponseID    string              `json:"responseId"`
}

type geminiCandidatesResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsageMeta  `json:"usageMetadata"`
	ModelVersion  string            `json:"modelVersion"`
	ResponseID    string            `json:"responseId"`
}

type geminiCandidate struct {
	Content      geminiRespContent `json:"content"`
	FinishReason string            `json:"finishReason"`
}

type geminiRespContent struct {
	Role  string           `json:"role"`
	Parts []geminiRespPart `json:"parts"`
}

type geminiRespPart struct {
	Text             string              `json:"text"`
	Thought          bool                `json:"thought"`
	ThoughtSignature string              `json:"thoughtSignature"`
	FunctionCall     *GeminiFunctionCall `json:"functionCall"`
	InlineData       *geminiRespInline   `json:"inlineData"`
}

type geminiRespInline struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiUsageMeta struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// parseGeminiSSE reads the Antigravity SSE stream and dispatches events onto the
// callback. It is JSON/SSE line-based — NOT the AWS binary event-stream format, so
// it must never be routed through parseEventStream.
func parseGeminiSSE(body io.Reader, callback *KiroStreamCallback) error {
	if callback == nil {
		callback = &KiroStreamCallback{}
	}

	scanner := bufio.NewScanner(body)
	// Allow long SSE data lines (default 64KB is too small for large tool args).
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var inputTokens, outputTokens int
	toolIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var parsed geminiSSEResponse
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			logger.Debugf("[Antigravity] skip unparseable SSE line: %v", err)
			continue
		}

		candidates := parsed.Candidates
		usage := parsed.UsageMetadata
		if parsed.Response != nil {
			candidates = parsed.Response.Candidates
			if parsed.Response.UsageMetadata != nil {
				usage = parsed.Response.UsageMetadata
			}
		}

		if usage != nil {
			if usage.PromptTokenCount > 0 {
				inputTokens = usage.PromptTokenCount
			}
			if usage.CandidatesTokenCount > 0 {
				outputTokens = usage.CandidatesTokenCount
			}
		}

		if len(candidates) == 0 {
			continue
		}
		candidate := candidates[0]

		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				if callback.OnToolUse != nil {
					callback.OnToolUse(KiroToolUse{
						ToolUseID: fmt.Sprintf("%s-%d", part.FunctionCall.Name, toolIndex),
						Name:      part.FunctionCall.Name,
						Input:     part.FunctionCall.Args,
					})
				}
				toolIndex++
				continue
			}
			if part.Text != "" && callback.OnText != nil {
				callback.OnText(part.Text, part.Thought)
			}
			if part.InlineData != nil && part.InlineData.Data != "" && callback.OnImage != nil {
				mime := part.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				callback.OnImage(part.InlineData.Data, mime, false)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("antigravity: read stream: %w", err)
	}

	if callback.OnComplete != nil {
		callback.OnComplete(inputTokens, outputTokens)
	}
	return nil
}
