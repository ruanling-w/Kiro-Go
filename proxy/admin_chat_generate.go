package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	chatClientRequestIDMaxBytes = 128
	chatContentMaxBytes         = 1 << 20
)

type chatGenerateRequest struct {
	ClientRequestID string   `json:"clientRequestId"`
	Content         string   `json:"content"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	AttachmentIDs   []string `json:"attachmentIds"`
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
	if req.Content == "" && len(req.AttachmentIDs) == 0 {
		fields = append(fields, comboFieldError{"content", "content or at least one attachment is required"})
	} else if len(req.Content) > chatContentMaxBytes || !utf8.ValidString(req.Content) {
		fields = append(fields, comboFieldError{"content", "must be valid UTF-8 and at most 1 MiB"})
	}
	if len(req.AttachmentIDs) > chatAttachmentMaxFiles {
		fields = append(fields, comboFieldError{"attachmentIds", "must contain at most four attachments"})
	}
	seenAttachments := make(map[string]bool, len(req.AttachmentIDs))
	for i := range req.AttachmentIDs {
		req.AttachmentIDs[i] = strings.TrimSpace(req.AttachmentIDs[i])
		id := req.AttachmentIDs[i]
		if id == "" || seenAttachments[id] {
			fields = append(fields, comboFieldError{"attachmentIds", "must contain unique non-empty IDs"})
			break
		}
		seenAttachments[id] = true
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

func chatRequestHash(content, provider, model string, attachmentIDs ...string) string {
	ids := append([]string(nil), attachmentIDs...)
	sort.Strings(ids)
	data, _ := json.Marshal(struct {
		Content, Provider, Model string
		AttachmentIDs            []string
	}{content, provider, model, ids})
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

func (h *Handler) chatMessagesWithAttachments(history []store.ChatMessage, current store.ChatMessage, attachments []store.ChatAttachment) ([]OpenAIMessage, error) {
	messages := make([]OpenAIMessage, 0, len(history)+1)
	messageIDs := make([]string, 0, len(history)+1)
	for _, message := range history {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		messages = append(messages, OpenAIMessage{Role: message.Role, Content: message.Content})
		messageIDs = append(messageIDs, message.ID)
	}
	messages = append(messages, OpenAIMessage{Role: "user", Content: current.Content})
	messageIDs = append(messageIDs, current.ID)

	byMessage := make(map[string][]store.ChatAttachment)
	for _, attachment := range attachments {
		if attachment.Kind == "image_input" && attachment.MessageID != "" {
			byMessage[attachment.MessageID] = append(byMessage[attachment.MessageID], attachment)
		}
	}
	if len(byMessage) == 0 {
		return messages, nil
	}
	assets, err := h.chatAssets()
	if err != nil {
		return nil, err
	}
	for i, messageID := range messageIDs {
		messageAttachments := byMessage[messageID]
		if len(messageAttachments) == 0 || messages[i].Role != "user" {
			continue
		}
		content, _ := messages[i].Content.(string)
		parts := make([]map[string]any, 0, len(messageAttachments)+1)
		if content != "" {
			parts = append(parts, map[string]any{"type": "text", "text": content})
		}
		for _, attachment := range messageAttachments {
			path, pathErr := assets.path(attachment.StorageKey)
			if pathErr != nil {
				return nil, pathErr
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || int64(len(data)) != attachment.SizeBytes || len(data) > chatAttachmentMaxBytes {
				return nil, errInvalidChatImage
			}
			parts = append(parts, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": "data:" + attachment.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(data)},
			})
		}
		messages[i].Content = parts
	}
	return messages, nil
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
	if len(req.AttachmentIDs) > 0 {
		vision, known := h.chatModelVisionCapability(req.Provider, req.Model)
		if known && !vision {
			unlock()
			chatAPIError(w, http.StatusUnprocessableEntity, "model_capability_mismatch", "the selected model does not support image input")
			return
		}
	}
	history, err := st.ListCompletedChatMessages(conversationID)
	if err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	allAttachments, err := st.ListChatAttachments(conversationID)
	if err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	turn, err := st.CreateChatTurnWithAttachments(conversationID, req.ClientRequestID,
		store.ChatMessage{ID: uuid.NewString(), Content: req.Content, Provider: req.Provider, Model: req.Model, RequestHash: chatRequestHash(req.Content, req.Provider, req.Model, req.AttachmentIDs...)},
		store.ChatMessage{ID: uuid.NewString(), Provider: req.Provider, Model: req.Model}, req.AttachmentIDs)
	var attachments []store.ChatAttachment
	if err == nil && turn.Created {
		for _, attachment := range allAttachments {
			if attachment.MessageID != "" {
				attachments = append(attachments, attachment)
				continue
			}
			for _, attachmentID := range req.AttachmentIDs {
				if attachment.ID == attachmentID {
					attachment.MessageID = turn.User.ID
					attachments = append(attachments, attachment)
					break
				}
			}
		}
	}
	unlock()
	if err != nil {
		if turn.Created {
			st, release, available := h.chatStore(w)
			if available {
				_ = st.AbortChatTurn(turn.User.ID)
				release()
			}
		}
		if errors.Is(err, store.ErrChatConflict) {
			chatAPIError(w, http.StatusConflict, "idempotency_conflict", "client request ID was already used for a different request or an attachment is already bound")
			return
		}
		writeChatStoreError(w, err)
		return
	}
	if !turn.Created {
		if acceptsChatSSE(r) {
			h.apiReplayChatTurnSSE(w, conversation, turn)
		} else {
			h.apiReplayChatTurn(w, conversation, turn)
		}
		return
	}
	messages, err := h.chatMessagesWithAttachments(history, turn.User, attachments)
	if err != nil {
		st, release, available := h.chatStore(w)
		if available {
			_ = st.AbortChatTurn(turn.User.ID)
			release()
		}
		chatAPIError(w, http.StatusUnprocessableEntity, "attachment_unavailable", "an attachment could not be read")
		return
	}
	if acceptsChatSSE(r) {
		h.apiStreamChatTurn(w, r, conversation, turn, messages, req)
		return
	}

	requestID := uuid.NewString()
	if !acquireChatSemaphore(r.Context(), h.chatTextSemaphore) {
		st, release, available := h.chatStore(w)
		if available {
			_ = st.AbortChatTurn(turn.User.ID)
			release()
		}
		chatAPIError(w, http.StatusTooManyRequests, "concurrency_limit", "too many chat generations are in progress")
		return
	}
	defer releaseChatSemaphore(h.chatTextSemaphore)
	result, executionErr := h.executeChatText(r.Context(), chatTextExecutionRequest{
		Messages: messages, Provider: req.Provider, Model: req.Model,
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
