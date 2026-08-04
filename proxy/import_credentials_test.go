package proxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/auth"
	"kiro-go/config"
	accountpool "kiro-go/pool"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// installCleanAuthClient replaces the global auth HTTP client with one whose
// Transport does not consult http.ProxyFromEnvironment — that function caches
// env vars on first call and would otherwise poison TestBuildKiroTransport*
// when tests run in the default order. Returns a cleanup that restores the
// previous client.
func installCleanAuthClient(t *testing.T) func() {
	t.Helper()
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}}
	prev := auth.SetGlobalAuthClientForTest(c)
	return func() { auth.SetGlobalAuthClientForTest(prev) }
}

// TestApiImportCredentialsRejectsWhenRefreshFails verifies the regression:
// previously, when auth.RefreshToken failed and the user supplied an accessToken,
// the handler stored that accessToken with ExpiresAt = now+300, producing an
// account that the pool would skip (Pick uses now > ExpiresAt-120 → ~3 min) and
// that the on-demand refresh path could never repair (Pick filters it out before
// ensureValidToken runs). The fix is to reject the import outright; the caller
// must provide a refreshToken that actually works.
func TestApiImportCredentialsRejectsWhenRefreshFails(t *testing.T) {
	cfgFile := t.TempDir() + "/config.json"
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	defer installCleanAuthClient(t)()

	// Stand up a fake OIDC endpoint that always 400s, simulating an unreachable
	// or invalid refresh.
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer fake.Close()

	oldOIDC := authOidcURL()
	auth.SetOIDCTokenURLForTest(func(string) string { return fake.URL })
	defer auth.SetOIDCTokenURLForTest(oldOIDC)

	h := &Handler{pool: accountpool.GetPool()}

	body := `{"refreshToken":"rt-broken","accessToken":"at-still-valid-elsewhere","clientId":"c","clientSecret":"s","authMethod":"idc","region":"us-east-1"}`
	req := httptest.NewRequest("POST", "/auth/credentials", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.apiImportCredentials(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when refresh fails, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(resp["error"], "Token refresh failed") {
		t.Fatalf("expected refresh-failed error, got %q", resp["error"])
	}

	// Crucial: no account should have been created. The previous bug stored a
	// half-broken account with ExpiresAt ~now+300 that would die in 3 minutes.
	if accs := config.GetAccounts(); len(accs) != 0 {
		t.Fatalf("expected no accounts to be persisted on failed import, got %+v", accs)
	}
}

// TestApiImportCredentialsUsesUpstreamExpiresAt verifies the happy path: when
// refresh succeeds, the persisted ExpiresAt reflects the upstream expiresIn,
// not a hard-coded 300s.
func TestApiImportCredentialsUsesUpstreamExpiresAt(t *testing.T) {
	cfgFile := t.TempDir() + "/config.json"
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	defer installCleanAuthClient(t)()

	const upstreamExpiresIn = 3600
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"accessToken":"at-new","refreshToken":"rt-rotated","expiresIn":%d,"profileArn":"arn:aws:codewhisperer:profile/test"}`, upstreamExpiresIn)
	}))
	defer fake.Close()

	oldOIDC := authOidcURL()
	auth.SetOIDCTokenURLForTest(func(string) string { return fake.URL })
	defer auth.SetOIDCTokenURLForTest(oldOIDC)

	h := &Handler{pool: accountpool.GetPool()}

	before := time.Now().Unix()
	body := `{"refreshToken":"rt-good","clientId":"c","clientSecret":"s","authMethod":"idc","region":"us-east-1"}`
	req := httptest.NewRequest("POST", "/auth/credentials", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.apiImportCredentials(rec, req)
	after := time.Now().Unix()

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on successful refresh, got %d body=%s", rec.Code, rec.Body.String())
	}

	accs := config.GetAccounts()
	if len(accs) != 1 {
		t.Fatalf("expected exactly one account persisted, got %d", len(accs))
	}
	got := accs[0]
	if got.AccessToken != "at-new" {
		t.Fatalf("expected upstream-issued accessToken, got %q", got.AccessToken)
	}
	if got.RefreshToken != "rt-rotated" {
		t.Fatalf("expected rotated refreshToken to be persisted, got %q", got.RefreshToken)
	}
	// Allow ±5s of drift but require the value to clearly come from upstream's
	// expiresIn rather than the old 300s fallback.
	expectMin := before + upstreamExpiresIn - 5
	expectMax := after + upstreamExpiresIn + 5
	if got.ExpiresAt < expectMin || got.ExpiresAt > expectMax {
		t.Fatalf("expected ExpiresAt ≈ now+%d ([%d..%d]), got %d", upstreamExpiresIn, expectMin, expectMax, got.ExpiresAt)
	}
	if got.ExpiresAt-time.Now().Unix() < 1500 {
		t.Fatalf("ExpiresAt too short — looks like the 300s fallback is still in play: %d (delta %d)", got.ExpiresAt, got.ExpiresAt-time.Now().Unix())
	}
}

// TestApiImportAccountsPartialFailure covers the batch endpoint (/import, the
// inverse of /export): a bad account must not sink the good ones, and it must be
// reported per-item rather than silently swallowed. It also pins the export-bundle
// un-mapping — nested `credentials`, the prettified "IdC" authMethod, and account
// level `idp` standing in for provider.
func TestApiImportAccountsPartialFailure(t *testing.T) {
	cfgFile := t.TempDir() + "/config.json"
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	defer installCleanAuthClient(t)()

	// The fake OIDC endpoint fails the refresh for exactly one refresh token.
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if strings.Contains(string(raw), "rt-broken") {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"accessToken":"at-new","refreshToken":"rt-rotated","expiresIn":3600}`)
	}))
	defer fake.Close()

	oldOIDC := authOidcURL()
	auth.SetOIDCTokenURLForTest(func(string) string { return fake.URL })
	defer auth.SetOIDCTokenURLForTest(oldOIDC)

	h := &Handler{pool: accountpool.GetPool()}

	body := `{"accounts":[
		{"nickname":"good","idp":"BuilderId","credentials":{"refreshToken":"rt-good","clientId":"c","clientSecret":"s","authMethod":"IdC","region":"us-east-1"}},
		{"nickname":"bad","credentials":{"refreshToken":"rt-broken","clientId":"c","clientSecret":"s","authMethod":"IdC","region":"us-east-1"}},
		{"nickname":"flat","refreshToken":"rt-good2","clientId":"c","clientSecret":"s","authMethod":"idc","region":"us-east-1"}
	]}`
	req := httptest.NewRequest("POST", "/import", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.apiImportAccounts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Success  bool `json:"success"`
		Accounts []struct {
			ID string `json:"id"`
		} `json:"accounts"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Accounts) != 2 {
		t.Fatalf("expected 2 imported accounts, got %d (%s)", len(resp.Accounts), rec.Body.String())
	}
	if len(resp.Errors) != 1 || !strings.Contains(resp.Errors[0], "bad:") {
		t.Fatalf("expected one per-item error labelled \"bad\", got %v", resp.Errors)
	}

	accs := config.GetAccounts()
	if len(accs) != 2 {
		t.Fatalf("expected exactly the 2 good accounts persisted, got %d", len(accs))
	}
	for _, a := range accs {
		// The export bundle's "IdC" must be un-mapped, or refresh dispatches wrong.
		if a.AuthMethod != "idc" {
			t.Fatalf("expected authMethod normalized to idc, got %q", a.AuthMethod)
		}
		if a.RefreshToken != "rt-rotated" {
			t.Fatalf("expected rotated refreshToken persisted, got %q", a.RefreshToken)
		}
	}
}

// TestNormalizeImportAuthMethod pins the mapping table, notably that the Entra
// spellings resolve to external_idp (the old switch coerced them to social, so an
// exported M365 account could never refresh) and that "IdC" survives its casing.
func TestNormalizeImportAuthMethod(t *testing.T) {
	cases := []struct {
		in, clientID, clientSecret, want string
	}{
		{"IdC", "c", "s", "idc"},
		{"builderid", "", "", "idc"},
		{"enterprise", "", "", "idc"},
		{"entra", "c", "", "external_idp"},
		{"azuread", "c", "", "external_idp"},
		{"external_idp", "c", "", "external_idp"},
		{"social", "", "", "social"},
		{"google", "", "", "social"},
		{"", "", "", "social"},
		{"", "c", "", "idc"},
		{"nonsense", "c", "s", "idc"},
		{"nonsense", "", "", "social"},
	}
	for _, tc := range cases {
		req := credentialImportRequest{AuthMethod: tc.in, ClientID: tc.clientID, ClientSecret: tc.clientSecret}
		normalizeImportAuthMethod(&req)
		if req.AuthMethod != tc.want {
			t.Errorf("normalize(%q, id=%q, secret=%q) = %q, want %q", tc.in, tc.clientID, tc.clientSecret, req.AuthMethod, tc.want)
		}
	}
}

// TestExternalIdpImportRejectsUnvalidatedEndpoint is the security regression: the
// token endpoint must be derived from the credential itself and gated on the IdP
// allow-list. A credential whose issuer points at an attacker host must NOT yield
// an endpoint — otherwise import becomes a refresh-token exfiltration primitive.
func TestExternalIdpImportRejectsUnvalidatedEndpoint(t *testing.T) {
	// A JWT (unsigned, payload-only is all the deriver reads) whose iss points at a
	// host that is not on allowedExternalIdpIssuerSuffixes.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://evil.example.com/tenant-id/v2.0"}`))
	evilToken := "x." + payload + ".y"

	req := credentialImportRequest{AuthMethod: "external_idp", ClientID: "c", AccessToken: evilToken}
	if _, _, ok := externalIdpImportEndpoints(&req); ok {
		t.Fatal("expected a non-allow-listed issuer to be rejected, but an endpoint was accepted")
	}

	// The legitimate shape still resolves.
	good := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://login.microsoftonline.com/tenant-id/v2.0"}`))
	req = credentialImportRequest{AuthMethod: "external_idp", ClientID: "c", AccessToken: "x." + good + ".y"}
	endpoint, scopes, ok := externalIdpImportEndpoints(&req)
	if !ok {
		t.Fatal("expected an allow-listed Microsoft issuer to be accepted")
	}
	if !strings.HasPrefix(endpoint, "https://login.microsoftonline.com/tenant-id/") {
		t.Fatalf("unexpected derived endpoint %q", endpoint)
	}
	if !strings.Contains(scopes, "offline_access") {
		t.Fatalf("expected offline_access in derived scopes, got %q", scopes)
	}
}

// authOidcURL captures the current oidc URL builder so the test can restore it.
func authOidcURL() func(string) string { return auth.GetOIDCTokenURLForTest() }
