package proxy

import (
	"encoding/json"
	"kiro-go/config"
	"strings"
	"testing"
)

func TestClaudeToOpenAIReasoningEffort(t *testing.T) {
	body, err := ClaudeToOpenAI(&ClaudeRequest{Model: "grok-4-high"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if body["reasoning_effort"] != "high" {
		t.Fatalf("reasoning_effort = %v", body["reasoning_effort"])
	}
	if _, exists := body["reasoning"]; exists {
		t.Fatal("chat completions request must not contain Responses API reasoning object")
	}
}

func TestExtractTextFromOpenAIMessageReasoningContent(t *testing.T) {
	content, reasoning := extractTextFromOpenAIMessage(OpenAIMessage{
		Content:          "answer",
		ReasoningContent: "thought",
	})
	if content != "answer" || reasoning != "thought" {
		t.Fatalf("content=%q reasoning=%q", content, reasoning)
	}
}

func TestOpenAIToOpenAIPreservesHostedToolsAndOptions(t *testing.T) {
	parallel := true
	req := &OpenAIRequest{
		Model:             "grok-4",
		Messages:          []OpenAIMessage{{Role: "user", Content: "search"}},
		Tools:             []OpenAITool{{Type: "web_search"}},
		ParallelToolCalls: &parallel,
		ReasoningEffort:   "high",
		StreamOptions:     map[string]interface{}{"include_usage": true},
	}
	body, err := OpenAIToOpenAI(req)
	if err != nil {
		t.Fatal(err)
	}
	tools, ok := body["tools"].([]interface{})
	if !ok || len(tools) != 1 || tools[0].(map[string]interface{})["type"] != "web_search" {
		t.Fatalf("hosted tools = %#v", body["tools"])
	}
	if body["parallel_tool_calls"] != true || body["reasoning_effort"] != "high" {
		t.Fatalf("options lost: %#v", body)
	}
}

func toMsgs(t *testing.T, body map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw, ok := body["messages"]
	if !ok {
		t.Fatal("body has no messages")
	}
	arr, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("messages is %T, want []map[string]interface{}", raw)
	}
	return arr
}

func TestClaudeToOpenAI_SystemAndText(t *testing.T) {
	req := &ClaudeRequest{
		Model:     "grok-4",
		MaxTokens: 100,
		System:    "you are helpful",
		Messages: []ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
	if msgs[0]["role"] != "system" || msgs[0]["content"] != "you are helpful" {
		t.Errorf("system msg = %v", msgs[0])
	}
	if msgs[1]["role"] != "user" || msgs[1]["content"] != "hello" {
		t.Errorf("user msg = %v", msgs[1])
	}
	if body["max_tokens"] != 100 {
		t.Errorf("max_tokens = %v, want 100", body["max_tokens"])
	}
}

func TestClaudeToOpenAI_ToolUseAndResult(t *testing.T) {
	// assistant turn with a tool_use block, then a user turn with a tool_result.
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			{Role: "user", Content: "what's the weather?"},
			{Role: "assistant", Content: []interface{}{
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "toolu_1",
					"name":  "get_weather",
					"input": map[string]interface{}{"city": "Hanoi"},
				},
			}},
			{Role: "user", Content: []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": "toolu_1",
					"content":     "sunny, 30C",
				},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	// user, assistant(tool_calls), tool
	if len(msgs) != 3 {
		t.Fatalf("got %d messages, want 3: %+v", len(msgs), msgs)
	}

	assistant := msgs[1]
	if assistant["role"] != "assistant" {
		t.Fatalf("msgs[1] role = %v, want assistant", assistant["role"])
	}
	tcs, ok := assistant["tool_calls"].([]map[string]interface{})
	if !ok || len(tcs) != 1 {
		t.Fatalf("tool_calls = %v", assistant["tool_calls"])
	}
	if tcs[0]["id"] != "toolu_1" {
		t.Errorf("tool_call id = %v", tcs[0]["id"])
	}
	fn := tcs[0]["function"].(map[string]interface{})
	if fn["name"] != "get_weather" {
		t.Errorf("fn name = %v", fn["name"])
	}
	// arguments must be a JSON string
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatalf("arguments not valid JSON: %v", err)
	}
	if args["city"] != "Hanoi" {
		t.Errorf("args city = %v", args["city"])
	}

	tool := msgs[2]
	if tool["role"] != "tool" {
		t.Errorf("msgs[2] role = %v, want tool", tool["role"])
	}
	if tool["tool_call_id"] != "toolu_1" {
		t.Errorf("tool_call_id = %v", tool["tool_call_id"])
	}
	if tool["content"] != "sunny, 30C" {
		t.Errorf("tool content = %v", tool["content"])
	}
}

// TestClaudeToOpenAI_ToolResultPrecedesUserText covers the ordering bug that
// broke the agentic loop: Claude Code packs tool_result blocks and ordinary text
// (system-reminders) into the SAME user turn. OpenAI/xAI require role:"tool"
// messages to sit immediately after the assistant turn that requested them, so
// the tool message must come before the user text.
func TestClaudeToOpenAI_ToolResultPrecedesUserText(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			{Role: "user", Content: "read the file"},
			{Role: "assistant", Content: []interface{}{
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "toolu_1",
					"name":  "read_file",
					"input": map[string]interface{}{"path": "a.go"},
				},
			}},
			// Claude Code shape: tool_result AND text in one user turn.
			{Role: "user", Content: []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": "toolu_1",
					"content":     "package main",
				},
				map[string]interface{}{
					"type": "text",
					"text": "<system-reminder>keep going</system-reminder>",
				},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	// user, assistant(tool_calls), tool, user(text)
	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4: %+v", len(msgs), msgs)
	}
	if msgs[1]["role"] != "assistant" {
		t.Fatalf("msgs[1] role = %v, want assistant", msgs[1]["role"])
	}
	if msgs[2]["role"] != "tool" {
		t.Fatalf("msgs[2] role = %v, want tool immediately after assistant", msgs[2]["role"])
	}
	if msgs[2]["tool_call_id"] != "toolu_1" {
		t.Errorf("tool_call_id = %v", msgs[2]["tool_call_id"])
	}
	if msgs[3]["role"] != "user" {
		t.Fatalf("msgs[3] role = %v, want user text after the tool result", msgs[3]["role"])
	}
	if msgs[3]["content"] != "<system-reminder>keep going</system-reminder>" {
		t.Errorf("user text = %v", msgs[3]["content"])
	}
}

// TestClaudeToOpenAI_DropsOrphanToolResult covers history trimmed by the client:
// a tool_result whose tool_use is gone would make xAI reject the whole request.
func TestClaudeToOpenAI_DropsOrphanToolResult(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			// No assistant tool_use anywhere — this result is an orphan.
			{Role: "user", Content: []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": "toolu_gone",
					"content":     "stale output",
				},
				map[string]interface{}{"type": "text", "text": "continue"},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	for _, m := range msgs {
		if m["role"] == "tool" {
			t.Fatalf("orphan tool message survived: %+v", msgs)
		}
	}
	if len(msgs) != 1 || msgs[0]["role"] != "user" || msgs[0]["content"] != "continue" {
		t.Fatalf("want single user message, got %+v", msgs)
	}
}

// A tool_result whose tool_use appeared in an earlier turn must be kept.
func TestClaudeToOpenAI_KeepsToolResultAcrossTurns(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			{Role: "assistant", Content: []interface{}{
				map[string]interface{}{
					"type": "tool_use", "id": "toolu_a", "name": "f",
					"input": map[string]interface{}{},
				},
			}},
			{Role: "user", Content: []interface{}{
				map[string]interface{}{"type": "tool_result", "tool_use_id": "toolu_a", "content": "ok"},
			}},
			{Role: "assistant", Content: []interface{}{
				map[string]interface{}{
					"type": "tool_use", "id": "toolu_b", "name": "g",
					"input": map[string]interface{}{},
				},
			}},
			{Role: "user", Content: []interface{}{
				map[string]interface{}{"type": "tool_result", "tool_use_id": "toolu_b", "content": "ok2"},
				// Orphan mixed in with a valid one.
				map[string]interface{}{"type": "tool_result", "tool_use_id": "toolu_ghost", "content": "??"},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	var toolIDs []string
	for _, m := range msgs {
		if m["role"] == "tool" {
			toolIDs = append(toolIDs, m["tool_call_id"].(string))
		}
	}
	if len(toolIDs) != 2 || toolIDs[0] != "toolu_a" || toolIDs[1] != "toolu_b" {
		t.Fatalf("tool ids = %v, want [toolu_a toolu_b]", toolIDs)
	}
}

// tool_use blocks missing an id or name would produce a malformed tool_calls
// entry that xAI rejects.
func TestClaudeToOpenAI_SkipsIncompleteToolUse(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			{Role: "assistant", Content: []interface{}{
				map[string]interface{}{"type": "tool_use", "id": "", "name": "f"},
				map[string]interface{}{"type": "tool_use", "id": "toolu_1", "name": ""},
				map[string]interface{}{"type": "text", "text": "hi"},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	if len(msgs) != 1 {
		t.Fatalf("got %d msgs, want 1: %+v", len(msgs), msgs)
	}
	if _, has := msgs[0]["tool_calls"]; has {
		t.Fatalf("incomplete tool_use produced tool_calls: %+v", msgs[0])
	}
	if msgs[0]["content"] != "hi" {
		t.Errorf("content = %v", msgs[0]["content"])
	}
}

// TestClaudeToolChoiceToOpenAI covers A3: Anthropic tool_choice objects must be
// rewritten into the OpenAI shape, and anything unrecognized dropped rather than
// forwarded verbatim (xAI 400s on the Claude form).
func TestClaudeToolChoiceToOpenAI(t *testing.T) {
	tools := []map[string]interface{}{
		{"type": "function", "function": map[string]interface{}{"name": "get_weather"}},
	}

	cases := []struct {
		name string
		in   interface{}
		want interface{}
	}{
		{"nil", nil, nil},
		{"claude auto", map[string]interface{}{"type": "auto"}, "auto"},
		{"claude any", map[string]interface{}{"type": "any"}, "required"},
		{"claude none", map[string]interface{}{"type": "none"}, "none"},
		{"string auto", "auto", "auto"},
		{"string required", "required", "required"},
		{"string any", "any", "required"},
		{"unknown string", "banana", nil},
		{"unknown type", map[string]interface{}{"type": "banana"}, nil},
		{"forced unknown tool", map[string]interface{}{"type": "tool", "name": "nope"}, nil},
		{"forced without name", map[string]interface{}{"type": "tool"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := claudeToolChoiceToOpenAI(c.in, tools)
			if got != c.want {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
		})
	}

	// Forced tool that IS declared → nested OpenAI shape.
	got := claudeToolChoiceToOpenAI(
		map[string]interface{}{"type": "tool", "name": "get_weather"}, tools)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("forced choice = %#v, want map", got)
	}
	if m["type"] != "function" {
		t.Errorf("type = %v, want function", m["type"])
	}
	fn, ok := m["function"].(map[string]interface{})
	if !ok || fn["name"] != "get_weather" {
		t.Errorf("function = %#v", m["function"])
	}

	// OpenAI nested form passes through unchanged in meaning.
	got = claudeToolChoiceToOpenAI(map[string]interface{}{
		"type":     "function",
		"function": map[string]interface{}{"name": "get_weather"},
	}, tools)
	m, ok = got.(map[string]interface{})
	if !ok || m["function"].(map[string]interface{})["name"] != "get_weather" {
		t.Errorf("nested form = %#v", got)
	}
}

// A Claude tool_choice object must never reach the body as-is.
func TestClaudeToOpenAI_RewritesToolChoice(t *testing.T) {
	req := &ClaudeRequest{
		Model:      "grok-4",
		Messages:   []ClaudeMessage{{Role: "user", Content: "weather?"}},
		Tools:      []ClaudeTool{{Name: "get_weather", InputSchema: map[string]interface{}{"type": "object"}}},
		ToolChoice: map[string]interface{}{"type": "any"},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	if body["tool_choice"] != "required" {
		t.Fatalf("tool_choice = %#v, want \"required\"", body["tool_choice"])
	}
}

func TestClaudeToOpenAI_ImageBlock(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4",
		Messages: []ClaudeMessage{
			{Role: "user", Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "describe this"},
				map[string]interface{}{
					"type": "image",
					"source": map[string]interface{}{
						"media_type": "image/png",
						"data":       "AAAA",
					},
				},
			}},
		},
	}
	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}
	msgs := toMsgs(t, body)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	parts, ok := msgs[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("content is %T, want multipart array", msgs[0]["content"])
	}
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	if parts[0]["type"] != "text" {
		t.Errorf("part0 type = %v", parts[0]["type"])
	}
	if parts[1]["type"] != "image_url" {
		t.Errorf("part1 type = %v", parts[1]["type"])
	}
	iu := parts[1]["image_url"].(map[string]string)
	if iu["url"] != "data:image/png;base64,AAAA" {
		t.Errorf("image url = %v", iu["url"])
	}
}

func TestClaudeStopReasonFromFinish(t *testing.T) {
	cases := map[string]string{
		"stop": "end_turn", "length": "max_tokens", "tool_calls": "tool_use",
		"content_filter": "refusal", "unknown": "", "": "",
	}
	for in, want := range cases {
		if got := claudeStopReasonFromFinish(in); got != want {
			t.Errorf("claudeStopReasonFromFinish(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseGrokOpenAISSEReportsUsageAndFinishReason(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"hello"},"finish_reason":null}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":17,"completion_tokens":3}}`,
		`data: [DONE]`, "",
	}, "\n\n")
	var text, finish string
	var inTok, outTok int
	err := parseGrokOpenAISSE(strings.NewReader(stream), &KiroStreamCallback{
		OnText:         func(s string, _ bool) { text += s },
		OnComplete:     func(in, out int) { inTok, outTok = in, out },
		OnFinishReason: func(reason string) { finish = reason },
	}, "grok-4")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" || finish != "length" || inTok != 17 || outTok != 3 {
		t.Fatalf("text=%q finish=%q usage=%d/%d", text, finish, inTok, outTok)
	}
}

func TestParseGrokOpenAISSEDoesNotFabricateUsage(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"content\":\"12345678\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	var inTok, outTok int
	err := parseGrokOpenAISSE(strings.NewReader(stream), &KiroStreamCallback{
		OnComplete: func(in, out int) { inTok, outTok = in, out },
	}, "grok-4")
	if err != nil {
		t.Fatal(err)
	}
	if inTok != 0 || outTok != 0 {
		t.Fatalf("fabricated usage: %d/%d", inTok, outTok)
	}
}

func TestResolveGrokModel(t *testing.T) {
	cases := map[string]string{
		"":                      "grok-4",
		"grok-3":                "grok-3",
		" grok-4.5-high ":       "grok-4.5",
		"grok-4.5-medium":       "grok-4.5",
		"grok-4.5-low":          "grok-4.5",
		"grok-4.5-xhigh":        "grok-4.5",
		"grok-4-thinking":       "grok-4",
		"grok-4-fast-reasoning": "grok-4-fast-reasoning",
		"grok-code-fast-1":      "grok-code-fast-1",
	}
	for in, want := range cases {
		if got := resolveGrokModel(in); got != want {
			t.Errorf("resolveGrokModel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestClaudeToOpenAI_SanitizesNullSchemaFields covers the xAI 400:
//
//	Schema validation failed: (root): null is not of types "boolean", "object"
//
// which clients trigger by emitting additionalProperties:null (and similar
// null schema fields) inside tool input_schema.
func TestClaudeToOpenAI_SanitizesNullSchemaFields(t *testing.T) {
	req := &ClaudeRequest{
		Model: "grok-4.5",
		Messages: []ClaudeMessage{
			{Role: "user", Content: "use tools"},
		},
		Tools: []ClaudeTool{
			{
				Name:        "read_file",
				Description: "Read a file",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":                 "string",
							"description":          "file path",
							"additionalProperties": nil, // ← the bad value
						},
						"options": map[string]interface{}{
							"type":                 []interface{}{"object", "null"},
							"additionalProperties": nil,
							"properties": map[string]interface{}{
								"offset": map[string]interface{}{
									"type":    []interface{}{"integer", "null"},
									"minimum": nil,
								},
							},
							"required": []interface{}{"offset", nil, ""},
						},
					},
					"additionalProperties": nil,
					"required":             []interface{}{"path"},
					"items":                nil,
				},
			},
			{
				Name:        "noop",
				Description: "no params",
				InputSchema: nil,
			},
		},
	}

	body, err := ClaudeToOpenAI(req, false)
	if err != nil {
		t.Fatalf("ClaudeToOpenAI: %v", err)
	}

	tools, ok := body["tools"].([]map[string]interface{})
	if !ok || len(tools) != 2 {
		t.Fatalf("tools = %#v", body["tools"])
	}

	// Walk the marshaled JSON and assert no null values remain anywhere under tools.
	raw, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if n := countNulls(decoded); n != 0 {
		t.Fatalf("expected 0 nulls in sanitized tools, found %d\n%s", n, string(raw))
	}

	fn0 := tools[0]["function"].(map[string]interface{})
	params0 := fn0["parameters"].(map[string]interface{})
	if _, still := params0["additionalProperties"]; still {
		t.Fatalf("root additionalProperties should be dropped when null, got %#v", params0["additionalProperties"])
	}
	if _, still := params0["items"]; still {
		t.Fatalf("null items should be dropped, got %#v", params0["items"])
	}

	props := params0["properties"].(map[string]interface{})
	pathSchema := props["path"].(map[string]interface{})
	if _, still := pathSchema["additionalProperties"]; still {
		t.Fatalf("nested additionalProperties:null should be dropped, got %#v", pathSchema)
	}

	options := props["options"].(map[string]interface{})
	// type: ["object","null"] → "object"
	if options["type"] != "object" {
		t.Fatalf("options.type = %#v, want \"object\"", options["type"])
	}
	reqList, _ := options["required"].([]interface{})
	if len(reqList) != 1 || reqList[0] != "offset" {
		t.Fatalf("options.required = %#v, want [offset]", reqList)
	}

	// nil InputSchema → empty object schema
	fn1 := tools[1]["function"].(map[string]interface{})
	params1 := fn1["parameters"].(map[string]interface{})
	if params1["type"] != "object" {
		t.Fatalf("noop parameters type = %#v", params1["type"])
	}
	if _, ok := params1["properties"].(map[string]interface{}); !ok {
		t.Fatalf("noop parameters missing properties: %#v", params1)
	}
}

func TestOpenAIToOpenAI_SanitizesNullSchemaFields(t *testing.T) {
	req := &OpenAIRequest{
		Model: "grok-4.5",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "hi"},
		},
		Tools: []OpenAITool{
			{
				Type: "function",
			},
		},
	}
	// Set nested function fields via the UnmarshalJSON-compatible path: build
	// the tool through JSON so the embedded struct is populated cleanly.
	rawTool := []byte(`{
		"type":"function",
		"function":{
			"name":"exec_command",
			"description":"Run a shell command",
			"parameters":{
				"type":"object",
				"properties":{
					"cmd":{"type":"string","additionalProperties":null}
				},
				"additionalProperties":null
			}
		}
	}`)
	if err := json.Unmarshal(rawTool, &req.Tools[0]); err != nil {
		t.Fatalf("unmarshal tool: %v", err)
	}

	body, err := OpenAIToOpenAI(req)
	if err != nil {
		t.Fatalf("OpenAIToOpenAI: %v", err)
	}

	raw, err := json.Marshal(body["tools"])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n := countNulls(decoded); n != 0 {
		t.Fatalf("expected 0 nulls in sanitized OpenAI tools, found %d\n%s", n, string(raw))
	}
}

// TestOpenAIToOpenAI_KeepsEmptyToolProperties guards the interaction between
// sanitizeGrokToolParameters (which deliberately emits `properties: {}` because
// xAI requires an object schema) and cleanEmptyOpenAIFields (which used to
// recurse and delete it again, producing HTTP 400).
func TestOpenAIToOpenAI_KeepsEmptyToolProperties(t *testing.T) {
	req := &OpenAIRequest{
		Model:    "grok-4",
		Messages: []OpenAIMessage{{Role: "user", Content: "list files"}},
		Tools:    []OpenAITool{{}},
	}
	rawTool := []byte(`{
		"type":"function",
		"function":{"name":"now","description":"current time","parameters":{"type":"object","properties":{}}}
	}`)
	if err := json.Unmarshal(rawTool, &req.Tools[0]); err != nil {
		t.Fatalf("unmarshal tool: %v", err)
	}

	body, err := OpenAIToOpenAI(req)
	if err != nil {
		t.Fatalf("OpenAIToOpenAI: %v", err)
	}

	tools, ok := body["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v", body["tools"])
	}
	fn := tools[0].(map[string]interface{})["function"].(map[string]interface{})
	params, ok := fn["parameters"].(map[string]interface{})
	if !ok {
		t.Fatalf("parameters = %#v", fn["parameters"])
	}
	if params["type"] != "object" {
		t.Errorf("parameters.type = %#v, want object", params["type"])
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties was stripped: %#v", params)
	}
	if len(props) != 0 {
		t.Errorf("properties = %#v, want empty object", props)
	}
}

// TestOpenAIToOpenAI_KeepsEmptyToolResultContent covers a tool that ran and
// returned nothing. `content: ""` is a valid role:"tool" message; stripping it
// leaves a message with no content field, which breaks the tool protocol.
func TestOpenAIToOpenAI_KeepsEmptyToolResultContent(t *testing.T) {
	req := &OpenAIRequest{
		Model: "grok-4",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "delete the temp file"},
			{Role: "assistant", Content: nil, ToolCalls: []ToolCall{
				func() ToolCall {
					var tc ToolCall
					tc.ID = "call_1"
					tc.Type = "function"
					tc.Function.Name = "rm"
					tc.Function.Arguments = `{"path":"/tmp/x"}`
					return tc
				}(),
			}},
			{Role: "tool", ToolCallID: "call_1", Content: ""},
		},
	}

	body, err := OpenAIToOpenAI(req)
	if err != nil {
		t.Fatalf("OpenAIToOpenAI: %v", err)
	}
	msgs, ok := body["messages"].([]interface{})
	if !ok || len(msgs) != 3 {
		t.Fatalf("messages = %#v", body["messages"])
	}

	toolMsg := msgs[2].(map[string]interface{})
	if toolMsg["role"] != "tool" {
		t.Fatalf("msgs[2] role = %v", toolMsg["role"])
	}
	if _, ok := toolMsg["content"]; !ok {
		t.Errorf("tool message lost its content field: %#v", toolMsg)
	}
	if toolMsg["tool_call_id"] != "call_1" {
		t.Errorf("tool_call_id = %v", toolMsg["tool_call_id"])
	}

	assistant := msgs[1].(map[string]interface{})
	tcs, ok := assistant["tool_calls"].([]interface{})
	if !ok || len(tcs) != 1 {
		t.Fatalf("tool_calls = %#v", assistant["tool_calls"])
	}
	fn := tcs[0].(map[string]interface{})["function"].(map[string]interface{})
	if fn["arguments"] != `{"path":"/tmp/x"}` {
		t.Errorf("arguments = %#v", fn["arguments"])
	}
}

// TestCleanEmptyOpenAIFieldsIsShallow pins the top-level-only contract.
func TestCleanEmptyOpenAIFieldsIsShallow(t *testing.T) {
	body := map[string]interface{}{
		"model": "grok-4",
		"user":  "",
		"stop":  []interface{}{},
		"tools": []interface{}{
			map[string]interface{}{"function": map[string]interface{}{
				"name":       "x",
				"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			}},
		},
		"messages": []interface{}{
			map[string]interface{}{"role": "tool", "tool_call_id": "c1", "content": ""},
		},
		"metadata": map[string]interface{}{},
	}
	cleanEmptyOpenAIFields(body)

	if _, still := body["user"]; still {
		t.Error(`empty "user" should be dropped`)
	}
	if _, still := body["stop"]; still {
		t.Error(`empty "stop" should be dropped`)
	}
	if _, still := body["metadata"]; still {
		t.Error(`empty "metadata" should be dropped`)
	}

	tools := body["tools"].([]interface{})
	params := tools[0].(map[string]interface{})["function"].(map[string]interface{})["parameters"].(map[string]interface{})
	if _, ok := params["properties"]; !ok {
		t.Error("must not recurse into tools: properties was stripped")
	}

	msgs := body["messages"].([]interface{})
	if _, ok := msgs[0].(map[string]interface{})["content"]; !ok {
		t.Error("must not recurse into messages: tool content was stripped")
	}
}

func TestSanitizeGrokToolParameters_PreservesBoolAdditionalProperties(t *testing.T) {
	in := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"x": map[string]interface{}{"type": "string"},
		},
	}
	out, ok := sanitizeGrokToolParameters(in).(map[string]interface{})
	if !ok {
		t.Fatalf("got %T", sanitizeGrokToolParameters(in))
	}
	if out["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %#v, want false", out["additionalProperties"])
	}
	// Caller schema must not be mutated.
	if in["additionalProperties"] != false {
		t.Fatalf("sanitizer mutated input")
	}
}

func countNulls(v interface{}) int {
	switch val := v.(type) {
	case nil:
		return 1
	case map[string]interface{}:
		n := 0
		for _, child := range val {
			n += countNulls(child)
		}
		return n
	case []interface{}:
		n := 0
		for _, child := range val {
			n += countNulls(child)
		}
		return n
	default:
		return 0
	}
}

func TestGrokModelInfosImageFlags(t *testing.T) {
	infos := grokModelInfos()
	if len(infos) == 0 {
		t.Fatal("static catalog empty")
	}
	var image *ModelInfo
	for i := range infos {
		if infos[i].ModelId == "grok-2-image-1212" {
			image = &infos[i]
			break
		}
	}
	if image == nil {
		t.Fatal("missing grok-2-image-1212 in static catalog")
	}
	if !modelSupportsImage(image.InputTypes) {
		t.Fatalf("image model InputTypes=%v want image", image.InputTypes)
	}
	for _, m := range infos {
		if m.ModelId == "grok-2-image-1212" {
			continue
		}
		if modelSupportsImage(m.InputTypes) {
			t.Fatalf("chat model %s should not be image-flagged, InputTypes=%v", m.ModelId, m.InputTypes)
		}
	}
}

func TestEnhanceGrokModelInfosFiltersAndFlags(t *testing.T) {
	in := []ModelInfo{
		{ModelId: "grok-4"},
		{ModelId: "grok-2-image-1212"},
		{ModelId: "something-else"},
		{ModelId: "grok-4"}, // dup
	}
	out := enhanceGrokModelInfos(in)
	if len(out) != 2 {
		t.Fatalf("want 2 grok models, got %d: %+v", len(out), out)
	}
	byID := map[string]ModelInfo{}
	for _, m := range out {
		byID[m.ModelId] = m
	}
	if byID["grok-4"].Description != "xAI Grok" {
		t.Fatalf("description not set: %+v", byID["grok-4"])
	}
	if !modelSupportsImage(byID["grok-2-image-1212"].InputTypes) {
		t.Fatalf("image flags missing: %+v", byID["grok-2-image-1212"])
	}
}

func TestEnhanceGrokFromOpenAIBody(t *testing.T) {
	body := []byte(`{"object":"list","data":[{"id":"grok-4"},{"id":"grok-2-image-1212"},{"id":"other"}]}`)
	ids, err := parseOpenAIModelIDs(body)
	if err != nil {
		t.Fatal(err)
	}
	infos := make([]ModelInfo, 0, len(ids))
	for _, id := range ids {
		infos = append(infos, ModelInfo{ModelId: id})
	}
	out := enhanceGrokModelInfos(infos)
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
}

func TestResolveGrokModelsFallsBackWithoutCreds(t *testing.T) {
	got := resolveGrokModels(&config.Account{Email: "x@y.z", Provider: "grok"})
	if len(got) == 0 {
		t.Fatal("expected static fallback")
	}
	// Same length as static catalog.
	if len(got) != len(grokModelInfos()) {
		t.Fatalf("fallback len=%d static=%d", len(got), len(grokModelInfos()))
	}
}
