package proxy

import (
	"strings"

	"kiro-go/store"
)

const usagePricingVersion = "9router-2026-08-13-v1"

// Rates are USD per one million tokens. This initial catalog is adapted from
// 9router/open-sse/providers/pricing.js. It is an estimate by model category;
// provider credits remain a separate unit and are never converted to USD.
type usagePrice struct {
	Input         float64
	Output        float64
	CacheRead     float64
	CacheCreation float64
}

var claudeUsagePrices = map[string]usagePrice{
	"claude-fable-5":    {Input: 10, Output: 50, CacheRead: 1, CacheCreation: 12.5},
	"claude-haiku-4.5":  {Input: 1, Output: 5, CacheRead: .1, CacheCreation: 1.25},
	"claude-opus-4.5":   {Input: 5, Output: 25, CacheRead: .5, CacheCreation: 6.25},
	"claude-opus-4.6":   {Input: 5, Output: 25, CacheRead: .5, CacheCreation: 6.25},
	"claude-opus-4.7":   {Input: 5, Output: 25, CacheRead: .5, CacheCreation: 6.25},
	"claude-opus-4.8":   {Input: 5, Output: 25, CacheRead: .5, CacheCreation: 6.25},
	"claude-opus-5":     {Input: 5, Output: 25, CacheRead: .5, CacheCreation: 6.25},
	"claude-sonnet-4":   {Input: 3, Output: 15, CacheRead: .3, CacheCreation: 3.75},
	"claude-sonnet-4.5": {Input: 3, Output: 15, CacheRead: .3, CacheCreation: 3.75},
	"claude-sonnet-4.6": {Input: 3, Output: 15, CacheRead: .3, CacheCreation: 3.75},
	"claude-sonnet-5":   {Input: 2, Output: 10, CacheRead: .2, CacheCreation: 2.5},
}

type usageCostEstimate struct {
	EstimatedCostUSD float64
	PricedTokens     int64
	UnpricedTokens   int64
	UnpricedLegacy   int64
	PricingCoverage  float64
	PricingComplete  bool
}

func normalizePricedModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimSuffix(model, "-thinking")
	model = strings.ReplaceAll(model, "-4-8", "-4.8")
	model = strings.ReplaceAll(model, "-4-7", "-4.7")
	return model
}

func priceForUsage(provider, model string) (usagePrice, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	// Kiro routes Claude models and 9router applies canonical model pricing when
	// no provider override exists. Unknown/remote providers stay unpriced.
	if provider != "kiro" && provider != "anthropic" {
		return usagePrice{}, false
	}
	price, ok := claudeUsagePrices[normalizePricedModel(model)]
	return price, ok
}

func estimateUsageCost(rows []store.UsageRollup) usageCostEstimate {
	var out usageCostEstimate
	for _, row := range rows {
		detailed := row.InputTokens + row.OutputTokens + row.CacheReadTokens + row.CacheCreationTokens
		out.UnpricedLegacy += row.LegacyTokens
		price, ok := priceForUsage(row.Provider, row.Model)
		if !ok {
			out.UnpricedTokens += detailed
			continue
		}
		out.PricedTokens += detailed
		out.EstimatedCostUSD += (float64(row.InputTokens)*price.Input +
			float64(row.OutputTokens)*price.Output +
			float64(row.CacheReadTokens)*price.CacheRead +
			float64(row.CacheCreationTokens)*price.CacheCreation) / 1_000_000
	}
	total := out.PricedTokens + out.UnpricedTokens + out.UnpricedLegacy
	if total == 0 {
		out.PricingCoverage = 1
	} else {
		out.PricingCoverage = float64(out.PricedTokens) / float64(total)
	}
	out.PricingComplete = out.UnpricedTokens == 0 && out.UnpricedLegacy == 0
	return out
}
