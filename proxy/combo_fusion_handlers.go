package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"kiro-go/config"

	"github.com/google/uuid"
)

type comboAttemptLogContext struct {
	RequestID       string
	ComboID         string
	ComboRevision   int64
	ComboStrategy   string
	CandidateModel  string
	EffectiveModel  string
	AttemptIndex    int
	FusionRole      string
	BeforeFirstByte bool
	SelectedModel   string
}

func (h *Handler) executeFusionModel(ctx context.Context, req *OpenAIRequest, model, provider string, thinking bool, estimated int, apiKeyID string, meta comboAttemptLogContext) (openAIComboResult, error) {
	if model == "" {
		return openAIComboResult{}, fmt.Errorf("fusion judge model is required")
	}
	if combo, ok := h.lookupComboSnapshot(model); ok {
		return openAIComboResult{}, fmt.Errorf("fusion judge model %q resolves to Combo %q", model, combo.Name)
	}
	candidate := *req
	candidate.Model, candidate.Stream, candidate.Tools = model, false, nil
	payload := OpenAIToKiro(&candidate, thinking)
	excluded := map[string]bool{}
	var last error
	for attempt := 0; attempt < maxAccountRetryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return openAIComboResult{}, err
		}
		account := h.pool.GetNextForModelAndProviderExcluding(model, provider, excluded)
		if account == nil {
			break
		}
		if err := h.ensureValidToken(account); err != nil {
			last = err
			h.recordComboAttempt(meta, account.ID, providerLabel(account), 0, 0, 0, err)
			excluded[account.ID] = true
			h.handleAccountFailure(account, err)
			continue
		}
		result, err := h.executeOpenAIComboAttempt(ctx, account, payload, model, thinking, estimated)
		if err != nil {
			last = err
			h.recordComboAttempt(meta, account.ID, providerLabel(account), 0, 0, 0, err)
			excluded[account.ID] = true
			h.handleAccountFailure(account, err)
			continue
		}
		result.accountID, result.provider = account.ID, providerLabel(account)
		h.recordBillableAttempt(apiKeyID, result.inputTokens, result.outputTokens, result.credits)
		h.pool.RecordSuccess(account.ID)
		h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
		h.recordComboAttempt(meta, account.ID, result.provider, result.inputTokens, result.outputTokens, result.credits, nil)
		return result, nil
	}
	if last == nil {
		last = errors.New("no available accounts")
	}
	return openAIComboResult{}, last
}

func (h *Handler) executeOpenAIFusion(ctx context.Context, req *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimated int, apiKeyID string) (openAIComboResult, error) {
	requestID := uuid.NewString()
	base := comboAttemptLogContext{RequestID: requestID, ComboID: route.Combo.ID, ComboRevision: route.Combo.Revision, ComboStrategy: route.Combo.Strategy, BeforeFirstByte: true}
	results, err := runFusionPanels(ctx, len(route.Candidates), fusionRunConfig{Quorum: route.Combo.FusionQuorum, Timeout: time.Duration(route.Combo.FusionTimeout) * time.Millisecond, Grace: fusionGracePeriod, Run: func(c context.Context, i int) (openAIComboResult, error) {
		meta := base
		meta.CandidateModel, meta.EffectiveModel, meta.AttemptIndex, meta.FusionRole = route.Candidates[i].Model, route.Candidates[i].Model, i+1, "panel"
		return h.executeFusionModel(c, req, route.Candidates[i].Model, route.Candidates[i].Provider, false, estimated, apiKeyID, meta)
	}})
	if err != nil {
		return openAIComboResult{}, err
	}
	if len(results) == 1 {
		return results[0].Result, nil
	}
	judge := *req
	judge.Messages = []OpenAIMessage{{Role: "system", Content: "Synthesize a single accurate answer."}, {Role: "user", Content: fusionJudgePrompt(results)}}
	judge.Tools, judge.Stream = nil, false
	base.CandidateModel, base.EffectiveModel, base.AttemptIndex, base.FusionRole, base.SelectedModel = route.Combo.JudgeModel, route.Combo.JudgeModel, len(route.Candidates)+1, "judge", route.Combo.JudgeModel
	return h.executeFusionModel(ctx, &judge, route.Combo.JudgeModel, route.Combo.JudgeProvider, thinking, estimateOpenAIRequestInputTokens(&judge), apiKeyID, base)
}

func claudeFusionOpenAIRequest(req *ClaudeRequest) *OpenAIRequest {
	payload := ClaudeToKiro(req, false)
	content := payload.ConversationState.CurrentMessage.UserInputMessage.Content
	return &OpenAIRequest{Model: req.Model, MaxTokens: req.MaxTokens, Messages: []OpenAIMessage{{Role: "user", Content: content}}}
}

func (h *Handler) handleClaudeFusion(ctx context.Context, w http.ResponseWriter, req *ClaudeRequest, route *comboRouteSnapshot, thinking bool, apiKeyID string) {
	openReq := claudeFusionOpenAIRequest(req)
	result, err := h.executeOpenAIFusion(ctx, openReq, route, thinking, estimateClaudeRequestInputTokens(req), apiKeyID)
	if err != nil {
		h.recordFailureWithDetailsMeta("claude", route.RequestedModel, "", err, "", apiKeyID, "")
		h.sendClaudeError(w, http.StatusServiceUnavailable, "api_error", err.Error())
		return
	}
	h.recordComboPublicSuccess(result.inputTokens, result.outputTokens, result.credits)
	resp := KiroToClaudeResponse(result.content, result.reasoning, false, result.toolUses, result.inputTokens, result.outputTokens, route.RequestedModel)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleResponsesFusion(ctx context.Context, w http.ResponseWriter, req *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimated int, apiKeyID, clientIP, respID string, responsesReq *ResponsesRequest, storedInput json.RawMessage, storeResponse bool) {
	result, err := h.executeOpenAIFusion(ctx, req, route, thinking, estimated, apiKeyID)
	if err != nil {
		h.recordFailureWithDetailsMeta("responses", route.RequestedModel, "", err, clientIP, apiKeyID, "")
		h.sendOpenAIError(w, http.StatusServiceUnavailable, "server_error", err.Error())
		return
	}
	h.recordComboPublicSuccess(result.inputTokens, result.outputTokens, result.credits)
	resp := buildResponsesObject(respID, route.RequestedModel, result.content, result.toolUses, result.inputTokens, result.outputTokens, responsesReq)
	resp.StoredInput, resp.Instructions = storedInput, responsesReq.Instructions
	if storeResponse {
		if saveErr := saveResponse(resp); saveErr != nil {
			logResponsesPersistFailure(resp.ID, saveErr)
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleOpenAIFusion(ctx context.Context, w http.ResponseWriter, req *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimated int, apiKeyID, clientIP string) {
	result, err := h.executeOpenAIFusion(ctx, req, route, thinking, estimated, apiKeyID)
	if err != nil {
		h.recordFailureWithDetailsMeta("openai", route.RequestedModel, "", err, clientIP, apiKeyID, "")
		h.sendOpenAIError(w, http.StatusServiceUnavailable, "server_error", err.Error())
		return
	}
	h.recordComboPublicSuccess(result.inputTokens, result.outputTokens, result.credits)
	resp := KiroToOpenAIResponseWithReasoning(result.content, result.reasoning, result.toolUses, result.inputTokens, result.outputTokens, route.RequestedModel, config.GetThinkingConfig().OpenAIFormat)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
