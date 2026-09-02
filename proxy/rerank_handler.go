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

// handleRerank processes POST /v1/rerank requests.
func (h *Handler) handleRerank(w http.ResponseWriter, r *http.Request) {
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

	var req RerankRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Failed to parse request JSON: %v", err))
		return
	}

	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		req.Model = "rerank-2.5" // Default recommended Voyage reranker
	}
	if strings.TrimSpace(req.Query) == "" {
		h.sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "query is required")
		return
	}

	atomic.AddInt64(&h.totalRequests, 1)

	var account *config.Account
	accPool := pool.GetPool()
	if accPool != nil {
		account = accPool.GetNextForModelAndProviderExcluding(req.Model, "voyage", nil)
		if account == nil {
			account = accPool.GetNextForModelAndProviderExcluding("", "voyage", nil)
		}
	}

	resp, err := CallVoyageRerank(r.Context(), account, &req)

	accountID := ""
	if account != nil {
		accountID = account.ID
	}
	provider := "voyage"
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordFailureWithDetailsMeta("rerank", req.Model, accountID, err, clientIP, apiKeyID, provider)
		h.sendOpenAIError(w, http.StatusInternalServerError, "api_error", err.Error())
		return
	}

	atomic.AddInt64(&h.successRequests, 1)
	tokens := resp.Usage.TotalTokens
	atomic.AddInt64(&h.totalTokens, int64(tokens))
	atomic.AddInt64(&h.totalInputTokens, int64(tokens))

	credits := float64(0)
	if account != nil {
		_ = config.AddVoyageAccountUsage(account.ID, req.Model, tokens)
		priceInfo := config.GetVoyageModelPriceInfo(req.Model)
		credits = float64(tokens) * priceInfo.PricePerMillion / 1_000_000.0
	}

	h.recordSuccessLogMeta("rerank", req.Model, accountID, logTokens{
		Input: tokens,
	}, credits, durationMs, clientIP, apiKeyID, provider)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
