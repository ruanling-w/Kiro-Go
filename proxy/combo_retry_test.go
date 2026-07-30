package proxy

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestComboAttemptFailureRetryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name string
		f    comboAttemptFailure
		want bool
	}{
		{"transport", comboFailure(0, errors.New("network"), true), true},
		{"rate limit", comboFailure(http.StatusTooManyRequests, errors.New("rate"), true), true},
		{"upstream", comboFailure(http.StatusServiceUnavailable, errors.New("upstream"), true), true},
		{"bad request", comboFailure(http.StatusBadRequest, errors.New("bad"), true), false},
		{"after first byte", comboFailure(http.StatusServiceUnavailable, errors.New("late"), false), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.f.Retryable(); got != tc.want {
				t.Fatalf("Retryable()=%v want %v", got, tc.want)
			}
		})
	}
}

func TestComboCandidateCursorStopsAfterCommit(t *testing.T) {
	cursor := newComboCandidateCursor([]string{"a", "b"})
	if model, ok := cursor.Current(); !ok || model != "a" {
		t.Fatalf("current=%q ok=%v", model, ok)
	}
	if !cursor.Advance(comboFailure(503, errors.New("retry"), true)) {
		t.Fatal("expected advance")
	}
	if model, ok := cursor.Current(); !ok || model != "b" {
		t.Fatalf("current=%q ok=%v", model, ok)
	}
	cursor.Commit()
	if cursor.Advance(comboFailure(503, errors.New("late"), true)) {
		t.Fatal("advanced after commit")
	}
	if _, ok := cursor.Current(); ok {
		t.Fatal("returned candidate after commit")
	}
}

func TestComboWaitHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if err := comboWait(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("canceled wait was not prompt")
	}
}
