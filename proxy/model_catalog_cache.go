package proxy

import (
	"sync"
	"time"
)

// defaultModelCatalogTTL bounds how long a per-account Kiro model catalog stays
// fresh before a live ListAvailableModels call is made again. Matches 9router's
// 5-minute per-credential catalog cache: short enough that a newly granted model
// shows up quickly, long enough to dedupe the bursty refreshes that the
// background sweep (30m) and lazy /v1/models fill would otherwise cause.
const defaultModelCatalogTTL = 5 * time.Minute

type modelCatalogEntry struct {
	Models    []ModelInfo
	ExpiresAt time.Time
}

// modelCatalogCache is a small per-account TTL cache for Kiro model catalogs.
// Expired entries are retained (not deleted) so GetStale can serve them as a
// fallback when a live fetch fails — this keeps an account's models visible
// through a transient AWS blip instead of dropping it from the aggregate list.
// Entries are only removed by Invalidate (token rotation) or overwritten by Set.
type modelCatalogCache struct {
	mu        sync.Mutex
	ttl       time.Duration
	byAccount map[string]modelCatalogEntry
}

func newModelCatalogCache(ttl time.Duration) *modelCatalogCache {
	if ttl <= 0 {
		ttl = defaultModelCatalogTTL
	}
	return &modelCatalogCache{
		ttl:       ttl,
		byAccount: make(map[string]modelCatalogEntry),
	}
}

// Get returns the cached catalog for id only if it has not expired.
func (c *modelCatalogCache) Get(id string) ([]ModelInfo, bool) {
	if c == nil || id == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.byAccount[id]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Models, true
}

// GetStale returns the cached catalog for id regardless of expiry. Used only as
// a fallback when a live fetch fails.
func (c *modelCatalogCache) GetStale(id string) ([]ModelInfo, bool) {
	if c == nil || id == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.byAccount[id]
	if !ok || len(entry.Models) == 0 {
		return nil, false
	}
	return entry.Models, true
}

// Set stores a fresh catalog for id, resetting its TTL.
func (c *modelCatalogCache) Set(id string, models []ModelInfo) {
	if c == nil || id == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byAccount[id] = modelCatalogEntry{
		Models:    models,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate drops any cached catalog for id. Call after rotating/importing a
// token so the next fetch is fresh (a rotated credential may see a different
// catalog).
func (c *modelCatalogCache) Invalidate(id string) {
	if c == nil || id == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.byAccount, id)
}
