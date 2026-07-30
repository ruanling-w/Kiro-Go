package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"kiro-go/config"
	"kiro-go/store"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var comboNameRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var builtInComboNames = map[string]bool{"auto": true, "gpt-4": true, "gpt-4o": true}

type comboRequest struct {
	Name          string   `json:"name"`
	Strategy      string   `json:"strategy"`
	StickyLimit   int      `json:"stickyLimit"`
	Revision      int64    `json:"revision"`
	Models        []string `json:"models"`
	FusionQuorum  int      `json:"fusionQuorum,omitempty"`
	FusionTimeout int      `json:"fusionTimeoutMs,omitempty"`
	JudgeModel    string   `json:"judgeModel,omitempty"`
}

type comboFieldError struct{ Field, Message string }

func comboError(w http.ResponseWriter, status int, message string, fields []comboFieldError) {
	w.WriteHeader(status)
	body := map[string]any{"error": message}
	if len(fields) > 0 {
		m := map[string]string{}
		for _, f := range fields {
			m[f.Field] = f.Message
		}
		body["fields"] = m
	}
	_ = json.NewEncoder(w).Encode(body)
}
func (h *Handler) knownComboModels() map[string]bool {
	known := map[string]bool{
		"auto": true, "gpt-4": true, "gpt-4o": true, "gpt-4-turbo": true, "gpt-3.5-turbo": true,
		"claude-3-5-sonnet": true, "claude-3-opus": true, "claude-3-sonnet": true,
	}
	thinkingSuffix := "-thinking"
	for _, model := range fallbackAnthropicModels(thinkingSuffix) {
		if id, ok := model["id"].(string); ok {
			known[strings.ToLower(id)] = true
		}
	}
	h.modelsCacheMu.RLock()
	cacheFresh := len(h.cachedModels) > 0 && h.modelsCacheTime > 0 && time.Since(time.Unix(h.modelsCacheTime, 0)) <= 30*time.Minute
	if cacheFresh {
		for _, m := range h.cachedModels {
			known[strings.ToLower(m.ModelId)] = true
		}
	}
	h.modelsCacheMu.RUnlock()
	if h.pool != nil {
		for _, account := range config.GetAccounts() {
			for _, model := range h.pool.GetModelList(account.ID) {
				known[strings.ToLower(strings.TrimSpace(model))] = true
			}
		}
	}
	return known
}

func (h *Handler) validateCombo(req comboRequest, id string) []comboFieldError {
	if !h.ensureCombosLoaded() {
		return []comboFieldError{{"models", "combo registry is unavailable"}}
	}
	var out []comboFieldError
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 128 || !comboNameRE.MatchString(name) {
		out = append(out, comboFieldError{"name", "must match ^[A-Za-z0-9_.-]+$ and be at most 128 characters"})
	}
	if req.Strategy != "fallback" && req.Strategy != "round-robin" && req.Strategy != "fusion" {
		out = append(out, comboFieldError{"strategy", "must be fallback, round-robin, or fusion"})
	}
	if req.StickyLimit < 1 || req.StickyLimit > 10000 {
		out = append(out, comboFieldError{"stickyLimit", "must be between 1 and 10000"})
	}
	if len(req.Models) < 1 || len(req.Models) > 8 {
		out = append(out, comboFieldError{"models", "must contain 1 to 8 models"})
	}
	h.combosMu.RLock()
	comboNames := make(map[string]bool, len(h.combosByName))
	for n, combo := range h.combosByName {
		if combo.ID != id {
			comboNames[n] = true
		}
	}
	h.combosMu.RUnlock()
	nameKey := strings.ToLower(name)
	if builtInComboNames[nameKey] {
		out = append(out, comboFieldError{"name", "is reserved"})
	} else if comboNames[nameKey] {
		out = append(out, comboFieldError{"name", "already exists"})
	}
	known := h.knownComboModels()
	if known[nameKey] {
		out = append(out, comboFieldError{"name", "conflicts with a direct model"})
	}
	seen := map[string]bool{}
	for i, m := range req.Models {
		key := strings.ToLower(strings.TrimSpace(m))
		if key == "" {
			out = append(out, comboFieldError{"models", "must not contain empty model names"})
			continue
		}
		if len(strings.TrimSpace(m)) > comboModelIDMaxBytes {
			out = append(out, comboFieldError{"models", "model IDs must be at most 128 bytes"})
			continue
		}
		if seen[key] {
			out = append(out, comboFieldError{"models", "must not contain duplicate models"})
			break
		}
		seen[key] = true
		if comboNames[key] {
			out = append(out, comboFieldError{"models", "nested combos are not allowed"})
		} else if !known[key] {
			out = append(out, comboFieldError{"models", "contains an unknown model at index " + strconv.Itoa(i)})
		}
	}
	if req.Strategy == "fusion" {
		if req.FusionQuorum < 1 || req.FusionQuorum > len(req.Models) {
			out = append(out, comboFieldError{"fusionQuorum", "must be between 1 and the model count"})
		}
		if req.FusionTimeout < 100 || req.FusionTimeout > 300000 {
			out = append(out, comboFieldError{"fusionTimeoutMs", "must be between 100 and 300000"})
		}
		judgeKey := strings.ToLower(strings.TrimSpace(req.JudgeModel))
		if judgeKey == "" || len(strings.TrimSpace(req.JudgeModel)) > comboModelIDMaxBytes || comboNames[judgeKey] || !known[judgeKey] {
			out = append(out, comboFieldError{"judgeModel", "must be a known non-Combo model"})
		}
	}
	return out
}

const (
	comboAdminBodyLimit  = 8 << 20
	comboModelIDMaxBytes = 128
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, comboAdminBodyLimit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func decodeCombo(w http.ResponseWriter, r *http.Request) (comboRequest, error) {
	var x comboRequest
	return x, decodeJSONBody(w, r, &x)
}
func comboFromRequest(x comboRequest, id string) store.Combo {
	c := store.Combo{ID: id, Name: strings.TrimSpace(x.Name), Strategy: x.Strategy, StickyLimit: x.StickyLimit, Revision: x.Revision}
	if x.Strategy == "fusion" {
		c.FusionQuorum = x.FusionQuorum
		c.FusionTimeout = x.FusionTimeout
		c.JudgeModel = strings.TrimSpace(x.JudgeModel)
	}
	for i, m := range x.Models {
		c.Models = append(c.Models, store.ComboModel{Model: strings.TrimSpace(m), Position: i})
	}
	return c
}
func comboStatus(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrStorageUnavailable):
		comboError(w, 503, "storage unavailable", nil)
	case errors.Is(err, store.ErrComboNotFound):
		comboError(w, 404, "combo not found", nil)
	case errors.Is(err, store.ErrComboNameConflict):
		comboError(w, 409, "combo name already exists", nil)
	case errors.Is(err, store.ErrComboConflict):
		comboError(w, 409, "revision conflict", nil)
	default:
		comboError(w, 503, "storage operation failed", nil)
	}
}
func (h *Handler) apiListCombos(w http.ResponseWriter, r *http.Request) {
	if combos, ok := h.comboSnapshot(); ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": combos})
		return
	}
	comboError(w, http.StatusServiceUnavailable, "storage unavailable", nil)
}
func (h *Handler) apiGetCombo(w http.ResponseWriter, r *http.Request, id string) {
	if !h.ensureCombosLoaded() {
		comboError(w, http.StatusServiceUnavailable, "storage unavailable", nil)
		return
	}
	h.combosMu.RLock()
	c, ok := h.combosByID[id]
	h.combosMu.RUnlock()
	if !ok {
		comboError(w, 404, "combo not found", nil)
		return
	}
	_ = json.NewEncoder(w).Encode(c)
}
func (h *Handler) apiCreateCombo(w http.ResponseWriter, r *http.Request) {
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		comboError(w, 503, "storage unavailable", nil)
		return
	}
	defer unlock()
	x, err := decodeCombo(w, r)
	if err != nil {
		comboError(w, 400, "invalid JSON", nil)
		return
	}
	if fs := h.validateCombo(x, ""); len(fs) > 0 {
		comboError(w, 422, "validation failed", fs)
		return
	}
	c, err := st.CreateCombo(comboFromRequest(x, uuid.NewString()))
	if err != nil {
		comboStatus(w, err)
		return
	}
	h.publishCombo(c)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}
func (h *Handler) apiUpdateCombo(w http.ResponseWriter, r *http.Request, id string) {
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		comboError(w, 503, "storage unavailable", nil)
		return
	}
	defer unlock()
	x, err := decodeCombo(w, r)
	if err != nil {
		comboError(w, 400, "invalid JSON", nil)
		return
	}
	if x.Revision < 1 {
		comboError(w, 400, "revision is required", []comboFieldError{{"revision", "must be positive"}})
		return
	}
	if fs := h.validateCombo(x, id); len(fs) > 0 {
		comboError(w, 422, "validation failed", fs)
		return
	}
	c, err := st.UpdateCombo(comboFromRequest(x, id))
	if err != nil {
		comboStatus(w, err)
		return
	}
	h.replaceCombo(c)
	_ = json.NewEncoder(w).Encode(c)
}
func (h *Handler) apiDeleteCombo(w http.ResponseWriter, r *http.Request, id string) {
	var x struct {
		Revision int64 `json:"revision"`
	}
	if err := decodeJSONBody(w, r, &x); err != nil || x.Revision < 1 {
		comboError(w, 400, "revision is required", nil)
		return
	}
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		comboError(w, 503, "storage unavailable", nil)
		return
	}
	defer unlock()
	if err := st.DeleteCombo(id, x.Revision); err != nil {
		comboStatus(w, err)
		return
	}
	h.removeCombo(id)
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) apiResetCombo(w http.ResponseWriter, r *http.Request, id string) {
	var x struct {
		Revision int64 `json:"revision"`
	}
	if err := decodeJSONBody(w, r, &x); err != nil || x.Revision < 1 {
		comboError(w, 400, "revision is required", nil)
		return
	}
	st, unlock := h.runtimeStoreForOperation()
	if st == nil {
		comboError(w, 503, "storage unavailable", nil)
		return
	}
	defer unlock()
	if err := st.ResetComboRotation(id, x.Revision); err != nil {
		comboStatus(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
