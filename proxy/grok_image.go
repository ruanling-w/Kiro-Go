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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kiro-go/config"
	"kiro-go/logger"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"time"
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

	bearer := grokBearer(account)
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

// fetchImageAsBase64 downloads and validates an image from a public HTTPS URL.
func fetchImageAsBase64(ctx context.Context, _ *http.Client, rawURL string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return "", errors.New("unsafe image URL")
	}
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	if port != "443" {
		return "", errors.New("unsafe image URL port")
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return "", errors.New("image URL host could not be resolved")
	}
	pinned := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if !publicImageAddress(address) {
			return "", errors.New("image URL resolved to a non-public address")
		}
		pinned = append(pinned, address)
	}
	var dialer net.Dialer
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12, ServerName: parsed.Hostname()},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DialContext: func(dialCtx context.Context, network, _ string) (net.Conn, error) {
			var errs []error
			for _, address := range pinned {
				conn, dialErr := dialer.DialContext(dialCtx, network, net.JoinHostPort(address.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				errs = append(errs, dialErr)
			}
			return nil, errors.Join(errs...)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
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
	if resp.ContentLength > chatAttachmentMaxBytes {
		return "", errInvalidChatImage
	}
	temp, err := os.CreateTemp("", "kiro-grok-image-*")
	if err != nil {
		return "", err
	}
	path := temp.Name()
	defer os.Remove(path)
	written, copyErr := io.Copy(temp, io.LimitReader(resp.Body, chatAttachmentMaxBytes+1))
	closeErr := temp.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written == 0 || written > chatAttachmentMaxBytes {
		return "", errInvalidChatImage
	}
	if _, err = validateStoredChatImage(path, written); err != nil {
		return "", errInvalidChatImage
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func publicImageAddress(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	blocked := []netip.Prefix{
		netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001:db8::/32"),
	}
	for _, prefix := range blocked {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
