package proxy

import (
	"encoding/json"
	"errors"
	"kiro-go/config"
	"net/http"
	"time"

	"github.com/google/uuid"
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

// comboStreamSink publishes SSE frames through a streamCommitGate and commits the
// gate the moment the first complete frame is handed to it. Staging one frame and
// then committing keeps a pre-commit attempt failure invisible to the client while
// still streaming the winning attempt live: after Commit the gate forwards writes
// straight to the real ResponseWriter.
//
// Because at most one frame is ever buffered, errStreamGateOverflow can only be
// raised pre-commit (a single frame larger than the gate limit). The sink returns
// it unchanged so the caller fails that attempt instead of truncating the frame.
type comboStreamSink struct {
	gate *streamCommitGate
}

func (s *comboStreamSink) Header() http.Header    { return s.gate.Header() }
func (s *comboStreamSink) WriteHeader(status int) { s.gate.WriteHeader(status) }

func (s *comboStreamSink) Write(p []byte) (int, error) {
	if s.gate.Committed() {
		return s.gate.Write(p)
	}
	if n, err := s.gate.Write(p); err != nil {
		return n, err
	}
	if err := s.gate.Commit(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *comboStreamSink) Flush() { s.gate.Flush() }

// handleOpenAIComboStream serves a streaming /v1/chat/completions request through
// a Combo. Candidate order is already resolved (round-robin reservation happened
// once in resolveComboRoute), and each candidate exhausts its own accounts with
// GetNextForModelExcluding — Combo never invokes the built-in ModelFallback, so
// two independent fallback loops cannot fight each other.
//
// The first published frame pins the attempt: before it, failures switch account
// and then candidate with nothing visible to the client; after it, the stream is
// terminated with the same error chunk + [DONE] the direct path emits and no
// account or model switch is allowed. Every wire "model" field is the Combo name
// the client sent (route.RequestedModel).
func (h *Handler) handleOpenAIComboStream(w http.ResponseWriter, original *OpenAIRequest, route *comboRouteSnapshot, thinking bool, estimatedInputTokens int, apiKeyID, clientIP string) {
	if _, ok := w.(http.Flusher); !ok {
		h.sendOpenAIError(w, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	start := time.Now()
	thinkingFormat := config.GetThinkingConfig().OpenAIFormat
	// One public request identity for every attempt.
	chatID := "chatcmpl-" + uuid.New().String()
	gate := newStreamCommitGate(w)

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

			usedProvider := providerLabel(account)
			gate.Discard()
			setOpenAIStreamHeaders(gate)
			sink := &comboStreamSink{gate: gate}
			sse := newOpenAISSEWriter(sink, sink)

			result := h.streamOpenAIAttempt(account, payload, openAIStreamAttempt{
				sse:            sse,
				chatID:         chatID,
				effectiveModel: candidate.Model,
				// Public identity stays the Combo name for every frame.
				publicModel:          route.RequestedModel,
				thinking:             thinking,
				thinkingFormat:       thinkingFormat,
				estimatedInputTokens: estimatedInputTokens,
			})

			if writeErr := result.writeErr; writeErr != nil {
				if gate.Committed() {
					// Bytes already reached the client; a partial write cannot be retried.
					return
				}
				// Nothing was published (typically errStreamGateOverflow on an
				// oversized first frame). Fail this attempt without truncating and
				// without blaming the account for a gateway-side limit.
				gate.Discard()
				lastErr = writeErr
				excluded[account.ID] = true
				continue
			}
			if err := result.upstreamErr; err != nil {
				if !gate.Committed() {
					gate.Discard()
					lastErr = err
					excluded[account.ID] = true
					h.handleAccountFailure(account, err)
					continue
				}
				// Committed: pinned to this account and model. Terminate the stream
				// explicitly instead of dropping the connection mid-response.
				h.handleAccountFailure(account, err)
				h.recordFailureWithDetailsMeta("openai", route.RequestedModel, account.ID, err, clientIP, apiKeyID, usedProvider)
				emitOpenAIStreamTerminalError(sse, chatID, route.RequestedModel, err)
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
			h.recordSuccessLogMeta("openai", route.RequestedModel, account.ID, result.inputTokens+result.outputTokens, result.credits, time.Since(start).Milliseconds(), clientIP, apiKeyID, usedProvider)
			return
		}
	}

	// Nothing was ever committed, so the client has seen no bytes and a normal
	// JSON error is still the correct protocol response.
	gate.Discard()
	if lastErr == nil {
		h.sendOpenAIError(w, http.StatusServiceUnavailable, "server_error", "No available accounts")
		return
	}
	h.recordFailureWithDetailsMeta("openai", route.RequestedModel, "", lastErr, clientIP, apiKeyID, "")
	h.sendOpenAIError(w, http.StatusBadGateway, "server_error", lastErr.Error())
}
