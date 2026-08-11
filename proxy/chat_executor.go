package proxy

import (
	"context"
	"errors"
	"kiro-go/config"
	"time"
)

var (
	errChatNoAvailableAccount = errors.New("chat: no available account")
	errChatUnsupportedOutput  = errors.New("chat: unsupported provider output")
)

type chatTextExecutionRequest struct {
	Messages  []OpenAIMessage
	Provider  string
	Model     string
	ClientIP  string
	RequestID string
}

type chatTextExecutionResult struct {
	Content             string
	Provider            string
	Model               string
	AccountID           string
	RequestID           string
	ProviderResponseID  string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	Credits             float64
}

func (h *Handler) executeChatText(ctx context.Context, req chatTextExecutionRequest) (chatTextExecutionResult, error) {
	if h.chatTextExecutor != nil {
		return h.chatTextExecutor(ctx, req)
	}
	openAIReq := &OpenAIRequest{Model: req.Model, Messages: req.Messages}
	estimatedInput := estimateOpenAIRequestInputTokens(openAIReq)
	start := time.Now()

	candidates := []struct{ model, provider string }{{req.Model, req.Provider}}
	if req.Provider == "combo" {
		route, err := h.resolveComboRoute(req.Model)
		if err != nil {
			return chatTextExecutionResult{}, err
		}
		if route == nil || route.Combo.Strategy == "fusion" {
			return chatTextExecutionResult{}, errChatUnsupportedOutput
		}
		candidates = candidates[:0]
		for _, candidate := range route.Candidates {
			candidates = append(candidates, struct{ model, provider string }{candidate.Model, candidate.Provider})
		}
	}

	var lastErr error
	for _, candidate := range candidates {
		candidateReq := *openAIReq
		candidateReq.Model = candidate.model
		candidatePayload := OpenAIToKiro(&candidateReq, isThinkingModel(candidate.model))
		excluded := map[string]bool{}
		for attempt := 0; attempt < maxAccountRetryAttempts; attempt++ {
			if h.pool == nil {
				break
			}
			account := h.pool.GetNextForModelAndProviderExcluding(candidate.model, candidate.provider, excluded)
			if account == nil {
				break
			}
			if err := h.ensureValidToken(account); err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			result, err := h.executeOpenAIComboAttempt(ctx, account, candidatePayload, candidate.model, isThinkingModel(candidate.model), estimatedInput)
			if err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			if len(result.toolUses) != 0 || result.content == "" {
				lastErr = errChatUnsupportedOutput
				excluded[account.ID] = true
				continue
			}
			provider := providerLabel(account)
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
			h.recordSuccessLogMeta("admin-chat", req.Model, account.ID, logTokens{Input: result.inputTokens, Output: result.outputTokens, CacheRead: result.usage.CacheRead, CacheCreation: result.usage.CacheCreation}, result.credits, time.Since(start).Milliseconds(), req.ClientIP, "", provider)
			return chatTextExecutionResult{
				Content: result.content, Provider: provider, Model: candidate.model,
				AccountID: account.ID, RequestID: req.RequestID,
				InputTokens: result.inputTokens, OutputTokens: result.outputTokens,
				CacheReadTokens: result.usage.CacheRead, CacheCreationTokens: result.usage.CacheCreation,
				Credits: result.credits,
			}, nil
		}
	}
	if lastErr != nil {
		h.recordFailureWithDetailsMeta("admin-chat", req.Model, "", lastErr, req.ClientIP, "", req.Provider)
		return chatTextExecutionResult{}, lastErr
	}
	return chatTextExecutionResult{}, errChatNoAvailableAccount
}

func isThinkingModel(model string) bool {
	_, thinking := ParseModelAndThinking(model, config.GetThinkingConfig().Suffix)
	return thinking
}
