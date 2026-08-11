package proxy

import (
	"bytes"
	"errors"
	"testing"
)

type chatSSETestFlusher struct{ calls int }

func (f *chatSSETestFlusher) Flush() { f.calls++ }

type chatSSEFailWriter struct{ err error }

func (w chatSSEFailWriter) Write([]byte) (int, error) { return 0, w.err }

func TestChatSSEWriterEventsAndTerminalGuard(t *testing.T) {
	var output bytes.Buffer
	flusher := &chatSSETestFlusher{}
	writer := newChatSSEWriter(&output, flusher)

	if err := writer.Event("response.delta", map[string]string{"delta": "line\n\"quoted\""}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Terminal("response.completed", map[string]string{"finishReason": "stop"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Event("response.delta", map[string]string{"delta": "ignored"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Done(); err != nil {
		t.Fatal(err)
	}

	want := "event: response.delta\ndata: {\"delta\":\"line\\n\\\"quoted\\\"\"}\n\n" +
		"event: response.completed\ndata: {\"finishReason\":\"stop\"}\n\n" +
		"event: done\ndata: {}\n\n"
	if output.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
	}
	if flusher.calls != 3 {
		t.Fatalf("flush calls=%d want=3", flusher.calls)
	}
}

func TestChatSSEWriterKeepsWriteError(t *testing.T) {
	expected := errors.New("broken connection")
	writer := newChatSSEWriter(chatSSEFailWriter{err: expected}, &chatSSETestFlusher{})
	if err := writer.Event("response.delta", map[string]string{"delta": "x"}); !errors.Is(err, expected) {
		t.Fatalf("first error=%v", err)
	}
	if err := writer.Done(); !errors.Is(err, expected) {
		t.Fatalf("sticky error=%v", err)
	}
	if !errors.Is(writer.Err(), expected) {
		t.Fatalf("Err=%v", writer.Err())
	}
}
