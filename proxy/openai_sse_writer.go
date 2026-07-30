package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type openAISSEWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func newOpenAISSEWriter(writer io.Writer, flusher http.Flusher) *openAISSEWriter {
	return &openAISSEWriter{writer: writer, flusher: flusher}
}

func (s *openAISSEWriter) JSON(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.writer, "data: %s\n\n", data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *openAISSEWriter) Done() error {
	if _, err := fmt.Fprint(s.writer, "data: [DONE]\n\n"); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
