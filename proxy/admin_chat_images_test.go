package proxy

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAdminChatImageGeneratePersistsAndReplays(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "codex", "gpt-5.5-image")
	png := testPNG(t)
	calls := 0
	h.chatImageExecutor = func(_ context.Context, req chatImageExecutionRequest) (chatImageExecutionResult, error) {
		calls++
		if req.Provider != "codex" || req.Model != "gpt-5.5-image" || req.Prompt != "draw a moon" {
			t.Fatalf("request=%+v", req)
		}
		return chatImageExecutionResult{Base64: base64.StdEncoding.EncodeToString(png), MIMEType: "image/png", Provider: req.Provider, Model: req.Model}, nil
	}
	path := "/admin/api/chat/conversations/" + conversation.ID + "/images/generate"
	body := `{"clientRequestId":"image-1","prompt":"draw a moon"}`
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, path, body, "pw"))
	if w.Code != http.StatusCreated {
		t.Fatalf("generate status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Attachments []chatAttachmentDTO `json:"attachments"`
		Replayed    bool                `json:"replayed"`
	}
	decodeChatResponse(t, w, &response)
	if response.Replayed || len(response.Attachments) != 1 || response.Attachments[0].Kind != "image_output" {
		t.Fatalf("response=%+v", response)
	}
	content := httptest.NewRecorder()
	h.handleAdminAPI(content, chatAdminRequest(http.MethodGet, response.Attachments[0].ContentURL, "", "pw"))
	if content.Code != http.StatusOK || content.Body.String() != string(png) {
		t.Fatalf("content status=%d", content.Code)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, path, body, "pw"))
	if w.Code != http.StatusOK || calls != 1 {
		t.Fatalf("replay status=%d calls=%d body=%s", w.Code, calls, w.Body.String())
	}
}

func TestAdminChatImageGenerateRejectsCapabilityMismatch(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	conversation := createGenerateTestConversation(t, h, "kiro", "claude-opus")
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations/"+conversation.ID+"/images/generate", `{"clientRequestId":"bad","prompt":"draw"}`, "pw"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
