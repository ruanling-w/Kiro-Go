package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	chatClientRequestIDMaxBytes = 128
	chatContentMaxBytes         = 1 << 20
)

type chatGenerateRequest struct {
	ClientRequestID string `json:"clientRequestId"`
	Content         string `json:"content"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
}

type chatGenerateResponse struct {
	Conversation     chatConversationDTO `json:"conversation"`
	UserMessage      chatMessageDTO      `json:"userMessage"`
	AssistantMessage chatMessageDTO      `json:"assistantMessage"`
	Replayed         bool                `json:"replayed"`
}

func normalizeChatProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "xai", "grok":
		return "grok"
	case "codex":
		return "codex"
	case "antigravity":
		return "antigravity"
	case "remotekiro":
		return "remotekiro"
	case "combo":
		return "combo"
	case "", "kiro":
		return "kiro"
	default:
		return ""
	}
}

func validateChatGenerateRequest(req *chatGenerateRequest, conversation store.ChatConversation) []comboFieldError {
	req.ClientRequestID = strings.TrimSpace(req.ClientRequestID)
	req.Content = strings.TrimSpace(req.Content)
	req.Provider = strings.TrimSpace(req.Provider)
	req.Model = strings.TrimSpace(req.Model)
	var fields []comboFieldError
	if req.ClientRequestID == "" || len(req.ClientRequestID) > chatClientRequestIDMaxBytes {
		fields = append(fields, comboFieldError{"clientRequestId", "is required and must be at most 128 bytes"})
	}
	if req.Content == "" || len(req.Content) > chatContentMaxBytes || !utf8.ValidString(req.Content) {
		fields = append(fields, comboFieldError{"content", "is required, must be valid UTF-8, and must be at most 1 MiB"})
	}
	if (req.Provider == "") != (req.Model == "") {
		fields = append(fields, comboFieldError{"provider", "provider and model must be supplied together"})
	}
	if req.Provider == "" && req.Model == "" {
		req.Provider, req.Model = conversation.Provider, conversation.Model
	}
	req.Provider = normalizeChatProvider(req.Provider)
	if req.Provider == "" {
		fields = append(fields, comboFieldError{"provider", "is unknown"})
	}
	if req.Model == "" || len(req.Model) > chatModelMaxBytes {
		fields = append(fields, comboFieldError{"model", "is required and must be at most 256 bytes"})
	}
	return fields
}

func chatRequestHash(content, provider, model string) string {
	data, _ := json.Marshal([]string{content, provider, model})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func chatHistoryMessages(history []store.ChatMessage, content string) []OpenAIMessage {
	messages := make([]OpenAIMessage, 0, len(history)+1)
	for _, message := range history {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		messages = append(messages, OpenAIMessage{Role: message.Role, Content: message.Content})
	}
	messages = append(messages, OpenAIMessage{Role: "user", Content: content})
	return messages
}

func chatAutoTitle(content string) string {
	const maxRunes = 80
	content = strings.Join(strings.Fields(content), " ")
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "…"
}

func (h *Handler) apiGenerateChatConversation(w http.ResponseWriter, r *http.Request, conversationID string) {
	var req chatGenerateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		chatAPIError(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}

	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	conversation, err := st.GetChatConversation(conversationID)
	if err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	if conversation.Status != "active" {
		unlock()
		chatAPIError(w, http.StatusConflict, "conversation_archived", "conversation is archived")
		return
	}
	if conversation.Mode != "chat" {
		unlock()
		chatAPIError(w, http.StatusConflict, "unsupported_conversation_mode", "conversation does not support text generation")
		return
	}
	if fields := validateChatGenerateRequest(&req, conversation); len(fields) > 0 {
		unlock()
		comboError(w, http.StatusUnprocessableEntity, "validation failed", fields)
		return
	}
	history, err := st.ListCompletedChatMessages(conversationID)
	if err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	turn, err := st.CreateChatTurn(conversationID, req.ClientRequestID,
		store.ChatMessage{ID: uuid.NewString(), Content: req.Content, Provider: req.Provider, Model: req.Model, RequestHash: chatRequestHash(req.Content, req.Provider, req.Model)},
		store.ChatMessage{ID: uuid.NewString(), Provider: req.Provider, Model: req.Model})
	unlock()
	if err != nil {
		if errors.Is(err, store.ErrChatConflict) {
			chatAPIError(w, http.StatusConflict, "idempotency_conflict", "client request ID was already used for a different request")
			return
		}
		writeChatStoreError(w, err)
		return
	}
	if !turn.Created {
		h.apiReplayChatTurn(w, conversation, turn)
		return
	}

	requestID := uuid.NewString()
	result, executionErr := h.executeChatText(r.Context(), chatTextExecutionRequest{
		Messages: chatHistoryMessages(history, req.Content), Provider: req.Provider, Model: req.Model,
		ClientIP: ClientIP(r, config.GetTrustProxyHeaders()), RequestID: requestID,
	})
	assistant := turn.Assistant
	assistant.RequestID = requestID
	status := http.StatusOK
	if executionErr == nil {
		assistant.Status = "complete"
		assistant.Content = result.Content
		assistant.Provider, assistant.Model = result.Provider, result.Model
		assistant.ProviderResponseID = result.ProviderResponseID
		assistant.RequestID = result.RequestID
		assistant.InputTokens, assistant.OutputTokens = result.InputTokens, result.OutputTokens
		assistant.CacheReadTokens, assistant.CacheCreationTokens = result.CacheReadTokens, result.CacheCreationTokens
	} else {
		assistant.Status = "error"
		assistant.ErrorCode = "provider_error"
		assistant.ErrorMessage = "provider request failed"
		status = http.StatusBadGateway
		if errors.Is(executionErr, context.Canceled) || errors.Is(executionErr, context.DeadlineExceeded) {
			assistant.Status = "stopped"
			assistant.ErrorCode = "generation_stopped"
			assistant.ErrorMessage = "generation was stopped"
			status = http.StatusRequestTimeout
		} else if errors.Is(executionErr, errChatNoAvailableAccount) {
			assistant.ErrorCode = "no_available_account"
			assistant.ErrorMessage = "no account is available for this provider and model"
			status = http.StatusServiceUnavailable
		} else if errors.Is(executionErr, errChatUnsupportedOutput) {
			assistant.ErrorCode = "unsupported_output"
			assistant.ErrorMessage = "the selected route did not return supported text output"
			status = http.StatusUnprocessableEntity
		}
	}

	st, unlock, ok = h.chatStore(w)
	if !ok {
		return
	}
	finalized, finalizeErr := st.FinalizeChatMessage(assistant)
	if finalizeErr == nil && executionErr == nil && conversation.Title == "" {
		conversation.Title = chatAutoTitle(req.Content)
		conversation, finalizeErr = st.UpdateChatConversation(conversation)
	}
	unlock()
	if finalizeErr != nil {
		writeChatStoreError(w, finalizeErr)
		return
	}
	if executionErr != nil {
		chatAPIError(w, status, assistant.ErrorCode, assistant.ErrorMessage)
		return
	}
	_ = json.NewEncoder(w).Encode(chatGenerateResponse{
		Conversation: chatConversationFromStore(conversation), UserMessage: chatMessageFromStore(turn.User),
		AssistantMessage: chatMessageFromStore(finalized), Replayed: false,
	})
}

func (h *Handler) apiReplayChatTurn(w http.ResponseWriter, conversation store.ChatConversation, turn store.ChatTurn) {
	if turn.Assistant.Status == "pending" || turn.Assistant.Status == "streaming" {
		chatAPIError(w, http.StatusConflict, "generation_in_progress", "generation is already in progress")
		return
	}
	_ = json.NewEncoder(w).Encode(chatGenerateResponse{
		Conversation: chatConversationFromStore(conversation), UserMessage: chatMessageFromStore(turn.User),
		AssistantMessage: chatMessageFromStore(turn.Assistant), Replayed: true,
	})
}
