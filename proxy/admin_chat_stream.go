package proxy

import (
	"context"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type chatSSEGenerationCreated struct {
	GenerationID       string `json:"generationId"`
	UserMessageID      string `json:"userMessageId"`
	AssistantMessageID string `json:"assistantMessageId"`
	Replayed           bool   `json:"replayed,omitempty"`
}

type chatSSECompleted struct {
	FinishReason string       `json:"finishReason"`
	Provider     string       `json:"provider"`
	Model        string       `json:"model"`
	Usage        chatSSEUsage `json:"usage"`
}

type chatSSEUsage struct {
	InputTokens         int `json:"inputTokens"`
	OutputTokens        int `json:"outputTokens"`
	CacheReadTokens     int `json:"cacheReadTokens"`
	CacheCreationTokens int `json:"cacheCreationTokens"`
}

type chatSSEError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func acceptsChatSSE(r *http.Request) bool {
	for _, value := range strings.Split(r.Header.Get("Accept"), ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]), "text/event-stream") {
			return true
		}
	}
	return false
}

func prepareChatSSE(w http.ResponseWriter) (*chatSSEWriter, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		chatAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "streaming is not supported by this server")
		return nil, false
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	return newChatSSEWriter(w, flusher), true
}

func chatSSECreated(turn store.ChatTurn, replayed bool) chatSSEGenerationCreated {
	return chatSSEGenerationCreated{
		GenerationID: turn.Assistant.ID, UserMessageID: turn.User.ID,
		AssistantMessageID: turn.Assistant.ID, Replayed: replayed,
	}
}

func chatSSECompletion(message store.ChatMessage) chatSSECompleted {
	return chatSSECompleted{
		FinishReason: "stop", Provider: message.Provider, Model: message.Model,
		Usage: chatSSEUsage{
			InputTokens: message.InputTokens, OutputTokens: message.OutputTokens,
			CacheReadTokens: message.CacheReadTokens, CacheCreationTokens: message.CacheCreationTokens,
		},
	}
}

func (h *Handler) apiReplayChatTurnSSE(w http.ResponseWriter, conversation store.ChatConversation, turn store.ChatTurn) {
	if turn.Assistant.Status == "pending" || turn.Assistant.Status == "streaming" {
		chatAPIError(w, http.StatusConflict, "generation_in_progress", "generation is already in progress")
		return
	}
	stream, ok := prepareChatSSE(w)
	if !ok {
		return
	}
	if stream.Event("generation.created", chatSSECreated(turn, true)) != nil {
		return
	}
	if turn.Assistant.Content != "" {
		if stream.Event("response.delta", map[string]string{"delta": turn.Assistant.Content}) != nil {
			return
		}
	}
	if turn.Assistant.Status == "complete" {
		_ = stream.Terminal("response.completed", chatSSECompletion(turn.Assistant))
	} else {
		_ = stream.Terminal("response.error", chatSSEError{Code: turn.Assistant.ErrorCode, Message: turn.Assistant.ErrorMessage, Retryable: false})
	}
	_ = stream.Done()
	_ = conversation
}

func (h *Handler) apiStreamChatTurn(w http.ResponseWriter, r *http.Request, conversation store.ChatConversation, turn store.ChatTurn, messages []OpenAIMessage, req chatGenerateRequest) {
	stream, ok := prepareChatSSE(w)
	if !ok {
		h.finalizeUnstartedChatStream(turn.Assistant, "streaming_unsupported", "streaming is not supported by this server")
		return
	}
	if err := stream.Event("generation.created", chatSSECreated(turn, false)); err != nil {
		h.finalizeStoppedChatStream(turn.Assistant, "", uuid.NewString())
		return
	}

	requestID := uuid.NewString()
	if !acquireChatSemaphore(r.Context(), h.chatTextSemaphore) {
		h.finalizeUnstartedChatStream(turn.Assistant, "concurrency_limit", "too many chat generations are in progress")
		_ = stream.Terminal("response.error", chatSSEError{Code: "concurrency_limit", Message: "too many chat generations are in progress", Retryable: true})
		_ = stream.Done()
		return
	}
	defer releaseChatSemaphore(h.chatTextSemaphore)
	result, executionErr := h.executeChatTextStream(r.Context(), chatTextExecutionRequest{
		Messages: messages, Provider: req.Provider, Model: req.Model,
		ClientIP: ClientIP(r, config.GetTrustProxyHeaders()), RequestID: requestID,
	}, chatTextStreamCallbacks{
		OnText: func(text string, thinking bool) error {
			event := "response.delta"
			if thinking {
				event = "response.reasoning_summary.delta"
			}
			return stream.Event(event, map[string]string{"delta": text})
		},
	})

	assistant := turn.Assistant
	assistant.Content = result.Content
	assistant.RequestID = requestID
	if result.Provider != "" {
		assistant.Provider, assistant.Model = result.Provider, result.Model
	}
	assistant.ProviderResponseID = result.ProviderResponseID
	assistant.InputTokens, assistant.OutputTokens = result.InputTokens, result.OutputTokens
	assistant.CacheReadTokens, assistant.CacheCreationTokens = result.CacheReadTokens, result.CacheCreationTokens
	if executionErr == nil {
		assistant.Status = "complete"
	} else {
		assistant.Status, assistant.ErrorCode, assistant.ErrorMessage = chatStreamFailure(executionErr, stream.Err() != nil)
	}

	finalized, updatedConversation, finalizeErr := h.finalizeChatStream(assistant, conversation, req.Content, executionErr == nil)
	if finalizeErr != nil {
		if stream.Err() == nil {
			_ = stream.Terminal("response.error", chatSSEError{Code: "persistence_error", Message: "generation could not be saved", Retryable: true})
			_ = stream.Done()
		}
		return
	}
	_ = updatedConversation
	if stream.Err() != nil {
		return
	}
	if executionErr == nil {
		_ = stream.Terminal("response.completed", chatSSECompletion(finalized))
	} else {
		_ = stream.Terminal("response.error", chatSSEError{Code: finalized.ErrorCode, Message: finalized.ErrorMessage, Retryable: finalized.Status == "error"})
	}
	_ = stream.Done()
}

func chatStreamFailure(err error, writeFailed bool) (status, code, message string) {
	if writeFailed || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "stopped", "generation_stopped", "generation was stopped"
	}
	if errors.Is(err, errChatNoAvailableAccount) {
		return "error", "no_available_account", "no account is available for this provider and model"
	}
	if errors.Is(err, errChatUnsupportedOutput) {
		return "error", "unsupported_output", "the selected route did not return supported text output"
	}
	return "error", "provider_error", "provider request failed"
}

func (h *Handler) finalizeChatStream(assistant store.ChatMessage, conversation store.ChatConversation, content string, updateTitle bool) (store.ChatMessage, store.ChatConversation, error) {
	st, unlock := h.runtimeStoreForOperation()
	defer unlock()
	finalized, err := st.FinalizeChatMessage(assistant)
	if err != nil {
		return store.ChatMessage{}, conversation, err
	}
	if updateTitle && conversation.Title == "" {
		conversation.Title = chatAutoTitle(content)
		conversation, err = st.UpdateChatConversation(conversation)
	}
	return finalized, conversation, err
}

func (h *Handler) finalizeStoppedChatStream(assistant store.ChatMessage, content, requestID string) {
	assistant.Status, assistant.Content, assistant.RequestID = "stopped", content, requestID
	assistant.ErrorCode, assistant.ErrorMessage = "generation_stopped", "generation was stopped"
	st, unlock := h.runtimeStoreForOperation()
	_, _ = st.FinalizeChatMessage(assistant)
	unlock()
}

func (h *Handler) finalizeUnstartedChatStream(assistant store.ChatMessage, code, message string) {
	assistant.Status, assistant.ErrorCode, assistant.ErrorMessage = "error", code, message
	st, unlock := h.runtimeStoreForOperation()
	_, _ = st.FinalizeChatMessage(assistant)
	unlock()
}
