package proxy

import (
	"context"
	"kiro-go/config"
	"strings"
)

// CallProvider dispatches a generation request to the upstream provider that
// owns the selected account. AWS Kiro/CodeWhisperer/AmazonQ accounts go through
// CallKiroAPI; Antigravity (Google Cloud Code / Gemini) accounts go through
// CallAntigravityAPI; Grok/xAI accounts go through CallGrokAPI; Remote Kiro-Go
// peers go through CallRemoteKiroAPI (OpenAI chat.completions passthrough).
// All share the provider-neutral KiroStreamCallback so all SSE-emitting logic
// in the handlers stays unchanged.
//
// The dispatch keys on account fields (Provider / AuthMethod / Grok* fields)
// rather than the requested model, because a single account is bound to exactly
// one upstream and model->account routing has already happened in the pool.
//
// ctx is the caller's request context. Every provider derives its cancelable
// upstream context from it, so a client that disconnects mid-stream tears down
// the upstream request instead of leaving it running to completion — without
// this, abandoned generations keep consuming account quota and connections.
func CallProvider(ctx context.Context, account *config.Account, payload *KiroPayload, callback *KiroStreamCallback) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if account != nil && isAntigravityAccount(account) {
		return CallAntigravityAPI(ctx, account, payload, callback)
	}
	if account != nil && isCodexAccount(account) {
		return CallCodexAPI(ctx, account, payload, callback)
	}
	if account != nil && isGrokOAuthAccount(account) {
		return CallGrokCLIAPI(ctx, account, payload, callback)
	}
	if account != nil && isGrokAccount(account) {
		return CallGrokAPI(ctx, account, payload, callback)
	}
	if account != nil && isRemoteKiroAccount(account) {
		return CallRemoteKiroAPI(ctx, account, payload, callback)
	}
	return CallKiroAPI(ctx, account, payload, callback)
}

// isRemoteKiroAccount reports whether an account proxies to another Kiro-Go
// (or OpenAI-compatible) peer via RemoteBaseURL + static sk AccessToken.
func isRemoteKiroAccount(account *config.Account) bool {
	if account == nil {
		return false
	}
	if strings.EqualFold(account.Provider, "remotekiro") ||
		strings.EqualFold(account.AuthMethod, "remotekiro") {
		return true
	}
	// Fallback: a configured remote base URL without other provider markers.
	return strings.TrimSpace(account.RemoteBaseURL) != ""
}

// isCodexAccount reports whether an account should be routed to the OpenAI Codex
// (ChatGPT) upstream. Both sign-in modes (OAuth and pasted access token) dispatch
// through CallCodexAPI.
func isCodexAccount(account *config.Account) bool {
	if account == nil {
		return false
	}
	return strings.EqualFold(account.Provider, "codex") ||
		strings.EqualFold(account.AuthMethod, "codex")
}

// isAntigravityAccount reports whether an account is served by the Antigravity
// (Google Cloud Code / Gemini) upstream.
func isAntigravityAccount(account *config.Account) bool {
	if account == nil {
		return false
	}
	return strings.EqualFold(account.AuthMethod, "antigravity") ||
		strings.EqualFold(account.Provider, "Antigravity")
}

// isGrokAccount reports whether an account belongs to Grok/xAI. Official API
// keys use api.x.ai; CallProvider checks Grok Build OAuth first and routes it to
// the distinct cli-chat-proxy transport.
func isGrokAccount(account *config.Account) bool {
	if account == nil {
		return false
	}
	if strings.EqualFold(account.Provider, "grok") ||
		strings.EqualFold(account.Provider, "xai") ||
		strings.EqualFold(account.AuthMethod, "grok") {
		return true
	}
	// Also treat accounts that carry an explicit Grok API key.
	if account.GrokAPIKey != "" {
		return true
	}
	return false
}

// isGrokOAuthAccount reports whether a Grok account authenticates via the Grok
// Build OAuth flow (Bearer access_token refreshed against xAI) rather than a
// static API key.
func isGrokOAuthAccount(account *config.Account) bool {
	if account == nil {
		return false
	}
	if strings.EqualFold(account.GrokAuthType, "oauth") {
		return true
	}
	// Fall back to credential shape: an OAuth account has tokens but no API key.
	if account.GrokAuthType == "" && account.GrokAPIKey == "" && account.RefreshToken != "" {
		return true
	}
	return false
}
