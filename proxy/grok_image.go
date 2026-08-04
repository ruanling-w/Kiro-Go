package proxy

// grok_image.go implements image generation for Grok/xAI accounts.
//
// Unlike chat completions, xAI serves images from a dedicated endpoint:
//   POST https://api.x.ai/v1/images/generations
//   Authorization: Bearer <token>
//   body: {model, prompt, n, response_format:"b64_json"}
//   resp: {data:[{b64_json|url}]}
//
// Mirrors 9router's OpenAI-compatible image adapter for xai
// (open-sse/handlers/imageProviders/openai.js + registry/xai.js imageConfig).

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"strings"
)

const grokImageURL = "https://api.x.ai/v1/images/generations"

// grokImageResponse is the xAI images-generation response shape.
type grokImageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CallGrokImageAPI generates an image via the xAI images endpoint and returns the
// first result as base64. Returns (b64, mimeType, error). URLs (if the account is
// configured to return them) are fetched and re-encoded to base64 so the caller
// always gets inline data.
//
// ctx is the caller's request context; the upstream call is derived from it so a
// client disconnect cancels the generation.
func CallGrokImageAPI(ctx context.Context, account *config.Account, req *CodexImageRequest) (b64 string, mimeType string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if account == nil {
		return "", "", fmt.Errorf("grok image: account is nil")
	}
	if req == nil {
		return "", "", fmt.Errorf("grok image: request is nil")
	}

	bearer := strings.TrimSpace(account.AccessToken)
	if bearer == "" {
		bearer = strings.TrimSpace(account.GrokAPIKey)
	}
	if bearer == "" {
		return "", "", fmt.Errorf("grok image: no credentials configured")
	}

	n := req.N
	if n <= 0 {
		n = 1
	}
	// xAI only accepts model/prompt/n/response_format (see registry/xai.js bodyFields).
	body := map[string]interface{}{
		"model":           req.Model,
		"prompt":          req.Prompt,
		"n":               n,
		"response_format": "b64_json",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("grok image: marshal request: %w", err)
	}

	client := GetClientForProxy(ResolveAccountProxyURL(account))

	// This client has Timeout: 0 (shared with the streaming paths), so the body
	// read needs its own idle watchdog — otherwise a stalled connection blocks
	// io.ReadAll indefinitely.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, grokImageURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+bearer)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("grok image: request failed: %w", err)
	}
	defer resp.Body.Close()

	idleReader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
	respBody, _ := io.ReadAll(idleReader)
	idleReader.Stop()
	if resp.StatusCode != http.StatusOK {
		return "", "", newUpstreamError("grok image", resp.StatusCode, string(respBody), "")
	}

	var parsed grokImageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", fmt.Errorf("grok image: decode response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", "", fmt.Errorf("grok image: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 {
		return "", "", fmt.Errorf("grok image: no image returned")
	}

	first := parsed.Data[0]
	if first.B64JSON != "" {
		return first.B64JSON, "image/png", nil
	}
	if first.URL != "" {
		fetched, fErr := fetchImageAsBase64(ctx, client, first.URL)
		if fErr != nil {
			return "", "", fmt.Errorf("grok image: fetch result url: %w", fErr)
		}
		if logger.GetLevel() == logger.LevelDebug {
			logger.Debugf("[GrokImage] fetched result from url")
		}
		return fetched, "image/png", nil
	}
	return "", "", fmt.Errorf("grok image: response had neither b64_json nor url")
}

// fetchImageAsBase64 downloads an image URL and returns its base64 encoding.
// client may be a Timeout: 0 streaming client, so the download is bounded by the
// same idle watchdog used on the SSE paths.
func fetchImageAsBase64(ctx context.Context, client *http.Client, url string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", newUpstreamError("", resp.StatusCode, "", fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
	idleReader := newIdleTimeoutReader(resp.Body, streamIdleTimeout, cancel)
	defer idleReader.Stop()
	data, err := io.ReadAll(idleReader)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
