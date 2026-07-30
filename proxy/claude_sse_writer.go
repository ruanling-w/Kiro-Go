package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type claudeSSEWriter struct {
	writer  io.Writer
	flusher http.Flusher
	err     error
}

func newClaudeSSEWriter(writer io.Writer, flusher http.Flusher) *claudeSSEWriter {
	return &claudeSSEWriter{writer: writer, flusher: flusher}
}

func (s *claudeSSEWriter) Send(event string, value any) error {
	if s.err != nil {
		return s.err
	}
	data, err := json.Marshal(value)
	if err != nil {
		s.err = err
		return err
	}
	if _, err := fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		s.err = err
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *claudeSSEWriter) Err() error { return s.err }
