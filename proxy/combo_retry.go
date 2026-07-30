package proxy

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type comboRetryClass string

const (
	comboRetryTransport comboRetryClass = "transport"
	comboRetryHTTP      comboRetryClass = "http"
	comboRetryTerminal  comboRetryClass = "terminal"
)

type comboAttemptFailure struct {
	Class           comboRetryClass
	Status          int
	ProviderCode    string
	RetryAfter      time.Duration
	BeforeFirstByte bool
	UsageConsumed   bool
	Cause           error
}

func (f comboAttemptFailure) Retryable() bool {
	if !f.BeforeFirstByte || f.UsageConsumed {
		return false
	}
	if f.Class == comboRetryTransport {
		return true
	}
	return f.Class == comboRetryHTTP && (f.Status == http.StatusRequestTimeout || f.Status == http.StatusTooManyRequests || f.Status == 500 || f.Status == 502 || f.Status == 503 || f.Status == 504)
}

func comboFailure(status int, cause error, beforeFirstByte bool) comboAttemptFailure {
	class := comboRetryHTTP
	if status == 0 {
		class = comboRetryTransport
	}
	if cause == nil {
		cause = errors.New("combo attempt failed")
	}
	return comboAttemptFailure{Class: class, Status: status, BeforeFirstByte: beforeFirstByte, Cause: cause}
}

type comboCandidateCursor struct {
	candidates []string
	index      int
	committed  bool
}

func newComboCandidateCursor(candidates []string) *comboCandidateCursor {
	return &comboCandidateCursor{candidates: append([]string(nil), candidates...)}
}

func (c *comboCandidateCursor) Current() (string, bool) {
	if c == nil || c.committed || c.index >= len(c.candidates) {
		return "", false
	}
	return c.candidates[c.index], true
}

func (c *comboCandidateCursor) Commit() {
	if c != nil {
		c.committed = true
	}
}

func (c *comboCandidateCursor) Advance(f comboAttemptFailure) bool {
	if c == nil || c.committed || !f.Retryable() {
		return false
	}
	c.index++
	return c.index < len(c.candidates)
}

func comboWait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
