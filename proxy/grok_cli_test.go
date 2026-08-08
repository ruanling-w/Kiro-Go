package proxy

import (
	"kiro-go/config"
	"testing"
)

func TestBuildGrokCLIResponsesRequestDropsOrphanOutput(t *testing.T) {
	req := &ClaudeRequest{Model: "grok-4-high", Messages: []ClaudeMessage{
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "tool_result", "tool_use_id": "missing", "content": "orphan"}, map[string]interface{}{"type": "text", "text": "continue"}}},
	}}
	body, err := BuildGrokCLIResponsesRequest(req, nil, req.Model, "session", true)
	if err != nil {
		t.Fatal(err)
	}
	input := body["input"].([]map[string]interface{})
	for _, item := range input {
		if item["type"] == "function_call_output" {
			t.Fatalf("orphan survived: %#v", input)
		}
	}
	if body["model"] != "grok-4" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["reasoning"].(map[string]interface{})["effort"] != "high" {
		t.Fatalf("reasoning = %#v", body["reasoning"])
	}
	if body["stream"] != true || body["store"] != false {
		t.Fatalf("stream/store = %#v", body)
	}
}

func TestBuildGrokCLIResponsesRequestKeepsToolLoop(t *testing.T) {
	call := ToolCall{ID: "call_1", Type: "function"}
	call.Function.Name = "read"
	call.Function.Arguments = `{"path":"a"}`
	req := &OpenAIRequest{Model: "grok-4", Messages: []OpenAIMessage{
		{Role: "assistant", ToolCalls: []ToolCall{call}},
		{Role: "tool", ToolCallID: "call_1", Content: "ok"},
	}}
	body, err := BuildGrokCLIResponsesRequest(nil, req, req.Model, "s", false)
	if err != nil {
		t.Fatal(err)
	}
	input := body["input"].([]map[string]interface{})
	if len(input) != 2 || input[0]["type"] != "function_call" || input[1]["type"] != "function_call_output" {
		t.Fatalf("input = %#v", input)
	}
}

func TestGrokCLITurnMonotonic(t *testing.T) {
	grokCLITurns.Lock()
	grokCLITurns.items = map[string]grokCLITurnState{}
	grokCLITurns.Unlock()
	input := []map[string]interface{}{{"type": "message", "role": "user"}}
	if got := resolveGrokCLITurn("s", input); got != 1 {
		t.Fatalf("first = %d", got)
	}
	if got := resolveGrokCLITurn("s", input); got != 2 {
		t.Fatalf("second = %d", got)
	}
}

func TestGrokCLISessionID(t *testing.T) {
	payload := &KiroPayload{}
	payload.ConversationState.ConversationID = "conversation"
	if got := grokCLISessionID(&config.Account{ID: "account"}, payload); got != "conversation" {
		t.Fatalf("session = %q", got)
	}
}

func TestGrokProviderClassification(t *testing.T) {
	oauth := &config.Account{Provider: "grok", GrokAuthType: "oauth", AccessToken: "token"}
	apiKey := &config.Account{Provider: "xai", GrokAuthType: "apikey", GrokAPIKey: "key"}
	legacy := &config.Account{Provider: "grok", AccessToken: "token", RefreshToken: "refresh"}
	if !isGrokOAuthAccount(oauth) || !isGrokOAuthAccount(legacy) {
		t.Fatal("OAuth Grok account not classified")
	}
	if isGrokOAuthAccount(apiKey) || !isGrokAccount(apiKey) {
		t.Fatal("API-key xAI account misclassified")
	}
}
