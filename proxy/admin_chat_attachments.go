package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kiro-go/logger"
	"kiro-go/store"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const chatAttachmentUnboundTTL = 24 * time.Hour

func (h *Handler) reconcileChatAssets() {
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		return
	}
	defer unlock()
	assets, err := h.chatAssets()
	if err != nil {
		logger.Warnf("chat asset reconciliation unavailable: %v", err)
		return
	}
	expired, err := st.DeleteUnboundChatAttachments(time.Now().Add(-chatAttachmentUnboundTTL).UnixMilli())
	if err != nil {
		logger.Warnf("chat attachment expiry failed: %v", err)
		return
	}
	attachments, err := st.ListAllChatAttachments()
	if err != nil {
		logger.Warnf("chat attachment reconciliation failed: %v", err)
		return
	}
	referenced := make(map[string]bool, len(attachments))
	for _, attachment := range attachments {
		path, pathErr := assets.path(attachment.StorageKey)
		info, statErr := os.Stat(path)
		valid := pathErr == nil && statErr == nil && info.Mode().IsRegular() && info.Size() == attachment.SizeBytes
		if valid && (attachment.Kind == "image_input" || attachment.Kind == "image_output") {
			_, validErr := validateStoredChatImage(path, attachment.SizeBytes)
			valid = validErr == nil
		}
		if !valid {
			if deleteErr := st.DeleteChatAttachment(attachment.ID); deleteErr != nil {
				logger.Warnf("remove invalid chat attachment metadata %s: %v", attachment.ID, deleteErr)
			}
			continue
		}
		referenced[attachment.StorageKey] = true
	}
	expiredKeys := make([]string, 0, len(expired))
	for _, attachment := range expired {
		expiredKeys = append(expiredKeys, attachment.StorageKey)
	}
	result, err := assets.reconcile(referenced, expiredKeys, time.Now().Add(-chatAttachmentUnboundTTL))
	if err != nil {
		logger.Warnf("chat asset reconciliation failed: %v", err)
		return
	}
	if result.RemovedOrphans+result.RemovedUploads+result.RemovedExpired > 0 {
		logger.Infof("chat assets reconciled: expired=%d orphan=%d upload=%d", result.RemovedExpired, result.RemovedOrphans, result.RemovedUploads)
	}
}

type chatAttachmentDTO struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	MIMEType       string `json:"mimeType"`
	SizeBytes      int64  `json:"sizeBytes"`
	Width          *int   `json:"width"`
	Height         *int   `json:"height"`
	CreatedAt      int64  `json:"createdAt"`
	ContentURL     string `json:"contentUrl"`
}

func chatAttachmentFromStore(a store.ChatAttachment) chatAttachmentDTO {
	return chatAttachmentDTO{
		ID: a.ID, ConversationID: a.ConversationID, MessageID: a.MessageID, Kind: a.Kind,
		Name: a.Name, MIMEType: a.MIMEType, SizeBytes: a.SizeBytes, Width: a.Width,
		Height: a.Height, CreatedAt: a.CreatedAt,
		ContentURL: "/admin/api/chat/conversations/" + a.ConversationID + "/attachments/" + a.ID + "/content",
	}
}

func (h *Handler) chatAssets() (*chatAssetStore, error) {
	return newChatAssetStore(h.chatAssetRoot)
}

func (h *Handler) apiUploadChatAttachments(w http.ResponseWriter, r *http.Request, conversationID string) {
	if !acquireChatSemaphore(r.Context(), h.chatUploadSemaphore) {
		chatAPIError(w, http.StatusTooManyRequests, "concurrency_limit", "too many attachment uploads are in progress")
		return
	}
	defer releaseChatSemaphore(h.chatUploadSemaphore)
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	if _, err := st.GetChatConversation(conversationID); err != nil {
		unlock()
		writeChatStoreError(w, err)
		return
	}
	unlock()

	r.Body = http.MaxBytesReader(w, r.Body, chatAttachmentRequestMax)
	reader, err := r.MultipartReader()
	if err != nil {
		chatAPIError(w, http.StatusBadRequest, "invalid_multipart", "multipart request is invalid")
		return
	}
	assets, err := h.chatAssets()
	if err != nil {
		chatAPIError(w, http.StatusServiceUnavailable, "asset_store_unavailable", "attachment storage is unavailable")
		return
	}
	created := make([]store.ChatAttachment, 0, chatAttachmentMaxFiles)
	cleanup := func() {
		for _, a := range created {
			if path, pathErr := assets.path(a.StorageKey); pathErr == nil {
				_ = os.Remove(path)
			}
			st, release, available := h.chatStore(w)
			if available {
				_ = st.DeleteChatAttachment(a.ID)
				release()
			}
		}
	}
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			cleanup()
			chatAPIError(w, http.StatusRequestEntityTooLarge, "attachment_request_too_large", "attachment request is too large")
			return
		}
		if part.FormName() != "files" || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		if len(created) >= chatAttachmentMaxFiles {
			_ = part.Close()
			cleanup()
			chatAPIError(w, http.StatusUnprocessableEntity, "too_many_attachments", "at most four images may be uploaded")
			return
		}
		storageKey, imageInfo, storeErr := assets.store(conversationID, part)
		_ = part.Close()
		if storeErr != nil {
			cleanup()
			if errors.Is(storeErr, errInvalidChatImage) {
				chatAPIError(w, http.StatusUnprocessableEntity, "invalid_image", "image must be a valid PNG, JPEG, or WebP up to 10 MiB")
			} else {
				chatAPIError(w, http.StatusServiceUnavailable, "asset_store_unavailable", "attachment could not be stored")
			}
			return
		}
		width, height := imageInfo.Width, imageInfo.Height
		a := store.ChatAttachment{
			ID: uuid.NewString(), ConversationID: conversationID, Kind: "image_input",
			Name: safeAttachmentName(part.FileName()), MIMEType: imageInfo.MIMEType,
			SizeBytes: imageInfo.Size, StorageKey: storageKey, Width: &width, Height: &height,
		}
		st, release, available := h.chatStore(w)
		if !available {
			if path, pathErr := assets.path(storageKey); pathErr == nil {
				_ = os.Remove(path)
			}
			cleanup()
			return
		}
		a, err = st.CreateChatAttachment(a)
		release()
		if err != nil {
			if path, pathErr := assets.path(storageKey); pathErr == nil {
				_ = os.Remove(path)
			}
			cleanup()
			writeChatStoreError(w, err)
			return
		}
		created = append(created, a)
	}
	if len(created) == 0 {
		chatAPIError(w, http.StatusUnprocessableEntity, "attachment_required", "at least one image is required")
		return
	}
	data := make([]chatAttachmentDTO, 0, len(created))
	for _, a := range created {
		data = append(data, chatAttachmentFromStore(a))
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func safeAttachmentName(value string) string {
	value = strings.TrimSpace(filepath.Base(strings.ReplaceAll(value, "\\", "/")))
	if value == "." || value == "" {
		return "image"
	}
	runes := []rune(value)
	if len(runes) > 200 {
		value = string(runes[:200])
	}
	return value
}

func (h *Handler) apiListChatAttachments(w http.ResponseWriter, conversationID string) {
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	defer unlock()
	if _, err := st.GetChatConversation(conversationID); err != nil {
		writeChatStoreError(w, err)
		return
	}
	attachments, err := st.ListChatAttachments(conversationID)
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	data := make([]chatAttachmentDTO, 0, len(attachments))
	for _, a := range attachments {
		data = append(data, chatAttachmentFromStore(a))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func (h *Handler) apiServeChatAttachment(w http.ResponseWriter, conversationID, attachmentID string) {
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	a, err := st.GetChatAttachment(conversationID, attachmentID)
	unlock()
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	assets, err := h.chatAssets()
	if err != nil {
		chatAPIError(w, http.StatusServiceUnavailable, "asset_store_unavailable", "attachment storage is unavailable")
		return
	}
	path, err := assets.path(a.StorageKey)
	if err != nil {
		chatAPIError(w, http.StatusNotFound, "attachment_not_found", "attachment not found")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		chatAPIError(w, http.StatusNotFound, "attachment_not_found", "attachment not found")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != a.SizeBytes {
		chatAPIError(w, http.StatusNotFound, "attachment_not_found", "attachment not found")
		return
	}
	w.Header().Set("Content-Type", a.MIMEType)
	w.Header().Set("Content-Length", strconv.FormatInt(a.SizeBytes, 10))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", mime.QEncoding.Encode("UTF-8", a.Name)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.CopyN(w, file, a.SizeBytes)
}

func (h *Handler) apiDeleteChatAttachment(w http.ResponseWriter, conversationID, attachmentID string) {
	st, unlock, ok := h.chatStore(w)
	if !ok {
		return
	}
	a, err := st.GetChatAttachment(conversationID, attachmentID)
	if err == nil && a.MessageID != "" {
		err = store.ErrChatConflict
	}
	if err == nil {
		err = st.DeleteChatAttachment(attachmentID)
	}
	unlock()
	if err != nil {
		writeChatStoreError(w, err)
		return
	}
	if assets, assetErr := h.chatAssets(); assetErr == nil {
		if path, pathErr := assets.path(a.StorageKey); pathErr == nil {
			_ = os.Remove(path)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
