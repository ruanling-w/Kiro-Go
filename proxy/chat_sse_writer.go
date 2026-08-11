package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type chatSSEWriter struct {
	writer     io.Writer
	flusher    http.Flusher
	err        error
	terminated bool
}

func newChatSSEWriter(writer io.Writer, flusher http.Flusher) *chatSSEWriter {
	return &chatSSEWriter{writer: writer, flusher: flusher}
}

func (s *chatSSEWriter) Event(event string, value any) error {
	if s.err != nil || s.terminated {
		return s.err
	}
	data, err := json.Marshal(value)
	if err != nil {
		s.err = err
		return err
	}
	if _, err = fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		s.err = err
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *chatSSEWriter) Terminal(event string, value any) error {
	if s.terminated {
		return s.err
	}
	if err := s.Event(event, value); err != nil {
		return err
	}
	s.terminated = true
	return nil
}

func (s *chatSSEWriter) Done() error {
	if s.err != nil {
		return s.err
	}
	data, err := json.Marshal(struct{}{})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.writer, "event: done\ndata: %s\n\n", data)
	if err != nil {
		s.err = err
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *chatSSEWriter) Err() error { return s.err }
