package proxy

// Tests for the session-token admin auth added in the FE rebuild: login/logout,
// session validation + expiry, constant-time password compare behavior, CSRF
// double-submit enforcement, and per-IP brute-force lockout.

import (
	"encoding/json"
	"kiro-go/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// resetAdminAuth clears the package-global session/attempt maps so each test
// starts from a clean slate (the manager is process-wide by design).
func resetAdminAuth() {
	adminAuth.mu.Lock()
	adminAuth.sessions = make(map[string]*adminSession)
	adminAuth.attempts = make(map[string]*loginAttempt)
	adminAuth.mu.Unlock()
}

func setAdminPassword(t *testing.T, pw string) {
	t.Helper()
	if err := config.UpdateSettingsPatch(nil, nil, pw); err != nil {
		t.Fatalf("set password: %v", err)
	}
}

func loginRequest(pw string) *http.Request {
	body, _ := json.Marshal(map[string]string{"password": pw})
	return httptest.NewRequest(http.MethodPost, "/admin/api/login", strings.NewReader(string(body)))
}

// cookieValue extracts a Set-Cookie value by name from a recorded response.
func cookieValue(res *http.Response, name string) string {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func TestAdminLoginSuccessSetsSessionAndCSRF(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "s3cret")

	h := &Handler{}
	w := httptest.NewRecorder()
	h.handleAdminLogin(w, loginRequest("s3cret"))

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", res.StatusCode)
	}
	if cookieValue(res, adminSessionCookie) == "" {
		t.Fatal("missing session cookie")
	}
	if cookieValue(res, adminCSRFCookie) == "" {
		t.Fatal("missing CSRF cookie")
	}
	// Session cookie must be HttpOnly; CSRF cookie must be readable.
	for _, c := range res.Cookies() {
		if c.Name == adminSessionCookie && !c.HttpOnly {
			t.Error("session cookie should be HttpOnly")
		}
		if c.Name == adminCSRFCookie && c.HttpOnly {
			t.Error("CSRF cookie must not be HttpOnly")
		}
	}
}

func TestAdminLoginWrongPasswordCountsFail(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "correct")

	h := &Handler{}
	w := httptest.NewRecorder()
	h.handleAdminLogin(w, loginRequest("wrong"))

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Result().StatusCode)
	}
	if cookieValue(w.Result(), adminSessionCookie) != "" {
		t.Fatal("no session cookie should be set on failure")
	}
}

func TestAdminLoginEmptyConfiguredPasswordFailsClosed(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "")

	h := &Handler{}
	w := httptest.NewRecorder()
	h.handleAdminLogin(w, loginRequest(""))

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("empty password must fail closed, got %d", w.Result().StatusCode)
	}
}

func TestAdminLockoutAfterRepeatedFailures(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "correct")

	h := &Handler{}
	// loginMaxFails wrong attempts trip the lockout.
	for i := 0; i < loginMaxFails; i++ {
		w := httptest.NewRecorder()
		h.handleAdminLogin(w, loginRequest("wrong"))
	}
	// Next attempt — even with the RIGHT password — is locked out (429).
	w := httptest.NewRecorder()
	h.handleAdminLogin(w, loginRequest("correct"))
	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 lockout, got %d", w.Result().StatusCode)
	}
}

func TestAdminSessionValidateAndExpiry(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()

	token, _, err := adminAuth.createSession("1.2.3.4")
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if _, ok := adminAuth.validate(token); !ok {
		t.Fatal("fresh session should validate")
	}
	if _, ok := adminAuth.validate("bogus"); ok {
		t.Fatal("bogus token should not validate")
	}

	// Force absolute expiry in the past → invalid + evicted.
	adminAuth.mu.Lock()
	adminAuth.sessions[hashToken(token)].expiresAt = time.Now().Add(-time.Second)
	adminAuth.mu.Unlock()
	if _, ok := adminAuth.validate(token); ok {
		t.Fatal("expired session should not validate")
	}
}

func TestAdminIdleExpiry(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()

	token, _, _ := adminAuth.createSession("1.2.3.4")
	adminAuth.mu.Lock()
	adminAuth.sessions[hashToken(token)].idleExpiry = time.Now().Add(-time.Second)
	adminAuth.mu.Unlock()
	if _, ok := adminAuth.validate(token); ok {
		t.Fatal("idle-expired session should not validate")
	}
}

func TestAdminInvalidateAllOnPasswordChange(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()

	token, _, _ := adminAuth.createSession("1.2.3.4")
	if _, ok := adminAuth.validate(token); !ok {
		t.Fatal("precondition: session valid")
	}
	adminAuth.invalidateAll()
	if _, ok := adminAuth.validate(token); ok {
		t.Fatal("session should be gone after invalidateAll")
	}
}

func TestAuthorizeAdminSessionCookie(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "pw")

	token, csrf, _ := adminAuth.createSession("1.2.3.4")
	h := &Handler{}

	// GET with valid session cookie → authorized, no CSRF needed.
	r := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	w := httptest.NewRecorder()
	if !h.authorizeAdmin(w, r) {
		t.Fatal("valid session GET should authorize")
	}

	// POST without CSRF header → 403.
	r = httptest.NewRequest(http.MethodPost, "/admin/api/settings", nil)
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	w = httptest.NewRecorder()
	if h.authorizeAdmin(w, r) {
		t.Fatal("mutating request without CSRF must be rejected")
	}
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Result().StatusCode)
	}

	// POST with matching CSRF header → authorized.
	r = httptest.NewRequest(http.MethodPost, "/admin/api/settings", nil)
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	r.Header.Set(adminCSRFHeader, csrf)
	w = httptest.NewRecorder()
	if !h.authorizeAdmin(w, r) {
		t.Fatal("mutating request with matching CSRF should authorize")
	}
}

func TestAuthorizeAdminLegacyHeaderFallback(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "legacy-pw")

	h := &Handler{}
	// Legacy header path: POST allowed without CSRF (no session to bind to).
	r := httptest.NewRequest(http.MethodPost, "/admin/api/settings", nil)
	r.Header.Set("X-Admin-Password", "legacy-pw")
	w := httptest.NewRecorder()
	if !h.authorizeAdmin(w, r) {
		t.Fatal("valid legacy header should authorize")
	}

	// Wrong header → 401.
	r = httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
	r.Header.Set("X-Admin-Password", "nope")
	w = httptest.NewRecorder()
	if h.authorizeAdmin(w, r) {
		t.Fatal("wrong legacy header must be rejected")
	}
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Result().StatusCode)
	}
}

func TestAdminLogoutDestroysSession(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()

	token, _, _ := adminAuth.createSession("1.2.3.4")
	h := &Handler{}
	r := httptest.NewRequest(http.MethodPost, "/admin/api/logout", nil)
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	w := httptest.NewRecorder()
	h.handleAdminLogout(w, r)

	if _, ok := adminAuth.validate(token); ok {
		t.Fatal("session should be destroyed after logout")
	}
}
