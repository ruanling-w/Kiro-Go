package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EmbeddingRequest is the standard OpenAI/Voyage-compatible embedding request payload.
type EmbeddingRequest struct {
	Model          string          `json:"model"`
	Input          json.RawMessage `json:"input"` // string or []string or []int or [][]int
	Dimensions     *int            `json:"dimensions,omitempty"`
	OutputDim      *int            `json:"output_dimension,omitempty"` // Voyage AI parameter
	InputType      string          `json:"input_type,omitempty"`       // Voyage AI: "query" | "document"
	Truncation     *bool           `json:"truncation,omitempty"`       // Voyage AI
	OutputDType    string          `json:"output_dtype,omitempty"`     // Voyage AI: "float" | "int8" | "uint8" | "binary" | "ubinary"
	EncodingFormat string          `json:"encoding_format,omitempty"` // OpenAI: "float" | "base64"
	User           string          `json:"user,omitempty"`
}

// ParseInputStrings normalizes the request input field into a slice of strings.
func (r *EmbeddingRequest) ParseInputStrings() ([]string, error) {
	if len(r.Input) == 0 {
		return nil, fmt.Errorf("input is required")
	}

	// Try single string
	var single string
	if err := json.Unmarshal(r.Input, &single); err == nil {
		return []string{single}, nil
	}

	// Try slice of strings
	var slice []string
	if err := json.Unmarshal(r.Input, &slice); err == nil {
		return slice, nil
	}

	// Try slice of arbitrary objects/interfaces (convert each to string)
	var anySlice []interface{}
	if err := json.Unmarshal(r.Input, &anySlice); err == nil {
		result := make([]string, 0, len(anySlice))
		for _, item := range anySlice {
			switch v := item.(type) {
			case string:
				result = append(result, v)
			default:
				bytes, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("invalid item in input array: %v", err)
				}
				result = append(result, string(bytes))
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("input must be a string or an array of strings")
}

// EffectiveDimensions returns dimensions or output_dimension if specified.
func (r *EmbeddingRequest) EffectiveDimensions() *int {
	if r.OutputDim != nil {
		return r.OutputDim
	}
	return r.Dimensions
}

// EmbeddingData represents a single embedding item.
type EmbeddingData struct {
	Object    string      `json:"object"`
	Index     int         `json:"index"`
	Embedding interface{} `json:"embedding"` // []float64 or []int (for quantized)
}

// EmbeddingUsage represents token usage for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResponse is the standard OpenAI/Voyage-compatible embedding response.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// RerankRequest represents a reranking request (Voyage AI / Cohere / Jina compatible).
type RerankRequest struct {
	Model           string          `json:"model"`
	Query           string          `json:"query"`
	Documents       json.RawMessage `json:"documents"` // []string or []map[string]interface{}
	TopK            *int            `json:"top_k,omitempty"`
	ReturnDocuments *bool           `json:"return_documents,omitempty"`
	Truncation      *bool           `json:"truncation,omitempty"`
}

// ParseDocuments normalizes documents to a list of strings.
func (r *RerankRequest) ParseDocuments() ([]string, error) {
	if len(r.Documents) == 0 {
		return nil, fmt.Errorf("documents is required")
	}

	// Try []string
	var strDocs []string
	if err := json.Unmarshal(r.Documents, &strDocs); err == nil {
		return strDocs, nil
	}

	// Try []map[string]interface{} (e.g. Cohere format [{"text": "..."}, ...])
	var mapDocs []map[string]interface{}
	if err := json.Unmarshal(r.Documents, &mapDocs); err == nil {
		result := make([]string, 0, len(mapDocs))
		for _, m := range mapDocs {
			if text, ok := m["text"].(string); ok {
				result = append(result, text)
			} else {
				bytes, _ := json.Marshal(m)
				result = append(result, string(bytes))
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("documents must be an array of strings or objects")
}

// RerankResult is a single reranked item.
type RerankResult struct {
	Index          int         `json:"index"`
	RelevanceScore float64     `json:"relevance_score"`
	Document       interface{} `json:"document,omitempty"`
}

// RerankUsage represents token usage for reranking.
type RerankUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// RerankResponse is the response format for reranking.
type RerankResponse struct {
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Results []RerankResult `json:"results"`
	Usage   RerankUsage    `json:"usage"`
}

// IsVoyageModel checks if a model name is a Voyage AI embedding or reranking model.
func IsVoyageModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "voyage") || strings.HasPrefix(m, "rerank")
}

// IsVoyageRerankModel checks if a model name is a Voyage AI reranking model.
func IsVoyageRerankModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "rerank")
}

// IsOpenAIEmbeddingModel checks if a model name is an OpenAI embedding model.
func IsOpenAIEmbeddingModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "text-embedding") || strings.Contains(m, "embedding-ada")
}

// IsGeminiEmbeddingModel checks if a model name is a Google / Gemini embedding model.
func IsGeminiEmbeddingModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "text-embedding-004") ||
		strings.HasPrefix(m, "embedding-001") ||
		strings.Contains(m, "gemini-embedding") ||
		strings.Contains(m, "embedding-gecko")
}
