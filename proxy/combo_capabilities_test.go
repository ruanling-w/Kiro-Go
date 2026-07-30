package proxy

import (
	"encoding/json"
	"reflect"
	"testing"

	"kiro-go/store"
)

func TestComboProtocolMediaRequirementsTrailingUserOnly(t *testing.T) {
	image := map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "data:image/png;base64,AA=="}}
	pdf := map[string]interface{}{"type": "document", "source": map[string]interface{}{"type": "base64", "media_type": "application/pdf", "data": "AA=="}}
	audio := map[string]interface{}{"type": "input_audio", "input_audio": map[string]interface{}{"data": "AA==", "format": "wav"}}
	video := map[string]interface{}{"type": "input_video", "video_url": "data:video/mp4;base64,AA=="}

	tests := []struct {
		name string
		got  comboCapabilities
		want comboCapabilities
	}{
		{"claude image current", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{image}}}}), comboCapabilityVision},
		{"claude pdf current", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{pdf}}}}), comboCapabilityPDF},
		{"claude audio current", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{audio}}}}), comboCapabilityAudio},
		{"claude video current", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{video}}}}), comboCapabilityVideo},
		{"claude history ignored", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{image}}, {Role: "assistant", Content: "seen"}, {Role: "user", Content: "now text"}}}), 0},
		{"claude trailing tool result ignores media history", claudeComboRequirements(&ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: []interface{}{pdf}}, {Role: "assistant", Content: "tool"}, {Role: "tool", Content: "result"}}}), 0},
		{"chat image current", openAIComboRequirements(&OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{image}}}}), comboCapabilityVision},
		{"chat pdf file", openAIComboRequirements(&OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{map[string]interface{}{"type": "input_file", "file_id": "file_123"}}}}}), comboCapabilityPDF},
		{"chat audio current", openAIComboRequirements(&OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{audio}}}}), comboCapabilityAudio},
		{"chat video current", openAIComboRequirements(&OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{video}}}}), comboCapabilityVideo},
		{"chat history ignored", openAIComboRequirements(&OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{video}}, {Role: "assistant", Content: "seen"}, {Role: "user", Content: "now text"}}}), 0},
		{"responses image", responsesComboRequirements(json.RawMessage(`[{"role":"user","content":[{"type":"input_image","image_url":"https://example.test/x.png"}]}]`)), comboCapabilityVision},
		{"responses pdf", responsesComboRequirements(json.RawMessage(`[{"role":"user","content":[{"type":"input_file","file_id":"file_pdf"}]}]`)), comboCapabilityPDF},
		{"responses audio", responsesComboRequirements(json.RawMessage(`[{"role":"user","content":[{"type":"input_audio","audio_url":"data:audio/wav;base64,AA=="}]}]`)), comboCapabilityAudio},
		{"responses video", responsesComboRequirements(json.RawMessage(`[{"role":"user","content":[{"type":"input_video","video_url":"https://example.test/x.mp4"}]}]`)), comboCapabilityVideo},
		{"responses history ignored", responsesComboRequirements(json.RawMessage(`[{"role":"user","content":[{"type":"input_image","image_url":"https://example.test/x.png"}]},{"role":"assistant","content":"seen"},{"role":"user","content":"now text"}]`)), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("requirements=%04b want=%04b", tt.got, tt.want)
			}
		})
	}
}

func TestReorderComboCandidatesStableNeverDropsUnknown(t *testing.T) {
	h := comboValidationHandler()
	h.cachedModels = []ModelInfo{
		{ModelId: "text-1", InputTypes: []string{"text"}},
		{ModelId: "vision-1", InputTypes: []string{"text", "image"}},
		{ModelId: "vision-2", InputTypes: []string{"vision"}},
		{ModelId: "all", InputTypes: []string{"image", "document", "audio", "video"}},
	}
	input := []store.ComboModel{{Model: "text-1"}, {Model: "vision-1"}, {Model: "unknown"}, {Model: "vision-2"}, {Model: "all"}}
	got := h.reorderComboCandidates(input, comboCapabilityVision)
	want := []store.ComboModel{{Model: "vision-1"}, {Model: "vision-2"}, {Model: "all"}, {Model: "text-1"}, {Model: "unknown"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
	if !reflect.DeepEqual(input, []store.ComboModel{{Model: "text-1"}, {Model: "vision-1"}, {Model: "unknown"}, {Model: "vision-2"}, {Model: "all"}}) {
		t.Fatalf("input mutated: %+v", input)
	}

	got = h.reorderComboCandidates(input, comboCapabilityPDF|comboCapabilityAudio|comboCapabilityVideo)
	want = []store.ComboModel{{Model: "all"}, {Model: "text-1"}, {Model: "vision-1"}, {Model: "unknown"}, {Model: "vision-2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("multi requirement got=%+v want=%+v", got, want)
	}
}

func TestApplyComboRequirementsPreservesIdentity(t *testing.T) {
	h := comboValidationHandler()
	h.cachedModels = []ModelInfo{{ModelId: "vision", InputTypes: []string{"image"}}}
	route := &comboRouteSnapshot{
		Combo:          store.Combo{ID: "c1", Name: "MyCombo", Revision: 7, Models: []store.ComboModel{{Model: "text"}, {Model: "vision"}}},
		RequestedModel: "mYcOmBo",
		Candidates:     []store.ComboModel{{Model: "text"}, {Model: "vision"}},
	}
	got := h.applyComboRequirements(route, comboCapabilityVision)
	if got.RequestedModel != "mYcOmBo" || got.Combo.Name != "MyCombo" || got.Combo.Revision != 7 {
		t.Fatalf("identity changed: %+v", got)
	}
	if got.Candidates[0].Model != "vision" || route.Candidates[0].Model != "text" {
		t.Fatalf("order=%+v original=%+v", got.Candidates, route.Candidates)
	}
}
