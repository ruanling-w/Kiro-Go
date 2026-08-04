package proxy

// admin_auth.go — session-token authentication for the admin panel.
//
// The legacy model sent the raw admin password as X-Admin-Password on EVERY
// request (also mirrored to a plaintext cookie / localStorage). This file
// replaces that with a proper login:
//
//   - POST /admin/api/login  → constant-time password check, then an HttpOnly
//     session cookie (server keeps only the SHA-256 of the token) + a readable
//     double-submit CSRF cookie.
//   - POST /admin/api/logout → destroys the server session and clears cookies.
//   - Every other /admin/api/* request authenticates via the session cookie;
//     mutating requests must echo the CSRF token in X-CSRF-Token.
//
// The old X-Admin-Password header is still accepted as a fallback so existing
// scripts / bookmarks keep working during migration; it bypasses CSRF (there is
// no session to bind to) but is still gated by the password.
//
// Brute-force protection: per-IP failed-login counting with exponential lockout,
// reusing ClientIP + the trust-proxy setting for correct IP attribution.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	adminSessionCookie = "kiro_admin_session"
	adminCSRFCookie    = "kiro_csrf"
	adminCSRFHeader    = "X-CSRF-Token"

	// Absolute cap on a session's life; the idle window slides on each use.
	adminSessionAbsoluteTTL = 24 * time.Hour
	adminSessionIdleTTL     = 30 * time.Minute

	// Login brute-force policy.
	loginMaxFails    = 5
	loginFailWindow  = 15 * time.Minute
	loginBaseLockout = 1 * time.Minute
	loginMaxLockout  = 30 * time.Minute
)

type adminSession struct {
	csrf       string
	createdAt  time.Time
	expiresAt  time.Time // absolute cap
	idleExpiry time.Time // sliding; renewed on each authenticated request
	ip         string
}

type loginAttempt struct {
	fails     int
	windowEnd time.Time
	lockUntil time.Time
}

type adminAuthManager struct {
	mu       sync.Mutex
	sessions map[string]*adminSession // key: hex(sha256(token))
	attempts map[string]*loginAttempt // key: client IP
}

// adminAuth is the process-wide session store. It is intentionally in-memory:
// admin sessions are short-lived and a restart forcing re-login is acceptable.
var adminAuth = newAdminAuthManager()

func newAdminAuthManager() *adminAuthManager {
	m := &adminAuthManager{
		sessions: make(map[string]*adminSession),
		attempts: make(map[string]*loginAttempt),
	}
	go m.cleanupLoop()
	return m
}

func (m *adminAuthManager) cleanupLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		m.mu.Lock()
		for k, s := range m.sessions {
			if now.After(s.expiresAt) || now.After(s.idleExpiry) {
				delete(m.sessions, k)
			}
		}
		for ip, a := range m.attempts {
			if now.After(a.windowEnd) && now.After(a.lockUntil) {
				delete(m.attempts, ip)
			}
		}
		m.mu.Unlock()
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// createSession mints a new session + CSRF token and stores the session keyed by
// the token's hash. Returns the raw token (for the cookie) and the CSRF value.
func (m *adminAuthManager) createSession(ip string) (token, csrf string, err error) {
	token, err = randomToken()
	if err != nil {
		return "", "", err
	}
	csrf, err = randomToken()
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	m.mu.Lock()
	m.sessions[hashToken(token)] = &adminSession{
		csrf:       csrf,
		createdAt:  now,
		expiresAt:  now.Add(adminSessionAbsoluteTTL),
		idleExpiry: now.Add(adminSessionIdleTTL),
		ip:         ip,
	}
	m.mu.Unlock()
	return token, csrf, nil
}

// validate looks up a session by raw token, enforces expiry, and slides the idle
// window forward. Returns the session (copy of csrf) when still valid.
func (m *adminAuthManager) validate(token string) (*adminSession, bool) {
	if token == "" {
		return nil, false
	}
	key := hashToken(token)
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[key]
	if !ok {
		return nil, false
	}
	if now.After(s.expiresAt) || now.After(s.idleExpiry) {
		delete(m.sessions, key)
		return nil, false
	}
	// Slide idle window, capped by the absolute expiry.
	s.idleExpiry = now.Add(adminSessionIdleTTL)
	if s.idleExpiry.After(s.expiresAt) {
		s.idleExpiry = s.expiresAt
	}
	return s, true
}

func (m *adminAuthManager) destroy(token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	delete(m.sessions, hashToken(token))
	m.mu.Unlock()
}

// invalidateAll drops every session. Called when the admin password changes so
// old sessions cannot outlive the credential.
func (m *adminAuthManager) invalidateAll() {
	m.mu.Lock()
	m.sessions = make(map[string]*adminSession)
	m.mu.Unlock()
}

// lockedOut reports whether the IP is currently locked out and for how long.
func (m *adminAuthManager) lockedOut(ip string) (bool, time.Duration) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.attempts[ip]
	if !ok {
		return false, 0
	}
	if now.Before(a.lockUntil) {
		return true, a.lockUntil.Sub(now)
	}
	return false, 0
}

// recordFail increments the failed-login counter for an IP and, past the
// threshold, sets an exponentially growing lockout.
func (m *adminAuthManager) recordFail(ip string) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.attempts[ip]
	if !ok || now.After(a.windowEnd) {
		a = &loginAttempt{windowEnd: now.Add(loginFailWindow)}
		m.attempts[ip] = a
	}
	a.fails++
	if a.fails >= loginMaxFails {
		// Exponential backoff: base * 2^(fails-threshold), capped.
		shift := a.fails - loginMaxFails
		lock := loginBaseLockout << shift
		if lock > loginMaxLockout || lock <= 0 {
			lock = loginMaxLockout
		}
		a.lockUntil = now.Add(lock)
	}
}

func (m *adminAuthManager) recordSuccess(ip string) {
	m.mu.Lock()
	delete(m.attempts, ip)
	m.mu.Unlock()
}

// ---- HTTP helpers ----

func adminRequestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setAdminCookies(w http.ResponseWriter, r *http.Request, token, csrf string) {
	secure := adminRequestIsSecure(r)
	maxAge := int(adminSessionAbsoluteTTL / time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    token,
		Path:     "/admin",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	// Readable by JS so the SPA can echo it back in X-CSRF-Token.
	http.SetCookie(w, &http.Cookie{
		Name:     adminCSRFCookie,
		Value:    csrf,
		Path:     "/admin",
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearAdminCookies(w http.ResponseWriter, r *http.Request) {
	secure := adminRequestIsSecure(r)
	for _, name := range []string{adminSessionCookie, adminCSRFCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/admin",
			MaxAge:   -1,
			HttpOnly: name == adminSessionCookie,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func adminSessionTokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(adminSessionCookie)
	if err != nil || c == nil {
		return ""
	}
	return c.Value
}

// handleAdminLogin authenticates a password and, on success, starts a session.
func (h *Handler) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r, config.GetTrustProxyHeaders())
	if locked, retry := adminAuth.lockedOut(ip); locked {
		w.Header().Set("Retry-After", secondsString(retry))
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many attempts, try again later"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	expected := config.GetPassword()
	// Constant-time compare to avoid leaking length/prefix via timing.
	match := subtle.ConstantTimeCompare([]byte(req.Password), []byte(expected)) == 1
	if !match || expected == "" {
		adminAuth.recordFail(ip)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid password"})
		return
	}

	token, csrf, err := adminAuth.createSession(ip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
		return
	}
	adminAuth.recordSuccess(ip)
	setAdminCookies(w, r, token, csrf)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAdminLogout destroys the current session and clears cookies.
func (h *Handler) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	adminAuth.destroy(adminSessionTokenFromRequest(r))
	clearAdminCookies(w, r)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// authorizeAdmin gates every /admin/api/* request except login/logout. It
// prefers the session cookie; mutating requests (POST/PUT/PATCH/DELETE) must
// also present a matching CSRF token. A valid legacy X-Admin-Password header is
// accepted as a fallback (no session ⇒ no CSRF binding, so CSRF is skipped for
// that path — it is still gated by the password itself).
//
// Returns true when authorized. On failure it writes the 401/403 response and
// returns false so the caller can stop.
func (h *Handler) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	// 1) Session cookie path.
	if token := adminSessionTokenFromRequest(r); token != "" {
		if sess, ok := adminAuth.validate(token); ok {
			if isMutatingMethod(r.Method) {
				if !csrfMatches(r, sess.csrf) {
					// Diagnostic: the usual cause is the kiro_csrf cookie never
					// reaching JS (Path=/admin, SameSite=Strict, Secure behind a
					// reverse proxy) so the client cannot echo the header at all.
					// Log presence only — never the token values.
					_, cookieErr := r.Cookie(adminCSRFCookie)
					logger.Warnf("[AdminAuth] CSRF mismatch on %s %s: header=%v cookie=%v secure=%v xfp=%q",
						r.Method, r.URL.Path,
						r.Header.Get(adminCSRFHeader) != "", cookieErr == nil,
						adminRequestIsSecure(r), r.Header.Get("X-Forwarded-Proto"))
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "CSRF token mismatch",
						"code":  "csrf_mismatch",
					})
					return false
				}
			}
			return true
		}
		// Stale/invalid session: clear the cookie so the client stops sending it.
		clearAdminCookies(w, r)
	}

	// 2) Legacy header fallback (constant-time compared).
	expected := config.GetPassword()
	if expected != "" {
		provided := r.Header.Get("X-Admin-Password")
		if provided == "" {
			if c, err := r.Cookie("admin_password"); err == nil && c != nil {
				provided = c.Value
			}
		}
		if provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
			return true
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
	return false
}

func csrfMatches(r *http.Request, expected string) bool {
	got := r.Header.Get(adminCSRFHeader)
	if got == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func secondsString(d time.Duration) string {
	s := int(d.Seconds())
	if s < 1 {
		s = 1
	}
	return strconv.Itoa(s)
}
