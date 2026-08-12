package proxy

import (
	"context"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func createGenerateTestConversation(t *testing.T, h *Handler, provider, model string) chatConversationDTO {
	t.Helper()
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations",
		`{"provider":"`+provider+`","model":"`+model+`"}`, "pw"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create conversation status=%d body=%s", w.Code, w.Body.String())
	}
	var conversation chatConversationDTO
	decodeChatResponse(t, w, &conversation)
	return conversation
}

func TestAdminChatGeneratePersistsHistoryUsageAndReplay(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "xai", "shared-model")
	var calls atomic.Int32
	h.chatTextExecutor = func(_ context.Context, req chatTextExecutionRequest) (chatTextExecutionResult, error) {
		calls.Add(1)
		if req.Provider != "grok" || req.Model != "shared-model" {
			t.Fatalf("identity=%s/%s", req.Provider, req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "Hello world" {
			t.Fatalf("messages=%+v", req.Messages)
		}
		return chatTextExecutionResult{
			Content: "Answer", Provider: "grok", Model: "shared-model", RequestID: req.RequestID,
			ProviderResponseID: "provider-response", InputTokens: 11, OutputTokens: 3,
			CacheReadTokens: 7, CacheCreationTokens: 2,
		}, nil
	}

	body := `{"clientRequestId":"request-1","content":"Hello world"}`
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", body, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", w.Code, w.Body.String())
	}
	var generated chatGenerateResponse
	decodeChatResponse(t, w, &generated)
	if generated.Replayed || generated.Conversation.Title != "Hello world" || generated.AssistantMessage.Content != "Answer" || generated.AssistantMessage.Status != "complete" {
		t.Fatalf("generated=%+v", generated)
	}
	if generated.AssistantMessage.InputTokens != 11 || generated.AssistantMessage.OutputTokens != 3 || generated.AssistantMessage.CacheReadTokens != 7 || generated.AssistantMessage.CacheCreationTokens != 2 || generated.AssistantMessage.ProviderResponseID != "provider-response" {
		t.Fatalf("assistant usage=%+v", generated.AssistantMessage)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", body, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", w.Code, w.Body.String())
	}
	decodeChatResponse(t, w, &generated)
	if !generated.Replayed || calls.Load() != 1 {
		t.Fatalf("replayed=%v calls=%d", generated.Replayed, calls.Load())
	}
}

func TestAdminChatGenerateIdempotencyValidationAndFailure(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")
	h.chatTextExecutor = func(_ context.Context, _ chatTextExecutionRequest) (chatTextExecutionResult, error) {
		return chatTextExecutionResult{}, errChatNoAvailableAccount
	}

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"failure","content":"prompt"}`, "pw"))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("failure status=%d body=%s", w.Code, w.Body.String())
	}
	st, unlock := h.runtimeStoreForOperation()
	page, err := st.ListChatMessages(conversation.ID, "", 10)
	unlock()
	var failedAssistant *store.ChatMessage
	for i := range page.Items {
		if page.Items[i].Role == "assistant" {
			failedAssistant = &page.Items[i]
			break
		}
	}
	if err != nil || len(page.Items) != 2 || failedAssistant == nil || failedAssistant.Status != "error" || failedAssistant.ErrorCode != "no_available_account" {
		t.Fatalf("messages=%+v err=%v", page.Items, err)
	}

	cases := []struct {
		body string
		want int
	}{
		{`{"clientRequestId":"","content":"prompt"}`, http.StatusUnprocessableEntity},
		{`{"clientRequestId":"x","content":""}`, http.StatusUnprocessableEntity},
		{`{"clientRequestId":"x","content":"prompt","provider":"grok"}`, http.StatusUnprocessableEntity},
		{`{"clientRequestId":"x","content":"prompt","unknown":true}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		w = httptest.NewRecorder()
		h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", tc.body, "pw"))
		if w.Code != tc.want {
			t.Errorf("body=%s status=%d want=%d response=%s", tc.body, w.Code, tc.want, w.Body.String())
		}
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"failure","content":"different"}`, "pw"))
	if w.Code != http.StatusConflict {
		t.Fatalf("hash conflict status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminChatGeneratePendingReplayAndCancellation(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")
	st, unlock := h.runtimeStoreForOperation()
	_, err := st.CreateChatTurn(conversation.ID, "pending",
		store.ChatMessage{ID: "pending-user", Content: "prompt", Provider: "kiro", Model: "claude", RequestHash: chatRequestHash("prompt", "kiro", "claude")},
		store.ChatMessage{ID: "pending-assistant", Provider: "kiro", Model: "claude"})
	unlock()
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"pending","content":"prompt"}`, "pw"))
	if w.Code != http.StatusConflict {
		t.Fatalf("pending status=%d body=%s", w.Code, w.Body.String())
	}

	h.chatTextExecutor = func(_ context.Context, _ chatTextExecutionRequest) (chatTextExecutionResult, error) {
		return chatTextExecutionResult{}, context.Canceled
	}
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"cancel","content":"cancel me"}`, "pw"))
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("cancel status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminChatGenerateRejectsImageForKnownNonVisionModel(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	if err := config.AddAccount(config.Account{ID: "kiro", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	h := newAdminChatTestHandler(t)
	h.modelsCacheMu.Lock()
	h.cachedModels = []ModelInfo{{ModelId: "text-only", ModelName: "Text only", InputTypes: []string{"text"}}}
	h.modelsCacheMu.Unlock()
	conversation := createGenerateTestConversation(t, h, "kiro", "text-only")

	st, unlock := h.runtimeStoreForOperation()
	attachment, err := st.CreateChatAttachment(store.ChatAttachment{
		ID: "attachment", ConversationID: conversation.ID, Kind: "image_input",
		Name: "image.png", MIMEType: "image/png", SizeBytes: 1, StorageKey: conversation.ID + "/image.png",
	})
	unlock()
	if err != nil {
		t.Fatal(err)
	}

	var called atomic.Bool
	h.chatTextExecutor = func(_ context.Context, _ chatTextExecutionRequest) (chatTextExecutionResult, error) {
		called.Store(true)
		return chatTextExecutionResult{}, nil
	}
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"no-vision","content":"describe","attachmentIds":["`+attachment.ID+`"]}`, "pw"))
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "model_capability_mismatch") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if called.Load() {
		t.Fatal("executor was called")
	}
	st, unlock = h.runtimeStoreForOperation()
	messages, err := st.ListChatMessages(conversation.ID, "", 10)
	unlock()
	if err != nil || len(messages.Items) != 0 {
		t.Fatalf("messages=%+v err=%v", messages.Items, err)
	}
}

func TestAdminChatGenerateConcurrencyLimitDoesNotPersistTurn(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatTextSemaphore = make(chan struct{}, 1)
	h.chatTextSemaphore <- struct{}{}
	conversation := createGenerateTestConversation(t, h, "kiro", "claude")

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"limited","content":"prompt"}`, "pw"))
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "concurrency_limit") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	st, unlock := h.runtimeStoreForOperation()
	messages, err := st.ListChatMessages(conversation.ID, "", 10)
	unlock()
	if err != nil || len(messages.Items) != 0 {
		t.Fatalf("messages=%+v err=%v", messages.Items, err)
	}
}

func TestChatExecutorInjectedError(t *testing.T) {
	expected := errors.New("upstream")
	h := &Handler{chatTextExecutor: func(context.Context, chatTextExecutionRequest) (chatTextExecutionResult, error) {
		return chatTextExecutionResult{}, expected
	}}
	_, err := h.executeChatText(context.Background(), chatTextExecutionRequest{})
	if !errors.Is(err, expected) {
		t.Fatalf("error=%v", err)
	}
}
