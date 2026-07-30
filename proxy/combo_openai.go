package proxy

import (
	"encoding/json"
	"errors"
	"kiro-go/config"
	"net/http"
	"time"
)

type openAIComboResult struct {
	content, reasoning string
	toolUses           []KiroToolUse
	inputTokens        int
	outputTokens       int
	credits            float64
	accountID          string
	provider           string
}

func (h *Handler) handleOpenAIComboNonStream(w http.ResponseWriter, original *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimatedInputTokens int, apiKeyID, clientIP string) {
	start := time.Now()
	var lastErr error
	for _, candidate := range route.Candidates {
		req := *original
		req.Model = candidate.Model
		payload := OpenAIToKiro(&req, thinking)
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
			result, err := h.executeOpenAIComboAttempt(account, payload, candidate.Model, thinking, estimatedInputTokens)
			if err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			result.accountID = account.ID
			result.provider = providerLabel(account)
			h.recordSuccessForApiKey(apiKeyID, result.inputTokens, result.outputTokens, result.credits)
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
			h.recordSuccessLogMeta("openai", route.RequestedModel, account.ID, result.inputTokens+result.outputTokens, result.credits, time.Since(start).Milliseconds(), clientIP, apiKeyID, result.provider)
			resp := KiroToOpenAIResponseWithReasoning(result.content, result.reasoning, result.toolUses, result.inputTokens, result.outputTokens, route.RequestedModel, config.GetThinkingConfig().OpenAIFormat)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}
	if lastErr == nil {
		h.sendOpenAIError(w, http.StatusServiceUnavailable, "server_error", "No available accounts")
		return
	}
	h.recordFailureWithDetailsMeta("openai", route.RequestedModel, "", lastErr, clientIP, apiKeyID, "")
	h.sendOpenAIError(w, http.StatusBadGateway, "server_error", lastErr.Error())
}

func (h *Handler) executeOpenAIComboAttempt(account *config.Account, payload *KiroPayload, model string, thinking bool, estimatedInputTokens int) (openAIComboResult, error) {
	var result openAIComboResult
	var realInputTokens int
	callback := &KiroStreamCallback{
		OnText: func(text string, isThinking bool) {
			if isThinking {
				result.reasoning += text
			} else {
				result.content += text
			}
		},
		OnToolUse: func(tu KiroToolUse) { result.toolUses = append(result.toolUses, tu) },
		OnImage: func(b64, mime string, partial bool) {
			if !partial && b64 != "" {
				result.content += imageMarkdownDataURI(b64, mime)
			}
		},
		OnComplete:     func(inTok, outTok int) { result.inputTokens, result.outputTokens = inTok, outTok },
		OnCredits:      func(c float64) { result.credits = c },
		OnContextUsage: func(pct float64) { realInputTokens = int(pct * float64(getContextWindowSize(model)) / 100) },
	}
	if err := CallProvider(account, payload, callback); err != nil {
		return openAIComboResult{}, err
	}
	var extracted string
	result.content, extracted = extractThinkingFromContent(result.content)
	if thinking && result.reasoning == "" {
		result.reasoning = extracted
	}
	if !thinking {
		result.reasoning = ""
	}
	if realInputTokens > 0 {
		result.inputTokens = realInputTokens
	} else if result.inputTokens <= 0 {
		result.inputTokens = estimatedInputTokens
	}
	result.outputTokens = estimateOpenAIOutputTokens(result.content, result.reasoning, result.toolUses)
	if result.content == "" && result.reasoning == "" && len(result.toolUses) == 0 {
		return openAIComboResult{}, errors.New("provider returned an empty response")
	}
	return result, nil
}
