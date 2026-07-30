package proxy

import (
	"encoding/json"
	"errors"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func TestResolveComboRouteFailedRequestConsumesReservation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	combo, err := st.CreateCombo(store.Combo{ID: "c1", Name: "rr", Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.publishCombo(combo)
	failedRequest, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	if failedRequest.Candidates[0].Model != "a" {
		t.Fatalf("failed request primary=%s", failedRequest.Candidates[0].Model)
	}
	// No provider call is needed: reservation is committed synchronously before
	// network execution, so abandoning this request simulates failure/cancellation.
	nextRequest, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	if nextRequest.Candidates[0].Model != "b" {
		t.Fatalf("failed request did not consume reservation: %+v", nextRequest.Candidates)
	}
}

func TestOpenAIStreamingComboRejectedWithoutReservation(t *testing.T) {
	if err := config.Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	combo, err := st.CreateCombo(store.Combo{ID: "c1", Name: "rr", Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.publishCombo(combo)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"rr","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	response := httptest.NewRecorder()
	h.handleOpenAIChat(response, request)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error.Type != "invalid_request_error" {
		t.Fatalf("body=%s err=%v", response.Body.String(), err)
	}
	first, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidates[0].Model != "a" {
		t.Fatalf("stream rejection consumed rotation: %+v", first.Candidates)
	}
}

func TestResolveComboRouteRegistryUnavailable(t *testing.T) {
	h := &Handler{combosByID: map[string]store.Combo{}, combosByName: map[string]store.Combo{}}
	if _, err := h.resolveComboRoute("unknown"); !errors.Is(err, errComboRegistryUnavailable) {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveComboRouteDirectModelWinsCollision(t *testing.T) {
	h := comboValidationHandler()
	h.publishCombo(store.Combo{ID: "collision", Name: "gpt-4", Revision: 1, Strategy: "fallback", Models: []store.ComboModel{{Model: "a"}}})
	route, err := h.resolveComboRoute("gpt-4")
	if err != nil || route != nil {
		t.Fatalf("direct model shadowed: route=%+v err=%v", route, err)
	}
}

func TestResolveComboRouteUnknownHealthyRegistryReturnsDirect(t *testing.T) {
	h := comboValidationHandler()
	route, err := h.resolveComboRoute("future-direct-model")
	if err != nil || route != nil {
		t.Fatalf("unknown direct route=%+v err=%v", route, err)
	}
}

func TestLookupComboSnapshotDoesNotReserveRoundRobin(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	combo, err := st.CreateCombo(store.Combo{ID: "c1", Name: "rr", Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.publishCombo(combo)
	if snapshot, ok := h.lookupComboSnapshot("RR"); !ok || snapshot.ID != combo.ID {
		t.Fatalf("snapshot=%+v ok=%v", snapshot, ok)
	}
	first, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidates[0].Model != "a" {
		t.Fatalf("read-only lookup consumed rotation: %+v", first.Candidates)
	}
}

func TestOpenAIStreamComboRejectionDoesNotReserveRotation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	combo, err := st.CreateCombo(store.Combo{ID: "c1", Name: "rr", Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	h := comboValidationHandler()
	h.runtimeStore = st
	h.publishCombo(combo)

	assertStreamingComboRejected(t, h)

	first, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.resolveComboRoute("rr")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || len(first.Candidates) == 0 || first.Candidates[0].Model != "a" {
		t.Fatalf("first reservation=%+v", first)
	}
	if second == nil || len(second.Candidates) == 0 || second.Candidates[0].Model != "b" {
		t.Fatalf("second reservation=%+v", second)
	}
}

func TestOpenAIStreamComboRejectionDoesNotRequireRuntimeStore(t *testing.T) {
	h := comboValidationHandler()
	h.publishCombo(store.Combo{ID: "c1", Name: "rr", Revision: 1, Strategy: "round-robin", StickyLimit: 1, Models: []store.ComboModel{{Model: "a"}, {Model: "b"}}})
	assertStreamingComboRejected(t, h)
}

func assertStreamingComboRejected(t *testing.T, h *Handler) {
	t.Helper()
	if err := config.Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"rr","stream":true,"messages":[{"role":"user","content":"hello"}]}`))
	rec := httptest.NewRecorder()
	h.handleOpenAIChat(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Type != "invalid_request_error" || response.Error.Message != "Streaming Combo routing is not enabled yet for rr" {
		t.Fatalf("error=%+v", response.Error)
	}
}

func TestResolveComboRouteDirectModelReturnsNil(t *testing.T) {
	h := comboValidationHandler()
	route, err := h.resolveComboRoute("gpt-4")
	if err != nil || route != nil {
		t.Fatalf("route=%+v err=%v", route, err)
	}
}
