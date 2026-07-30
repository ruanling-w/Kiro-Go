package proxy

import (
	"errors"
	"kiro-go/store"
	"strings"
)

var errComboRegistryUnavailable = errors.New("combo registry unavailable")

type comboRouteSnapshot struct {
	Combo          store.Combo
	RequestedModel string
	Candidates     []store.ComboModel
}

func cloneCombo(c store.Combo) store.Combo {
	copy := c
	copy.Models = append([]store.ComboModel(nil), c.Models...)
	return copy
}

func (h *Handler) lookupComboSnapshot(requestedModel string) (*store.Combo, bool) {
	if h == nil {
		return nil, false
	}
	name := strings.TrimSpace(requestedModel)
	if name == "" || h.knownComboModels()[strings.ToLower(name)] || !h.ensureCombosLoaded() {
		return nil, false
	}
	h.combosMu.RLock()
	combo, ok := h.combosByName[strings.ToLower(name)]
	h.combosMu.RUnlock()
	if !ok {
		return nil, false
	}
	copy := cloneCombo(combo)
	return &copy, true
}

// resolveComboRoute resolves a configured Combo without changing direct-model
// behavior. Callers retain requestedModel for public response identity and use
// Candidates only for upstream execution.
func (h *Handler) resolveComboRoute(requestedModel string) (*comboRouteSnapshot, error) {
	name := strings.TrimSpace(requestedModel)
	if name == "" {
		return nil, nil
	}
	// A direct model always wins over a colliding Combo identity.
	if h.knownComboModels()[strings.ToLower(name)] {
		return nil, nil
	}
	if !h.ensureCombosLoaded() {
		return nil, errComboRegistryUnavailable
	}
	h.combosMu.RLock()
	combo, ok := h.combosByName[strings.ToLower(name)]
	h.combosMu.RUnlock()
	if !ok {
		return nil, nil
	}
	combo = cloneCombo(combo)
	candidates := append([]store.ComboModel(nil), combo.Models...)
	if combo.Strategy == "round-robin" && len(candidates) > 1 {
		st, unlock := h.runtimeStoreForOperation()
		if st == nil {
			return nil, store.ErrStorageUnavailable
		}
		reserved, err := st.ReserveComboRotation(combo.ID, combo.Revision)
		unlock()
		if err != nil {
			return nil, err
		}
		candidates = append([]store.ComboModel(nil), reserved...)
	}
	return &comboRouteSnapshot{Combo: combo, RequestedModel: requestedModel, Candidates: candidates}, nil
}

func (h *Handler) applyComboRequirements(route *comboRouteSnapshot, required comboCapabilities) *comboRouteSnapshot {
	if route == nil {
		return nil
	}
	copy := *route
	copy.Combo = cloneCombo(route.Combo)
	copy.Candidates = h.reorderComboCandidates(route.Candidates, required)
	return &copy
}
