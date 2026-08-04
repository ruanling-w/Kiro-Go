package proxy

import (
	"context"
	"net/http"
	"strings"
)

type responsesComboSink struct{ gate *streamCommitGate }

func (s *responsesComboSink) Header() http.Header  { return s.gate.Header() }
func (s *responsesComboSink) WriteHeader(code int) { s.gate.WriteHeader(code) }
func (s *responsesComboSink) Flush()               { s.gate.Flush() }
func (s *responsesComboSink) Write(p []byte) (int, error) {
	if s.gate.Committed() {
		return s.gate.Write(p)
	}
	n, err := s.gate.Write(p)
	if err != nil {
		return n, err
	}
	text := string(p)
	if strings.Contains(text, "response.output_text.delta") || strings.Contains(text, "response.function_call_arguments.delta") {
		if err := s.gate.Commit(); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// Responses Combo attempts are isolated behind a first-public-event gate. The
// existing Responses executor supplies protocol event conversion and terminal
// response.failed handling; after the first output event the gate pins it.
func (h *Handler) handleResponsesComboStream(ctx context.Context, w http.ResponseWriter, original *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimatedInputTokens int, apiKeyID, clientIP, respID string, req *ResponsesRequest, storedInput []byte, storeResponse bool) {
	if _, ok := w.(http.Flusher); !ok {
		h.sendOpenAIError(w, 500, "server_error", "Streaming not supported")
		return
	}
	for _, candidate := range route.Candidates {
		attempt := *original
		attempt.Model = candidate.Model
		gate := newStreamCommitGate(w)
		sink := &responsesComboSink{gate: gate}
		h.handleResponsesStreamModels(ctx, sink, OpenAIToKiro(&attempt, thinking), candidate.Model, route.RequestedModel, true, thinking, estimatedInputTokens, apiKeyID, clientIP, respID, req, storedInput, storeResponse)
		if gate.Committed() {
			return
		}
		gate.Discard()
	}
	h.sendOpenAIError(w, http.StatusBadGateway, "server_error", "all Combo streaming candidates failed")
}
