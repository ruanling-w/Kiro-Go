package proxy

import (
	"encoding/json"
	"kiro-go/config"
	"kiro-go/store"
	"net/http/httptest"
	"testing"
	"time"
)

func listedModel(t *testing.T, h *Handler, path, id string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	h.handleModels(w, httptest.NewRequest("GET", path, nil))
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, model := range body.Data {
		if model["id"] == id {
			return model
		}
	}
	return nil
}

func advertisementHandler(combos ...store.Combo) *Handler {
	h := &Handler{combosByID: map[string]store.Combo{}, combosByName: map[string]store.Combo{}, combosLoaded: true}
	h.cachedModels = []ModelInfo{{ModelId: "Physical", InputTypes: []string{"text"}}, {ModelId: "judge", InputTypes: []string{"text"}}}
	h.modelsCacheTime = time.Now().Unix()
	for _, c := range combos {
		h.publishCombo(c)
	}
	return h
}

func TestComboAdvertisementGateAndAliases(t *testing.T) {
	if err := config.Init(t.TempDir() + "/config.json"); err != nil {
		t.Fatal(err)
	}
	h := advertisementHandler(store.Combo{ID: "c", Name: "MyCombo", Strategy: "fallback", StickyLimit: 1, Models: []store.ComboModel{{Model: "physical"}}})
	if got := listedModel(t, h, "/v1/models", "MyCombo"); got != nil {
		t.Fatal("combo advertised with gate off")
	}
	config.Get().ComboModelAdvertisement = true
	for _, path := range []string{"/v1/models", "/models"} {
		got := listedModel(t, h, path, "MyCombo")
		if got == nil || got["owned_by"] != "kiro" || got["strategy"] != "fallback" {
			t.Fatalf("%s combo=%v", path, got)
		}
	}
}

func TestComboAdvertisementFiltersCollisionsAndUnroutable(t *testing.T) {
	if err := config.Init(t.TempDir() + "/config.json"); err != nil {
		t.Fatal(err)
	}
	config.Get().ComboModelAdvertisement = true
	h := advertisementHandler(
		store.Combo{ID: "collision", Name: "pHySiCaL", Strategy: "fallback", StickyLimit: 1, Models: []store.ComboModel{{Model: "Physical"}}},
		store.Combo{ID: "unknown", Name: "UnknownCandidate", Strategy: "fallback", StickyLimit: 1, Models: []store.ComboModel{{Model: "missing"}}},
		store.Combo{ID: "bad", Name: "InvalidStrategy", Strategy: "bogus", Models: []store.ComboModel{{Model: "Physical"}}},
		store.Combo{ID: "fusion", Name: "BadFusion", Strategy: "fusion", Models: []store.ComboModel{{Model: "Physical"}}, FusionQuorum: 1, FusionTimeout: 1000, JudgeModel: "missing"},
	)
	for _, id := range []string{"pHySiCaL", "UnknownCandidate", "InvalidStrategy", "BadFusion"} {
		if got := listedModel(t, h, "/v1/models", id); got != nil {
			t.Fatalf("advertised %s", id)
		}
	}
	if got := listedModel(t, h, "/v1/models", "Physical"); got == nil || got["owned_by"] != "anthropic" {
		t.Fatalf("physical model lost: %v", got)
	}
}

func TestComboAdvertisementPreservesCaseAndCaseUniqueness(t *testing.T) {
	if err := config.Init(t.TempDir() + "/config.json"); err != nil {
		t.Fatal(err)
	}
	config.Get().ComboModelAdvertisement = true
	h := advertisementHandler(store.Combo{ID: "one", Name: "CaseCombo", Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "Physical"}}})
	if got := listedModel(t, h, "/models", "CaseCombo"); got == nil {
		t.Fatal("case-preserved combo missing")
	}
	if got := listedModel(t, h, "/models", "casecombo"); got != nil {
		t.Fatal("lowercase duplicate advertised")
	}
}

func TestComboAdvertisementRegistryUnavailableKeepsPhysicalModels(t *testing.T) {
	if err := config.Init(t.TempDir() + "/config.json"); err != nil {
		t.Fatal(err)
	}
	config.Get().ComboModelAdvertisement = true
	h := advertisementHandler()
	h.combosLoaded = false
	h.runtimeStore = nil
	if got := listedModel(t, h, "/v1/models", "Physical"); got == nil {
		t.Fatal("registry failure removed physical list")
	}
}
