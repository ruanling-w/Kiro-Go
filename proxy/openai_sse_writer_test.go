package proxy

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAISSEWriterJSONAndDone(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := newOpenAISSEWriter(rec, rec)
	if err := writer.JSON(map[string]string{"content": "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Done(); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data: {"content":"hello"}`) || !strings.HasSuffix(body, "data: [DONE]\n\n") {
		t.Fatalf("body=%q", body)
	}
}

func TestOpenAISSEWriterStagesThroughCommitGate(t *testing.T) {
	rec := httptest.NewRecorder()
	gate := newStreamCommitGate(rec)
	writer := newOpenAISSEWriter(gate, gate)
	if err := writer.JSON(map[string]string{"content": "hidden"}); err != nil {
		t.Fatal(err)
	}
	if rec.Body.Len() != 0 {
		t.Fatal("SSE leaked before commit")
	}
	gate.Discard()
	if err := writer.JSON(map[string]string{"content": "visible"}); err != nil {
		t.Fatal(err)
	}
	if err := gate.Commit(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rec.Body.String(), "hidden") || !strings.Contains(rec.Body.String(), "visible") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}
