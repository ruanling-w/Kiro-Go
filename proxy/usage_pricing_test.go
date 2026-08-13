package proxy

import (
	"math"
	"testing"

	"kiro-go/store"
)

func TestEstimateUsageCostSeparatesCoverage(t *testing.T) {
	got := estimateUsageCost([]store.UsageRollup{
		{
			Provider: "kiro", Model: "claude-sonnet-4.6-thinking",
			InputTokens: 100, OutputTokens: 50,
			CacheReadTokens: 200, CacheCreationTokens: 30,
		},
		{Provider: "unknown", Model: "custom", InputTokens: 40, OutputTokens: 10},
		{Provider: "", Model: "", LegacyTokens: 20},
	})
	wantCost := (100*3.0 + 50*15.0 + 200*.3 + 30*3.75) / 1_000_000
	if math.Abs(got.EstimatedCostUSD-wantCost) > 1e-12 {
		t.Fatalf("cost=%g want=%g", got.EstimatedCostUSD, wantCost)
	}
	if got.PricedTokens != 380 || got.UnpricedTokens != 50 || got.UnpricedLegacy != 20 {
		t.Fatalf("coverage counts=%+v", got)
	}
	if got.PricingComplete {
		t.Fatal("partial estimate reported complete")
	}
	if math.Abs(got.PricingCoverage-float64(380)/450) > 1e-12 {
		t.Fatalf("coverage=%g", got.PricingCoverage)
	}
}

func TestPriceForUsageDoesNotGuessRemoteProvider(t *testing.T) {
	if _, ok := priceForUsage("remotekiro", "claude-sonnet-4.6"); ok {
		t.Fatal("remote provider must remain unpriced without a verified override")
	}
	if _, ok := priceForUsage("kiro", "unknown-model"); ok {
		t.Fatal("unknown model must remain unpriced")
	}
	if _, ok := priceForUsage("kiro", "claude-opus-4-8-thinking"); !ok {
		t.Fatal("controlled Kiro model alias was not priced")
	}
}
