package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamCommitGateBuffersAndCommits(t *testing.T) {
	rec := httptest.NewRecorder()
	gate := newStreamCommitGate(rec)
	gate.Header().Set("Content-Type", "text/event-stream")
	gate.WriteHeader(http.StatusOK)
	if _, err := gate.Write([]byte("data: first\n\n")); err != nil {
		t.Fatal(err)
	}
	if rec.Body.Len() != 0 || gate.Committed() {
		t.Fatal("buffer leaked before commit")
	}
	if err := gate.Commit(); err != nil {
		t.Fatal(err)
	}
	if !gate.Committed() || rec.Code != http.StatusOK || rec.Body.String() != "data: first\n\n" {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestStreamCommitGateDiscardHidesAttempt(t *testing.T) {
	rec := httptest.NewRecorder()
	gate := newStreamCommitGate(rec)
	_, _ = gate.Write([]byte("failed attempt"))
	gate.Discard()
	_, _ = gate.Write([]byte("success"))
	if err := gate.Commit(); err != nil {
		t.Fatal(err)
	}
	if rec.Body.String() != "success" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestStreamCommitGateEnforcesBufferLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	gate := newStreamCommitGate(rec)
	gate.limit = 3
	if _, err := gate.Write(bytes.Repeat([]byte("x"), 4)); err != errStreamGateOverflow {
		t.Fatalf("err=%v", err)
	}
}
