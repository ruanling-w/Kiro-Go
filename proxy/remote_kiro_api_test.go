package proxy

import (
	"context"
	"encoding/json"
	"io"
	"kiro-go/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withPrivateRemoteAllowed(t *testing.T) {
	t.Helper()
	t.Setenv("KIRO_ALLOW_PRIVATE_REMOTE", "1")
}

func ensureConfigForRemoteTests(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := config.Init(dir + "/config.json"); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	InitKiroHttpClient("")
}

func TestParseOpenAIModelIDs(t *testing.T) {
	body := []byte(`{"object":"list","data":[{"id":"claude-sonnet-4.5"},{"id":"claude-sonnet-4.5"},{"id":"  "},{"id":"gpt-4o"}]}`)
	ids, err := parseOpenAIModelIDs(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "claude-sonnet-4.5" || ids[1] != "gpt-4o" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestFetchRemoteKiroModels(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("auth=%q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"},{"id":"m2"}]}`))
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-test",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	ids, err := FetchRemoteKiroModels(acc)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestCallRemoteKiroAPINonStreamOpenAI(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-1",
			"object":"chat.completion",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello remote"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
		}`))
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-x",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	payload := &KiroPayload{
		SourceOpenAI: &OpenAIRequest{
			Model: "claude-sonnet-4.5",
			Messages: []OpenAIMessage{
				{Role: "user", Content: "hi"},
			},
			Stream: false,
		},
	}
	// Seed Kiro shape model so resolvePayloadModelForGrok finds it.
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = "claude-sonnet-4.5"

	var text strings.Builder
	var completed bool
	err := CallRemoteKiroAPI(context.Background(), acc, payload, &KiroStreamCallback{
		OnText: func(s string, _ bool) { text.WriteString(s) },
		OnComplete: func(in, out int) {
			completed = true
			if in != 3 || out != 2 {
				t.Errorf("tokens in=%d out=%d", in, out)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !completed {
		t.Fatal("OnComplete not called")
	}
	if text.String() != "hello remote" {
		t.Fatalf("text=%q", text.String())
	}
	if gotBody["model"] != "claude-sonnet-4.5" {
		t.Fatalf("model rewritten: %v", gotBody["model"])
	}
	if gotBody["stream"] != false {
		t.Fatalf("stream=%v", gotBody["stream"])
	}
}

func TestCallRemoteKiroAPIStream(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, "data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-x",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	payload := &KiroPayload{
		SourceOpenAI: &OpenAIRequest{
			Model:    "m",
			Messages: []OpenAIMessage{{Role: "user", Content: "x"}},
			Stream:   true,
		},
	}
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = "m"

	var text strings.Builder
	err := CallRemoteKiroAPI(context.Background(), acc, payload, &KiroStreamCallback{
		OnText:     func(s string, _ bool) { text.WriteString(s) },
		OnComplete: func(in, out int) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "hi" {
		t.Fatalf("text=%q", text.String())
	}
}

func TestCallRemoteKiroAPIClaudeSource(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	var gotPath string
	var gotBody map[string]interface{}
	var gotAPIKey, gotAnthropicVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-api-key")
		gotAnthropicVersion = r.Header.Get("anthropic-version")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		// Anthropic non-stream shape — NOT OpenAI chat.completions.
		_, _ = w.Write([]byte(`{
			"id":"msg_1",
			"type":"message",
			"role":"assistant",
			"content":[{"type":"text","text":"ok"}],
			"model":"claude-sonnet-4.5",
			"stop_reason":"end_turn",
			"usage":{"input_tokens":4,"output_tokens":1}
		}`))
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-x",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	payload := &KiroPayload{
		SourceClaude: &ClaudeRequest{
			Model:     "claude-sonnet-4.5",
			MaxTokens: 64,
			Messages: []ClaudeMessage{
				{Role: "user", Content: "hello"},
			},
			Stream: false,
		},
	}
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = "claude-sonnet-4.5"

	var text strings.Builder
	var inTok, outTok int
	err := CallRemoteKiroAPI(context.Background(), acc, payload, &KiroStreamCallback{
		OnText: func(s string, thinking bool) {
			if thinking {
				t.Fatalf("unexpected thinking: %q", s)
			}
			text.WriteString(s)
		},
		OnComplete: func(in, out int) { inTok, outTok = in, out },
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path=%q want /v1/messages (Claude must not be converted to OpenAI)", gotPath)
	}
	if gotAPIKey != "sk-x" {
		t.Fatalf("x-api-key=%q", gotAPIKey)
	}
	if gotAnthropicVersion == "" {
		t.Fatal("missing anthropic-version header")
	}
	if gotBody["model"] != "claude-sonnet-4.5" {
		t.Fatalf("model=%v", gotBody["model"])
	}
	if _, isOpenAI := gotBody["messages"]; !isOpenAI {
		// Claude body uses "messages" too — ensure max_tokens (Anthropic snake) present
		// and OpenAI-only fields like "stream" bool still ok. Main guard is path above.
	}
	if gotBody["max_tokens"] == nil {
		t.Fatalf("expected Anthropic max_tokens in body, got %v", gotBody)
	}
	if text.String() != "ok" {
		t.Fatalf("text=%q", text.String())
	}
	if inTok != 4 || outTok != 1 {
		t.Fatalf("tokens in=%d out=%d", inTok, outTok)
	}
}

func TestCallRemoteKiroAPIClaudeStream(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"usage\":{\"input_tokens\":9,\"output_tokens\":0}}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi \"}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"remote\"}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		fl.Flush()
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-x",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	payload := &KiroPayload{
		SourceClaude: &ClaudeRequest{
			Model:     "claude-opus-5",
			MaxTokens: 32,
			Messages:  []ClaudeMessage{{Role: "user", Content: "x"}},
			Stream:    true,
		},
	}
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = "claude-opus-5"

	var text strings.Builder
	var inTok, outTok int
	err := CallRemoteKiroAPI(context.Background(), acc, payload, &KiroStreamCallback{
		OnText:     func(s string, _ bool) { text.WriteString(s) },
		OnComplete: func(in, out int) { inTok, outTok = in, out },
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "hi remote" {
		t.Fatalf("text=%q", text.String())
	}
	if inTok != 9 || outTok != 2 {
		t.Fatalf("tokens in=%d out=%d", inTok, outTok)
	}
}

func TestCallRemoteKiroAPIClaudeEmptyIsError(t *testing.T) {
	withPrivateRemoteAllowed(t)
	ensureConfigForRemoteTests(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 200 OK but no content — previously logged as success and client retried.
		_, _ = w.Write([]byte(`{
			"id":"msg_empty",
			"type":"message",
			"role":"assistant",
			"content":[],
			"model":"claude-opus-5",
			"stop_reason":"end_turn",
			"usage":{"input_tokens":40,"output_tokens":0}
		}`))
	}))
	defer srv.Close()

	acc := &config.Account{
		RemoteBaseURL: srv.URL,
		AccessToken:   "sk-x",
		AuthMethod:    "remotekiro",
		Provider:      "remotekiro",
	}
	payload := &KiroPayload{
		SourceClaude: &ClaudeRequest{
			Model:     "claude-opus-5",
			MaxTokens: 16,
			Messages:  []ClaudeMessage{{Role: "user", Content: "x"}},
		},
	}
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = "claude-opus-5"

	err := CallRemoteKiroAPI(context.Background(), acc, payload, &KiroStreamCallback{
		OnText:     func(string, bool) {},
		OnComplete: func(int, int) { t.Fatal("OnComplete must not fire on empty") },
	})
	if err == nil || !strings.Contains(err.Error(), "empty claude") {
		t.Fatalf("err=%v", err)
	}
}

func TestIsRemoteKiroAccountAndProviderLabel(t *testing.T) {
	acc := &config.Account{AuthMethod: "remotekiro", Provider: "remotekiro"}
	if !isRemoteKiroAccount(acc) {
		t.Fatal("expected remotekiro")
	}
	if providerLabel(acc) != "remotekiro" {
		t.Fatalf("label=%q", providerLabel(acc))
	}
	// Fallback on RemoteBaseURL alone.
	acc2 := &config.Account{RemoteBaseURL: "https://x.example"}
	// Without private allow, detection still true; label uses provider field empty → remotekiro via AuthMethod empty + RemoteBaseURL path in isRemote.
	if !isRemoteKiroAccount(acc2) {
		t.Fatal("expected RemoteBaseURL fallback")
	}
}

func TestClassifyRemoteKiroFailureIsSoft(t *testing.T) {
	acc := &config.Account{ID: "r1", AuthMethod: "remotekiro", Provider: "remotekiro"}
	got := classifyAccountFailure(acc, errString("HTTP 403 forbidden"), false)
	if got != EventSoft {
		t.Fatalf("event=%q", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
