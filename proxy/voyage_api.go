package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/auth"
	"kiro-go/config"
	"kiro-go/logger"
	"net/http"
	"os"
	"strings"
	"time"
)

const voyageDefaultBaseURL = "https://api.voyageai.com"

// voyageModelInfos returns the Voyage AI models for provider catalog and model listing.
func voyageModelInfos() []ModelInfo {
	return []ModelInfo{
		{ModelId: "voyage-4-large", ModelName: "Voyage 4 Large", Description: "Voyage AI general-purpose and multilingual text embedding model (context: 32k, dim: 1024/256/512/2048)"},
		{ModelId: "voyage-4", ModelName: "Voyage 4", Description: "Voyage AI text embedding model optimized for retrieval quality (context: 32k, dim: 1024/256/512/2048)"},
		{ModelId: "voyage-4-lite", ModelName: "Voyage 4 Lite", Description: "Voyage AI text embedding model optimized for latency and cost (context: 32k, dim: 1024/256/512/2048)"},
		{ModelId: "voyage-code-4", ModelName: "Voyage Code 4", Description: "Voyage AI text embedding model optimized for code retrieval (context: 32k, dim: 1024/256/512/2048)"},
		{ModelId: "voyage-4-nano", ModelName: "Voyage 4 Nano", Description: "Voyage AI lightweight text embedding model (context: 32k, dim: 1024/256/512/2048)"},
		{ModelId: "voyage-3.5", ModelName: "Voyage 3.5", Description: "Voyage AI text embedding model (context: 32k, dim: 1024)"},
		{ModelId: "voyage-3.5-lite", ModelName: "Voyage 3.5 Lite", Description: "Voyage AI fast text embedding model (context: 32k, dim: 1024)"},
		{ModelId: "voyage-3", ModelName: "Voyage 3", Description: "Voyage AI text embedding model (context: 32k, dim: 1024)"},
		{ModelId: "voyage-code-3", ModelName: "Voyage Code 3", Description: "Voyage AI code embedding model (context: 32k, dim: 1024)"},
		{ModelId: "voyage-finance-2", ModelName: "Voyage Finance 2", Description: "Voyage AI finance domain-specific embedding model (context: 32k, dim: 1024)"},
		{ModelId: "voyage-law-2", ModelName: "Voyage Law 2", Description: "Voyage AI legal domain-specific embedding model (context: 16k, dim: 1024)"},
		{ModelId: "voyage-multilingual-2", ModelName: "Voyage Multilingual 2", Description: "Voyage AI multilingual embedding model (context: 32k, dim: 1024)"},
		{ModelId: "rerank-2.5", ModelName: "Voyage Rerank 2.5", Description: "Voyage AI generalist reranker with instruction following & multilingual (context: 32k)"},
		{ModelId: "rerank-2.5-lite", ModelName: "Voyage Rerank 2.5 Lite", Description: "Voyage AI fast reranker with instruction following & multilingual (context: 32k)"},
		{ModelId: "rerank-2", ModelName: "Voyage Rerank 2", Description: "Voyage AI second-generation reranker (context: 16k)"},
		{ModelId: "rerank-2-lite", ModelName: "Voyage Rerank 2 Lite", Description: "Voyage AI lightweight reranker (context: 8k)"},
	}
}

// voyageModelIDs returns the model ids supported by Voyage AI.
func voyageModelIDs() []string {
	infos := voyageModelInfos()
	ids := make([]string, len(infos))
	for i, info := range infos {
		ids[i] = info.ModelId
	}
	return ids
}

// resolveVoyageAPIKey extracts the API key from account or environment.
func resolveVoyageAPIKey(account *config.Account) string {
	if account != nil {
		if strings.TrimSpace(account.VoyageAPIKey) != "" {
			return strings.TrimSpace(account.VoyageAPIKey)
		}
		if strings.TrimSpace(account.AccessToken) != "" {
			return strings.TrimSpace(account.AccessToken)
		}
	}
	return strings.TrimSpace(os.Getenv("VOYAGE_API_KEY"))
}

// CallVoyageEmbedding executes a text embedding request against Voyage AI.
func CallVoyageEmbedding(ctx context.Context, account *config.Account, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	apiKey := resolveVoyageAPIKey(account)
	if apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is not configured (missing in account and VOYAGE_API_KEY env)")
	}

	inputs, err := req.ParseInputStrings()
	if err != nil {
		return nil, fmt.Errorf("voyage: %w", err)
	}

	// Prepare payload for Voyage AI REST API
	bodyMap := map[string]interface{}{
		"model": req.Model,
		"input": inputs,
	}
	if req.InputType != "" {
		bodyMap["input_type"] = req.InputType
	}
	if req.Truncation != nil {
		bodyMap["truncation"] = *req.Truncation
	}
	if dim := req.EffectiveDimensions(); dim != nil {
		bodyMap["output_dimension"] = *dim
	}
	if req.OutputDType != "" {
		bodyMap["output_dtype"] = req.OutputDType
	}
	if req.EncodingFormat != "" {
		bodyMap["encoding_format"] = req.EncodingFormat
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("voyage: marshal request: %w", err)
	}

	url := voyageDefaultBaseURL + "/v1/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("voyage: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", "Kiro-Go/"+config.Version)

	proxyURL := ""
	if account != nil {
		proxyURL = account.ProxyURL
	}
	client := auth.GetAuthClientForProxy(proxyURL)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voyage: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("voyage: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errObj)
		errMsg := errObj.Detail
		if errMsg == "" {
			errMsg = errObj.Message
		}
		if errMsg == "" {
			errMsg = errObj.Error
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		logger.Warnf("[Voyage] Embeddings error (status %d): %s", resp.StatusCode, errMsg)
		return nil, fmt.Errorf("voyage upstream error (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	// Voyage returns standard OpenAI format:
	// {"object": "list", "data": [{"object": "embedding", "embedding": [...], "index": 0}], "model": "...", "usage": {"total_tokens": 10}}
	var voyageResp struct {
		Object string `json:"object"`
		Data   []struct {
			Object    string      `json:"object"`
			Index     int         `json:"index"`
			Embedding interface{} `json:"embedding"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &voyageResp); err != nil {
		return nil, fmt.Errorf("voyage: unmarshal response: %w", err)
	}

	res := &EmbeddingResponse{
		Object: "list",
		Model:  voyageResp.Model,
		Data:   make([]EmbeddingData, len(voyageResp.Data)),
		Usage: EmbeddingUsage{
			PromptTokens: voyageResp.Usage.PromptTokens,
			TotalTokens:  voyageResp.Usage.TotalTokens,
		},
	}
	if res.Usage.PromptTokens == 0 && res.Usage.TotalTokens > 0 {
		res.Usage.PromptTokens = res.Usage.TotalTokens
	}
	if res.Model == "" {
		res.Model = req.Model
	}

	for i, d := range voyageResp.Data {
		res.Data[i] = EmbeddingData{
			Object:    "embedding",
			Index:     d.Index,
			Embedding: d.Embedding,
		}
	}

	return res, nil
}

// CallVoyageRerank executes a reranking request against Voyage AI.
func CallVoyageRerank(ctx context.Context, account *config.Account, req *RerankRequest) (*RerankResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	apiKey := resolveVoyageAPIKey(account)
	if apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is not configured (missing in account and VOYAGE_API_KEY env)")
	}

	docs, err := req.ParseDocuments()
	if err != nil {
		return nil, fmt.Errorf("voyage: %w", err)
	}

	bodyMap := map[string]interface{}{
		"model":     req.Model,
		"query":     req.Query,
		"documents": docs,
	}
	if req.TopK != nil {
		bodyMap["top_k"] = *req.TopK
	}
	if req.Truncation != nil {
		bodyMap["truncation"] = *req.Truncation
	}
	if req.ReturnDocuments != nil {
		bodyMap["return_documents"] = *req.ReturnDocuments
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("voyage: marshal rerank request: %w", err)
	}

	url := voyageDefaultBaseURL + "/v1/rerank"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("voyage: create rerank request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", "Kiro-Go/"+config.Version)

	proxyURL := ""
	if account != nil {
		proxyURL = account.ProxyURL
	}
	client := auth.GetAuthClientForProxy(proxyURL)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voyage: rerank request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("voyage: read rerank response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errObj)
		errMsg := errObj.Detail
		if errMsg == "" {
			errMsg = errObj.Message
		}
		if errMsg == "" {
			errMsg = errObj.Error
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		logger.Warnf("[Voyage] Rerank error (status %d): %s", resp.StatusCode, errMsg)
		return nil, fmt.Errorf("voyage upstream error (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	// Voyage returns response with "data" or "results":
	// {"object": "list", "data": [{"index": 0, "relevance_score": 0.94, "document": "..."}], "model": "...", "usage": {"total_tokens": 42}}
	var rawResp struct {
		Object  string `json:"object"`
		Model   string `json:"model"`
		Data    []RerankResult `json:"data"`
		Results []RerankResult `json:"results"`
		Usage   struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		TotalTokens int `json:"total_tokens"`
	}

	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("voyage: unmarshal rerank response: %w", err)
	}

	results := rawResp.Results
	if len(results) == 0 && len(rawResp.Data) > 0 {
		results = rawResp.Data
	}

	totalTokens := rawResp.Usage.TotalTokens
	if totalTokens == 0 && rawResp.TotalTokens > 0 {
		totalTokens = rawResp.TotalTokens
	}

	modelName := rawResp.Model
	if modelName == "" {
		modelName = req.Model
	}

	return &RerankResponse{
		Object:  "list",
		Model:   modelName,
		Results: results,
		Usage: RerankUsage{
			TotalTokens: totalTokens,
		},
	}, nil
}

// refreshVoyageInfo refreshes account stats and recalculates Voyage free tier quota balances.
func refreshVoyageInfo(account *config.Account, info *config.AccountInfo) (*config.AccountInfo, error) {
	buckets := config.CalculateVoyageBuckets(account.VoyageUsage)
	account.VoyageQuota = buckets

	totalUsedTokens := int64(0)
	for _, used := range account.VoyageUsage {
		totalUsedTokens += used
	}

	info.SubscriptionType = "voyage-api"
	info.SubscriptionTitle = "Voyage AI Free Tier"
	info.UsageCurrent = float64(totalUsedTokens) / 1_000_000.0
	info.UsageLimit = 200.0 // 200M primary free tier quota limit
	if info.UsageLimit > 0 {
		info.UsagePercent = (info.UsageCurrent / info.UsageLimit) * 100.0
		if info.UsagePercent > 100.0 {
			info.UsagePercent = 100.0
		}
	}
	info.LastRefresh = time.Now().Unix()

	return info, nil
}
