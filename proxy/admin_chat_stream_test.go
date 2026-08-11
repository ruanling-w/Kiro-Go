package proxy

import (
	"context"
	"errors"
	"kiro-go/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func chatSSEGenerateRequest(method, path, body, password string) *http.Request {
	r := chatAdminRequest(method, path, body, password)
	r.Header.Set("Accept", "text/event-stream")
	return r
}

func findAssistantMessage(t *testing.T, h *Handler, conversationID string) store.ChatMessage {
	t.Helper()
	st, unlock := h.runtimeStoreForOperation()
	page, err := st.ListChatMessages(conversationID, "", 10)
	unlock()
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range page.Items {
		if message.Role == "assistant" {
			return message
		}
	}
	t.Fatal("assistant message not found")
	return store.ChatMessage{}
}

func TestAdminChatGenerateStreamsAndPersistsCompletion(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")
	h.chatTextStreamExecutor = func(_ context.Context, req chatTextExecutionRequest, callbacks chatTextStreamCallbacks) (chatTextExecutionResult, error) {
		if err := callbacks.OnText("Reason", true); err != nil {
			return chatTextExecutionResult{}, err
		}
		if err := callbacks.OnText("Hello ", false); err != nil {
			return chatTextExecutionResult{}, err
		}
		if err := callbacks.OnText("world", false); err != nil {
			return chatTextExecutionResult{}, err
		}
		return chatTextExecutionResult{
			Content: "Hello world", Provider: "kiro", Model: "claude", RequestID: req.RequestID,
			InputTokens: 9, OutputTokens: 2, CacheReadTokens: 4, CacheCreationTokens: 1,
		}, nil
	}

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatSSEGenerateRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"stream-1","content":"Hello"}`, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type=%q", got)
	}
	body := w.Body.String()
	for _, expected := range []string{
		"event: generation.created", "event: response.reasoning_summary.delta", `{"delta":"Reason"}`,
		"event: response.delta", `{"delta":"Hello "}`, `{"delta":"world"}`,
		"event: response.completed", `"inputTokens":9`, `"cacheReadTokens":4`, "event: done",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q in stream:\n%s", expected, body)
		}
	}
	if strings.Index(body, "event: generation.created") > strings.Index(body, "event: response.delta") || strings.Index(body, "event: response.completed") > strings.Index(body, "event: done") {
		t.Fatalf("incorrect event order:\n%s", body)
	}
	assistant := findAssistantMessage(t, h, conversation.ID)
	if assistant.Status != "complete" || assistant.Content != "Hello world" || assistant.InputTokens != 9 || assistant.CacheReadTokens != 4 {
		t.Fatalf("assistant=%+v", assistant)
	}
}

func TestAdminChatGenerateStreamPersistsPartialError(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")
	h.chatTextStreamExecutor = func(_ context.Context, req chatTextExecutionRequest, callbacks chatTextStreamCallbacks) (chatTextExecutionResult, error) {
		if err := callbacks.OnText("partial", false); err != nil {
			return chatTextExecutionResult{}, err
		}
		return chatTextExecutionResult{Content: "partial", Provider: "kiro", Model: "claude", RequestID: req.RequestID}, errors.New("upstream failed")
	}

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatSSEGenerateRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"stream-error","content":"Hello"}`, "pw"))
	body := w.Body.String()
	if !strings.Contains(body, `event: response.error`) || !strings.Contains(body, `"code":"provider_error"`) || !strings.Contains(body, "event: done") {
		t.Fatalf("stream:\n%s", body)
	}
	assistant := findAssistantMessage(t, h, conversation.ID)
	if assistant.Status != "error" || assistant.Content != "partial" || assistant.ErrorCode != "provider_error" {
		t.Fatalf("assistant=%+v", assistant)
	}
}

func TestAdminChatGenerateStreamReplayDoesNotExecute(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")
	calls := 0
	h.chatTextStreamExecutor = func(_ context.Context, req chatTextExecutionRequest, callbacks chatTextStreamCallbacks) (chatTextExecutionResult, error) {
		calls++
		if err := callbacks.OnText("answer", false); err != nil {
			return chatTextExecutionResult{}, err
		}
		return chatTextExecutionResult{Content: "answer", Provider: "kiro", Model: "claude", RequestID: req.RequestID}, nil
	}
	path := "/admin/api/chat/conversations/" + conversation.ID + "/generate"
	body := `{"clientRequestId":"stream-replay","content":"Hello"}`
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.handleAdminAPI(w, chatSSEGenerateRequest(http.MethodPost, path, body, "pw"))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "event: response.completed") {
			t.Fatalf("iteration=%d status=%d stream=%s", i, w.Code, w.Body.String())
		}
		if i == 1 && !strings.Contains(w.Body.String(), `"replayed":true`) {
			t.Fatalf("replay marker missing: %s", w.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("calls=%d want=1", calls)
	}
}
