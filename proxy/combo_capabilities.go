package proxy

import (
	"encoding/json"
	"strings"

	"kiro-go/store"
)

type comboCapabilities uint8

const (
	comboCapabilityVision comboCapabilities = 1 << iota
	comboCapabilityPDF
	comboCapabilityAudio
	comboCapabilityVideo
)

func (h *Handler) reorderComboCandidates(candidates []store.ComboModel, required comboCapabilities) []store.ComboModel {
	ordered := append([]store.ComboModel(nil), candidates...)
	if required == 0 || len(ordered) < 2 {
		return ordered
	}
	capabilities := h.comboModelCapabilities()
	preferred := make([]store.ComboModel, 0, len(ordered))
	other := make([]store.ComboModel, 0, len(ordered))
	for _, candidate := range ordered {
		caps, known := capabilities[strings.ToLower(strings.TrimSpace(candidate.Model))]
		if known && caps&required == required {
			preferred = append(preferred, candidate)
		} else {
			other = append(other, candidate)
		}
	}
	return append(preferred, other...)
}

func (h *Handler) comboModelCapabilities() map[string]comboCapabilities {
	out := make(map[string]comboCapabilities)
	if h == nil {
		return out
	}
	h.modelsCacheMu.RLock()
	models := append([]ModelInfo(nil), h.cachedModels...)
	h.modelsCacheMu.RUnlock()
	for _, model := range models {
		var caps comboCapabilities
		for _, inputType := range model.InputTypes {
			switch strings.ToLower(strings.TrimSpace(inputType)) {
			case "image", "vision":
				caps |= comboCapabilityVision
			case "pdf", "document":
				caps |= comboCapabilityPDF
			case "audio":
				caps |= comboCapabilityAudio
			case "video":
				caps |= comboCapabilityVideo
			}
		}
		out[strings.ToLower(strings.TrimSpace(model.ModelId))] |= caps
	}
	return out
}

func claudeComboRequirements(req *ClaudeRequest) comboCapabilities {
	if req == nil {
		return 0
	}
	if len(req.Messages) == 0 || !strings.EqualFold(strings.TrimSpace(req.Messages[len(req.Messages)-1].Role), "user") {
		return 0
	}
	return mediaRequirements(req.Messages[len(req.Messages)-1].Content)
}

func openAIComboRequirements(req *OpenAIRequest) comboCapabilities {
	if req == nil {
		return 0
	}
	if len(req.Messages) == 0 || !strings.EqualFold(strings.TrimSpace(req.Messages[len(req.Messages)-1].Role), "user") {
		return 0
	}
	return mediaRequirements(req.Messages[len(req.Messages)-1].Content)
}

func responsesComboRequirements(input json.RawMessage) comboCapabilities {
	var value interface{}
	if json.Unmarshal(input, &value) != nil {
		return 0
	}
	items, ok := value.([]interface{})
	if !ok {
		items = []interface{}{value}
	}
	if len(items) == 0 {
		return 0
	}
	item, ok := items[len(items)-1].(map[string]interface{})
	if !ok {
		return 0
	}
	role, _ := item["role"].(string)
	typ, _ := item["type"].(string)
	if strings.EqualFold(strings.TrimSpace(role), "user") || (role == "" && strings.HasPrefix(strings.ToLower(typ), "input_")) {
		return mediaRequirements(item)
	}
	return 0
}

func mediaRequirements(value interface{}) comboCapabilities {
	var required comboCapabilities
	walkMediaValue(value, &required)
	return required
}

func walkMediaValue(value interface{}, required *comboCapabilities) {
	switch v := value.(type) {
	case []ClaudeContentBlock:
		for i := range v {
			walkMediaValue(v[i], required)
		}
	case ClaudeContentBlock:
		inspectMediaType(v.Type, required)
		if v.Source != nil {
			inspectMediaType(v.Source.MediaType, required)
			inspectMediaReference(v.Source.Data, required)
		}
		walkMediaValue(v.Content, required)
	case []interface{}:
		for _, item := range v {
			walkMediaValue(item, required)
		}
	case map[string]interface{}:
		if typ, ok := v["type"].(string); ok {
			inspectMediaType(typ, required)
		}
		for _, key := range []string{"media_type", "mime_type", "format", "filename", "file_name", "url", "image_url", "audio_url", "video_url", "file_id", "source", "content"} {
			if item, ok := v[key]; ok {
				walkMediaValue(item, required)
			}
		}
	case map[string]string:
		for key, item := range v {
			if key == "type" || key == "media_type" || key == "mime_type" || key == "format" {
				inspectMediaType(item, required)
			} else {
				inspectMediaReference(item, required)
			}
		}
	case string:
		inspectMediaReference(v, required)
	case json.RawMessage:
		var decoded interface{}
		if json.Unmarshal(v, &decoded) == nil {
			walkMediaValue(decoded, required)
		}
	}
}

func inspectMediaType(value string, required *comboCapabilities) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "pdf") || value == "document" || value == "input_file" || value == "file":
		*required |= comboCapabilityPDF
	case strings.Contains(value, "audio"):
		*required |= comboCapabilityAudio
	case strings.Contains(value, "video"):
		*required |= comboCapabilityVideo
	case strings.Contains(value, "image") || strings.Contains(value, "vision"):
		*required |= comboCapabilityVision
	}
}

func inspectMediaReference(value string, required *comboCapabilities) {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(lower, "data:application/pdf") || strings.Contains(lower, ".pdf"):
		*required |= comboCapabilityPDF
	case strings.HasPrefix(lower, "data:audio/"):
		*required |= comboCapabilityAudio
	case strings.HasPrefix(lower, "data:video/"):
		*required |= comboCapabilityVideo
	case strings.HasPrefix(lower, "data:image/"):
		*required |= comboCapabilityVision
	}
}
