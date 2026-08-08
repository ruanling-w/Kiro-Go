package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type grokRoundTripFunc func(*http.Request) (*http.Response, error)

func (f grokRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func grokTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestDoGrokCallWithRetryReplaysBody(t *testing.T) {
	const payload = `{"model":"grok-4","messages":[{"role":"user","content":"hi"}]}`
	attempts := 0
	client := &http.Client{Transport: grokRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		got, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != payload {
			t.Fatalf("attempt %d body = %q", attempts, got)
		}
		if attempts < 3 {
			return grokTestResponse(http.StatusServiceUnavailable, `{"error":"busy"}`), nil
		}
		return grokTestResponse(http.StatusOK, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`), nil
	})}
	req, err := http.NewRequest(http.MethodPost, "https://example.test", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := doGrokCallWithRetry(ctx, client, req, cancel, false, &KiroStreamCallback{}, "grok-4"); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDoGrokCallWithRetryDoesNotRetry400(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: grokRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return grokTestResponse(http.StatusBadRequest, `{"error":"bad"}`), nil
	})}
	req, _ := http.NewRequest(http.MethodPost, "https://example.test", bytes.NewBufferString(`{}`))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := doGrokCallWithRetry(ctx, client, req, cancel, false, &KiroStreamCallback{}, "grok-4"); err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDoGrokCallWithRetryCancellationInterruptsBackoff(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: grokRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("temporary transport failure")
	})}
	req, _ := http.NewRequest(http.MethodPost, "https://example.test", bytes.NewBufferString(`{}`))
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	started := time.Now()
	err := doGrokCallWithRetry(ctx, client, req, cancel, false, &KiroStreamCallback{}, "grok-4")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if elapsed := time.Since(started); elapsed >= 400*time.Millisecond {
		t.Fatalf("cancellation took %v", elapsed)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDoGrokCallWithRetryDoesNotRetrySuccessfulStream(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: grokRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return grokTestResponse(http.StatusOK, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"), nil
	})}
	req, _ := http.NewRequest(http.MethodPost, "https://example.test", bytes.NewBufferString(`{}`))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := doGrokCallWithRetry(ctx, client, req, cancel, true, &KiroStreamCallback{}, "grok-4"); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}
