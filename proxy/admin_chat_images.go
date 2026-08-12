package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

const chatImagePromptMaxBytes = 1 << 16

type chatImageGenerateRequest struct {
	ClientRequestID string `json:"clientRequestId"`
	Prompt          string `json:"prompt"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	Size            string `json:"size,omitempty"`
	Quality         string `json:"quality,omitempty"`
}

type chatImageExecutionResult struct {
	Base64   string
	MIMEType string
	Provider string
	Model    string
}

type chatImageExecutionRequest struct {
	Provider string
	Model    string
	Prompt   string
	Size     string
	Quality  string
}

func validateChatImageGenerateRequest(req *chatImageGenerateRequest, conversation store.ChatConversation) []comboFieldError {
	req.ClientRequestID = strings.TrimSpace(req.ClientRequestID)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Provider = strings.TrimSpace(req.Provider)
	req.Model = strings.TrimSpace(req.Model)
	req.Size = strings.TrimSpace(req.Size)
	req.Quality = strings.TrimSpace(req.Quality)
	var fields []comboFieldError
	if req.ClientRequestID == "" || len(req.ClientRequestID) > chatClientRequestIDMaxBytes {
		fields = append(fields, comboFieldError{"clientRequestId", "is required and must be at most 128 bytes"})
	}
	if req.Prompt == "" || len(req.Prompt) > chatImagePromptMaxBytes {
		fields = append(fields, comboFieldError{"prompt", "is required and must be at most 64 KiB"})
	}
	if req.Provider == "" && req.Model == "" {
		req.Provider, req.Model = conversation.Provider, conversation.Model
	}
	req.Provider = normalizeChatProvider(req.Provider)
	if req.Provider == "" || req.Model == "" {
		fields = append(fields, comboFieldError{"provider", "provider and model are required"})
	} else if !chatImageModelForProvider(req.Provider, req.Model) {
		fields = append(fields, comboFieldError{"model", "model does not support image generation for this provider"})
	}
	if req.Size != "" && req.Size != "1024x1024" && req.Size != "1536x1024" && req.Size != "1024x1536" && req.Size != "auto" {
		fields = append(fields, comboFieldError{"size", "is not supported"})
	}
	if req.Quality != "" && req.Quality != "low" && req.Quality != "medium" && req.Quality != "high" && req.Quality != "auto" {
		fields = append(fields, comboFieldError{"quality", "is not supported"})
	}
	return fields
}

func chatImageModelForProvider(provider, model string) bool {
	switch provider {
	case "grok":
		return isGrokImageModel(model)
	case "antigravity":
		return isAntigravityImageModel(model)
	case "codex":
		return isCodexImageModel(model)
	default:
		return false
	}
}

func (h *Handler) executeChatImage(ctx context.Context, req chatImageExecutionRequest) (chatImageExecutionResult, error) {
	if h.chatImageExecutor != nil {
		return h.chatImageExecutor(ctx, req)
	}
	var want func(*config.Account) bool
	var call func(context.Context, *config.Account, *CodexImageRequest) (string, string, error)
	switch req.Provider {
	case "grok":
		want, call = isGrokAccount, CallGrokImageAPI
	case "antigravity":
		want, call = isAntigravityAccount, CallAntigravityImageAPI
	case "codex":
		want, call = isCodexAccount, CallCodexImageAPI
	default:
		return chatImageExecutionResult{}, errors.New("unsupported image provider")
	}
	upstream := &CodexImageRequest{Model: req.Model, Prompt: req.Prompt, N: 1, Size: req.Size, Quality: req.Quality, OutputFormat: "png"}
	excluded := make(map[string]bool)
	var lastErr error
	for attempt := 0; attempt < maxAccountRetryAttempts; attempt++ {
		account := h.pool.GetNextForModelExcluding(req.Model, excluded)
		if account == nil {
			break
		}
		if !want(account) {
			excluded[account.ID] = true
			continue
		}
		if err := h.ensureValidToken(account); err != nil {
			lastErr = err
			excluded[account.ID] = true
			h.handleAccountFailure(account, err)
			continue
		}
		b64, mimeType, err := call(ctx, account, upstream)
		if err == nil {
			return chatImageExecutionResult{Base64: b64, MIMEType: mimeType, Provider: req.Provider, Model: req.Model}, nil
		}
		lastErr = err
		excluded[account.ID] = true
		h.handleAccountFailure(account, err)
	}
	if lastErr == nil {
		lastErr = errors.New("no available account for image generation")
	}
	return chatImageExecutionResult{}, lastErr
}

func (h *Handler) apiGenerateChatImage(w http.ResponseWriter, r *http.Request, conversationID string) {
	var req chatImageGenerateRequest
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
	if fields := validateChatImageGenerateRequest(&req, conversation); len(fields) > 0 {
		unlock()
		comboError(w, http.StatusUnprocessableEntity, "validation failed", fields)
		return
	}
	hash := chatRequestHash(req.Prompt, req.Provider, req.Model, req.Size, req.Quality)
	turn, err := st.CreateChatTurn(conversationID, req.ClientRequestID,
		store.ChatMessage{ID: uuid.NewString(), Content: req.Prompt, Provider: req.Provider, Model: req.Model, RequestHash: hash},
		store.ChatMessage{ID: uuid.NewString(), Provider: req.Provider, Model: req.Model})
	unlock()
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	if !turn.Created {
		st, release, available := h.chatStore(w)
		if !available {
			return
		}
		attachments, listErr := st.ListChatAttachments(conversationID)
		release()
		if listErr != nil {
			writeChatStoreError(w, listErr)
			return
		}
		data := make([]chatAttachmentDTO, 0, 1)
		for _, attachment := range attachments {
			if attachment.MessageID == turn.Assistant.ID {
				data = append(data, chatAttachmentFromStore(attachment))
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"userMessage": chatMessageFromStore(turn.User), "assistantMessage": chatMessageFromStore(turn.Assistant), "attachments": data, "replayed": true})
		return
	}

	if !acquireChatSemaphore(r.Context(), h.chatImageSemaphore) {
		turn.Assistant.Status, turn.Assistant.ErrorCode, turn.Assistant.ErrorMessage = "error", "concurrency_limit", "too many image generations are in progress"
		if st, release, available := h.chatStore(w); available {
			turn.Assistant, _ = st.FinalizeChatMessage(turn.Assistant)
			release()
		}
		chatAPIError(w, http.StatusTooManyRequests, "concurrency_limit", "too many image generations are in progress")
		return
	}
	defer releaseChatSemaphore(h.chatImageSemaphore)
	result, execErr := h.executeChatImage(r.Context(), chatImageExecutionRequest{Provider: req.Provider, Model: req.Model, Prompt: req.Prompt, Size: req.Size, Quality: req.Quality})
	if execErr != nil {
		if errors.Is(execErr, context.Canceled) || errors.Is(r.Context().Err(), context.Canceled) {
			turn.Assistant.Status, turn.Assistant.ErrorCode, turn.Assistant.ErrorMessage = "stopped", "generation_cancelled", "image generation stopped"
			if st, release, available := h.chatStore(w); available {
				turn.Assistant, _ = st.FinalizeChatMessage(turn.Assistant)
				release()
			}
			return
		}
		turn.Assistant.Status, turn.Assistant.ErrorCode, turn.Assistant.ErrorMessage = "error", "image_generation_failed", "image generation failed"
		st, release, available := h.chatStore(w)
		if available {
			turn.Assistant, _ = st.FinalizeChatMessage(turn.Assistant)
			release()
		}
		chatAPIError(w, http.StatusBadGateway, "image_generation_failed", "image generation failed")
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Base64)
	if err != nil {
		h.failChatImageTurn(w, turn, "invalid image returned by provider")
		return
	}
	assets, err := h.chatAssets()
	if err != nil {
		h.failChatImageTurn(w, turn, "image storage is unavailable")
		return
	}
	storageKey, imageInfo, err := assets.store(conversationID, bytes.NewReader(decoded))
	if err != nil {
		h.failChatImageTurn(w, turn, "invalid image returned by provider")
		return
	}
	width, height := imageInfo.Width, imageInfo.Height
	attachment := store.ChatAttachment{ID: uuid.NewString(), ConversationID: conversationID, MessageID: turn.Assistant.ID, Kind: "image_output", Name: "generated." + strings.TrimPrefix(imageInfo.MIMEType, "image/"), MIMEType: imageInfo.MIMEType, SizeBytes: imageInfo.Size, StorageKey: storageKey, Width: &width, Height: &height}
	turn.Assistant.Status, turn.Assistant.Provider, turn.Assistant.Model = "complete", result.Provider, result.Model
	st, release, available := h.chatStore(w)
	if !available {
		if path, pathErr := assets.path(storageKey); pathErr == nil {
			_ = os.Remove(path)
		}
		return
	}
	attachment, err = st.CreateChatAttachment(attachment)
	if err == nil {
		turn.Assistant, err = st.FinalizeChatMessage(turn.Assistant)
	}
	release()
	if err != nil {
		if path, pathErr := assets.path(storageKey); pathErr == nil {
			_ = os.Remove(path)
		}
		writeChatStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"userMessage": chatMessageFromStore(turn.User), "assistantMessage": chatMessageFromStore(turn.Assistant), "attachments": []chatAttachmentDTO{chatAttachmentFromStore(attachment)}, "replayed": false})
}

func (h *Handler) failChatImageTurn(w http.ResponseWriter, turn store.ChatTurn, message string) {
	turn.Assistant.Status, turn.Assistant.ErrorCode, turn.Assistant.ErrorMessage = "error", "invalid_image_output", message
	if st, release, ok := h.chatStore(w); ok {
		_, _ = st.FinalizeChatMessage(turn.Assistant)
		release()
	}
	chatAPIError(w, http.StatusBadGateway, "invalid_image_output", message)
}
