package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/config"
	"kiro-go/pool"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// handleEmbeddings processes POST /v1/embeddings requests.
func (h *Handler) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendOpenAIError(w, http.StatusMethodNotAllowed, "invalid_request_error", "Method not allowed")
		return
	}

	startTime := time.Now()
	clientIP := clientIPFromContext(r.Context())
	if clientIP == "" {
		clientIP = ClientIP(r, config.GetTrustProxyHeaders())
	}
	apiKeyID := apiKeyIDFromContext(r.Context())

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}

	var req EmbeddingRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Failed to parse request JSON: %v", err))
		return
	}

	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		h.sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	atomic.AddInt64(&h.totalRequests, 1)

	// Determine provider and route
	var (
		account  *config.Account
		provider string
		resp     *EmbeddingResponse
	)

	accPool := pool.GetPool()

	switch {
	case IsVoyageModel(req.Model):
		provider = "voyage"
		if accPool != nil {
			account = accPool.GetNextForModelAndProviderExcluding(req.Model, "voyage", nil)
			if account == nil {
				account = accPool.GetNextForModelAndProviderExcluding("", "voyage", nil)
			}
		}
		resp, err = CallVoyageEmbedding(r.Context(), account, &req)

	case IsGeminiEmbeddingModel(req.Model):
		provider = "antigravity"
		if accPool != nil {
			account = accPool.GetNextForModelAndProviderExcluding(req.Model, "antigravity", nil)
			if account == nil {
				account = accPool.GetNextForModelAndProviderExcluding("", "antigravity", nil)
			}
		}
		resp, err = CallGeminiEmbedding(r.Context(), account, &req)

	case IsOpenAIEmbeddingModel(req.Model):
		provider = "openai"
		if accPool != nil {
			account = accPool.GetNextForModelAndProviderExcluding(req.Model, "remotekiro", nil)
			if account == nil {
				account = accPool.GetNextForModelAndProviderExcluding(req.Model, "codex", nil)
			}
		}
		resp, err = CallOpenAIEmbedding(r.Context(), account, &req)

	default:
		// Default routing: Check if a Voyage account is available first, otherwise try OpenAI
		if accPool != nil {
			account = accPool.GetNextForModelAndProviderExcluding(req.Model, "voyage", nil)
		}
		if account != nil || IsVoyageModel(req.Model) {
			provider = "voyage"
			resp, err = CallVoyageEmbedding(r.Context(), account, &req)
		} else {
			provider = "openai"
			if accPool != nil {
				account = accPool.GetNextForModelAndProviderExcluding(req.Model, "", nil)
			}
			resp, err = CallOpenAIEmbedding(r.Context(), account, &req)
		}
	}

	accountID := ""
	if account != nil {
		accountID = account.ID
	}

	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordFailureWithDetailsMeta("embeddings", req.Model, accountID, err, clientIP, apiKeyID, provider)
		h.sendOpenAIError(w, http.StatusInternalServerError, "api_error", err.Error())
		return
	}

	atomic.AddInt64(&h.successRequests, 1)
	tokens := resp.Usage.TotalTokens
	if tokens == 0 {
		tokens = resp.Usage.PromptTokens
	}
	atomic.AddInt64(&h.totalTokens, int64(tokens))
	atomic.AddInt64(&h.totalInputTokens, int64(tokens))

	credits := float64(0)
	if provider == "voyage" && account != nil {
		_ = config.AddVoyageAccountUsage(account.ID, req.Model, tokens)
		priceInfo := config.GetVoyageModelPriceInfo(req.Model)
		credits = float64(tokens) * priceInfo.PricePerMillion / 1_000_000.0
	}

	h.recordSuccessLogMeta("embeddings", req.Model, accountID, logTokens{
		Input: tokens,
	}, credits, durationMs, clientIP, apiKeyID, provider)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
