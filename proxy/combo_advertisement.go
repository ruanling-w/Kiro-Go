package proxy

import (
	"kiro-go/config"
	"kiro-go/store"
	"strings"
)

// appendAdvertisedCombos is deliberately read-only: listing never reserves
// round-robin rotation and a registry failure leaves physical models intact.
func appendAdvertisedCombos(h *Handler, models []map[string]interface{}) []map[string]interface{} {
	cfg := config.Get()
	if h == nil || cfg == nil || !cfg.ComboModelAdvertisement || !h.ensureCombosLoaded() {
		return models
	}
	physical := make(map[string]bool, len(models))
	for _, m := range models {
		if id, ok := m["id"].(string); ok {
			physical[strings.ToLower(id)] = true
		}
	}
	h.combosMu.RLock()
	combos := make([]store.Combo, 0, len(h.combosByName))
	for _, c := range h.combosByName {
		combos = append(combos, cloneCombo(c))
	}
	h.combosMu.RUnlock()
	for _, c := range combos {
		key := strings.ToLower(strings.TrimSpace(c.Name))
		if key == "" || physical[key] || !advertisedComboRoutable(h, c) {
			continue
		}
		info := buildModelInfo(c.Name, "kiro", false)
		info["combo"] = true
		info["strategy"] = c.Strategy
		info["capabilities"] = map[string]bool{"combo": true}
		info["info"] = map[string]interface{}{"meta": map[string]interface{}{"combo": true, "strategy": c.Strategy}}
		models = append(models, info)
		physical[key] = true
	}
	return models
}

func advertisedComboRoutable(h *Handler, c store.Combo) bool {
	name := strings.TrimSpace(c.Name)
	if name == "" || len(name) > 128 || !comboNameRE.MatchString(name) || c.StickyLimit < 1 || c.StickyLimit > 10000 || (c.Strategy != "fallback" && c.Strategy != "round-robin" && c.Strategy != "fusion") || len(c.Models) < 1 || len(c.Models) > 8 {
		return false
	}
	known := h.knownComboModels()
	seen := make(map[string]bool, len(c.Models))
	for _, candidate := range c.Models {
		model := strings.TrimSpace(candidate.Model)
		key := strings.ToLower(model)
		if key == "" || len(model) > comboModelIDMaxBytes || seen[key] || !known[key] {
			return false
		}
		seen[key] = true
	}
	if c.Strategy == "fusion" {
		judge := strings.ToLower(strings.TrimSpace(c.JudgeModel))
		if c.FusionQuorum < 1 || c.FusionQuorum > len(c.Models) || c.FusionTimeout < 100 || c.FusionTimeout > 300000 || judge == "" || !known[judge] {
			return false
		}
	}
	return true
}
