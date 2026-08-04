package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (h *Handler) handleResponsesComboNonStream(ctx context.Context, w http.ResponseWriter, original *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimatedInput int, apiKeyID, clientIP, respID string, req *ResponsesRequest, storedInput json.RawMessage, storeResponse bool) {
	start := time.Now()
	var lastErr error
	for _, candidate := range route.Candidates {
		candidateReq := *original
		candidateReq.Model = candidate.Model
		payload := OpenAIToKiro(&candidateReq, thinking)
		excluded := map[string]bool{}
		for attempt := 0; attempt < maxAccountRetryAttempts; attempt++ {
			account := h.pool.GetNextForModelExcluding(candidate.Model, excluded)
			if account == nil {
				break
			}
			if err := h.ensureValidToken(account); err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			result, err := h.executeOpenAIComboAttempt(ctx, account, payload, candidate.Model, thinking, estimatedInput)
			if err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			h.recordSuccessForApiKey(apiKeyID, result.inputTokens, result.outputTokens, result.credits)
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
			h.recordSuccessLogMeta("responses", route.RequestedModel, account.ID, result.inputTokens+result.outputTokens, result.credits, time.Since(start).Milliseconds(), clientIP, apiKeyID, providerLabel(account))
			respObj := buildResponsesObject(respID, route.RequestedModel, result.content, result.toolUses, result.inputTokens, result.outputTokens, req)
			respObj.StoredInput = storedInput
			respObj.Instructions = req.Instructions
			if storeResponse {
				if saveErr := saveResponse(respObj); saveErr != nil {
					logResponsesPersistFailure(respObj.ID, saveErr)
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(respObj)
			return
		}
	}
	if lastErr == nil {
		h.logComboPoolEmpty("responses-combo", route)
		h.sendOpenAIError(w, http.StatusServiceUnavailable, "server_error", "No available accounts")
		return
	}
	h.recordFailureWithDetailsMeta("responses", route.RequestedModel, "", lastErr, clientIP, apiKeyID, "")
	h.sendOpenAIError(w, http.StatusInternalServerError, "server_error", lastErr.Error())
}
