package proxy

import (
	"encoding/json"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newAdminChatTestHandler(t *testing.T) *Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "chat-api.db"))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{runtimeStore: st}
	t.Cleanup(func() { _ = st.Close() })
	return h
}

func chatAdminRequest(method, target, body, password string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if password != "" {
		r.Header.Set("X-Admin-Password", password)
	}
	return r
}

func decodeChatResponse(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(w.Result().Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v body=%q", err, w.Body.String())
	}
}

func TestAdminChatConversationCRUDAndMessages(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations", `{"title":" First ","provider":"Kiro","model":"claude"}`, "pw"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var created chatConversationDTO
	decodeChatResponse(t, w, &created)
	if created.ID == "" || created.Title != "First" || created.Provider != "kiro" || created.Mode != "chat" || created.Status != "active" {
		t.Fatalf("created=%+v", created)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodPatch, "/admin/api/chat/conversations/"+created.ID, `{"title":"Renamed","status":"archived","pinned":true}`, "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", w.Code, w.Body.String())
	}
	var patched chatConversationDTO
	decodeChatResponse(t, w, &patched)
	if patched.Title != "Renamed" || patched.Model != "claude" || !patched.Pinned || patched.ArchivedAt == nil || patched.CreatedAt != created.CreatedAt {
		t.Fatalf("patched=%+v", patched)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations?status=archived&limit=1", "", "pw"))
	var list struct {
		Data []chatConversationDTO `json:"data"`
	}
	decodeChatResponse(t, w, &list)
	if w.Code != http.StatusOK || len(list.Data) != 1 || list.Data[0].ID != created.ID {
		t.Fatalf("list status=%d data=%+v", w.Code, list.Data)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations/"+created.ID+"/messages", "", "pw"))
	var messages struct {
		Data []chatMessageDTO `json:"data"`
	}
	decodeChatResponse(t, w, &messages)
	if w.Code != http.StatusOK || messages.Data == nil || len(messages.Data) != 0 {
		t.Fatalf("messages status=%d data=%#v", w.Code, messages.Data)
	}

	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodDelete, "/admin/api/chat/conversations/"+created.ID, "", "pw"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations/"+created.ID, "", "pw"))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "conversation_not_found") {
		t.Fatalf("missing status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminChatConversationDeleteRemovesAssetDirectory(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "kiro", "vision")
	_ = uploadChatImage(t, h, conversation.ID, "photo.png", testPNG(t))
	dir := filepath.Join(h.chatAssetRoot, conversation.ID)
	if err := os.WriteFile(filepath.Join(dir, ".orphan.upload"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodDelete, "/admin/api/chat/conversations/"+conversation.ID, "", "pw"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("asset directory remains: %v", err)
	}
}

func TestAdminChatValidationRoutesAndUnavailable(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)

	cases := []struct {
		method, target, body string
		status               int
	}{
		{http.MethodPost, "/admin/api/chat/conversations", `{"unknown":true}`, http.StatusBadRequest},
		{http.MethodPost, "/admin/api/chat/conversations", `{} {}`, http.StatusBadRequest},
		{http.MethodPost, "/admin/api/chat/conversations", `{"mode":"video"}`, http.StatusUnprocessableEntity},
		{http.MethodGet, "/admin/api/chat/conversations?limit=0", "", http.StatusUnprocessableEntity},
		{http.MethodGet, "/admin/api/chat/conversations?cursor=bad", "", http.StatusBadRequest},
		{http.MethodGet, "/admin/api/chat/conversations/missing/messages", "", http.StatusNotFound},
		{http.MethodGet, "/admin/api/chat/conversations/x/messages/extra", "", http.StatusNotFound},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		h.handleAdminAPI(w, chatAdminRequest(tc.method, tc.target, tc.body, "pw"))
		if w.Code != tc.status {
			t.Errorf("%s %s status=%d want=%d body=%s", tc.method, tc.target, w.Code, tc.status, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	(&Handler{}).handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations", "", "pw"))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "chat_store_unavailable") {
		t.Fatalf("unavailable status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminChatRouteAuthAndCSRF(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)

	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/conversations", "", ""))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", w.Code)
	}

	token, csrf, _ := adminAuth.createSession("192.0.2.1")
	r := chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations", `{}`, "")
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status=%d", w.Code)
	}

	r = chatAdminRequest(http.MethodPost, "/admin/api/chat/conversations", `{}`, "")
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	r.Header.Set(adminCSRFHeader, csrf)
	w = httptest.NewRecorder()
	h.handleAdminAPI(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("csrf create status=%d body=%s", w.Code, w.Body.String())
	}
}

func chatSessionRequest(method, target, body, token, csrf string) *http.Request {
	r := chatAdminRequest(method, target, body, "")
	r.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: token})
	if csrf != "" {
		r.Header.Set(adminCSRFHeader, csrf)
	}
	return r
}

func TestAdminChatMutationsRequireSessionCSRFBeforeSideEffects(t *testing.T) {
	mustInitConfig(t)
	resetAdminAuth()
	setAdminPassword(t, "pw")
	h := newAdminChatTestHandler(t)
	h.chatAssetRoot = filepath.Join(t.TempDir(), "assets")
	conversation := createGenerateTestConversation(t, h, "codex", "gpt-5.5-image")
	token, _, err := adminAuth.createSession("192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, method, target, body string
	}{
		{"text generation", http.MethodPost, "/admin/api/chat/conversations/" + conversation.ID + "/generate", `{"clientRequestId":"csrf-text","content":"hello"}`},
		{"attachment upload", http.MethodPost, "/admin/api/chat/conversations/" + conversation.ID + "/attachments", ""},
		{"attachment delete", http.MethodDelete, "/admin/api/chat/conversations/" + conversation.ID + "/attachments/missing", ""},
		{"image generation", http.MethodPost, "/admin/api/chat/conversations/" + conversation.ID + "/images/generate", `{"clientRequestId":"csrf-image","prompt":"draw"}`},
		{"conversation delete", http.MethodDelete, "/admin/api/chat/conversations/" + conversation.ID, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, csrf := range []string{"", "wrong-token"} {
				w := httptest.NewRecorder()
				h.handleAdminAPI(w, chatSessionRequest(tc.method, tc.target, tc.body, token, csrf))
				if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "csrf_mismatch") {
					t.Fatalf("csrf=%q status=%d body=%s", csrf, w.Code, w.Body.String())
				}
			}
		})
	}

	st, release := h.runtimeStoreForOperation()
	messages, err := st.ListChatMessages(conversation.ID, "", 10)
	attachments, attachmentErr := st.ListChatAttachments(conversation.ID)
	_, conversationErr := st.GetChatConversation(conversation.ID)
	release()
	if err != nil || attachmentErr != nil || conversationErr != nil || len(messages.Items) != 0 || len(attachments) != 0 {
		t.Fatalf("messages=%+v attachments=%+v errors=%v/%v/%v", messages.Items, attachments, err, attachmentErr, conversationErr)
	}
}

func TestAdminChatModelCatalogPreservesProviderIdentity(t *testing.T) {
	mustInitConfig(t)
	setAdminPassword(t, "pw")
	for _, account := range []config.Account{
		{ID: "grok", Provider: "xai", Enabled: true},
		{ID: "codex", Provider: "codex", Enabled: true},
		{ID: "disabled", Provider: "antigravity", Enabled: false},
	} {
		if err := config.AddAccount(account); err != nil {
			t.Fatal(err)
		}
	}
	h := newAdminChatTestHandler(t)
	w := httptest.NewRecorder()
	h.handleAdminAPI(w, chatAdminRequest(http.MethodGet, "/admin/api/chat/models", "", "pw"))
	if w.Code != http.StatusOK {
		t.Fatalf("models status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Data []chatModelDTO `json:"data"`
	}
	decodeChatResponse(t, w, &response)
	if len(response.Data) == 0 {
		t.Fatal("empty model catalog")
	}
	seenProvider := map[string]bool{}
	seenID := map[string]bool{}
	for _, model := range response.Data {
		if seenID[model.ID] {
			t.Fatalf("duplicate id %q", model.ID)
		}
		seenID[model.ID] = true
		seenProvider[model.Provider] = true
		if model.ID != model.Provider+":"+model.Model {
			t.Fatalf("non-composite model=%+v", model)
		}
	}
	if !seenProvider["grok"] || !seenProvider["codex"] || seenProvider["antigravity"] || seenProvider["kiro"] {
		t.Fatalf("providers=%v", seenProvider)
	}
}
