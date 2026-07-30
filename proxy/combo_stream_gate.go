package proxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

var errStreamGateOverflow = errors.New("stream pre-commit buffer exceeded")

const streamGateBufferLimit = 256 << 10

// streamCommitGate stages response headers and bytes until a serving attempt is
// accepted. Discard makes pre-commit retries invisible; Commit pins the attempt
// and forwards all subsequent writes directly to the client.
type streamCommitGate struct {
	dst       http.ResponseWriter
	buffer    bytes.Buffer
	header    http.Header
	status    int
	committed bool
	limit     int
}

func newStreamCommitGate(dst http.ResponseWriter) *streamCommitGate {
	return &streamCommitGate{dst: dst, header: make(http.Header), limit: streamGateBufferLimit}
}

func (g *streamCommitGate) Header() http.Header { return g.header }

func (g *streamCommitGate) WriteHeader(status int) {
	if g.committed {
		return
	}
	if g.status == 0 {
		g.status = status
	}
}

func (g *streamCommitGate) Write(p []byte) (int, error) {
	if g.committed {
		return g.dst.Write(p)
	}
	if g.buffer.Len()+len(p) > g.limit {
		return 0, errStreamGateOverflow
	}
	return g.buffer.Write(p)
}

func (g *streamCommitGate) Flush() {
	if !g.committed {
		return
	}
	if f, ok := g.dst.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *streamCommitGate) Commit() error {
	if g.committed {
		return nil
	}
	g.copyHeaders()
	status := g.status
	if status == 0 {
		status = http.StatusOK
	}
	g.dst.WriteHeader(status)
	g.committed = true
	if g.buffer.Len() > 0 {
		data := g.buffer.Bytes()
		n, err := g.dst.Write(data)
		if err != nil {
			return err
		}
		if n != len(data) {
			return io.ErrShortWrite
		}
	}
	g.buffer.Reset()
	g.Flush()
	return nil
}

func (g *streamCommitGate) Discard() {
	if g.committed {
		return
	}
	g.buffer.Reset()
	g.header = make(http.Header)
	g.status = 0
}

func (g *streamCommitGate) Committed() bool { return g.committed }

func (g *streamCommitGate) copyHeaders() {
	for key := range g.dst.Header() {
		g.dst.Header().Del(key)
	}
	for key, values := range g.header {
		for _, value := range values {
			g.dst.Header().Add(key, value)
		}
	}
}
