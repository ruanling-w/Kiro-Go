package pool

import (
	"errors"
	"kiro-go/config"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOverLimitAccountsAreSkippedByDefault(t *testing.T) {
	p := &AccountPool{}
	normal := config.Account{ID: "normal"}
	overLimit := config.Account{ID: "over", UsageCurrent: 10, UsageLimit: 10}

	p.accounts = []config.Account{normal, overLimit}

	for i := 0; i < 5; i++ {
		acc := p.GetNext()
		if acc == nil {
			t.Fatalf("expected an account")
		}
		if acc.ID == "over" {
			t.Fatalf("expected over-limit account to be skipped when upstream OverageStatus is empty")
		}
	}
}

func TestOverLimitAccountsCanBeSelectedWhenUpstreamOverageEnabled(t *testing.T) {
	p := &AccountPool{}
	overLimit := config.Account{
		ID:            "over",
		UsageCurrent:  10,
		UsageLimit:    10,
		OverageStatus: "ENABLED",
	}

	p.accounts = []config.Account{overLimit}

	acc := p.GetNext()
	if acc == nil {
		t.Fatalf("expected upstream-enabled overage account to be selectable")
	}
	if acc.ID != "over" {
		t.Fatalf("expected overage account, got %q", acc.ID)
	}
}

func TestOverLimitAccountsRemainSkippedWhenUpstreamOverageDisabled(t *testing.T) {
	p := &AccountPool{}
	overLimit := config.Account{
		ID:            "over",
		UsageCurrent:  10,
		UsageLimit:    10,
		OverageStatus: "DISABLED",
	}

	p.accounts = []config.Account{overLimit}

	if acc := p.GetNext(); acc != nil {
		t.Fatalf("expected nil when upstream OverageStatus=DISABLED, got %q", acc.ID)
	}
}

func TestGetNextKeepsFiveMinuteTokenAvailable(t *testing.T) {
	p := &AccountPool{}
	account := config.Account{
		ID:          "acct-1",
		AccessToken: "access-token",
		ExpiresAt:   time.Now().Unix() + 300,
	}

	p.accounts = []config.Account{account}

	got := p.GetNext()
	if got == nil {
		t.Fatalf("expected five-minute token to be available")
	}
	if got.ID != account.ID {
		t.Fatalf("expected account %q, got %q", account.ID, got.ID)
	}
}

// ---------------------------------------------------------------------------
// IsAuthFailure
// ---------------------------------------------------------------------------

func TestIsAuthFailureRecognizes401And403(t *testing.T) {
	positives := []string{
		"HTTP 401 from server",
		"received 403 Forbidden",
		"bad credentials",
		"invalid_grant",
		"invalid_token",
		"token expired",
		"token has expired",
		"unauthorized",
	}
	for _, msg := range positives {
		if !IsAuthFailure(errors.New(msg)) {
			t.Errorf("IsAuthFailure(%q) = false, want true", msg)
		}
	}
}

func TestIsAuthFailureIgnoresFalsePositives(t *testing.T) {
	// hasStatusToken only excludes digit boundaries; e.g. "4011" contains "401"
	// but the trailing '1' is a digit so it does NOT match.
	negatives := []string{
		"status code 4011 found", // digit immediately after 401 → not a standalone token
		"error 14013 exceeded",   // digit before and after 401
		"some random error",
		"status 200 OK",
	}
	for _, msg := range negatives {
		if IsAuthFailure(errors.New(msg)) {
			t.Errorf("IsAuthFailure(%q) = true, want false", msg)
		}
	}
}

func TestIsAuthFailureNilError(t *testing.T) {
	if IsAuthFailure(nil) {
		t.Fatal("IsAuthFailure(nil) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// IsSuspensionError
// ---------------------------------------------------------------------------

func TestIsSuspensionErrorDetectsKnownMessages(t *testing.T) {
	positives := []string{
		"account temporarily_suspended",
		"account temporarily suspended",
		"no available kiro profile",
		"No Available Kiro Profile", // case-insensitive
	}
	for _, msg := range positives {
		if !IsSuspensionError(errors.New(msg)) {
			t.Errorf("IsSuspensionError(%q) = false, want true", msg)
		}
	}
}

func TestIsSuspensionErrorIgnoresUnrelatedErrors(t *testing.T) {
	negatives := []string{
		"some other error",
		"unauthorized",
		"429 too many requests",
	}
	for _, msg := range negatives {
		if IsSuspensionError(errors.New(msg)) {
			t.Errorf("IsSuspensionError(%q) = true, want false", msg)
		}
	}
}

func TestIsSuspensionErrorNilError(t *testing.T) {
	if IsSuspensionError(nil) {
		t.Fatal("IsSuspensionError(nil) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// GetNextForModelExcluding
// ---------------------------------------------------------------------------

func newTestPool(accounts ...config.Account) *AccountPool {
	p := &AccountPool{
		cooldowns:   make(map[string]time.Time),
		errorCounts: make(map[string]int),
		modelLists:  make(map[string]map[string]bool),
	}
	p.accounts = accounts
	return p
}

func TestGetNextForModelExcludingSkipsExcludedAccounts(t *testing.T) {
	p := newTestPool(
		config.Account{ID: "a"},
		config.Account{ID: "b"},
	)
	excluded := map[string]bool{"a": true}
	for i := 0; i < 5; i++ {
		acc := p.GetNextForModelExcluding("model", excluded)
		if acc == nil {
			t.Fatal("expected account b, got nil")
		}
		if acc.ID == "a" {
			t.Fatalf("excluded account a was returned on iteration %d", i)
		}
	}
}

func TestGetNextForModelExcludingReturnsNilWhenAllExcluded(t *testing.T) {
	p := newTestPool(config.Account{ID: "only"})
	acc := p.GetNextForModelExcluding("model", map[string]bool{"only": true})
	if acc != nil {
		t.Fatalf("expected nil when only account is excluded, got %q", acc.ID)
	}
}

func TestGetNextForModelExcludingReturnsNilOnEmptyPool(t *testing.T) {
	p := newTestPool()
	acc := p.GetNextForModelExcluding("model", map[string]bool{})
	if acc != nil {
		t.Fatalf("expected nil for empty pool, got %q", acc.ID)
	}
}

func TestGetNextForModelExcludingHonorsProviderQualifiedModel(t *testing.T) {
	p := newTestPool(
		config.Account{ID: "kiro", Provider: "kiro"},
		config.Account{ID: "grok", Provider: "grok"},
	)
	p.SetModelList("kiro", []string{"shared-model"})
	p.SetModelList("grok", []string{"shared-model"})

	for i := 0; i < 5; i++ {
		acc := p.GetNextForModelExcluding("grok::shared-model", nil)
		if acc == nil || acc.ID != "grok" {
			t.Fatalf("qualified model chose %#v, want grok account", acc)
		}
	}
}

func TestGetNextForModelExcludingRejectsMalformedProviderQualifiedModel(t *testing.T) {
	p := newTestPool(config.Account{ID: "grok", Provider: "grok"})
	if acc := p.GetNextForModelExcluding("grok::", nil); acc != nil {
		t.Fatalf("malformed candidate chose %#v", acc)
	}
}

// ---------------------------------------------------------------------------
// DisableAccount
// ---------------------------------------------------------------------------

func TestDisableAccountSetsCooldown(t *testing.T) {
	// Initialize a temporary config so SetAccountBanStatus can persist safely.
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	p := newTestPool()
	p.DisableAccount("test-id", "test reason")

	p.mu.RLock()
	cooldown, ok := p.cooldowns["test-id"]
	p.mu.RUnlock()

	if !ok {
		t.Fatal("expected cooldown to be set after DisableAccount")
	}
	// Safety-net cooldown must be at least 23 hours from now.
	minExpected := time.Now().Add(23 * time.Hour)
	if cooldown.Before(minExpected) {
		t.Fatalf("expected cooldown >= 23h in future, got %v", cooldown)
	}
}

func TestGetNextExcludingSkipsExcludedAccount(t *testing.T) {
	p := &AccountPool{
		accounts: []config.Account{
			{ID: "a", Enabled: true},
			{ID: "b", Enabled: true},
		},
		cooldowns:    make(map[string]time.Time),
		errorCounts:  make(map[string]int),
		modelLists:   make(map[string]map[string]bool),
		currentIndex: ^uint64(0),
	}

	acc := p.GetNextExcluding(map[string]bool{"a": true})
	if acc == nil || acc.ID != "b" {
		t.Fatalf("expected account b, got %#v", acc)
	}
}

func TestGetNextForModelExcludingSkipsExcludedAccount(t *testing.T) {
	p := &AccountPool{
		accounts: []config.Account{
			{ID: "a", Enabled: true},
			{ID: "b", Enabled: true},
		},
		cooldowns:    make(map[string]time.Time),
		errorCounts:  make(map[string]int),
		modelLists:   make(map[string]map[string]bool),
		currentIndex: ^uint64(0),
	}
	p.SetModelList("a", []string{"claude-sonnet-4.5"})
	p.SetModelList("b", []string{"claude-sonnet-4.5"})

	acc := p.GetNextForModelExcluding("claude-sonnet-4.5", map[string]bool{"a": true})
	if acc == nil || acc.ID != "b" {
		t.Fatalf("expected account b, got %#v", acc)
	}
}

// ---------------------------------------------------------------------------
// Reload over-usage filtering
// ---------------------------------------------------------------------------

func TestReloadKeepsOverQuotaAccountWhenAllowOverUsage(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.AddAccount(config.Account{
		ID:           "over",
		Enabled:      true,
		UsageCurrent: 10,
		UsageLimit:   10,
	}); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	if err := config.UpdateAllowOverUsage(true); err != nil {
		t.Fatalf("UpdateAllowOverUsage: %v", err)
	}

	p := newTestPool()
	p.Reload()

	if got := p.GetNext(); got == nil || got.ID != "over" {
		t.Fatalf("expected over-quota account to remain routable when allowOverUsage=true, got %#v", got)
	}
}

func TestReloadDropsOverQuotaAccountWhenAllowOverUsageDisabled(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.AddAccount(config.Account{
		ID:           "over",
		Enabled:      true,
		UsageCurrent: 10,
		UsageLimit:   10,
	}); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}

	p := newTestPool()
	p.Reload()

	if got := p.GetNext(); got != nil {
		t.Fatalf("expected over-quota account to be dropped, got %q", got.ID)
	}
}

// UnavailableReason is the operator-facing half of a 503. Each distinct cause
// must be named, because "No available accounts" alone cannot distinguish a
// drained pool from a model nobody advertises.
func TestUnavailableReasonNamesTheCause(t *testing.T) {
	t.Run("no accounts configured", func(t *testing.T) {
		p := &AccountPool{cooldowns: map[string]time.Time{}, modelLists: map[string]map[string]bool{}}
		if got := p.UnavailableReason("m", nil); !strings.Contains(got, "no accounts configured") {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("all disabled leaves an empty weighted list", func(t *testing.T) {
		p := &AccountPool{cooldowns: map[string]time.Time{}, modelLists: map[string]map[string]bool{}}
		p.totalAccounts = 3
		if got := p.UnavailableReason("m", nil); !strings.Contains(got, "all 3 configured accounts are disabled") {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("cooldown reports the soonest expiry", func(t *testing.T) {
		p := &AccountPool{cooldowns: map[string]time.Time{}, modelLists: map[string]map[string]bool{}}
		p.accounts = []config.Account{{ID: "a"}, {ID: "b"}}
		p.totalAccounts = 2
		p.cooldowns["a"] = time.Now().Add(30 * time.Second)
		p.cooldowns["b"] = time.Now().Add(10 * time.Minute)

		got := p.UnavailableReason("m", nil)
		if !strings.Contains(got, "2 enabled accounts") || !strings.Contains(got, "2 in error cooldown") {
			t.Fatalf("got %q", got)
		}
		if !strings.Contains(got, "soonest expires in") {
			t.Fatalf("expected a soonest-expiry hint, got %q", got)
		}
	})

	t.Run("model filter is distinguished from cooldown", func(t *testing.T) {
		p := &AccountPool{cooldowns: map[string]time.Time{}, modelLists: map[string]map[string]bool{}}
		p.accounts = []config.Account{{ID: "a"}, {ID: "b"}}
		p.totalAccounts = 2
		// Both have a non-empty catalog that omits the requested model, so
		// accountHasModel is strict rather than optimistic.
		p.modelLists["a"] = map[string]bool{"other-model": true}
		p.modelLists["b"] = map[string]bool{"other-model": true}

		got := p.UnavailableReason("wanted-model", nil)
		if !strings.Contains(got, `2 do not list model "wanted-model"`) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("quota and exclusion are reported separately", func(t *testing.T) {
		p := &AccountPool{cooldowns: map[string]time.Time{}, modelLists: map[string]map[string]bool{}}
		p.accounts = []config.Account{
			{ID: "tried"},
			{ID: "over", UsageCurrent: 10, UsageLimit: 10},
		}
		p.totalAccounts = 2

		got := p.UnavailableReason("", map[string]bool{"tried": true})
		if !strings.Contains(got, "1 already tried this request") {
			t.Fatalf("got %q", got)
		}
		if !strings.Contains(got, "1 over quota") {
			t.Fatalf("got %q", got)
		}
	})
}

func TestNamespacedAccountRouting(t *testing.T) {
	p := &AccountPool{
		accounts: []config.Account{
			{ID: "acc-bai", Nickname: "bai", Provider: "remotekiro", CustomModels: []string{"gpt-5.6-sol", "deepseek-r1"}},
			{ID: "acc-coe", Nickname: "coe", Provider: "codex", Email: "einnam20@gmail.com", CustomModels: []string{"gpt-5.6-sol", "claude-sonnet-4.5"}},
			{ID: "acc-ag", Nickname: "ag1", Provider: "antigravity", CustomModels: []string{"claude-sonnet-4.5"}},
		},
		cooldowns:  map[string]time.Time{},
		modelLists: map[string]map[string]bool{},
	}
	p.totalAccounts = len(p.accounts)

	// 1. Direct slash namespace: "bai/gpt-5.6-sol" -> must route to acc-bai
	acc := p.GetNextForModel("bai/gpt-5.6-sol")
	if acc == nil || acc.ID != "acc-bai" {
		t.Fatalf("expected acc-bai, got %+v", acc)
	}

	// 2. Direct slash namespace: "coe/gpt-5.6-sol" -> must route to acc-coe
	acc = p.GetNextForModel("coe/gpt-5.6-sol")
	if acc == nil || acc.ID != "acc-coe" {
		t.Fatalf("expected acc-coe, got %+v", acc)
	}

	// 3. Double colon namespace: "bai::deepseek-r1" -> must route to acc-bai
	acc = p.GetNextForModel("bai::deepseek-r1")
	if acc == nil || acc.ID != "acc-bai" {
		t.Fatalf("expected acc-bai, got %+v", acc)
	}

	// 4. Double colon with email: "einnam20@gmail.com::gpt-5.6-sol" -> must route to acc-coe
	acc = p.GetNextForModel("einnam20@gmail.com::gpt-5.6-sol")
	if acc == nil || acc.ID != "acc-coe" {
		t.Fatalf("expected acc-coe, got %+v", acc)
	}

	// 5. Provider routing: "antigravity::claude-sonnet-4.5" -> must route to acc-ag
	acc = p.GetNextForModel("antigravity::claude-sonnet-4.5")
	if acc == nil || acc.ID != "acc-ag" {
		t.Fatalf("expected acc-ag, got %+v", acc)
	}

	// 6. Strict custom models: "bai/claude-sonnet-4.5" -> acc-bai doesn't have it -> returns nil
	acc = p.GetNextForModel("bai/claude-sonnet-4.5")
	if acc != nil {
		t.Fatalf("expected nil for unsupported model on bai, got %+v", acc)
	}

	// 7. Non-existent namespace with slash model name: "meta-llama/llama-3" -> fallback checks model name
	p.accounts[0].CustomModels = append(p.accounts[0].CustomModels, "meta-llama/llama-3")
	acc = p.GetNextForModel("meta-llama/llama-3")
	if acc == nil || acc.ID != "acc-bai" {
		t.Fatalf("expected acc-bai for meta-llama/llama-3, got %+v", acc)
	}
}

