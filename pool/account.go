// Package pool 账号池管理
// 实现轮询负载均衡、错误冷却、Token 刷新
package pool

import (
	"fmt"
	"kiro-go/config"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const tokenRefreshSkewSeconds int64 = 120

// AccountPool 账号池
type AccountPool struct {
	mu            sync.RWMutex
	accounts      []config.Account
	totalAccounts int
	currentIndex  uint64
	cooldowns     map[string]time.Time       // 账号冷却时间
	errorCounts   map[string]int             // 连续错误计数
	modelLists    map[string]map[string]bool // accountID → set of modelIDs (from ListAvailableModels)
}

var (
	pool     *AccountPool
	poolOnce sync.Once
)

// GetPool 获取全局账号池单例
func GetPool() *AccountPool {
	poolOnce.Do(func() {
		pool = &AccountPool{
			cooldowns:   make(map[string]time.Time),
			errorCounts: make(map[string]int),
			modelLists:  make(map[string]map[string]bool),
		}
		pool.Reload()
	})
	return pool
}

// Reload rebuilds the weighted account list from config.
// Weight <= 1 → 1 entry; weight >= 2 → weight entries.
// Over-quota accounts are dropped unless either the per-account upstream
// Overages switch (OverageStatus=ENABLED) or the global AllowOverUsage
// setting permits over-quota routing.
func (p *AccountPool) Reload() {
	p.mu.Lock()
	defer p.mu.Unlock()
	enabled := config.GetEnabledAccounts()
	allowOverUsage := config.GetAllowOverUsage()
	var weighted []config.Account
	for _, a := range enabled {
		if isQuotaBlocked(a, allowOverUsage) {
			continue
		}
		w := effectiveWeight(a.Weight)
		for j := 0; j < w; j++ {
			weighted = append(weighted, a)
		}
	}
	p.accounts = weighted
	p.totalAccounts = len(enabled)
}

// GetNext 获取下一个可用账号（加权轮询）
func (p *AccountPool) GetNext() *config.Account {
	return p.GetNextExcluding(nil)
}

// GetNextExcluding 获取下一个可用账号（加权轮询），并跳过指定账号。
func (p *AccountPool) GetNextExcluding(excluded map[string]bool) *config.Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.accounts) == 0 {
		return nil
	}

	allowOverUsage := config.GetAllowOverUsage()
	now := time.Now()
	n := len(p.accounts)
	seen := make(map[string]bool)

	// 加权轮询查找可用账号
	for i := 0; i < n; i++ {
		idx := atomic.AddUint64(&p.currentIndex, 1) % uint64(n)
		acc := &p.accounts[idx]

		if excluded != nil && excluded[acc.ID] {
			seen[acc.ID] = true
			continue
		}
		if seen[acc.ID] {
			continue
		}

		// 跳过冷却中的账号
		if cooldown, ok := p.cooldowns[acc.ID]; ok && now.Before(cooldown) {
			seen[acc.ID] = true
			continue
		}

		// 跳过即将过期的 Token
		if acc.ExpiresAt > 0 && time.Now().Unix() > acc.ExpiresAt-tokenRefreshSkewSeconds {
			seen[acc.ID] = true
			continue
		}

		// Skip accounts whose quota is exhausted, unless overrides apply.
		if isQuotaBlocked(*acc, allowOverUsage) {
			seen[acc.ID] = true
			continue
		}

		// Skip accounts that do not support text generation (e.g. Voyage AI embeddings/rerank only)
		if BucketOf(acc) == "voyage" {
			seen[acc.ID] = true
			continue
		}

		return acc
	}

	// 无可用账号，返回冷却时间最短的（排除额度用尽的，除非允许超额）
	var best *config.Account
	var earliest time.Time
	for i := range p.accounts {
		acc := &p.accounts[i]
		if excluded != nil && excluded[acc.ID] {
			continue
		}
		if isQuotaBlocked(*acc, allowOverUsage) {
			continue
		}
		if BucketOf(acc) == "voyage" {
			continue
		}
		if cooldown, ok := p.cooldowns[acc.ID]; ok {
			if best == nil || cooldown.Before(earliest) {
				best = acc
				earliest = cooldown
			}
		} else {
			return acc
		}
	}
	return best
}

// SetModelList 缓存账号支持的模型集合（由 handler 在刷新后调用）
func (p *AccountPool) SetModelList(accountID string, modelIDs []string) {
	set := make(map[string]bool, len(modelIDs))
	for _, id := range modelIDs {
		set[strings.ToLower(strings.TrimSpace(id))] = true
	}
	p.mu.Lock()
	p.modelLists[accountID] = set
	p.mu.Unlock()
}

// GetModelList 返回该账号缓存的模型 ID 列表（供 admin API 使用）。
// 若尚无缓存则返回空切片。
func (p *AccountPool) GetModelList(accountID string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set, ok := p.modelLists[accountID]
	if !ok || len(set) == 0 {
		return []string{}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

// accountHasModel 检查账号是否支持指定模型。
// 若该账号尚无模型列表（冷启动），视为支持所有模型。
func (p *AccountPool) accountHasModel(accountID, model string) bool {
	list, ok := p.modelLists[accountID]
	if !ok || len(list) == 0 {
		return true // 冷启动：列表未就绪，乐观放行
	}
	return list[strings.ToLower(strings.TrimSpace(model))]
}

// BucketOf returns the normalized provider bucket for an account.
func BucketOf(account *config.Account) string {
	if account == nil {
		return ""
	}
	p := strings.ToLower(strings.TrimSpace(account.Provider))
	switch p {
	case "antigravity", "grok", "xai", "codex", "remotekiro", "remote-kiro", "voyage", "voyageai":
		if p == "xai" {
			return "grok"
		}
		if p == "remote-kiro" {
			return "remotekiro"
		}
		if p == "voyageai" {
			return "voyage"
		}
		return p
	}

	a := strings.ToLower(strings.TrimSpace(account.AuthMethod))
	switch a {
	case "antigravity", "grok", "codex", "remotekiro", "remote-kiro", "voyage", "voyageai":
		if a == "remote-kiro" {
			return "remotekiro"
		}
		if a == "voyageai" {
			return "voyage"
		}
		return a
	}

	if account.VoyageAPIKey != "" {
		return "voyage"
	}
	if account.RemoteBaseURL != "" {
		return "remotekiro"
	}
	return "kiro"
}

// parseModelRouting extracts an optional provider/account constraint from a model or Combo candidate.
// Supports both "::" and "/" as delimiters (e.g. "bai::gpt-4o", "bai/gpt-4o", "codex.einnam::gpt-5.3").
func parseModelRouting(candidate string) (targetProvider, targetModel string) {
	raw := strings.TrimSpace(candidate)
	if raw == "" {
		return "", ""
	}

	// 1. Explicit "::" delimiter
	if before, after, ok := strings.Cut(raw, "::"); ok {
		targetProvider = strings.ToLower(strings.TrimSpace(before))
		targetModel = strings.TrimSpace(after)
		if targetProvider == "" || targetModel == "" || strings.Contains(targetModel, "::") {
			return "", ""
		}
		return targetProvider, targetModel
	}

	// 2. Namespace "/" delimiter (e.g. "bai/gpt-5.6-sol")
	if before, after, ok := strings.Cut(raw, "/"); ok {
		prefix := strings.ToLower(strings.TrimSpace(before))
		model := strings.TrimSpace(after)
		if prefix != "" && model != "" {
			return prefix, model
		}
	}

	return "", raw
}

// isVoyageModel reports whether the given model name is a Voyage AI embedding/reranking model.
func isVoyageModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "voyage-") || strings.HasPrefix(m, "voyage_") || strings.HasPrefix(m, "rerank-") || strings.HasPrefix(m, "rerank_")
}

// accountMatchesModel checks if an account supports the given model.
func (p *AccountPool) accountMatchesModel(acc *config.Account, model string) bool {
	if acc == nil {
		return false
	}
	cleanModel := strings.ToLower(strings.TrimSpace(model))

	// If account has explicit CustomModels (Strict Whitelist), enforce it strictly
	if len(acc.CustomModels) > 0 {
		for _, cm := range acc.CustomModels {
			if strings.EqualFold(strings.TrimSpace(cm), cleanModel) {
				return true
			}
		}
		return false
	}

	isVoyage := BucketOf(acc) == "voyage"
	isVoyageMod := isVoyageModel(model)
	if isVoyage && !isVoyageMod {
		return false
	}
	if !isVoyage && isVoyageMod {
		list, ok := p.modelLists[acc.ID]
		if !ok || len(list) == 0 {
			return false
		}
		return list[cleanModel]
	}
	return p.accountHasModel(acc.ID, cleanModel)
}

// GetNextForModel 获取下一个支持指定模型的可用账号。
// model 应为去掉 thinking 后缀的实际模型名。
// 若无账号有该模型列表数据，行为与 GetNext 相同（乐观路由）。
func (p *AccountPool) GetNextForModel(model string) *config.Account {
	return p.GetNextForModelExcluding(model, nil)
}

// GetNextForModelExcluding 获取下一个支持指定模型的可用账号，并跳过指定账号。
func (p *AccountPool) GetNextForModelExcluding(model string, excluded map[string]bool) *config.Account {
	return p.GetNextForModelAndProviderExcluding(model, "", excluded)
}

// GetNextForModelAndProviderExcluding selects an account that supports model and
// belongs to target provider/account. An empty provider preserves legacy cross-provider routing.
func (p *AccountPool) GetNextForModelAndProviderExcluding(model, provider string, excluded map[string]bool) *config.Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.accounts) == 0 {
		return nil
	}

	allowOverUsage := config.GetAllowOverUsage()
	now := time.Now()
	n := len(p.accounts)
	seen := make(map[string]bool)

	targetProvider, targetModel := parseModelRouting(model)
	if targetModel == "" {
		return nil
	}

	// If targetProvider was parsed from a slash (like "meta-llama/llama-3") but no account matches it,
	// fall back to treating the entire candidate as the model name.
	if targetProvider != "" && targetProvider != provider {
		matchedAny := false
		for i := range p.accounts {
			if accountMatchesTarget(&p.accounts[i], targetProvider) {
				matchedAny = true
				break
			}
		}
		if !matchedAny && !strings.Contains(model, "::") {
			targetProvider = ""
			targetModel = model
		}
	}

	for i := 0; i < n; i++ {
		idx := atomic.AddUint64(&p.currentIndex, 1) % uint64(n)
		acc := &p.accounts[idx]

		if excluded != nil && excluded[acc.ID] {
			seen[acc.ID] = true
			continue
		}
		if seen[acc.ID] {
			continue
		}

		if targetProvider == "" {
			targetProvider = provider
		}
		if targetProvider != "" && !accountMatchesTarget(acc, targetProvider) {
			seen[acc.ID] = true
			continue
		}
		if !p.accountMatchesModel(acc, targetModel) {
			seen[acc.ID] = true
			continue
		}
		if cooldown, ok := p.cooldowns[acc.ID]; ok && now.Before(cooldown) {
			seen[acc.ID] = true
			continue
		}
		if acc.ExpiresAt > 0 && time.Now().Unix() > acc.ExpiresAt-tokenRefreshSkewSeconds {
			seen[acc.ID] = true
			continue
		}
		if isQuotaBlocked(*acc, allowOverUsage) {
			seen[acc.ID] = true
			continue
		}
		return acc
	}

	// fallback：找冷却时间最短且支持该模型的账号
	var best *config.Account
	var earliest time.Time
	for i := range p.accounts {
		acc := &p.accounts[i]
		if excluded != nil && excluded[acc.ID] {
			continue
		}
		if targetProvider == "" {
			targetProvider = provider
		}
		if targetProvider != "" && !accountMatchesTarget(acc, targetProvider) {
			continue
		}
		if !p.accountMatchesModel(acc, targetModel) {
			continue
		}
		if isQuotaBlocked(*acc, allowOverUsage) {
			continue
		}
		if cooldown, ok := p.cooldowns[acc.ID]; ok {
			if best == nil || cooldown.Before(earliest) {
				best = acc
				earliest = cooldown
			}
		} else {
			return acc
		}
	}
	return best
}

// UnavailableReason explains why no account could be routed for model, for
// logging alongside a 503. A bare "No available accounts" is indistinguishable
// between "operator disabled everything", "all cooling down after upstream
// errors", "all over quota", and "no account advertises this model" — which is
// exactly the ambiguity that made the pool-draining bug hard to see from the
// outside. Pass an empty model to skip the model filter.
//
// The counts are a snapshot, so an account can be counted under only its first
// matching reason; the total is what matters, not the exact attribution.
func (p *AccountPool) UnavailableReason(model string, excluded map[string]bool) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.totalAccounts == 0 {
		return "no accounts configured"
	}
	if len(p.accounts) == 0 {
		return fmt.Sprintf("all %d configured accounts are disabled or over quota (pool is empty after Reload)", p.totalAccounts)
	}

	allowOverUsage := config.GetAllowOverUsage()
	now := time.Now()
	seen := make(map[string]bool)

	var total, skipExcluded, skipModel, skipCooldown, skipExpired, skipQuota int
	var soonest time.Time
	for i := range p.accounts {
		acc := &p.accounts[i]
		if seen[acc.ID] {
			continue
		}
		seen[acc.ID] = true
		total++

		switch {
		case excluded != nil && excluded[acc.ID]:
			skipExcluded++
		case model != "" && !p.accountMatchesModel(acc, model):
			skipModel++
		case isQuotaBlocked(*acc, allowOverUsage):
			skipQuota++
		case acc.ExpiresAt > 0 && now.Unix() > acc.ExpiresAt-tokenRefreshSkewSeconds:
			skipExpired++
		default:
			if cooldown, ok := p.cooldowns[acc.ID]; ok && now.Before(cooldown) {
				skipCooldown++
				if soonest.IsZero() || cooldown.Before(soonest) {
					soonest = cooldown
				}
			}
		}
	}

	parts := make([]string, 0, 5)
	if skipExcluded > 0 {
		parts = append(parts, fmt.Sprintf("%d already tried this request", skipExcluded))
	}
	if skipModel > 0 {
		parts = append(parts, fmt.Sprintf("%d do not list model %q", skipModel, model))
	}
	if skipCooldown > 0 {
		msg := fmt.Sprintf("%d in error cooldown", skipCooldown)
		if !soonest.IsZero() {
			msg += fmt.Sprintf(" (soonest expires in %s)", time.Until(soonest).Truncate(time.Second))
		}
		parts = append(parts, msg)
	}
	if skipExpired > 0 {
		parts = append(parts, fmt.Sprintf("%d have an expiring/expired token", skipExpired))
	}
	if skipQuota > 0 {
		parts = append(parts, fmt.Sprintf("%d over quota", skipQuota))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d enabled accounts, none matched (selection raced with a pool reload)", total)
	}
	return fmt.Sprintf("%d enabled accounts: %s", total, strings.Join(parts, ", "))
}

// GetByID 根据 ID 获取账号
func (p *AccountPool) GetByID(id string) *config.Account {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for i := range p.accounts {
		if p.accounts[i].ID == id {
			return &p.accounts[i]
		}
	}
	return nil
}

// RecordSuccess 记录请求成功，清除冷却
func (p *AccountPool) RecordSuccess(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cooldowns, id)
	p.errorCounts[id] = 0
}

// RecordError 记录请求错误，设置冷却
func (p *AccountPool) RecordError(id string, isQuotaError bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCounts[id]++

	if isQuotaError {
		// 配额错误，冷却 1 小时
		p.cooldowns[id] = time.Now().Add(time.Hour)
	} else if p.errorCounts[id] >= 3 {
		// 连续 3 次错误，冷却 1 分钟
		p.cooldowns[id] = time.Now().Add(time.Minute)
	}
}

// IsAuthFailure reports whether an error indicates the refresh token / credentials
// have been revoked or invalidated upstream (401, 403 with auth markers, etc.).
// These accounts cannot be recovered automatically and must be re-authenticated.
func IsAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	// Match HTTP status codes only when they appear as standalone tokens to avoid
	// false positives from arbitrary digits in the error body (e.g. request IDs).
	if hasStatusToken(msg, "401") || hasStatusToken(msg, "403") {
		return true
	}
	if strings.Contains(lower, "bad credentials") ||
		strings.Contains(lower, "invalid_grant") ||
		strings.Contains(lower, "invalid grant") ||
		strings.Contains(lower, "invalid_token") ||
		strings.Contains(lower, "invalid token") ||
		strings.Contains(lower, "token expired") ||
		strings.Contains(lower, "token has expired") ||
		strings.Contains(lower, "unauthorized") {
		return true
	}
	return false
}

// hasStatusToken returns true when status appears in s with non-digit boundaries
// on both sides, so "401" matches "HTTP 401 from ..." but not "request_401abc".
func hasStatusToken(s, status string) bool {
	for {
		idx := strings.Index(s, status)
		if idx < 0 {
			return false
		}
		leftOK := idx == 0 || !isDigit(s[idx-1])
		rightIdx := idx + len(status)
		rightOK := rightIdx >= len(s) || !isDigit(s[rightIdx])
		if leftOK && rightOK {
			return true
		}
		s = s[idx+len(status):]
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// IsSuspensionError reports whether the error indicates the account has been
// temporarily suspended by upstream or has no available Kiro profile.
// Unlike auth failures (revoked credentials), these may be transient, but
// the account should be disabled until an operator re-enables it.
func IsSuspensionError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "temporarily_suspended") ||
		strings.Contains(lower, "temporarily suspended") ||
		strings.Contains(lower, "no available kiro profile")
}

// DisableAccount marks an account as disabled (auth revoked / unrecoverable),
// removes it from the in-memory pool so subsequent requests skip it, and
// persists the change via config.SetAccountBanStatus.
func (p *AccountPool) DisableAccount(id, reason string) {
	if err := config.SetAccountBanStatus(id, "DISABLED", reason); err != nil {
		// best effort — even if persistence fails, drop it from memory
		_ = err
	}
	p.mu.Lock()
	// Long cooldown as a safety net in case Reload races
	p.cooldowns[id] = time.Now().Add(24 * time.Hour)
	p.mu.Unlock()
	p.Reload()
}

// MarkOverLimit marks an account as over usage limit (after a 402 / OVERAGE response).
// With the upstream OverageStatus model, the live status is refreshed via
// FetchOverageStatus from the request handler; here we just cooldown briefly so
// the next attempt picks a different account, then reload.
func (p *AccountPool) MarkOverLimit(id string) {
	p.mu.Lock()
	p.cooldowns[id] = time.Now().Add(time.Hour)
	p.mu.Unlock()
	p.Reload()
}

// UpdateToken 更新账号 Token
func (p *AccountPool) UpdateToken(id, accessToken, refreshToken string, expiresAt int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.accounts {
		if p.accounts[i].ID == id {
			p.accounts[i].AccessToken = accessToken
			if refreshToken != "" {
				p.accounts[i].RefreshToken = refreshToken
			}
			p.accounts[i].ExpiresAt = expiresAt
		}
	}
}

// Count 返回账号总数
func (p *AccountPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.totalAccounts > 0 {
		return p.totalAccounts
	}

	seen := make(map[string]bool)
	for _, acc := range p.accounts {
		seen[acc.ID] = true
	}
	return len(seen)
}

// AvailableCount 返回可用账号数
func (p *AccountPool) AvailableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	now := time.Now()
	count := 0
	seen := make(map[string]bool)
	for _, acc := range p.accounts {
		if seen[acc.ID] {
			continue
		}
		seen[acc.ID] = true
		if cooldown, ok := p.cooldowns[acc.ID]; ok && now.Before(cooldown) {
			continue
		}
		count++
	}
	return count
}

// UpdateStats 更新账号统计
func (p *AccountPool) UpdateStats(id string, tokens int, credits float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var updated bool
	var requestCount, errorCount, totalTokens int
	var totalCredits float64
	var lastUsed int64
	for i := range p.accounts {
		if p.accounts[i].ID == id {
			if !updated {
				p.accounts[i].RequestCount++
				p.accounts[i].TotalTokens += tokens
				p.accounts[i].TotalCredits += credits
				p.accounts[i].LastUsed = time.Now().Unix()

				requestCount = p.accounts[i].RequestCount
				errorCount = p.accounts[i].ErrorCount
				totalTokens = p.accounts[i].TotalTokens
				totalCredits = p.accounts[i].TotalCredits
				lastUsed = p.accounts[i].LastUsed
				updated = true
				continue
			}
			p.accounts[i].RequestCount = requestCount
			p.accounts[i].ErrorCount = errorCount
			p.accounts[i].TotalTokens = totalTokens
			p.accounts[i].TotalCredits = totalCredits
			p.accounts[i].LastUsed = lastUsed
		}
	}
	if updated {
		go config.UpdateAccountStats(id, requestCount, errorCount, totalTokens, totalCredits, lastUsed)
	}
}

// GetAllAccounts 获取所有账号副本
func (p *AccountPool) GetAllAccounts() []config.Account {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]config.Account, len(p.accounts))
	copy(result, p.accounts)
	return result
}

func isOverUsageLimit(acc config.Account) bool {
	return acc.UsageLimit > 0 && acc.UsageCurrent >= acc.UsageLimit
}

// isQuotaBlocked reports whether an over-quota account should be skipped:
// the per-account upstream Overages switch (OverageStatus=ENABLED) and the
// global allowOverUsage setting are the two ways to keep it routable.
func isQuotaBlocked(acc config.Account, allowOverUsage bool) bool {
	return isOverUsageLimit(acc) && !isUpstreamOverageEnabled(acc) && !allowOverUsage
}

// isUpstreamOverageEnabled reports whether the upstream Overages switch is ON for this account.
// "ENABLED" → true; anything else (DISABLED, UNKNOWN, empty) → false.
func isUpstreamOverageEnabled(acc config.Account) bool {
	return strings.EqualFold(acc.OverageStatus, "ENABLED")
}

// accountMatchesTarget checks if an account matches a routing target, which can be:
// 1. Account Nickname / Prefix (e.g. "bai", "coe", "keycrop")
// 2. Account ID (e.g. "c46f8591-b394-40ed-ad19-4babaebc7880")
// 3. Account Email or Email username (e.g. "einnam20@gmail.com", "einnam20")
// 4. Composite "<provider>.<identifier>" (e.g. "codex.einnam20", "remotekiro.bai")
// 5. Provider bucket name (e.g. "antigravity", "codex", "remotekiro", "grok", "voyage", "kiro")
func accountMatchesTarget(account *config.Account, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return true
	}
	if account == nil {
		return false
	}

	// 1. Match by Nickname
	if account.Nickname != "" && strings.EqualFold(strings.TrimSpace(account.Nickname), target) {
		return true
	}

	// 2. Match by Account ID
	if strings.EqualFold(strings.TrimSpace(account.ID), target) {
		return true
	}

	// 3. Match by Email or Email prefix
	if account.Email != "" {
		email := strings.TrimSpace(account.Email)
		if strings.EqualFold(email, target) {
			return true
		}
		if username, _, ok := strings.Cut(email, "@"); ok && strings.EqualFold(username, target) {
			return true
		}
	}

	// 4. Match by Composite format "<provider>.<identifier>"
	if prov, ident, found := strings.Cut(target, "."); found {
		if accountMatchesProvider(account, prov) {
			if strings.EqualFold(strings.TrimSpace(account.ID), ident) {
				return true
			}
			if account.Nickname != "" && strings.EqualFold(strings.TrimSpace(account.Nickname), ident) {
				return true
			}
			if account.Email != "" {
				email := strings.TrimSpace(account.Email)
				if strings.EqualFold(email, ident) {
					return true
				}
				if username, _, ok := strings.Cut(email, "@"); ok && strings.EqualFold(username, ident) {
					return true
				}
			}
		}
		return false
	}

	// 5. Match by Provider
	return AccountMatchesProvider(account, target)
}

func AccountMatchesTarget(account *config.Account, target string) bool {
	return accountMatchesTarget(account, target)
}

func accountMatchesProvider(account *config.Account, provider string) bool {
	return AccountMatchesProvider(account, provider)
}

func AccountMatchesProvider(account *config.Account, provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return true
	}
	if account == nil {
		return false
	}
	accountProvider := strings.ToLower(strings.TrimSpace(account.Provider))
	authMethod := strings.ToLower(strings.TrimSpace(account.AuthMethod))
	switch {
	case accountProvider == "remotekiro" || authMethod == "remotekiro" || strings.TrimSpace(account.RemoteBaseURL) != "":
		return provider == "remotekiro"
	case accountProvider == "codex" || authMethod == "codex":
		return provider == "codex"
	case accountProvider == "antigravity" || authMethod == "antigravity":
		return provider == "antigravity"
	case accountProvider == "grok" || accountProvider == "xai" || authMethod == "grok" || account.GrokAPIKey != "":
		return provider == "grok"
	case accountProvider == "voyage" || accountProvider == "voyageai" || authMethod == "voyage" || account.VoyageAPIKey != "":
		return provider == "voyage"
	default:
		return provider == "kiro"
	}
}

// CleanRoutingModel strips any namespace or provider prefix (e.g. "bai/", "coe::", "codex/").
func CleanRoutingModel(candidate string) string {
	raw := strings.TrimSpace(candidate)
	if raw == "" {
		return ""
	}
	if _, after, ok := strings.Cut(raw, "::"); ok {
		if strings.TrimSpace(after) != "" {
			return strings.TrimSpace(after)
		}
	}
	if before, after, ok := strings.Cut(raw, "/"); ok {
		prefix := strings.ToLower(strings.TrimSpace(before))
		model := strings.TrimSpace(after)
		if prefix != "" && model != "" {
			for _, acc := range config.GetAccounts() {
				if AccountMatchesTarget(&acc, prefix) {
					return model
				}
			}
			if AccountMatchesProvider(nil, prefix) || prefix == "antigravity" || prefix == "codex" || prefix == "remotekiro" || prefix == "grok" || prefix == "voyage" || prefix == "kiro" || prefix == "openai" || prefix == "anthropic" || prefix == "google" || prefix == "xai" || prefix == "voyageai" {
				return model
			}
		}
	}
	return raw
}

func effectiveWeight(weight int) int {
	if weight < 1 {
		return 1
	}
	return weight
}
