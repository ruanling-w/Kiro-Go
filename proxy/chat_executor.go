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

type chatTextStreamCallbacks struct {
	OnText      func(text string, thinking bool) error
	OnCommitted func(provider, model string)
}

type chatStreamAttemptResult struct {
	chatTextExecutionResult
	committed bool
	toolUses  []KiroToolUse
}

type chatCandidate struct{ model, provider string }

func (h *Handler) chatCandidates(req chatTextExecutionRequest) ([]chatCandidate, error) {
	candidates := []chatCandidate{{req.Model, req.Provider}}
	if req.Provider != "combo" {
		return candidates, nil
	}
	route, err := h.resolveComboRoute(req.Model)
	if err != nil {
		return nil, err
	}
	if route == nil || route.Combo.Strategy == "fusion" {
		return nil, errChatUnsupportedOutput
	}
	candidates = candidates[:0]
	for _, candidate := range route.Candidates {
		candidates = append(candidates, chatCandidate{candidate.Model, candidate.Provider})
	}
	return candidates, nil
}

func (h *Handler) executeChatTextStream(ctx context.Context, req chatTextExecutionRequest, callbacks chatTextStreamCallbacks) (chatTextExecutionResult, error) {
	if h.chatTextStreamExecutor != nil {
		return h.chatTextStreamExecutor(ctx, req, callbacks)
	}
	openAIReq := &OpenAIRequest{Model: req.Model, Messages: req.Messages}
	estimatedInput := estimateOpenAIRequestInputTokens(openAIReq)
	candidates, err := h.chatCandidates(req)
	if err != nil {
		return chatTextExecutionResult{}, err
	}
	start := time.Now()
	var lastErr error
	for _, candidate := range candidates {
		candidateReq := *openAIReq
		candidateReq.Model = candidate.model
		payload := OpenAIToKiro(&candidateReq, isThinkingModel(candidate.model))
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
			result, attemptErr := h.executeChatTextStreamAttempt(ctx, account, payload, candidate.model, estimatedInput, callbacks)
			provider := providerLabel(account)
			result.Provider, result.Model = provider, candidate.model
			result.AccountID, result.RequestID = account.ID, req.RequestID
			if attemptErr != nil {
				lastErr = attemptErr
				if result.committed || errors.Is(attemptErr, context.Canceled) || errors.Is(attemptErr, context.DeadlineExceeded) {
					return result.chatTextExecutionResult, attemptErr
				}
				excluded[account.ID] = true
				h.handleAccountFailure(account, attemptErr)
				continue
			}
			if len(result.toolUses) != 0 || result.Content == "" {
				lastErr = errChatUnsupportedOutput
				if result.committed {
					return result.chatTextExecutionResult, lastErr
				}
				excluded[account.ID] = true
				continue
			}
			result.Provider, result.Model = provider, candidate.model
			result.AccountID, result.RequestID = account.ID, req.RequestID
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.InputTokens+result.OutputTokens, result.Credits)
			h.recordSuccessLogMeta("admin-chat", req.Model, account.ID, logTokens{Input: result.InputTokens, Output: result.OutputTokens, CacheRead: result.CacheReadTokens, CacheCreation: result.CacheCreationTokens}, result.Credits, time.Since(start).Milliseconds(), req.ClientIP, "", provider)
			return result.chatTextExecutionResult, nil
		}
	}
	if lastErr != nil {
		h.recordFailureWithDetailsMeta("admin-chat", req.Model, "", lastErr, req.ClientIP, "", req.Provider)
		return chatTextExecutionResult{}, lastErr
	}
	return chatTextExecutionResult{}, errChatNoAvailableAccount
}

func (h *Handler) executeChatTextStreamAttempt(ctx context.Context, account *config.Account, payload *KiroPayload, model string, estimatedInput int, callbacks chatTextStreamCallbacks) (chatStreamAttemptResult, error) {
	var result chatStreamAttemptResult
	var realInput int
	var callbackErr error
	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	callback := &KiroStreamCallback{
		OnText: func(text string, thinking bool) {
			if text == "" || callbackErr != nil {
				return
			}
			if callbacks.OnText != nil {
				if err := callbacks.OnText(text, thinking); err != nil {
					callbackErr = err
					cancel()
					return
				}
			}
			if !result.committed {
				result.committed = true
				if callbacks.OnCommitted != nil {
					callbacks.OnCommitted(providerLabel(account), model)
				}
			}
			if !thinking {
				result.Content += text
			}
		},
		OnToolUse:  func(tool KiroToolUse) { result.toolUses = append(result.toolUses, tool) },
		OnComplete: func(input, output int) { result.InputTokens, result.OutputTokens = input, output },
		OnUsage: func(usage tokenUsage) {
			result.CacheReadTokens, result.CacheCreationTokens = usage.CacheRead, usage.CacheCreation
			if usage.Input > 0 {
				result.InputTokens = usage.Input
			}
			if usage.Output > 0 {
				result.OutputTokens = usage.Output
			}
		},
		OnCredits:      func(credits float64) { result.Credits = credits },
		OnContextUsage: func(percent float64) { realInput = int(percent * float64(getContextWindowSize(model)) / 100) },
	}
	if err := CallProvider(attemptCtx, account, payload, callback); err != nil {
		if callbackErr != nil {
			return result, callbackErr
		}
		return result, err
	}
	if callbackErr != nil {
		return result, callbackErr
	}
	if realInput > 0 {
		result.InputTokens = realInput
	} else if result.InputTokens <= 0 {
		result.InputTokens = estimatedInput
	}
	if result.OutputTokens <= 0 {
		result.OutputTokens = estimateOpenAIOutputTokens(result.Content, "", result.toolUses)
	}
	return result, nil
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
