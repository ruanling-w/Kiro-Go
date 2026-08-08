package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/config"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	grokCLIResponsesURL = "https://cli-chat-proxy.grok.com/responses"
	grokCLIIdentifier   = "grok-cli"
	grokCLIVersion      = "1.0.0"
)

func CallGrokCLIAPI(ctx context.Context, account *config.Account, payload *KiroPayload, callback *KiroStreamCallback) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if callback == nil {
		callback = &KiroStreamCallback{}
	}
	if account == nil || payload == nil {
		return fmt.Errorf("grok cli: account and payload are required")
	}
	bearer := strings.TrimSpace(account.AccessToken)
	if bearer == "" {
		return fmt.Errorf("grok cli: no OAuth access token")
	}
	model := resolvePayloadModelForGrok(payload)
	sessionID := grokCLISessionID(account, payload)
	body, err := BuildGrokCLIResponsesRequest(payload.SourceClaude, payload.SourceOpenAI, model, sessionID, payload.SourceThinking)
	if err != nil {
		return err
	}
	input, _ := body["input"].([]map[string]interface{})
	turn := resolveGrokCLITurn(sessionID, input)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("grok cli: marshal request: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	template, err := http.NewRequestWithContext(ctx, http.MethodPost, grokCLIResponsesURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	template.Header.Set("Content-Type", "application/json")
	template.Header.Set("Accept", "text/event-stream")
	template.Header.Set("Authorization", "Bearer "+bearer)
	template.Header.Set("User-Agent", fmt.Sprintf("%s/%s (%s/%s)", grokCLIIdentifier, grokCLIVersion, runtime.GOOS, runtime.GOARCH))
	template.Header.Set("x-grok-client-identifier", grokCLIIdentifier)
	template.Header.Set("x-grok-client-version", grokCLIVersion)
	template.Header.Set("x-grok-session-id", sessionID)
	template.Header.Set("x-grok-conv-id", sessionID)
	template.Header.Set("x-grok-req-id", uuid.New().String())
	template.Header.Set("x-grok-turn-idx", strconv.Itoa(turn))
	template.Header.Set("x-grok-model-override", body["model"].(string))
	if account.Email != "" {
		template.Header.Set("x-email", account.Email)
	}
	if account.UserId != "" {
		template.Header.Set("x-userid", account.UserId)
	}
	return doGrokCLICallWithRetry(ctx, GetClientForProxy(ResolveAccountProxyURL(account)), template, cancel, callback, model)
}

func doGrokCLICallWithRetry(ctx context.Context, client *http.Client, template *http.Request, cancel context.CancelFunc, callback *KiroStreamCallback, model string) error {
	var lastErr error
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < grokMaxRetries; attempt++ {
		req, err := cloneGrokRequest(ctx, template)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("grok cli: request failed: %w", err)
		} else if resp.StatusCode == http.StatusOK {
			reader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
			err = parseResponsesSSE(reader, callback, model, false)
			reader.Stop()
			resp.Body.Close()
			return err
		} else {
			status := resp.StatusCode
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if status == http.StatusPaymentRequired {
				return fmt.Errorf("grok cli: spending limit or subscription credits exhausted: %s", strings.TrimSpace(string(data)))
			}
			lastErr = newUpstreamError("grok cli", status, string(data), "")
			if !isRetryableGrokStatus(status) {
				return lastErr
			}
		}
		if attempt == grokMaxRetries-1 {
			break
		}
		if err := waitForGrokRetry(ctx, backoff); err != nil {
			return fmt.Errorf("grok cli: retry canceled: %w", err)
		}
		backoff *= 2
	}
	return lastErr
}
