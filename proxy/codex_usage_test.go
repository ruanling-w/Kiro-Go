package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestParseCodexWhamUsage_BasicWindows(t *testing.T) {
	body := []byte(`{
		"plan_type": "plus",
		"rate_limit": {
			"limit_reached": false,
			"primary_window": {
				"used_percent": 72.5,
				"reset_at": 1710003600
			},
			"secondary_window": {
				"percent_used": 41,
				"resets_at": "2026-07-14T00:00:00Z"
			}
		},
		"rate_limit_reset_credits": { "available_count": 2 },
		"code_review_rate_limit": {
			"limit_reached": true,
			"primary_window": { "used_percent": 100, "reset_at": 1710007200 }
		}
	}`)

	snap, err := parseCodexWhamUsage(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if snap.Plan != "plus" {
		t.Fatalf("plan=%q", snap.Plan)
	}
	if snap.ResetCredits != 2 {
		t.Fatalf("credits=%d", snap.ResetCredits)
	}
	if snap.LimitReached {
		t.Fatalf("expected LimitReached=false for normal window")
	}
	if len(snap.Windows) < 2 {
		t.Fatalf("expected >=2 windows, got %d: %#v", len(snap.Windows), snap.Windows)
	}

	byKey := map[string]configWindow{}
	for _, w := range snap.Windows {
		byKey[w.Key] = configWindow{w.UsedPct, w.ResetAt, w.LimitHit, w.Label}
	}
	if byKey["session"].UsedPct < 72 || byKey["session"].UsedPct > 73 {
		t.Fatalf("session used=%v", byKey["session"].UsedPct)
	}
	if byKey["session"].ResetAt == "" {
		t.Fatalf("session reset empty")
	}
	if byKey["weekly"].UsedPct != 41 {
		t.Fatalf("weekly used=%v", byKey["weekly"].UsedPct)
	}
	if byKey["weekly"].ResetAt != "2026-07-14T00:00:00Z" {
		t.Fatalf("weekly reset=%q", byKey["weekly"].ResetAt)
	}
	if w, ok := byKey["review_session"]; !ok || !w.LimitHit || w.UsedPct != 100 {
		t.Fatalf("review_session=%#v", w)
	}

	// Round-trip through JSON to ensure Account fields are serializable.
	raw, err := json.Marshal(snap.Windows)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 10 {
		t.Fatalf("marshal short: %s", raw)
	}
}

type configWindow struct {
	UsedPct  float64
	ResetAt  string
	LimitHit bool
	Label    string
}

func TestParseCodexWhamUsage_InvalidJSON(t *testing.T) {
	if _, err := parseCodexWhamUsage([]byte(`{`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestCodexUsageHTTPErrorClassification(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		err := &CodexUsageHTTPError{Status: status, Body: "expired"}
		if !isCodexUsageAuthError(err) {
			t.Fatalf("status %d should be auth error", status)
		}
		wrapped := fmt.Errorf("wrapped: %w", err)
		var typed *CodexUsageHTTPError
		if !errors.As(wrapped, &typed) {
			t.Fatalf("typed error was not preserved: %v", wrapped)
		}
	}
	for _, status := range []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusInternalServerError} {
		if isCodexUsageAuthError(&CodexUsageHTTPError{Status: status}) {
			t.Fatalf("status %d should remain a transient error", status)
		}
	}
}

func TestCodexUsageHTTPErrorBoundedBody(t *testing.T) {
	body := strings.Repeat("x", 1000)
	err := &CodexUsageHTTPError{Status: http.StatusInternalServerError, Body: truncateForErr(body, 240)}
	if len(err.Error()) > 280 {
		t.Fatalf("error was not bounded: %d bytes", len(err.Error()))
	}
}

func TestParseCodexWhamUsage_UnrecognizedSchema(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"error":"temporarily unavailable"}`),
		[]byte(`{"summary":{}}`),
		[]byte(`{"rate_limit":{}}`),
		[]byte(`{"rate_limit_reset_credits":null}`),
		[]byte(`{"rate_limit_reset_credits":{"available_count":null}}`),
		[]byte(`{"plan_type":123}`),
	}
	for _, payload := range payloads {
		if _, err := parseCodexWhamUsage(payload); err == nil {
			t.Fatalf("expected unrecognized schema error for %s", payload)
		}
	}
}

func TestParseCodexWhamUsage_EmptyPrimaryFallsBack(t *testing.T) {
	body := []byte(`{
		"rate_limit": {},
		"rate_limits_by_limit_id": {
			"codex": {
				"primary_window": {"used_percent": 25}
			}
		}
	}`)
	snap, err := parseCodexWhamUsage(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(snap.Windows) != 1 || snap.Windows[0].UsedPct != 25 {
		t.Fatalf("expected fallback window, got %#v", snap.Windows)
	}
}
