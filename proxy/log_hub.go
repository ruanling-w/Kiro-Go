package proxy

import "sync"

// logHub fans out request-log entries to live SSE subscribers (admin Logs page).
// Every log — success or failure — passes through appendRequestLog, which calls
// broadcast while holding requestLogsMu to share one startup boundary with the
// snapshot. Sends are non-blocking over buffered channels so a slow or stalled
// subscriber can never back up the request hot path.
type logHub struct {
	mu   sync.Mutex
	subs map[chan RequestLog]struct{}
}

func newLogHub() *logHub {
	return &logHub{subs: make(map[chan RequestLog]struct{})}
}

// subscribe registers a new subscriber and returns its buffered channel.
func (hb *logHub) subscribe() chan RequestLog {
	ch := make(chan RequestLog, 64)
	hb.mu.Lock()
	hb.subs[ch] = struct{}{}
	hb.mu.Unlock()
	return ch
}

// unsubscribe removes a subscriber and closes its channel.
func (hb *logHub) unsubscribe(ch chan RequestLog) {
	hb.mu.Lock()
	if _, ok := hb.subs[ch]; ok {
		delete(hb.subs, ch)
		close(ch)
	}
	hb.mu.Unlock()
}

// broadcast delivers entry to every subscriber without blocking. A subscriber
// whose buffer is full simply drops this entry rather than stalling the caller.
func (hb *logHub) broadcast(entry RequestLog) {
	hb.mu.Lock()
	for ch := range hb.subs {
		select {
		case ch <- entry:
		default:
		}
	}
	hb.mu.Unlock()
}
