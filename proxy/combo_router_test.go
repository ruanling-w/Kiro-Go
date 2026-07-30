package proxy

import (
	"errors"
	"kiro-go/store"
	"path/filepath"
	"testing"
)

func TestResolveComboRouteFallbackSnapshotsModels(t *testing.T) {
	h := comboValidationHandler()
	h.publishCombo(store.Combo{ID: "c1", Name: "Primary", Revision: 1, Strategy: "fallback", Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	route, err := h.resolveComboRoute("pRiMaRy")
	if err != nil {
		t.Fatal(err)
	}
	if route == nil || route.RequestedModel != "pRiMaRy" || route.Candidates[0].Model != "a" {
		t.Fatalf("route=%+v", route)
	}
	h.combosMu.Lock()
	combo := h.combosByID["c1"]
	combo.Models[0].Model = "changed"
	h.combosByID["c1"] = combo
	h.combosMu.Unlock()
	if route.Candidates[0].Model != "a" || route.Combo.Models[0].Model != "a" {
		t.Fatalf("route mutated after publication: %+v", route)
	}
}

func TestResolveComboRouteRoundRobinReservesOnce(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	combo, err := st.CreateCombo(store.Combo{ID: "c1", Name: "rr", Revision: 1, Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.publishCombo(combo)
	first, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.resolveComboRoute("RR")
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidates[0].Model != "a" || second.Candidates[0].Model != "b" {
		t.Fatalf("first=%+v second=%+v", first.Candidates, second.Candidates)
	}
}

func TestResolveComboRouteRegistryUnavailable(t *testing.T) {
	h := &Handler{combosByID: map[string]store.Combo{}, combosByName: map[string]store.Combo{}}
	if _, err := h.resolveComboRoute("unknown"); !errors.Is(err, errComboRegistryUnavailable) {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveComboRouteDirectModelReturnsNil(t *testing.T) {
	h := comboValidationHandler()
	route, err := h.resolveComboRoute("gpt-4")
	if err != nil || route != nil {
		t.Fatalf("route=%+v err=%v", route, err)
	}
}
