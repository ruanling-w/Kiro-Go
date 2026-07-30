package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type openAISSEWriter struct {
	writer     io.Writer
	flusher    http.Flusher
	err        error
	terminated bool
}

func newOpenAISSEWriter(writer io.Writer, flusher http.Flusher) *openAISSEWriter {
	return &openAISSEWriter{writer: writer, flusher: flusher}
}

func (s *openAISSEWriter) JSON(value any) error {
	if s.err != nil || s.terminated {
		return s.err
	}
	data, err := json.Marshal(value)
	if err != nil {
		s.err = err
		return err
	}
	if _, err := fmt.Fprintf(s.writer, "data: %s\n\n", data); err != nil {
		s.err = err
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *openAISSEWriter) Done() error {
	if s.terminated {
		return s.err
	}
	if s.err != nil {
		return s.err
	}
	if _, err := fmt.Fprint(s.writer, "data: [DONE]\n\n"); err != nil {
		s.err = err
		return err
	}
	s.terminated = true
	s.flusher.Flush()
	return nil
}

func (s *openAISSEWriter) Err() error { return s.err }
