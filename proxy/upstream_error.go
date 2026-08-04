package proxy

// upstream_error.go carries the upstream HTTP status alongside the response body
// so account failover can classify failures by status code instead of grepping
// the error string.
//
// Why this exists: every provider embeds the raw upstream body in its error
// (`fmt.Errorf("...upstream error %d: %s", status, body)`), and
// account_failover.go used to classify by substring — `strings.Contains(msg,
// "429")` or `strings.Contains(msg, "http 403")`. A body that merely mentions
// those digits (a request id, a byte count, a nested error) would put the
// account into a 1-hour cooldown or a permanent ban, draining the pool after a
// couple of requests and surfacing as 503 "No available accounts".
//
// Providers construct these via newUpstreamError; classification reads the
// status with errors.As and only falls back to substring matching when no
// typed error is present (token refresh paths, transport errors, upstreams
// that never produced a status).

import (
	"errors"
	"fmt"
	"strings"
)

// upstreamBodyLimit bounds how much upstream body is kept in the error message.
// Long HTML error pages add no diagnostic value and are the main source of
// accidental substring matches in any remaining string-based checks.
const upstreamBodyLimit = 512

// UpstreamError is a failure response from a provider upstream, retaining the
// HTTP status so it can be classified without string matching.
type UpstreamError struct {
	Provider string
	Status   int
	Body     string
	// Context is an optional prefix describing where the call went, e.g.
	// "HTTP 429 from cloudcode-pa.googleapis.com".
	Context string
}

func (e *UpstreamError) Error() string {
	prefix := e.Context
	if prefix == "" {
		prefix = fmt.Sprintf("upstream error %d", e.Status)
	}
	if e.Provider != "" {
		prefix = e.Provider + ": " + prefix
	}
	if e.Body == "" {
		return prefix
	}
	return prefix + ": " + e.Body
}

// newUpstreamError builds an UpstreamError with the body truncated to
// upstreamBodyLimit. context may be empty.
func newUpstreamError(provider string, status int, body, context string) *UpstreamError {
	return &UpstreamError{
		Provider: provider,
		Status:   status,
		Body:     truncateUpstreamBody(body),
		Context:  context,
	}
}

func truncateUpstreamBody(body string) string {
	body = strings.TrimSpace(body)
	if len(body) <= upstreamBodyLimit {
		return body
	}
	return body[:upstreamBodyLimit] + "... (truncated)"
}

// asUpstreamError extracts an *UpstreamError from err, if present.
func asUpstreamError(err error) (*UpstreamError, bool) {
	var ue *UpstreamError
	if errors.As(err, &ue) {
		return ue, true
	}
	return nil, false
}

// upstreamStatus returns the upstream HTTP status carried by err, or 0 when err
// is not (and does not wrap) an *UpstreamError.
func upstreamStatus(err error) int {
	if ue, ok := asUpstreamError(err); ok {
		return ue.Status
	}
	return 0
}

// upstreamBody returns the (truncated) upstream body carried by err, or "".
func upstreamBody(err error) string {
	if ue, ok := asUpstreamError(err); ok {
		return ue.Body
	}
	return ""
}
