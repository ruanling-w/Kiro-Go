package proxy

import (
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"strings"
	"time"
)

const maxAccountRetryAttempts = 3

// accountFailureKind is the classification of an upstream failure. It decides
// both the pool penalty (soft cooldown / quota cooldown / permanent ban) and the
// Telegram event, so misclassification is expensive: a wrong "auth" verdict
// disables the account outright.
type accountFailureKind int

const (
	failureSoft accountFailureKind = iota
	failureOverage
	failureQuota
	failureSuspended
	failureProfileUnavailable
	failureAuth
)

// classifyFailureKind decides what an upstream failure means for the account.
//
// When the provider produced an *UpstreamError the HTTP status is authoritative
// and no digit/word matching is done on the body. Substring matching only runs
// for untyped errors — token refresh failures, transport errors, and any call
// site that never saw a status code. This is the fix for the pool-draining bug:
// providers embed the raw upstream body in their error text, so a body that
// merely contained "429" or the word "forbidden" (a request id, a nested error,
// an HTML error page) used to trigger a 1-hour cooldown or a permanent ban.
//
// Order matters. Suspension and profile-unavailable are body markers that arrive
// *with* a 401/403 status, so they must be tested before the status-based auth
// check or a suspended account would be misreported as bad credentials.
func classifyFailureKind(err error) accountFailureKind {
	if err == nil {
		return failureSoft
	}
	msg := err.Error()

	if isSuspensionErrorMessage(msg) {
		return failureSuspended
	}
	if isProfileUnavailableErrorMessage(msg) {
		return failureProfileUnavailable
	}

	if ue, ok := asUpstreamError(err); ok {
		switch {
		case ue.Status == http.StatusPaymentRequired && strings.Contains(strings.ToLower(ue.Body), "overage"):
			return failureOverage
		case ue.Status == http.StatusTooManyRequests:
			return failureQuota
		case ue.Status == http.StatusUnauthorized || ue.Status == http.StatusForbidden:
			return failureAuth
		}
		// Any other status (5xx, 400, 404 …) is a soft failure: rotate the
		// account, never ban it.
		return failureSoft
	}

	switch {
	case isOverageErrorMessage(msg):
		return failureOverage
	case isQuotaErrorMessage(msg):
		return failureQuota
	case isAuthErrorMessage(msg):
		return failureAuth
	}
	return failureSoft
}

func isQuotaErrorMessage(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "429") || strings.Contains(msg, "quota")
}

func isOverageErrorMessage(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "402") && strings.Contains(msg, "overage")
}

func isSuspensionErrorMessage(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "temporarily_suspended") ||
		strings.Contains(msg, "temporarily is suspended") ||
		strings.Contains(msg, "account suspended")
}

func isProfileUnavailableErrorMessage(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "no available kiro profile")
}

func isAuthErrorMessage(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "http 401") ||
		strings.Contains(msg, "http 403") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "token invalid") ||
		strings.Contains(msg, "token expired") ||
		strings.Contains(msg, "invalid_grant") ||
		strings.Contains(msg, "access token expired") ||
		strings.Contains(msg, "refresh token expired")
}

// logPoolEmpty records why no account was routable before a 503 is returned.
// The client-facing message is intentionally opaque ("No available accounts");
// this is the operator-facing half, and it is the difference between "the pool
// drained itself" and "nobody advertises that model".
func (h *Handler) logPoolEmpty(api, model string, excluded map[string]bool) {
	if h == nil || h.pool == nil {
		return
	}
	logger.Warnf("[%s] 503 no available accounts for model %q — %s", api, model, h.pool.UnavailableReason(model, excluded))
}

// logComboPoolEmpty is the Combo equivalent: a Combo route tries each candidate
// model with its own exclusion set, so the reason is reported per candidate.
func (h *Handler) logComboPoolEmpty(api string, route *comboRouteSnapshot) {
	if h == nil || h.pool == nil || route == nil {
		return
	}
	for _, candidate := range route.Candidates {
		logger.Warnf("[%s] 503 no available accounts for combo %q candidate %q — %s",
			api, route.RequestedModel, candidate.Model, h.pool.UnavailableReason(candidate.Model, nil))
	}
}

func (h *Handler) disableAccount(account *config.Account, banStatus, banReason string) {
	if account == nil {
		return
	}

	updatedAccount := *account
	if !updatedAccount.Enabled && updatedAccount.BanStatus == banStatus && updatedAccount.BanReason == banReason {
		return
	}

	updatedAccount.Enabled = false
	updatedAccount.BanStatus = banStatus
	updatedAccount.BanReason = banReason
	updatedAccount.BanTime = time.Now().Unix()

	if err := config.UpdateAccount(account.ID, updatedAccount); err != nil {
		logger.Warnf("[AccountFailover] Failed to disable %s: %v", account.Email, err)
		return
	}

	logger.Warnf("[AccountFailover] Disabled %s: %s", account.Email, banReason)
	h.pool.Reload()
}

func (h *Handler) disableAccountOverage(account *config.Account) {
	if account == nil {
		return
	}

	snap, fetchErr := FetchOverageStatus(account)
	if fetchErr != nil {
		logger.Warnf("[AccountFailover] Failed to refresh overage status for %s: %v", account.Email, fetchErr)
		return
	}
	if persistErr := PersistOverageSnapshot(account.ID, snap); persistErr != nil {
		logger.Warnf("[AccountFailover] Failed to persist overage snapshot for %s: %v", account.Email, persistErr)
		return
	}

	logger.Warnf("[AccountFailover] Refreshed overage status for %s after upstream overage limit error: %s", account.Email, snap.Status)
	h.pool.Reload()
}

// classifyAccountFailure maps a failure to a Telegram event type without side effects.
// fromTokenRefresh prefers EventTokenRefresh for auth/default failures on the refresh path.
func classifyAccountFailure(account *config.Account, err error, fromTokenRefresh bool) string {
	if account == nil || err == nil {
		return ""
	}
	if isKiroAPIKeyAccount(account) || isRemoteKiroAccount(account) {
		return EventSoft
	}
	switch classifyFailureKind(err) {
	case failureOverage:
		return EventOverage
	case failureQuota:
		return EventQuota
	case failureSuspended:
		return EventBan
	case failureProfileUnavailable:
		return EventSoft
	case failureAuth:
		if fromTokenRefresh {
			return EventTokenRefresh
		}
		return EventBan
	default:
		if fromTokenRefresh {
			return EventTokenRefresh
		}
		return EventSoft
	}
}

func (h *Handler) handleAccountFailure(account *config.Account, err error) {
	h.handleAccountFailureEx(account, err, false)
}

// handleBackgroundFailure records a failure observed by a background maintenance
// loop (model-catalog refresh, periodic account-info refresh) rather than by a
// real client request.
//
// These loops touch every enabled account on a timer, so letting them ban is how
// the pool empties with no traffic at all: one upstream blip during a sweep
// disables every account it reached, and the next client request gets 503 "No
// available accounts". A background failure therefore only ever applies a soft
// cooldown — the account rotates out briefly and the next real request re-tests
// it. Quota is still honoured because a genuine 429 is not a blip and the
// cooldown is what keeps us from hammering it.
//
// Bans stay on the request path (handleAccountFailure), where a failure is
// corroborated by an actual client call.
func (h *Handler) handleBackgroundFailure(account *config.Account, err error, source string) {
	if account == nil || err == nil {
		return
	}
	kind := classifyFailureKind(err)
	h.pool.RecordError(account.ID, kind == failureQuota)
	if kind == failureAuth || kind == failureSuspended {
		logger.Warnf("[%s] %s looks unauthenticated/suspended (%v) — soft cooldown only; a client request will confirm before any ban", source, account.Email, err)
	}
	NotifyAccountEvent(account, EventSoft, err.Error())
}

func (h *Handler) handleAccountFailureEx(account *config.Account, err error, fromTokenRefresh bool) {
	if account == nil || err == nil {
		return
	}

	// Kiro CLI API-key (ksk_) accounts have no OAuth token to refresh and their
	// 403s are often transient (region backoff). Soft-fail only; never auto-ban.
	// Remote Kiro-Go peers are the same: static sk, peer may be down or rate-limit —
	// rotate with a soft cooldown, never disable the local account entry.
	if isKiroAPIKeyAccount(account) || isRemoteKiroAccount(account) {
		h.pool.RecordError(account.ID, false)
		NotifyAccountEvent(account, EventSoft, err.Error())
		return
	}

	errMsg := err.Error()
	eventType := classifyAccountFailure(account, err, fromTokenRefresh)
	switch classifyFailureKind(err) {
	case failureOverage:
		h.disableAccountOverage(account)
		h.pool.RecordError(account.ID, false)
	case failureQuota:
		h.pool.RecordError(account.ID, true)
	case failureSuspended:
		h.disableAccount(account, "BANNED", "AWS temporarily suspended - unusual user activity detected")
	case failureProfileUnavailable:
		// Profile ARN may be transiently unresolvable (upstream blip, stale token).
		// Treat as a soft failure: short cooldown so the next request rotates account,
		// but never auto-disable — operators can still investigate via warn logs.
		h.pool.RecordError(account.ID, false)
	case failureAuth:
		h.disableAccount(account, "BANNED", "Authentication failed - token invalid or expired")
	default:
		h.pool.RecordError(account.ID, false)
	}

	NotifyAccountEvent(account, eventType, errMsg)
}
