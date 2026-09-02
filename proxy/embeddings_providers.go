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
)

// CallOpenAIEmbedding dispatches an embedding request to OpenAI or RemoteKiro upstream.
func CallOpenAIEmbedding(ctx context.Context, account *config.Account, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	baseURL := "https://api.openai.com"
	apiKey := ""
	proxyURL := ""

	if account != nil {
		proxyURL = account.ProxyURL
		if account.RemoteBaseURL != "" {
			baseURL = strings.TrimRight(account.RemoteBaseURL, "/")
		}
		if account.AccessToken != "" {
			apiKey = account.AccessToken
		}
	}

	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("openai: API key is not configured for embedding")
	}

	inputs, err := req.ParseInputStrings()
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}

	bodyMap := map[string]interface{}{
		"model": req.Model,
		"input": inputs,
	}
	if req.Dimensions != nil {
		bodyMap["dimensions"] = *req.Dimensions
	}
	if req.EncodingFormat != "" {
		bodyMap["encoding_format"] = req.EncodingFormat
	}
	if req.User != "" {
		bodyMap["user"] = req.User
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	url := baseURL + "/v1/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", "Kiro-Go/"+config.Version)

	client := auth.GetAuthClientForProxy(proxyURL)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errObj)
		errMsg := errObj.Error.Message
		if errMsg == "" {
			errMsg = string(respBody)
		}
		logger.Warnf("[OpenAI] Embeddings error (status %d): %s", resp.StatusCode, errMsg)
		return nil, fmt.Errorf("openai upstream error (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	return &embResp, nil
}

// CallGeminiEmbedding dispatches an embedding request to Google / Gemini / Antigravity.
func CallGeminiEmbedding(ctx context.Context, account *config.Account, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	token := ""
	proxyURL := ""
	if account != nil {
		token = account.AccessToken
		proxyURL = account.ProxyURL
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}
	if token == "" {
		return nil, fmt.Errorf("gemini: access token or GEMINI_API_KEY is not configured")
	}

	inputs, err := req.ParseInputStrings()
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = "text-embedding-004"
	}
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}

	// Prepare batchEmbedContents request
	type ContentPart struct {
		Text string `json:"text"`
	}
	type Content struct {
		Parts []ContentPart `json:"parts"`
	}
	type EmbedReq struct {
		Model                string   `json:"model"`
		Content              Content  `json:"content"`
		OutputDimensionality *int     `json:"outputDimensionality,omitempty"`
	}
	type BatchEmbedReq struct {
		Requests []EmbedReq `json:"requests"`
	}

	dim := req.EffectiveDimensions()
	batchReq := BatchEmbedReq{
		Requests: make([]EmbedReq, len(inputs)),
	}
	for i, inp := range inputs {
		batchReq.Requests[i] = EmbedReq{
			Model: modelName,
			Content: Content{
				Parts: []ContentPart{{Text: inp}},
			},
			OutputDimensionality: dim,
		}
	}

	bodyBytes, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal batch embed request: %w", err)
	}

	// Determine endpoint URL: If token starts with AIza (Google API Key), use Generative Language v1beta.
	// Otherwise (OAuth Bearer token), use Google Cloud Code PA or Generative Language with Bearer.
	var url string
	isAPIKey := strings.HasPrefix(token, "AIza")
	if isAPIKey {
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:batchEmbedContents?key=%s", modelName, token)
	} else {
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:batchEmbedContents", modelName)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if !isAPIKey {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	httpReq.Header.Set("User-Agent", "Kiro-Go/"+config.Version)

	client := auth.GetAuthClientForProxy(proxyURL)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errObj struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errObj)
		errMsg := errObj.Error.Message
		if errMsg == "" {
			errMsg = string(respBody)
		}
		logger.Warnf("[Gemini] Embeddings error (status %d): %s", resp.StatusCode, errMsg)
		return nil, fmt.Errorf("gemini upstream error (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	var geminiResp struct {
		Embeddings []struct {
			Values []float64 `json:"values"`
		} `json:"embeddings"`
	}
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("gemini: unmarshal response: %w", err)
	}

	totalTokens := 0
	for _, inp := range inputs {
		totalTokens += len(strings.Fields(inp)) // approximate if not provided
	}

	res := &EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data:   make([]EmbeddingData, len(geminiResp.Embeddings)),
		Usage: EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}
	for i, emb := range geminiResp.Embeddings {
		res.Data[i] = EmbeddingData{
			Object:    "embedding",
			Index:     i,
			Embedding: emb.Values,
		}
	}

	return res, nil
}
