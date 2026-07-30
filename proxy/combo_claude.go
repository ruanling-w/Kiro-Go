package proxy

import (
	"encoding/json"
	"errors"
	"kiro-go/config"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type claudeComboResult struct {
	content      string
	thinking     string
	toolUses     []KiroToolUse
	inputTokens  int
	outputTokens int
	credits      float64
}

func (h *Handler) handleClaudeComboNonStream(w http.ResponseWriter, original *ClaudeRequest, route *comboRouteSnapshot, thinking bool, thinkingOpts claudeThinkingResponseOptions, apiKeyID, clientIP string) {
	start := time.Now()
	var lastErr error
	for _, candidate := range route.Candidates {
		req := *original
		req.Model = candidate.Model
		effective := cloneClaudeRequestForThinking(&req, thinking)
		estimatedInput := estimateClaudeRequestInputTokens(effective)
		cacheProfile := h.promptCache.BuildClaudeProfile(effective, estimatedInput)
		payload := ClaudeToKiro(&req, thinking)
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
			result, err := h.executeClaudeComboAttempt(account, payload, candidate.Model, thinking, estimatedInput)
			if err != nil {
				lastErr = err
				excluded[account.ID] = true
				h.handleAccountFailure(account, err)
				continue
			}
			h.recordSuccessForApiKey(apiKeyID, result.inputTokens, result.outputTokens, result.credits)
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
			h.promptCache.Update(account.ID, cacheProfile)
			h.recordSuccessLogMeta("claude", route.RequestedModel, account.ID, result.inputTokens+result.outputTokens, result.credits, time.Since(start).Milliseconds(), clientIP, apiKeyID, providerLabel(account))
			responseThinking := result.thinking
			includeEmpty := thinking && thinkingOpts.OmitDisplay && responseThinking != ""
			if includeEmpty {
				responseThinking = ""
			}
			content := result.content
			if thinking && responseThinking != "" {
				switch thinkingOpts.Format {
				case "think":
					content = "<think>" + responseThinking + "</think>" + content
					responseThinking = ""
				case "reasoning_content":
					content = responseThinking + content
					responseThinking = ""
				}
			}
			resp := KiroToClaudeResponse(content, responseThinking, includeEmpty, result.toolUses, result.inputTokens, result.outputTokens, route.RequestedModel)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}
	if lastErr == nil {
		h.sendClaudeError(w, http.StatusServiceUnavailable, "api_error", "No available accounts")
		return
	}
	h.recordFailureWithDetailsMeta("claude", route.RequestedModel, "", lastErr, clientIP, apiKeyID, "")
	h.sendClaudeError(w, http.StatusBadGateway, "api_error", lastErr.Error())
}

func setClaudeStreamHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// handleClaudeComboStream resolves candidate order once, retries only before the
// first Claude event, and pins the account/model as soon as message_start is public.
func (h *Handler) handleClaudeComboStream(w http.ResponseWriter, original *ClaudeRequest, route *comboRouteSnapshot, thinking bool, thinkingOpts claudeThinkingResponseOptions, apiKeyID, clientIP string) {
	if _, ok := w.(http.Flusher); !ok {
		h.sendClaudeError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
		return
	}
	start := time.Now()
	msgID := "msg_" + uuid.New().String()
	gate := newStreamCommitGate(w)
	var lastErr error
	for _, candidate := range route.Candidates {
		req := *original
		req.Model = candidate.Model
		effective := cloneClaudeRequestForThinking(&req, thinking)
		estimatedInput := estimateClaudeRequestInputTokens(effective)
		cacheProfile := h.promptCache.BuildClaudeProfile(effective, estimatedInput)
		payload := ClaudeToKiro(&req, thinking)
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
			provider := providerLabel(account)
			cacheUsage := h.promptCache.Compute(account.ID, cacheProfile)
			gate.Discard()
			setClaudeStreamHeaders(gate)
			sink := &comboStreamSink{gate: gate}
			sse := newClaudeSSEWriter(sink, sink)
			result := h.streamClaudeAttempt(account, payload, claudeStreamAttempt{sse: sse, messageID: msgID, effectiveModel: candidate.Model, publicModel: route.RequestedModel, thinking: thinking, thinkingOpts: thinkingOpts, estimatedInputTokens: estimatedInput, cacheProfile: cacheProfile, cacheUsage: cacheUsage})
			if result.writeErr != nil {
				if gate.Committed() {
					return
				}
				gate.Discard()
				lastErr = result.writeErr
				excluded[account.ID] = true
				continue
			}
			if result.upstreamErr != nil {
				lastErr = result.upstreamErr
				if !gate.Committed() {
					gate.Discard()
					excluded[account.ID] = true
					h.handleAccountFailure(account, result.upstreamErr)
					continue
				}
				h.handleAccountFailure(account, result.upstreamErr)
				h.recordFailureWithDetailsMeta("claude", route.RequestedModel, account.ID, result.upstreamErr, clientIP, apiKeyID, provider)
				_ = sse.Send("error", map[string]interface{}{"type": "error", "error": map[string]string{"type": "api_error", "message": result.upstreamErr.Error()}})
				return
			}
			if !result.finished {
				if !gate.Committed() {
					gate.Discard()
					lastErr = errors.New("provider returned an empty stream")
					excluded[account.ID] = true
					continue
				}
				return
			}
			h.recordSuccessForApiKey(apiKeyID, result.inputTokens, result.outputTokens, result.credits)
			h.pool.RecordSuccess(account.ID)
			h.pool.UpdateStats(account.ID, result.inputTokens+result.outputTokens, result.credits)
			h.promptCache.Update(account.ID, cacheProfile)
			h.recordSuccessLogMeta("claude", route.RequestedModel, account.ID, result.inputTokens+result.outputTokens, result.credits, time.Since(start).Milliseconds(), clientIP, apiKeyID, provider)
			return
		}
	}
	gate.Discard()
	if lastErr == nil {
		h.sendClaudeError(w, http.StatusServiceUnavailable, "api_error", "No available accounts")
		return
	}
	h.recordFailureWithDetailsMeta("claude", route.RequestedModel, "", lastErr, clientIP, apiKeyID, "")
	h.sendClaudeError(w, http.StatusBadGateway, "api_error", lastErr.Error())
}

func (h *Handler) executeClaudeComboAttempt(account *config.Account, payload *KiroPayload, model string, thinking bool, estimatedInput int) (claudeComboResult, error) {
	var result claudeComboResult
	var realInput int
	callback := &KiroStreamCallback{
		OnText: func(text string, isThinking bool) {
			if isThinking {
				result.thinking += text
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
		OnContextUsage: func(pct float64) { realInput = int(pct * float64(getContextWindowSize(model)) / 100) },
	}
	if err := CallProvider(account, payload, callback); err != nil {
		return claudeComboResult{}, err
	}
	var extracted string
	result.content, extracted = extractThinkingFromContent(result.content)
	if thinking && result.thinking == "" {
		result.thinking = extracted
	}
	if !thinking {
		result.thinking = ""
	}
	if realInput > 0 {
		result.inputTokens = realInput
	} else if result.inputTokens <= 0 {
		result.inputTokens = estimatedInput
	}
	if result.content == "" && result.thinking == "" && len(result.toolUses) == 0 {
		return claudeComboResult{}, errors.New("provider returned an empty response")
	}
	result.outputTokens = estimateClaudeOutputTokens(result.content, result.thinking, result.toolUses)
	return result, nil
}
