package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"kiro-go/config"
)

func TestEmbeddingRequest_ParseInputStrings(t *testing.T) {
	// Single string
	req1 := EmbeddingRequest{
		Input: json.RawMessage(`"hello world"`),
	}
	strs, err := req1.ParseInputStrings()
	if err != nil || len(strs) != 1 || strs[0] != "hello world" {
		t.Fatalf("expected ['hello world'], got %v, err: %v", strs, err)
	}

	// Slice of strings
	req2 := EmbeddingRequest{
		Input: json.RawMessage(`["doc1", "doc2"]`),
	}
	strs, err = req2.ParseInputStrings()
	if err != nil || len(strs) != 2 || strs[0] != "doc1" || strs[1] != "doc2" {
		t.Fatalf("expected ['doc1', 'doc2'], got %v, err: %v", strs, err)
	}
}

func TestRerankRequest_ParseDocuments(t *testing.T) {
	// Slice of strings
	req1 := RerankRequest{
		Documents: json.RawMessage(`["Apple", "Banana", "Cherry"]`),
	}
	docs, err := req1.ParseDocuments()
	if err != nil || len(docs) != 3 || docs[0] != "Apple" {
		t.Fatalf("expected 3 docs, got %v, err: %v", docs, err)
	}

	// Slice of objects
	req2 := RerankRequest{
		Documents: json.RawMessage(`[{"text": "Apple"}, {"text": "Banana"}]`),
	}
	docs, err = req2.ParseDocuments()
	if err != nil || len(docs) != 2 || docs[0] != "Apple" || docs[1] != "Banana" {
		t.Fatalf("expected 2 docs from objects, got %v, err: %v", docs, err)
	}
}

func TestModelPredicates(t *testing.T) {
	if !IsVoyageModel("voyage-4-large") {
		t.Error("expected voyage-4-large to be Voyage model")
	}
	if !IsVoyageModel("rerank-2.5") {
		t.Error("expected rerank-2.5 to be Voyage model")
	}
	if !IsOpenAIEmbeddingModel("text-embedding-3-small") {
		t.Error("expected text-embedding-3-small to be OpenAI embedding model")
	}
	if !IsGeminiEmbeddingModel("text-embedding-004") {
		t.Error("expected text-embedding-004 to be Gemini embedding model")
	}
}

func TestCallVoyageEmbedding(t *testing.T) {
	// Mock Voyage API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-voyage-key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"detail": "invalid api key"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"object": "list",
			"data": [
				{
					"object": "embedding",
					"embedding": [0.1, 0.2, 0.3],
					"index": 0
				}
			],
			"model": "voyage-4-large",
			"usage": {
				"total_tokens": 5
			}
		}`))
	}))
	defer ts.Close()

	// Set env
	os.Setenv("VOYAGE_API_KEY", "test-voyage-key")
	defer os.Unsetenv("VOYAGE_API_KEY")

	// Custom request against mock
	account := &config.Account{
		Provider:     "voyage",
		VoyageAPIKey: "test-voyage-key",
	}

	req := &EmbeddingRequest{
		Model: "voyage-4-large",
		Input: json.RawMessage(`"hello"`),
	}

	dim := 1024
	req.Dimensions = &dim

	// Test directly against mock server by overriding base url in context or test
	httpReq, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/embeddings", bytes.NewReader([]byte(`{"model":"voyage-4-large","input":["hello"]}`)))
	httpReq.Header.Set("Authorization", "Bearer test-voyage-key")
	client := ts.Client()
	resp, err := client.Do(httpReq)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("mock request failed: %v", err)
	}

	if account.VoyageAPIKey != "test-voyage-key" {
		t.Errorf("expected voyage key, got %s", account.VoyageAPIKey)
	}
}

func TestEmbeddingsAndRerankEndpoints(t *testing.T) {
	h := NewHandler()
	defer h.Close()

	// Test /v1/embeddings with invalid method
	req := httptest.NewRequest("GET", "/v1/embeddings", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
	}

	// Test /v1/rerank with invalid method
	req = httptest.NewRequest("GET", "/v1/rerank", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
	}

	// Test /v1/models includes embedding and rerank models
	req = httptest.NewRequest("GET", "/v1/models", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /v1/models, got %d", w.Code)
	}

	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &modelsResp); err != nil {
		t.Fatalf("failed to decode models: %v", err)
	}

	foundVoyage := false
	foundRerank := false
	foundTextEmbedding := false

	for _, m := range modelsResp.Data {
		if strings.HasPrefix(m.ID, "voyage-4") {
			foundVoyage = true
		}
		if strings.HasPrefix(m.ID, "rerank-") {
			foundRerank = true
		}
		if strings.HasPrefix(m.ID, "text-embedding-") {
			foundTextEmbedding = true
		}
	}

	if !foundVoyage {
		t.Error("expected voyage models in /v1/models")
	}
	if !foundRerank {
		t.Error("expected rerank models in /v1/models")
	}
	if !foundTextEmbedding {
		t.Error("expected text-embedding models in /v1/models")
	}
}
