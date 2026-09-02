package config

import (
	"testing"
)

func TestVoyagePricingAndBuckets(t *testing.T) {
	usage := map[string]int64{
		"voyage-4-large":   50_000_000,   // within 200M free limit
		"voyage-finance-2": 60_000_000,   // exceeded 50M free limit by 10M
		"voyage-3.5":       10_000_000,   // 0 free limit, all 10M charged
	}

	buckets := CalculateVoyageBuckets(usage)
	if len(buckets) == 0 {
		t.Fatal("expected non-empty voyage buckets")
	}

	var bucketV4, bucketFin, bucketV35 *VoyageQuotaBucket
	for i := range buckets {
		switch buckets[i].Model {
		case "voyage-4-large":
			bucketV4 = &buckets[i]
		case "voyage-finance-2":
			bucketFin = &buckets[i]
		case "voyage-3.5":
			bucketV35 = &buckets[i]
		}
	}

	if bucketV4 == nil || bucketFin == nil || bucketV35 == nil {
		t.Fatalf("expected to find all test buckets")
	}

	// Test voyage-4-large
	if bucketV4.FreeLimitTokens != 200_000_000 {
		t.Errorf("expected 200M free tokens, got %d", bucketV4.FreeLimitTokens)
	}
	if bucketV4.RemainingFreeTokens != 150_000_000 {
		t.Errorf("expected 150M remaining free tokens, got %d", bucketV4.RemainingFreeTokens)
	}
	if bucketV4.CostUSD != 0 {
		t.Errorf("expected cost 0 within free tier, got %f", bucketV4.CostUSD)
	}
	if bucketV4.IsFreeExhausted {
		t.Errorf("expected isFreeExhausted false")
	}
	if bucketV4.UsedPercent != 25.0 {
		t.Errorf("expected used percent 25.0%%, got %f", bucketV4.UsedPercent)
	}

	// Test voyage-finance-2
	if bucketFin.FreeLimitTokens != 50_000_000 {
		t.Errorf("expected 50M free tokens, got %d", bucketFin.FreeLimitTokens)
	}
	if bucketFin.RemainingFreeTokens != 0 {
		t.Errorf("expected 0 remaining free tokens, got %d", bucketFin.RemainingFreeTokens)
	}
	// 10M excess tokens * $0.12 / 1M = $1.20
	if bucketFin.CostUSD != 1.20 {
		t.Errorf("expected cost $1.20, got %f", bucketFin.CostUSD)
	}
	if !bucketFin.IsFreeExhausted {
		t.Errorf("expected isFreeExhausted true")
	}

	// Test voyage-3.5 (no free limit)
	if bucketV35.FreeLimitTokens != 0 {
		t.Errorf("expected 0 free limit tokens, got %d", bucketV35.FreeLimitTokens)
	}
	// 10M tokens * $0.06 / 1M = $0.60
	if bucketV35.CostUSD != 0.60 {
		t.Errorf("expected cost $0.60, got %f", bucketV35.CostUSD)
	}
}

func TestAddVoyageAccountUsage(t *testing.T) {
	cfgLock.Lock()
	testAcc := Account{
		ID:           "test-voyage-acc-1",
		Email:        "voyage@test.com",
		Provider:     "voyage",
		VoyageAPIKey: "va-test-123",
	}
	cfg.Accounts = append(cfg.Accounts, testAcc)
	cfgLock.Unlock()

	err := AddVoyageAccountUsage("test-voyage-acc-1", "voyage-4-large", 100_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	accs := GetAccounts()
	var updated *Account
	for _, a := range accs {
		if a.ID == "test-voyage-acc-1" {
			updated = &a
			break
		}
	}

	if updated == nil {
		t.Fatal("account not found")
	}
	if updated.VoyageUsage["voyage-4-large"] != 100_000 {
		t.Errorf("expected 100000 used tokens, got %d", updated.VoyageUsage["voyage-4-large"])
	}
	if len(updated.VoyageQuota) == 0 {
		t.Errorf("expected calculated voyageQuota buckets")
	}
}
