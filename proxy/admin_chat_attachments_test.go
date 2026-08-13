package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for y := range 3 {
		for x := range 2 {
			img.Set(x, y, color.White)
		}
	}
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func uploadChatImage(t *testing.T, h *Handler, conversationID, filename string, data []byte) chatAttachmentDTO {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversationID+"/attachments", "", "pw")
	r.Body = io.NopCloser(bytes.NewReader(body.Bytes()))
	r.ContentLength = int64(body.Len())
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Data []chatAttachmentDTO `json:"data"`
	}
	decodeChatResponse(t, w, &response)
	if len(response.Data) != 1 {
		t.Fatalf("attachments=%+v", response.Data)
	}
	return response.Data[0]
}

func pngWithDimensions(t *testing.T, width, height uint32) []byte {
	t.Helper()
	data := append([]byte(nil), testPNG(t)...)
	binary.BigEndian.PutUint32(data[16:20], width)
	binary.BigEndian.PutUint32(data[20:24], height)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}

func uploadChatMultipart(t *testing.T, h *Handler, conversationID string, files map[string][]byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, data := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversationID+"/attachments", "", "pw")
	r.Body = io.NopCloser(bytes.NewReader(body.Bytes()))
	r.ContentLength = int64(body.Len())
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, r)
	return w
}

func assertNoChatAttachments(t *testing.T, h *Handler, conversationID string) {
	t.Helper()
	st, unlock := h.runtimeStoreForOperation()
	attachments, err := st.ListChatAttachments(conversationID)
	unlock()
	if err != nil || len(attachments) != 0 {
		t.Fatalf("attachments=%+v err=%v", attachments, err)
	}
	entries, err := os.ReadDir(filepath.Join(h.chatAssetRoot, conversationID))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("asset entries=%v", entries)
	}
}

func TestAdminChatAttachmentUploadLimitsRollback(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "kiro", "vision")
	imageData := testPNG(t)

	files := map[string][]byte{}
	for i := range 5 {
		files[fmt.Sprintf("image-%d.png", i)] = imageData
	}
	w := uploadChatMultipart(t, h, conversation.ID, files)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "too_many_attachments") {
		t.Fatalf("five files status=%d body=%s", w.Code, w.Body.String())
	}
	assertNoChatAttachments(t, h, conversation.ID)

	w = uploadChatMultipart(t, h, conversation.ID, map[string][]byte{"large.png": make([]byte, chatAttachmentMaxBytes+1)})
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "invalid_image") {
		t.Fatalf("large file status=%d body=%s", w.Code, w.Body.String())
	}
	assertNoChatAttachments(t, h, conversation.ID)
}

func TestAdminChatAttachmentSniffsMIMEAndRejectsPixelBomb(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "kiro", "vision")

	attachment := uploadChatImage(t, h, conversation.ID, "claimed.jpg", testPNG(t))
	if attachment.MIMEType != "image/png" {
		t.Fatalf("mime=%q", attachment.MIMEType)
	}

	w := uploadChatMultipart(t, h, conversation.ID, map[string][]byte{"bomb.png": pngWithDimensions(t, 10_000, 5_000)})
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "invalid_image") {
		t.Fatalf("pixel bomb status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminChatAttachmentUploadServeDeleteAndVision(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "kiro", "vision-model")
	attachment := uploadChatImage(t, h, conversation.ID, "../../photo.png", testPNG(t))
	if attachment.Name != "photo.png" || attachment.MIMEType != "image/png" || attachment.Width == nil || *attachment.Width != 2 || attachment.Height == nil || *attachment.Height != 3 {
		t.Fatalf("attachment=%+v", attachment)
	}

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, attachment.ContentURL, "", "pw"))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || !bytes.Equal(w.Body.Bytes(), testPNG(t)) {
		t.Fatalf("content status=%d headers=%v", w.Code, w.Header())
	}

	var generation int
	h.chatTextExecutor = func(_ context.Context, req chatTextExecutionRequest) (chatTextExecutionResult, error) {
		generation++
		messageIndex := len(req.Messages) - 1
		if generation == 2 {
			messageIndex = 0
			if len(req.Messages) != 3 || req.Messages[2].Content != "what did you see?" {
				t.Fatalf("historical messages=%#v", req.Messages)
			}
		}
		parts, ok := req.Messages[messageIndex].Content.([]map[string]any)
		if !ok || len(parts) != 2 {
			t.Fatalf("content=%#v", req.Messages[messageIndex].Content)
		}
		imagePart := parts[1]["image_url"].(map[string]any)["url"].(string)
		if !strings.HasPrefix(imagePart, "data:image/png;base64,") {
			t.Fatalf("image part prefix=%q", imagePart)
		}
		if _, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(imagePart, "data:image/png;base64,")); err != nil {
			t.Fatal(err)
		}
		return chatTextExecutionResult{Content: "seen", Provider: "kiro", Model: "vision-model"}, nil
	}
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"vision","content":"describe","attachmentIds":["`+attachment.ID+`"]}`, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/generate", `{"clientRequestId":"vision-history","content":"what did you see?"}`, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("historical generate status=%d body=%s", w.Code, w.Body.String())
	}
	if generation != 2 {
		t.Fatalf("generation calls=%d", generation)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodDelete, "/admin/api/chat/conversations/"+conversation.ID+"/attachments/"+attachment.ID, "", "pw"))
	if w.Code != http.StatusConflict {
		t.Fatalf("bound delete status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestChatAssetReconciliationRemovesExpiredAndOrphanFiles(t *testing.T) {
	root := t.TempDir()
	assets, err := newChatAssetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	conversationID := "conversation"
	dir := filepath.Join(root, conversationID)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"referenced":    testPNG(t),
		"expired":       testPNG(t),
		"orphan":        testPNG(t),
		".stale.upload": []byte("partial"),
	} {
		if err = os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stale := filepath.Join(dir, ".stale.upload")
	old := time.Now().Add(-2 * chatAttachmentUnboundTTL)
	if err = os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	result, err := assets.reconcile(
		map[string]bool{conversationID + "/referenced": true},
		[]string{conversationID + "/expired"},
		time.Now().Add(-chatAttachmentUnboundTTL),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedExpired != 1 || result.RemovedOrphans != 1 || result.RemovedUploads != 1 {
		t.Fatalf("result=%+v", result)
	}
	if _, err = os.Stat(filepath.Join(dir, "referenced")); err != nil {
		t.Fatalf("referenced file removed: %v", err)
	}
}

func TestChatAssetPathRejectsTraversalAndSymlink(t *testing.T) {
	root := t.TempDir()
	assets, err := newChatAssetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = assets.path("../outside"); !errors.Is(err, errInvalidChatImage) {
		t.Fatalf("traversal error=%v", err)
	}
	if err = os.Symlink(t.TempDir(), filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err = assets.path("linked/file"); !errors.Is(err, errInvalidChatImage) {
		t.Fatalf("symlink error=%v", err)
	}
}

func TestAdminChatAttachmentRejectsInvalidAndCrossConversation(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	first := createGenerateTestConversation(t, h, "kiro", "vision")
	second := createGenerateTestConversation(t, h, "kiro", "vision")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("files", "fake.png")
	_, _ = part.Write([]byte("<svg><script>alert(1)</script></svg>"))
	_ = writer.Close()
	r := chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+first.ID+"/attachments", "", "pw")
	r.Body = io.NopCloser(bytes.NewReader(body.Bytes()))
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status=%d body=%s", w.Code, w.Body.String())
	}

	attachment := uploadChatImage(t, h, first.ID, "ok.png", testPNG(t))
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations/"+second.ID+"/attachments/"+attachment.ID+"/content", "", "pw"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross content status=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+second.ID+"/generate", `{"clientRequestId":"cross","content":"x","attachmentIds":["`+attachment.ID+`"]}`, "pw"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross claim status=%d body=%s", w.Code, w.Body.String())
	}
}
