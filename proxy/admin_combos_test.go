package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kiro-go/config"
	"kiro-go/store"
)

func comboValidationHandler() *Handler {
	return &Handler{combosByID: map[string]store.Combo{}, combosByName: map[string]store.Combo{}, combosLoaded: true}
}

func TestValidateComboRejectsReservedUnknownEmptyAndNestedJudge(t *testing.T) {
	h := comboValidationHandler()
	h.combosByName["primary"] = store.Combo{ID: "c1", Name: "Primary"}
	base := comboRequest{Name: "safe", Strategy: "fallback", StickyLimit: 1, Models: []string{"gpt-4"}}
	cases := []struct {
		name  string
		req   comboRequest
		field string
	}{
		{"reserved", func() comboRequest { x := base; x.Name = "gpt-4o"; return x }(), "name"},
		{"unknown", func() comboRequest { x := base; x.Models = []string{"made-up"}; return x }(), "models"},
		{"empty", func() comboRequest { x := base; x.Models = []string{"   "}; return x }(), "models"},
		{"trimmed nested judge", comboRequest{Name: "safe", Strategy: "fusion", StickyLimit: 1, Models: []string{"gpt-4"}, FusionQuorum: 1, FusionTimeout: 100, JudgeModel: " Primary "}, "judgeModel"},
		{"unknown judge", comboRequest{Name: "safe", Strategy: "fusion", StickyLimit: 1, Models: []string{"gpt-4"}, FusionQuorum: 1, FusionTimeout: 100, JudgeModel: "unknown"}, "judgeModel"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			for _, e := range h.validateCombo(tc.req, "") {
				if e.Field == tc.field {
					found = true
				}
			}
			if !found {
				t.Fatalf("errors=%+v", h.validateCombo(tc.req, ""))
			}
		})
	}
}

func TestValidateComboRejectsDirectNameCollisionAndOversizedModelIDs(t *testing.T) {
	h := comboValidationHandler()
	base := comboRequest{Name: "gpt-4-turbo", Strategy: "fallback", StickyLimit: 1, Models: []string{"gpt-4"}}
	if errors := h.validateCombo(base, ""); !hasComboFieldError(errors, "name") {
		t.Fatalf("direct collision errors=%+v", errors)
	}
	base.Name = "safe"
	base.Models = []string{strings.Repeat("x", comboModelIDMaxBytes+1)}
	if errors := h.validateCombo(base, ""); !hasComboFieldError(errors, "models") {
		t.Fatalf("oversized model errors=%+v", errors)
	}
	base.Strategy = "fusion"
	base.Models = []string{"gpt-4"}
	base.FusionQuorum = 1
	base.FusionTimeout = 100
	base.JudgeModel = strings.Repeat("x", comboModelIDMaxBytes+1)
	if errors := h.validateCombo(base, ""); !hasComboFieldError(errors, "judgeModel") {
		t.Fatalf("oversized judge errors=%+v", errors)
	}
}

func hasComboFieldError(errors []comboFieldError, field string) bool {
	for _, err := range errors {
		if err.Field == field {
			return true
		}
	}
	return false
}

func TestComboFromRequestNormalizesNonFusionFields(t *testing.T) {
	got := comboFromRequest(comboRequest{Name: "safe", Strategy: "fallback", StickyLimit: 1, FusionQuorum: 2, FusionTimeout: 1234, JudgeModel: "gpt-4", Models: []string{"gpt-4"}}, "c1")
	if got.FusionQuorum != 0 || got.FusionTimeout != 0 || got.JudgeModel != "" {
		t.Fatalf("non-fusion fields not normalized: %+v", got)
	}
}

func TestDecodeComboRejectsOversizedBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat(" ", comboAdminBodyLimit+1)))
	w := httptest.NewRecorder()
	if _, err := decodeCombo(w, r); err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestResetComboRequiresRevisionBody(t *testing.T) {
	h := comboValidationHandler()
	for _, body := range []string{"", `{}`, `{"revision":0}`, `{"revision":1,"extra":true}`, `{"revision":1} {"revision":2}`} {
		w := httptest.NewRecorder()
		h.apiResetCombo(w, httptest.NewRequest("POST", "/", strings.NewReader(body)), "c1")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body=%q status=%d response=%s", body, w.Code, w.Body.String())
		}
	}
}
func TestDecodeComboRejectsTrailingJSON(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"x"} {"name":"y"}`))
	w := httptest.NewRecorder()
	if _, err := decodeCombo(w, r); err == nil {
		t.Fatal("expected trailing JSON error")
	}
}

func TestDeleteComboRejectsTrailingJSON(t *testing.T) {
	h := comboValidationHandler()
	r := httptest.NewRequest("DELETE", "/", strings.NewReader(`{"revision":1} {"revision":2}`))
	w := httptest.NewRecorder()
	h.apiDeleteCombo(w, r, "c1")
	if w.Code != 400 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestComboRouteRequiresExactSegments(t *testing.T) {
	if comboRoute("/combos/a/b", "GET", "GET", false) {
		t.Fatal("nested ID route matched")
	}
	if comboRoute("/combos/a/b/reset-rotation", "POST", "POST", true) {
		t.Fatal("nested reset route matched")
	}
	if !comboRoute("/combos/a/reset-rotation", "POST", "POST", true) {
		t.Fatal("valid reset route did not match")
	}
}

func TestReplaceComboPublishesSingleRegistryTransition(t *testing.T) {
	h := comboValidationHandler()
	h.publishCombo(store.Combo{ID: "c1", Name: "Old"})
	h.replaceCombo(store.Combo{ID: "c1", Name: "New"})
	if _, ok := h.combosByName["old"]; ok {
		t.Fatal("old name retained")
	}
	if got := h.combosByName["new"]; got.ID != "c1" {
		t.Fatalf("new combo=%+v", got)
	}
}

func TestComboCRUDDoesNotAdvertiseUnroutableModelIDs(t *testing.T) {
	if err := config.Init(t.TempDir() + "/config.json"); err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.cachedModels = []ModelInfo{{ModelId: "routable-model"}}
	h.modelsCacheTime = time.Now().Unix()
	assertNotAdvertised := func(names ...string) {
		t.Helper()
		w := httptest.NewRecorder()
		h.handleModels(w, httptest.NewRequest("GET", "/v1/models", nil))
		var response struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode models response: %v", err)
		}
		for _, model := range response.Data {
			for _, name := range names {
				if model["id"] == name {
					t.Fatalf("unroutable combo %q advertised in /v1/models", name)
				}
			}
		}
	}

	h.publishCombo(store.Combo{ID: "c1", Name: "created-combo"})
	assertNotAdvertised("created-combo")
	h.replaceCombo(store.Combo{ID: "c1", Name: "updated-combo"})
	assertNotAdvertised("created-combo", "updated-combo")
	h.removeCombo("c1")
	assertNotAdvertised("created-combo", "updated-combo")
}

func TestValidateComboIgnoresEmptyAndStaleCachedModels(t *testing.T) {
	h := comboValidationHandler()
	req := comboRequest{Name: "safe", Strategy: "fallback", StickyLimit: 1, Models: []string{"retired-model"}}

	h.cachedModels = nil
	h.modelsCacheTime = time.Now().Unix()
	if errors := h.validateCombo(req, ""); len(errors) == 0 {
		t.Fatal("empty model cache accepted unknown model")
	}

	h.cachedModels = []ModelInfo{{ModelId: "retired-model"}}
	h.modelsCacheTime = time.Now().Add(-31 * time.Minute).Unix()
	if errors := h.validateCombo(req, ""); len(errors) == 0 {
		t.Fatal("stale model cache accepted retired model")
	}

	h.modelsCacheTime = time.Now().Unix()
	if errors := h.validateCombo(req, ""); len(errors) != 0 {
		t.Fatalf("fresh model cache rejected known model: %+v", errors)
	}
}

func TestAPIGetComboHydratesRegistryBeforeLookup(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/runtime.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	created, err := st.CreateCombo(store.Combo{ID: "persisted", Name: "persisted-combo", Strategy: "fallback", StickyLimit: 1, Models: []store.ComboModel{{Model: "gpt-4"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.combosLoaded = false

	w := httptest.NewRecorder()
	h.apiGetCombo(w, httptest.NewRequest("GET", "/admin/api/combos/persisted", nil), "persisted")
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got store.Combo
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID || got.Name != created.Name || len(got.Models) != 1 {
		t.Fatalf("hydrated combo=%+v, want %+v", got, created)
	}
}
