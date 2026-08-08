package proxy

import "testing"

// TestRecordSuccessLogMetaTokenBreakdown verifies the success log records the
// input/output split plus prompt-cache counts, and keeps Tokens as the in+out
// total for backward-compatible consumers.
func TestRecordSuccessLogMetaTokenBreakdown(t *testing.T) {
	h := &Handler{}
	h.recordSuccessLogMeta("claude", "claude-sonnet-4.5", "acct", logTokens{
		Input:         120,
		Output:        40,
		CacheRead:     100,
		CacheCreation: 20,
	}, 1.5, 12, "1.2.3.4", "key-1", "kiro")

	logs := h.getRequestLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	e := logs[0]
	if e.InputTokens != 120 || e.OutputTokens != 40 {
		t.Fatalf("in/out mismatch: in=%d out=%d", e.InputTokens, e.OutputTokens)
	}
	if e.Tokens != 160 {
		t.Fatalf("Tokens must equal in+out (160), got %d", e.Tokens)
	}
	if e.CacheReadTokens != 100 || e.CacheCreationTokens != 20 {
		t.Fatalf("cache mismatch: read=%d write=%d", e.CacheReadTokens, e.CacheCreationTokens)
	}
	if e.Cached {
		t.Fatalf("live upstream turn must not be flagged Cached")
	}
	if e.Status != "success" || e.Credits != 1.5 || e.Provider != "kiro" {
		t.Fatalf("meta mismatch: %+v", e)
	}
}

// TestRecordSuccessLogMetaCachedFlag verifies a response-cache hit is flagged
// Cached so the UI can distinguish it independently of the provider string.
func TestRecordSuccessLogMetaCachedFlag(t *testing.T) {
	h := &Handler{}
	h.recordSuccessLogMeta("openai", "gpt-4o", "", logTokens{
		Input:  10,
		Output: 5,
		Cached: true,
	}, 0, 0, "", "key-1", "cache")

	logs := h.getRequestLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	e := logs[0]
	if !e.Cached {
		t.Fatalf("cache hit must set Cached=true")
	}
	if e.Tokens != 15 {
		t.Fatalf("Tokens must equal in+out (15), got %d", e.Tokens)
	}
	if e.Credits != 0 {
		t.Fatalf("cache hit must be credit-free, got %v", e.Credits)
	}
}
