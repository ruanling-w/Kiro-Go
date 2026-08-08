package proxy

import (
	"fmt"
	"strings"
)

var grokCLIHostedTools = map[string]bool{
	"web_search": true, "x_search": true, "web_search_preview": true,
	"file_search": true, "image_generation": true, "code_interpreter": true,
	"mcp": true, "local_shell": true,
}

func resolveGrokCLIModel(model string) (string, string) {
	m := strings.TrimSpace(model)
	if m == "" {
		return "grok-4", "high"
	}
	lower := strings.ToLower(m)
	for _, effort := range []string{"xhigh", "high", "medium", "low"} {
		if strings.HasSuffix(lower, "-"+effort) {
			return m[:len(m)-len(effort)-1], effort
		}
	}
	if strings.HasSuffix(lower, "-max") {
		return m[:len(m)-4], "xhigh"
	}
	if strings.HasSuffix(lower, "-thinking") {
		return m[:len(m)-9], "high"
	}
	return m, "high"
}

func BuildGrokCLIResponsesRequest(sourceClaude *ClaudeRequest, sourceOpenAI *OpenAIRequest, model, sessionID string, thinking bool) (map[string]interface{}, error) {
	upstream, effort := resolveGrokCLIModel(model)
	var input []map[string]interface{}
	var instructions string
	var tools []map[string]interface{}
	var toolChoice interface{}

	switch {
	case sourceClaude != nil:
		instructions = extractClaudeSystemPrompt(sourceClaude.System)
		input = claudeMessagesToResponsesInput(sourceClaude.Messages)
		tools = claudeToolsToResponsesTools(sourceClaude.Tools)
		toolChoice = sourceClaude.ToolChoice
	case sourceOpenAI != nil:
		instructions, input = openAIMessagesToResponsesInput(sourceOpenAI.Messages)
		tools = grokCLIOpenAITools(sourceOpenAI.Tools)
		toolChoice = sourceOpenAI.ToolChoice
	default:
		return nil, fmt.Errorf("grok cli: no source request")
	}

	input = normalizeGrokCLIInput(stripCodexStoredItems(input))
	if len(input) == 0 {
		input = []map[string]interface{}{{"type": "message", "role": "user", "content": []map[string]interface{}{{"type": "input_text", "text": "..."}}}}
	}
	body := map[string]interface{}{
		"model": upstream, "input": input, "stream": true, "store": false,
		"prompt_cache_key": sessionID,
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	if thinking || effort != "" {
		body["reasoning"] = map[string]interface{}{"effort": effort, "summary": "auto"}
		body["include"] = []string{"reasoning.encrypted_content"}
	}
	if len(tools) > 0 {
		body["tools"] = tools
		if choice := normalizeGrokCLIToolChoice(toolChoice, tools); choice != nil {
			body["tool_choice"] = choice
		}
	}
	if sourceClaude != nil && sourceClaude.MaxTokens > 0 {
		body["max_output_tokens"] = sourceClaude.MaxTokens
	}
	if sourceOpenAI != nil {
		if sourceOpenAI.MaxTokens > 0 {
			body["max_output_tokens"] = sourceOpenAI.MaxTokens
		}
		if sourceOpenAI.ParallelToolCalls != nil {
			body["parallel_tool_calls"] = *sourceOpenAI.ParallelToolCalls
		}
	}
	return body, nil
}

func grokCLIOpenAITools(tools []OpenAITool) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		typ := strings.ToLower(strings.TrimSpace(tool.Type))
		if grokCLIHostedTools[typ] && tool.Function.Name == "" {
			out = append(out, map[string]interface{}{"type": typ})
			continue
		}
		if tool.Function.Name == "" {
			continue
		}
		params := tool.Function.Parameters
		if params == nil {
			params = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		entry := map[string]interface{}{"type": "function", "name": tool.Function.Name, "parameters": params}
		if tool.Function.Description != "" {
			entry["description"] = tool.Function.Description
		}
		out = append(out, entry)
	}
	return out
}

func normalizeGrokCLIInput(input []map[string]interface{}) []map[string]interface{} {
	calls := map[string]bool{}
	cleaned := make([]map[string]interface{}, 0, len(input))
	for _, item := range input {
		typ, _ := item["type"].(string)
		if typ == "function_call" {
			id, _ := item["call_id"].(string)
			name, _ := item["name"].(string)
			if id == "" || strings.TrimSpace(name) == "" {
				continue
			}
			calls[id] = true
		}
		cleaned = append(cleaned, item)
	}
	out := cleaned[:0]
	for _, item := range cleaned {
		if item["type"] == "function_call_output" {
			id, _ := item["call_id"].(string)
			if !calls[id] {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func normalizeGrokCLIToolChoice(choice interface{}, tools []map[string]interface{}) interface{} {
	if choice == nil {
		return nil
	}
	if s, ok := choice.(string); ok {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "auto" || s == "none" || s == "required" {
			return s
		}
		return nil
	}
	m, ok := choice.(map[string]interface{})
	if !ok {
		return nil
	}
	typ, _ := m["type"].(string)
	if typ == "any" {
		return "required"
	}
	name, _ := m["name"].(string)
	if name == "" {
		if fn, ok := m["function"].(map[string]interface{}); ok {
			name, _ = fn["name"].(string)
		}
	}
	for _, tool := range tools {
		if tool["type"] == "function" && tool["name"] == name {
			return map[string]interface{}{"type": "function", "name": name}
		}
		if tool["type"] == typ && grokCLIHostedTools[typ] {
			return map[string]interface{}{"type": typ}
		}
	}
	return nil
}
