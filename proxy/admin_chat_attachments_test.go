package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
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

	h.chatTextExecutor = func(_ context.Context, req chatTextExecutionRequest) (chatTextExecutionResult, error) {
		parts, ok := req.Messages[len(req.Messages)-1].Content.([]map[string]any)
		if !ok || len(parts) != 2 {
			t.Fatalf("content=%#v", req.Messages[len(req.Messages)-1].Content)
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
	h.handleAdminAPI(w, chatAdminRequest(http.MethodDelete, "/admin/api/chat/conversations/"+conversation.ID+"/attachments/"+attachment.ID, "", "pw"))
	if w.Code != http.StatusConflict {
		t.Fatalf("bound delete status=%d body=%s", w.Code, w.Body.String())
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
