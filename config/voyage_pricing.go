package config

import (
	"sort"
	"strings"
	"time"
)

// VoyageQuotaBucket represents the per-model free token quota tracking for Voyage AI accounts.
type VoyageQuotaBucket struct {
	Model               string  `json:"model"`
	DisplayName         string  `json:"displayName,omitempty"`
	Category            string  `json:"category,omitempty"` // "embedding" | "reranker"
	UsedTokens          int64   `json:"usedTokens"`
	FreeLimitTokens     int64   `json:"freeLimitTokens"`
	RemainingFreeTokens int64   `json:"remainingFreeTokens"`
	UsedPercent         float64 `json:"usedPercent"` // 0.0 - 100.0%
	CostUSD             float64 `json:"costUsd"`     // Charged cost after free tokens
	PricePerMillion     float64 `json:"pricePerMillion"`
	IsFreeExhausted     bool    `json:"isFreeExhausted"`
}

// VoyageModelPriceInfo contains pricing and free tier limit for a Voyage model according to https://docs.voyageai.com/docs/pricing
type VoyageModelPriceInfo struct {
	Model           string
	DisplayName     string
	Category        string
	FreeLimitTokens int64
	PricePerMillion float64
}

// VoyageModelPricing defines the official pricing and free tier token limit per model.
var VoyageModelPricing = map[string]VoyageModelPriceInfo{
	// 200M Free Tokens Text Embeddings
	"voyage-4-large":        {Model: "voyage-4-large", DisplayName: "Voyage 4 Large", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.12},
	"voyage-4":              {Model: "voyage-4", DisplayName: "Voyage 4", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.06},
	"voyage-4-lite":         {Model: "voyage-4-lite", DisplayName: "Voyage 4 Lite", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.02},
	"voyage-code-4":         {Model: "voyage-code-4", DisplayName: "Voyage Code 4", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.12},
	"voyage-context-4":      {Model: "voyage-context-4", DisplayName: "Voyage Context 4", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.12},
	"voyage-4-nano":         {Model: "voyage-4-nano", DisplayName: "Voyage 4 Nano", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.02},

	// 50M Free Tokens Text Embeddings
	"voyage-finance-2":      {Model: "voyage-finance-2", DisplayName: "Voyage Finance 2", Category: "embedding", FreeLimitTokens: 50_000_000, PricePerMillion: 0.12},
	"voyage-law-2":          {Model: "voyage-law-2", DisplayName: "Voyage Law 2", Category: "embedding", FreeLimitTokens: 50_000_000, PricePerMillion: 0.12},
	"voyage-code-2":         {Model: "voyage-code-2", DisplayName: "Voyage Code 2", Category: "embedding", FreeLimitTokens: 50_000_000, PricePerMillion: 0.12},
	"voyage-multilingual-2": {Model: "voyage-multilingual-2", DisplayName: "Voyage Multilingual 2", Category: "embedding", FreeLimitTokens: 50_000_000, PricePerMillion: 0.12},

	// Multimodal Embeddings (200M free text tokens)
	"voyage-multimodal-3.5": {Model: "voyage-multimodal-3.5", DisplayName: "Voyage Multimodal 3.5", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.12},
	"voyage-multimodal-3":   {Model: "voyage-multimodal-3", DisplayName: "Voyage Multimodal 3", Category: "embedding", FreeLimitTokens: 200_000_000, PricePerMillion: 0.12},

	// 200M Free Tokens Rerankers
	"rerank-2.5":            {Model: "rerank-2.5", DisplayName: "Voyage Rerank 2.5", Category: "reranker", FreeLimitTokens: 200_000_000, PricePerMillion: 0.05},
	"rerank-2.5-lite":       {Model: "rerank-2.5-lite", DisplayName: "Voyage Rerank 2.5 Lite", Category: "reranker", FreeLimitTokens: 200_000_000, PricePerMillion: 0.02},
	"rerank-2":              {Model: "rerank-2", DisplayName: "Voyage Rerank 2", Category: "reranker", FreeLimitTokens: 200_000_000, PricePerMillion: 0.05},
	"rerank-2-lite":         {Model: "rerank-2-lite", DisplayName: "Voyage Rerank 2 Lite", Category: "reranker", FreeLimitTokens: 200_000_000, PricePerMillion: 0.02},

	// Older models (No free tokens)
	"voyage-3-large":        {Model: "voyage-3-large", DisplayName: "Voyage 3 Large", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.18},
	"voyage-3.5":            {Model: "voyage-3.5", DisplayName: "Voyage 3.5", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.06},
	"voyage-3.5-lite":       {Model: "voyage-3.5-lite", DisplayName: "Voyage 3.5 Lite", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.02},
	"voyage-3":              {Model: "voyage-3", DisplayName: "Voyage 3", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.06},
	"voyage-3-lite":         {Model: "voyage-3-lite", DisplayName: "Voyage 3 Lite", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.02},
	"voyage-code-3":         {Model: "voyage-code-3", DisplayName: "Voyage Code 3", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.18},
	"voyage-context-3":      {Model: "voyage-context-3", DisplayName: "Voyage Context 3", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.18},
	"voyage-large-2-instruct": {Model: "voyage-large-2-instruct", DisplayName: "Voyage Large 2 Instruct", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.12},
	"voyage-large-2":        {Model: "voyage-large-2", DisplayName: "Voyage Large 2", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.12},
	"voyage-2":              {Model: "voyage-2", DisplayName: "Voyage 2", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.10},
	"voyage-01":             {Model: "voyage-01", DisplayName: "Voyage 01", Category: "embedding", FreeLimitTokens: 0, PricePerMillion: 0.10},
	"rerank-1":              {Model: "rerank-1", DisplayName: "Voyage Rerank 1", Category: "reranker", FreeLimitTokens: 0, PricePerMillion: 0.05},
	"rerank-lite-1":         {Model: "rerank-lite-1", DisplayName: "Voyage Rerank Lite 1", Category: "reranker", FreeLimitTokens: 0, PricePerMillion: 0.02},
}

// NormalizeVoyageModel returns lowercase trimmed model name.
func NormalizeVoyageModel(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

// GetVoyageModelPriceInfo returns pricing info for a model, with defaults if unknown.
func GetVoyageModelPriceInfo(model string) VoyageModelPriceInfo {
	m := NormalizeVoyageModel(model)
	if info, ok := VoyageModelPricing[m]; ok {
		return info
	}
	category := "embedding"
	if strings.HasPrefix(m, "rerank") {
		category = "reranker"
	}
	return VoyageModelPriceInfo{
		Model:           model,
		DisplayName:     model,
		Category:        category,
		FreeLimitTokens: 0,
		PricePerMillion: 0.12,
	}
}

// CalculateVoyageBuckets computes the quota buckets for all configured Voyage models given token usage map.
func CalculateVoyageBuckets(usage map[string]int64) []VoyageQuotaBucket {
	// Include all defined Voyage pricing models, plus any custom models found in usage
	modelSet := make(map[string]bool)
	for k := range VoyageModelPricing {
		modelSet[k] = true
	}
	for k := range usage {
		modelSet[NormalizeVoyageModel(k)] = true
	}

	buckets := make([]VoyageQuotaBucket, 0, len(modelSet))

	for m := range modelSet {
		info := GetVoyageModelPriceInfo(m)
		usedTokens := int64(0)
		if usage != nil {
			if u, ok := usage[m]; ok {
				usedTokens = u
			}
		}

		var remainingFree int64
		var usedPercent float64
		var costUSD float64
		var isExhausted bool

		if info.FreeLimitTokens > 0 {
			if usedTokens <= info.FreeLimitTokens {
				remainingFree = info.FreeLimitTokens - usedTokens
				usedPercent = float64(usedTokens) / float64(info.FreeLimitTokens) * 100.0
				costUSD = 0
				isExhausted = false
			} else {
				remainingFree = 0
				usedPercent = 100.0
				excess := usedTokens - info.FreeLimitTokens
				costUSD = float64(excess) * info.PricePerMillion / 1_000_000.0
				isExhausted = true
			}
		} else {
			remainingFree = 0
			usedPercent = 100.0
			costUSD = float64(usedTokens) * info.PricePerMillion / 1_000_000.0
			isExhausted = true
		}

		buckets = append(buckets, VoyageQuotaBucket{
			Model:               info.Model,
			DisplayName:         info.DisplayName,
			Category:            info.Category,
			UsedTokens:          usedTokens,
			FreeLimitTokens:     info.FreeLimitTokens,
			RemainingFreeTokens: remainingFree,
			UsedPercent:         usedPercent,
			CostUSD:             costUSD,
			PricePerMillion:     info.PricePerMillion,
			IsFreeExhausted:     isExhausted,
		})
	}

	// Sort buckets: active models with usage first, then by category, then by name
	sort.Slice(buckets, func(i, j int) bool {
		if (buckets[i].UsedTokens > 0) != (buckets[j].UsedTokens > 0) {
			return buckets[i].UsedTokens > buckets[j].UsedTokens
		}
		if buckets[i].Category != buckets[j].Category {
			return buckets[i].Category < buckets[j].Category
		}
		return buckets[i].Model < buckets[j].Model
	})

	return buckets
}

// AddVoyageAccountUsage records token usage for a Voyage AI account and persists it.
func AddVoyageAccountUsage(id string, model string, tokens int) error {
	if tokens <= 0 {
		return nil
	}

	cfgLock.Lock()
	defer cfgLock.Unlock()

	normModel := NormalizeVoyageModel(model)

	for i, a := range cfg.Accounts {
		if a.ID == id {
			if cfg.Accounts[i].VoyageUsage == nil {
				cfg.Accounts[i].VoyageUsage = make(map[string]int64)
			}
			cfg.Accounts[i].VoyageUsage[normModel] += int64(tokens)

			// Recompute quota buckets
			buckets := CalculateVoyageBuckets(cfg.Accounts[i].VoyageUsage)
			cfg.Accounts[i].VoyageQuota = buckets

			// Compute total tokens and total credits / cost across all models
			totalUsedTokens := int64(0)
			totalCost := float64(0)
			totalFreeLimit := int64(0)

			for _, b := range buckets {
				totalUsedTokens += b.UsedTokens
				totalCost += b.CostUSD
				totalFreeLimit += b.FreeLimitTokens
			}

			cfg.Accounts[i].TotalTokens = int(totalUsedTokens)
			cfg.Accounts[i].TotalCredits = totalCost
			cfg.Accounts[i].LastUsed = time.Now().Unix()

			// Expose usage in millions of tokens for standard UI meters
			cfg.Accounts[i].UsageCurrent = float64(totalUsedTokens) / 1_000_000.0
			cfg.Accounts[i].UsageLimit = float64(totalFreeLimit) / 1_000_000.0
			if cfg.Accounts[i].UsageLimit > 0 {
				cfg.Accounts[i].UsagePercent = cfg.Accounts[i].UsageCurrent / cfg.Accounts[i].UsageLimit
			}

			markDirty()
			return nil
		}
	}
	return nil
}
