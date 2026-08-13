package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	chatTitleMaxBytes     = 500
	chatProviderMaxBytes  = 64
	chatModelMaxBytes     = 256
	chatProjectIDMaxBytes = 128
)

type chatConversationDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Mode       string `json:"mode"`
	Status     string `json:"status"`
	Pinned     bool   `json:"pinned"`
	ProjectID  string `json:"projectId"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	ArchivedAt *int64 `json:"archivedAt"`
}

type chatMessageDTO struct {
	ID                  string              `json:"id"`
	ConversationID      string              `json:"conversationId"`
	ParentMessageID     string              `json:"parentMessageId"`
	ClientRequestID     string              `json:"clientRequestId"`
	Role                string              `json:"role"`
	Content             string              `json:"content"`
	Provider            string              `json:"provider"`
	Model               string              `json:"model"`
	Status              string              `json:"status"`
	ErrorCode           string              `json:"errorCode"`
	ErrorMessage        string              `json:"errorMessage"`
	ProviderResponseID  string              `json:"providerResponseId"`
	RequestID           string              `json:"requestId"`
	InputTokens         int                 `json:"inputTokens"`
	OutputTokens        int                 `json:"outputTokens"`
	CacheReadTokens     int                 `json:"cacheReadTokens"`
	CacheCreationTokens int                 `json:"cacheCreationTokens"`
	CreatedAt           int64               `json:"createdAt"`
	UpdatedAt           int64               `json:"updatedAt"`
	Attachments         []chatAttachmentDTO `json:"attachments,omitempty"`
}

type chatConversationRequest struct {
	Title     *string `json:"title"`
	Provider  *string `json:"provider"`
	Model     *string `json:"model"`
	Mode      *string `json:"mode"`
	Status    *string `json:"status"`
	Pinned    *bool   `json:"pinned"`
	ProjectID *string `json:"projectId"`
}

type chatCapabilitiesDTO struct {
	Vision          bool `json:"vision"`
	ImageGeneration bool `json:"imageGeneration"`
	Reasoning       bool `json:"reasoning"`
	Tools           bool `json:"tools"`
	Web             bool `json:"web"`
}

type chatModelDTO struct {
	ID           string              `json:"id"`
	Provider     string              `json:"provider"`
	Model        string              `json:"model"`
	DisplayName  string              `json:"displayName"`
	Availability string              `json:"availability"`
	Capabilities chatCapabilitiesDTO `json:"capabilities"`
}

func chatConversationFromStore(c store.ChatConversation) chatConversationDTO {
	return chatConversationDTO{
		ID: c.ID, Title: c.Title, Provider: c.Provider, Model: c.Model,
		Mode: c.Mode, Status: c.Status, Pinned: c.Pinned, ProjectID: c.ProjectID,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, ArchivedAt: c.ArchivedAt,
	}
}

func chatMessageFromStore(m store.ChatMessage) chatMessageDTO {
	return chatMessageDTO{
		ID: m.ID, ConversationID: m.ConversationID, ParentMessageID: m.ParentMessageID,
		ClientRequestID: m.ClientRequestID, Role: m.Role, Content: m.Content,
		Provider: m.Provider, Model: m.Model, Status: m.Status, ErrorCode: m.ErrorCode,
		ErrorMessage: m.ErrorMessage, ProviderResponseID: m.ProviderResponseID,
		RequestID: m.RequestID, InputTokens: m.InputTokens, OutputTokens: m.OutputTokens,
		CacheReadTokens: m.CacheReadTokens, CacheCreationTokens: m.CacheCreationTokens,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func chatAPIError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}

func writeChatStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrChatNotFound):
		chatAPIError(w, http.StatusNotFound, "conversation_not_found", "conversation not found")
	case errors.Is(err, store.ErrChatMessageNotFound):
		chatAPIError(w, http.StatusNotFound, "message_not_found", "message not found")
	case errors.Is(err, store.ErrChatAttachmentNotFound):
		chatAPIError(w, http.StatusNotFound, "attachment_not_found", "attachment not found")
	case errors.Is(err, store.ErrChatInvalidCursor):
		chatAPIError(w, http.StatusBadRequest, "invalid_cursor", "cursor is invalid")
	case errors.Is(err, store.ErrChatConflict):
		chatAPIError(w, http.StatusConflict, "chat_conflict", "chat state conflict")
	default:
		chatAPIError(w, http.StatusServiceUnavailable, "chat_store_unavailable", "chat storage is unavailable")
	}
}

func validateChatConversationRequest(req chatConversationRequest) []comboFieldError {
	var out []comboFieldError
	checkString := func(field string, value *string, max int) {
		if value != nil && len(strings.TrimSpace(*value)) > max {
			out = append(out, comboFieldError{Field: field, Message: "must be at most " + strconv.Itoa(max) + " bytes"})
		}
	}
	checkString("title", req.Title, chatTitleMaxBytes)
	checkString("provider", req.Provider, chatProviderMaxBytes)
	checkString("model", req.Model, chatModelMaxBytes)
	checkString("projectId", req.ProjectID, chatProjectIDMaxBytes)
	if req.Mode != nil {
		mode := strings.ToLower(strings.TrimSpace(*req.Mode))
		if mode != "chat" && mode != "image" {
			out = append(out, comboFieldError{Field: "mode", Message: "must be chat or image"})
		}
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status != "active" && status != "archived" {
			out = append(out, comboFieldError{Field: "status", Message: "must be active or archived"})
		}
	}
	return out
}

func applyChatConversationRequest(c *store.ChatConversation, req chatConversationRequest) {
	if req.Title != nil {
		c.Title = strings.TrimSpace(*req.Title)
	}
	if req.Provider != nil {
		c.Provider = strings.ToLower(strings.TrimSpace(*req.Provider))
	}
	if req.Model != nil {
		c.Model = strings.TrimSpace(*req.Model)
	}
	if req.Mode != nil {
		c.Mode = strings.ToLower(strings.TrimSpace(*req.Mode))
	}
	if req.Status != nil {
		c.Status = strings.ToLower(strings.TrimSpace(*req.Status))
	}
	if req.Pinned != nil {
		c.Pinned = *req.Pinned
	}
	if req.ProjectID != nil {
		c.ProjectID = strings.TrimSpace(*req.ProjectID)
	}
}

// handleAdminChatRoute handles exact routes under /admin/api/chat.
func (h *Handler) handleAdminChatRoute(w http.ResponseWriter, r *http.Request, path string) bool {
	if path == "/chat/models" && r.Method == http.MethodGet {
		h.apiListChatModels(w)
		return true
	}
	if path == "/chat/conversations" {
		switch r.Method {
		case http.MethodGet:
			h.apiListChatConversations(w, r)
			return true
		case http.MethodPost:
			h.apiCreateChatConversation(w, r)
			return true
		}
		return false
	}
	const prefix = "/chat/conversations/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 2 && parts[0] != "" {
		switch {
		case parts[1] == "messages" && r.Method == http.MethodGet:
			h.apiListChatMessages(w, r, parts[0])
			return true
		case parts[1] == "generate" && r.Method == http.MethodPost:
			h.apiGenerateChatConversation(w, r, parts[0])
			return true
		case parts[1] == "attachments" && r.Method == http.MethodGet:
			h.apiListChatAttachments(w, parts[0])
			return true
		case parts[1] == "attachments" && r.Method == http.MethodPost:
			h.apiUploadChatAttachments(w, r, parts[0])
			return true
		}
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "images" && parts[2] == "generate" && r.Method == http.MethodPost {
		h.apiGenerateChatImage(w, r, parts[0])
		return true
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "attachments" && parts[2] != "" && r.Method == http.MethodDelete {
		h.apiDeleteChatAttachment(w, parts[0], parts[2])
		return true
	}
	if len(parts) == 4 && parts[0] != "" && parts[1] == "attachments" && parts[2] != "" && parts[3] == "content" && r.Method == http.MethodGet {
		h.apiServeChatAttachment(w, parts[0], parts[2])
		return true
	}
	if len(parts) != 1 || parts[0] == "" {
		return false
	}
	switch r.Method {
	case http.MethodGet:
		h.apiGetChatConversation(w, parts[0])
	case http.MethodPatch:
		h.apiPatchChatConversation(w, r, parts[0])
	case http.MethodDelete:
		h.apiDeleteChatConversation(w, parts[0])
	default:
		return false
	}
	return true
}

func (h *Handler) chatStore(w http.ResponseWriter) (*store.Store, func(), bool) {
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		chatAPIError(w, http.StatusServiceUnavailable, "chat_store_unavailable", "chat storage is unavailable")
		return nil, unlock, false
	}
	return st, unlock, true
}

func (h *Handler) apiCreateChatConversation(w http.ResponseWriter, r *http.Request) {
	var req chatConversationRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		chatAPIError(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	if fields := validateChatConversationRequest(req); len(fields) > 0 {
		comboError(w, http.StatusUnprocessableEntity, "validation failed", fields)
		return
	}
	c := store.ChatConversation{ID: uuid.NewString(), Mode: "chat", Status: "active"}
	applyChatConversationRequest(&c, req)
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	defer unlock()
	created, err := st.CreateChatConversation(c)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(chatConversationFromStore(created))
}

func (h *Handler) apiListChatConversations(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && status != "active" && status != "archived" {
		chatAPIError(w, http.StatusUnprocessableEntity, "validation_failed", "status must be active or archived")
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if len(search) > chatTitleMaxBytes {
		chatAPIError(w, http.StatusUnprocessableEntity, "validation_failed", "search must be at most 500 bytes")
		return
	}
	limit, ok := parseChatLimit(r.URL.Query().Get("limit"), 50, 100)
	if !ok {
		chatAPIError(w, http.StatusUnprocessableEntity, "validation_failed", "limit must be between 1 and 100")
		return
	}
	st, unlock, available := h.chatStore(w)
	if !available {
		return
	}
	defer unlock()
	page, err := st.ListChatConversations(status, search, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	data := make([]chatConversationDTO, 0, len(page.Items))
	for _, c := range page.Items {
		data = append(data, chatConversationFromStore(c))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "nextCursor": page.NextCursor})
}

func (h *Handler) apiGetChatConversation(w http.ResponseWriter, id string) {
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	defer unlock()
	c, err := st.GetChatConversation(id)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(chatConversationFromStore(c))
}

func (h *Handler) apiPatchChatConversation(w http.ResponseWriter, r *http.Request, id string) {
	var req chatConversationRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		chatAPIError(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	if fields := validateChatConversationRequest(req); len(fields) > 0 {
		comboError(w, http.StatusUnprocessableEntity, "validation failed", fields)
		return
	}
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	defer unlock()
	c, err := st.GetChatConversation(id)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	oldStatus := c.Status
	applyChatConversationRequest(&c, req)
	if c.Status == "archived" && oldStatus != "archived" {
		now := time.Now().UnixMilli()
		c.ArchivedAt = &now
	} else if c.Status == "active" {
		c.ArchivedAt = nil
	}
	c, err = st.UpdateChatConversation(c)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(chatConversationFromStore(c))
}

func (h *Handler) apiDeleteChatConversation(w http.ResponseWriter, id string) {
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	if err := st.DeleteChatConversation(id); err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	unlock()
	if assets, err := h.chatAssets(); err == nil {
		_ = assets.removeConversation(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) apiListChatMessages(w http.ResponseWriter, r *http.Request, conversationID string) {
	limit, ok := parseChatLimit(r.URL.Query().Get("limit"), 100, 200)
	if !ok {
		chatAPIError(w, http.StatusUnprocessableEntity, "validation_failed", "limit must be between 1 and 200")
		return
	}
	st, unlock, available := h.chatStore(w)
	if !available {
		return
	}
	defer unlock()
	if _, err := st.GetChatConversation(conversationID); err != nil {
		writeChatStoreError(w, err)
		return
	}
	page, err := st.ListChatMessages(conversationID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	data := make([]chatMessageDTO, 0, len(page.Items))
	attachments, err := st.ListChatAttachments(conversationID)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	byMessage := make(map[string][]chatAttachmentDTO)
	for _, attachment := range attachments {
		if attachment.MessageID != "" {
			byMessage[attachment.MessageID] = append(byMessage[attachment.MessageID], chatAttachmentFromStore(attachment))
		}
	}
	for _, m := range page.Items {
		dto := chatMessageFromStore(m)
		dto.Attachments = byMessage[m.ID]
		data = append(data, dto)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "nextCursor": page.NextCursor})
}

func acquireChatSemaphore(ctx context.Context, semaphore chan struct{}) bool {
	if semaphore == nil {
		return true
	}
	select {
	case semaphore <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

func releaseChatSemaphore(semaphore chan struct{}) {
	if semaphore != nil {
		<-semaphore
	}
}

func parseChatLimit(value string, defaultValue, maxValue int) (int, bool) {
	if value == "" {
		return defaultValue, true
	}
	limit, err := strconv.Atoi(value)
	return limit, err == nil && limit >= 1 && limit <= maxValue
}

func chatProviderBucket(account *config.Account) string {
	switch {
	case isGrokAccount(account):
		return "grok"
	case isCodexAccount(account):
		return "codex"
	case isAntigravityAccount(account):
		return "antigravity"
	case isRemoteKiroAccount(account):
		return "remotekiro"
	default:
		return "kiro"
	}
}

func (h *Handler) chatModelCatalog() []chatModelDTO {
	enabled := config.GetEnabledAccounts()
	providers := make(map[string]bool)
	for i := range enabled {
		providers[chatProviderBucket(&enabled[i])] = true
	}
	models := make([]chatModelDTO, 0)
	seen := make(map[string]bool)
	appendInfos := func(provider string, infos []ModelInfo, thinkingVariants bool) {
		appendOne := func(m ModelInfo, id, name string) {
			id = strings.TrimSpace(id)
			key := provider + "\x00" + strings.ToLower(id)
			if id == "" || seen[key] {
				return
			}
			seen[key] = true
			vision := modelSupportsImage(m.InputTypes)
			models = append(models, chatModelDTO{
				ID: provider + ":" + id, Provider: provider, Model: id,
				DisplayName: name, Availability: "available",
				Capabilities: chatCapabilitiesDTO{
					Vision: vision, ImageGeneration: isGrokImageModel(id) || isCodexImageModel(id) || isAntigravityImageModel(id),
					Reasoning: strings.Contains(strings.ToLower(id), "thinking") || strings.Contains(strings.ToLower(id), "reasoning"),
				},
			})
		}
		thinkingSuffix := config.GetThinkingConfig().Suffix
		if thinkingSuffix == "" {
			thinkingSuffix = "-thinking"
		}
		for _, m := range infos {
			name := strings.TrimSpace(m.ModelName)
			if name == "" {
				name = m.ModelId
			}
			appendOne(m, m.ModelId, name)
			if thinkingVariants && !strings.HasSuffix(strings.ToLower(m.ModelId), strings.ToLower(thinkingSuffix)) {
				appendOne(m, m.ModelId+thinkingSuffix, name+" (thinking)")
			}
		}
	}
	if providers["grok"] {
		appendInfos("grok", grokModelInfos(), false)
	}
	if providers["codex"] {
		appendInfos("codex", codexModelInfos(), false)
	}
	if providers["antigravity"] {
		appendInfos("antigravity", antigravityModelInfos(), false)
	}
	if providers["remotekiro"] {
		infos, _ := h.remoteKiroProviderModelInfos()
		appendInfos("remotekiro", infos, false)
	}
	if providers["kiro"] {
		infos, _ := h.kiroProviderModelInfos()
		appendInfos("kiro", infos, true)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider == models[j].Provider {
			return strings.ToLower(models[i].Model) < strings.ToLower(models[j].Model)
		}
		return models[i].Provider < models[j].Provider
	})
	return models
}

func (h *Handler) chatModelVisionCapability(provider, model string) (bool, bool) {
	for _, candidate := range h.chatModelCatalog() {
		if candidate.Provider == provider && strings.EqualFold(candidate.Model, model) {
			return candidate.Capabilities.Vision, true
		}
	}
	return false, false
}

func (h *Handler) apiListChatModels(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{"data": h.chatModelCatalog()})
}
