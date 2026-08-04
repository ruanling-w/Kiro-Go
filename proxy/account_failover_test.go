package proxy

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"kiro-go/config"
	accountpool "kiro-go/pool"
)

func TestAccountFailureClassifiers(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) bool
		msg  string
	}{
		{name: "quota", fn: isQuotaErrorMessage, msg: "HTTP 429: quota exhausted"},
		{name: "overage", fn: isOverageErrorMessage, msg: "HTTP 402 from Kiro IDE: OVERAGE limit exceeded"},
		{name: "suspension", fn: isSuspensionErrorMessage, msg: "Your User ID temporarily is suspended"},
		{name: "profile", fn: isProfileUnavailableErrorMessage, msg: "no available Kiro profile"},
		{name: "auth", fn: isAuthErrorMessage, msg: "Authentication failed - token invalid or expired"},
	}

	for _, tc := range tests {
		if !tc.fn(tc.msg) {
			t.Fatalf("%s classifier did not match %q", tc.name, tc.msg)
		}
	}
}

func TestClassifyAccountFailure(t *testing.T) {
	acc := &config.Account{ID: "a1", AuthMethod: "social"}
	ksk := &config.Account{ID: "k1", AuthMethod: "api_key"}

	cases := []struct {
		name    string
		acc     *config.Account
		err     error
		refresh bool
		want    string
	}{
		{"quota", acc, errors.New("HTTP 429 from kiro: quota"), false, EventQuota},
		{"overage", acc, errors.New("HTTP 402 from kiro: overage limit"), false, EventOverage},
		{"suspend", acc, errors.New("account temporarily_suspended"), false, EventBan},
		{"profile", acc, errors.New("no available Kiro profile"), false, EventSoft},
		{"auth request", acc, errors.New("HTTP 403 from kiro: forbidden"), false, EventBan},
		{"auth refresh", acc, errors.New("HTTP 401 from oidc: unauthorized"), true, EventTokenRefresh},
		{"default soft", acc, errors.New("connection reset"), false, EventSoft},
		{"default refresh", acc, errors.New("connection reset"), true, EventTokenRefresh},
		{"ksk soft", ksk, errors.New("HTTP 403 from kiro"), false, EventSoft},
	}
	for _, tc := range cases {
		got := classifyAccountFailure(tc.acc, tc.err, tc.refresh)
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// A typed *UpstreamError classifies by HTTP status, so a body that merely
// mentions "429" or "forbidden" no longer cools down or bans the account. This
// was the pool-draining bug: two such failures emptied a small pool and every
// subsequent request returned 503 "No available accounts".
func TestClassifyFailureKindUsesStatusNotBody(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want accountFailureKind
	}{
		{
			"500 body mentioning 429",
			newUpstreamError("kiro", 500, `{"message":"internal error","requestId":"req-429-abc"}`, ""),
			failureSoft,
		},
		{
			"502 body mentioning forbidden",
			newUpstreamError("grok", 502, "upstream says: forbidden by origin", ""),
			failureSoft,
		},
		{
			"400 body mentioning quota",
			newUpstreamError("codex", 400, "invalid request: quota field is not supported", ""),
			failureSoft,
		},
		{"real 429", newUpstreamError("kiro", 429, "slow down", ""), failureQuota},
		{"real 401", newUpstreamError("kiro", 401, "bad token", ""), failureAuth},
		{"real 403", newUpstreamError("kiro", 403, "nope", ""), failureAuth},
		{"real 402 overage", newUpstreamError("kiro", 402, "OVERAGE limit exceeded", ""), failureOverage},
		{
			"402 without overage marker is soft",
			newUpstreamError("kiro", 402, "payment required", ""),
			failureSoft,
		},
		// Body markers that ride along with a 401/403 must win over the status,
		// or a suspended account is reported as bad credentials.
		{
			"suspension beats 403 status",
			newUpstreamError("kiro", 403, "Your User ID temporarily is suspended", ""),
			failureSuspended,
		},
		{
			"profile unavailable beats 403 status",
			newUpstreamError("kiro", 403, "no available Kiro profile", ""),
			failureProfileUnavailable,
		},
		// Untyped errors (token refresh, transport) still fall back to substrings.
		{"untyped auth", errors.New("HTTP 403 from oidc: forbidden"), failureAuth},
		{"untyped quota", errors.New("HTTP 429: quota exhausted"), failureQuota},
		{"untyped transport", errors.New("connection reset by peer"), failureSoft},
		{"nil", nil, failureSoft},
	}
	for _, tc := range cases {
		if got := classifyFailureKind(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

// Background maintenance loops (model-catalog refresh, periodic account-info
// refresh) sweep every enabled account on a timer. If they were allowed to ban,
// a single upstream blip during a sweep would disable the whole pool with no
// client traffic at all, and the next request would get 503 "No available
// accounts". handleBackgroundFailure must therefore only ever soft-cool.
func TestHandleBackgroundFailureNeverDisablesAccount(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	// Every failure that the request path would treat as fatal.
	fatal := []struct {
		id  string
		err error
	}{
		{"auth-typed", newUpstreamError("kiro", 403, "nope", "")},
		{"auth-untyped", errors.New("HTTP 401 from oidc: unauthorized")},
		{"suspended", errors.New("Your User ID temporarily is suspended")},
		{"overage", errors.New("HTTP 402 from kiro: OVERAGE limit exceeded")},
	}
	for _, tc := range fatal {
		if err := config.AddAccount(config.Account{
			ID:           tc.id,
			Email:        tc.id + "@example.com",
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			AuthMethod:   "social",
			Provider:     "BuilderId",
			ExpiresAt:    time.Now().Unix() + 3600,
			Enabled:      true,
		}); err != nil {
			t.Fatalf("AddAccount(%s): %v", tc.id, err)
		}
	}

	p := accountpool.GetPool()
	p.Reload()
	h := &Handler{pool: p}

	accounts := config.GetAccounts()
	for _, tc := range fatal {
		var account *config.Account
		for i := range accounts {
			if accounts[i].ID == tc.id {
				account = &accounts[i]
				break
			}
		}
		if account == nil {
			t.Fatalf("account %s not found after AddAccount", tc.id)
		}
		h.handleBackgroundFailure(account, tc.err, "ModelsCache")
	}

	for _, a := range config.GetAccounts() {
		if !a.Enabled {
			t.Fatalf("background failure disabled account %s (ban=%q reason=%q); background sweeps must soft-cool only", a.ID, a.BanStatus, a.BanReason)
		}
	}
}
