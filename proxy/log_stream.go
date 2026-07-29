package proxy

import (
	"net/http"
	"time"
)

// subscribeLogStream atomically establishes the live subscription boundary and
// returns the current buffer newest-first. appendRequestLog holds the same lock
// through its non-blocking broadcast, so an entry is either in this snapshot or
// in the subscriber channel, never lost between the two.
func (h *Handler) subscribeLogStream() (chan RequestLog, []RequestLog) {
	h.requestLogsMu.Lock()
	defer h.requestLogsMu.Unlock()

	ch := h.logHub.subscribe()
	snapshot := make([]RequestLog, len(h.requestLogs))
	for i, entry := range h.requestLogs {
		snapshot[len(h.requestLogs)-1-i] = entry
	}
	return ch, snapshot
}

// apiStreamLogs streams request logs to the admin Logs page over SSE.
//
// handleAdminAPI already gated auth (session cookie) and set a JSON
// Content-Type; we override it to text/event-stream here. The connection first
// receives a "snapshot" event with the current ring buffer (newest-first), then
// a "log" event per new entry. A periodic comment heartbeat keeps the
// connection alive through idle proxy timeouts.
func (h *Handler) apiStreamLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ch, snapshot := h.subscribeLogStream()
	defer h.logHub.unsubscribe(ch)

	// Initial backfill: current buffer, newest-first. The subscription was
	// registered under the same lock used by appendRequestLog, so no entry can
	// fall between this snapshot and subsequent live events.
	h.sendSSE(w, flusher, "snapshot", snapshot)

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				return
			}
			h.sendSSE(w, flusher, "log", entry)
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
