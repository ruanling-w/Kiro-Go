package proxy

import (
	"crypto/sha256"
	"kiro-go/config"
	"sync"
	"time"
)

// defaultResponseCacheTTL bounds how long a cached upstream answer may be
// replayed before it is considered stale. Short enough that a re-asked prompt
// does not return a very old answer; long enough to absorb the retry storms and
// duplicate requests that clients (and agent loops) generate.
const defaultResponseCacheTTL = 10 * time.Minute

// defaultResponseCacheMaxPerKey caps how many distinct answers are retained per
// API key so a single noisy key cannot grow the cache without bound. When the
// cap is hit the oldest entry (by creation time) is evicted.
const defaultResponseCacheMaxPerKey = 256

// cachedResponse is a fully-materialized answer ready to be replayed to a client
// without contacting any upstream. It stores the semantic content (text +
// thinking) and the numbers needed to reproduce the original wire response, not
// the raw provider frames — the handlers rebuild frames via the same builders
// used for a live response, so a cache hit is byte-identical in shape.
type cachedResponse struct {
	Content         string
	ThinkingContent string
	InputTokens     int
	OutputTokens    int
	FinishReason    string // OpenAI finish vocabulary ("", "stop", "length", ...)
	Model           string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type responseCache struct {
	mu        sync.Mutex
	ttl       time.Duration
	maxPerKey int
	byKey     map[string]map[[32]byte]cachedResponse
}

func newResponseCache(ttl time.Duration, maxPerKey int) *responseCache {
	if ttl <= 0 {
		ttl = defaultResponseCacheTTL
	}
	if maxPerKey <= 0 {
		maxPerKey = defaultResponseCacheMaxPerKey
	}
	return &responseCache{
		ttl:       ttl,
		maxPerKey: maxPerKey,
		byKey:     make(map[string]map[[32]byte]cachedResponse),
	}
}

// Get returns a non-expired cached answer for (apiKeyID, key). Anonymous
// requests (empty apiKeyID) are never cached, so they always miss — this keeps
// one caller's answers from leaking to another.
func (c *responseCache) Get(apiKeyID string, key [32]byte) (cachedResponse, bool) {
	if c == nil || apiKeyID == "" {
		return cachedResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := c.byKey[apiKeyID]
	if entries == nil {
		return cachedResponse{}, false
	}
	entry, ok := entries[key]
	if !ok {
		return cachedResponse{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(entries, key)
		if len(entries) == 0 {
			delete(c.byKey, apiKeyID)
		}
		return cachedResponse{}, false
	}
	return entry, true
}

// Set stores an answer for (apiKeyID, key). No-op for anonymous requests.
func (c *responseCache) Set(apiKeyID string, key [32]byte, v cachedResponse) {
	if c == nil || apiKeyID == "" {
		return
	}
	now := time.Now()
	v.CreatedAt = now
	v.ExpiresAt = now.Add(c.ttl)

	c.mu.Lock()
	defer c.mu.Unlock()
	entries := c.byKey[apiKeyID]
	if entries == nil {
		entries = make(map[[32]byte]cachedResponse)
		c.byKey[apiKeyID] = entries
	}
	entries[key] = v
	c.evictLocked(entries)
}

// evictLocked prunes expired entries and, if still over the per-key cap, drops
// the oldest entries by creation time until within bounds.
func (c *responseCache) evictLocked(entries map[[32]byte]cachedResponse) {
	now := time.Now()
	for k, e := range entries {
		if now.After(e.ExpiresAt) {
			delete(entries, k)
		}
	}
	for len(entries) > c.maxPerKey {
		var oldestKey [32]byte
		var oldest time.Time
		first := true
		for k, e := range entries {
			if first || e.CreatedAt.Before(oldest) {
				oldestKey = k
				oldest = e.CreatedAt
				first = false
			}
		}
		delete(entries, oldestKey)
	}
}

// buildClaudeResponseCacheKey fingerprints the request fields that determine the
// answer: the resolved model, whether thinking mode is on (the model string has
// its -thinking suffix stripped before this point, so thinking must be hashed
// separately or a thinking and non-thinking turn would collide), the system
// prompt, the full message list, and the sampling params. Tool definitions are
// intentionally absent — tool requests are never cached, so they never reach
// here. Reuses the canonical JSON + length-prefixed hashing from
// cache_tracker.go so the key is stable across map ordering.
func buildClaudeResponseCacheKey(req *ClaudeRequest, thinking bool) [32]byte {
	hasher := sha256.New()
	writeHashChunk(hasher, "claude")
	writeHashChunk(hasher, req.Model)
	writeHashChunk(hasher, boolChunk(thinking))
	writeHashChunk(hasher, canonicalizeCacheValue(req.System))
	writeHashChunk(hasher, canonicalizeCacheValue(req.Messages))
	writeHashChunk(hasher, canonicalizeCacheValue(map[string]interface{}{
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
	}))
	var out [32]byte
	copy(out[:], hasher.Sum(nil))
	return out
}

// buildOpenAIResponseCacheKey is the OpenAI counterpart of
// buildClaudeResponseCacheKey.
func buildOpenAIResponseCacheKey(req *OpenAIRequest, thinking bool) [32]byte {
	hasher := sha256.New()
	writeHashChunk(hasher, "openai")
	writeHashChunk(hasher, req.Model)
	writeHashChunk(hasher, boolChunk(thinking))
	writeHashChunk(hasher, canonicalizeCacheValue(req.Messages))
	writeHashChunk(hasher, canonicalizeCacheValue(map[string]interface{}{
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
		"stop":        req.Stop,
	}))
	var out [32]byte
	copy(out[:], hasher.Sum(nil))
	return out
}

func boolChunk(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// responseCacheIntent carries a computed cache key plus the API key it is scoped
// to. A nil *responseCacheIntent passed into a handler means "do not cache this
// request" — the single sentinel keeps the four direct handlers from each
// re-deriving the cacheability rules.
type responseCacheIntent struct {
	apiKeyID string
	key      [32]byte
}

// claudeCacheIntent returns a cache intent for a direct (non-combo) Claude
// request, or nil when the request must not be cached. A request is cacheable
// only when: the response cache is enabled, the caller is authenticated (an
// apiKeyID scopes the entry so answers never leak across keys), and the request
// carries no tools (tool_use is not idempotent — a replayed tool call would skip
// the side effect the client expects).
func (h *Handler) claudeCacheIntent(req *ClaudeRequest, thinking bool, apiKeyID string) *responseCacheIntent {
	if h == nil || h.responseCache == nil || apiKeyID == "" {
		return nil
	}
	if !config.GetResponseCacheEnabled() {
		return nil
	}
	if len(req.Tools) > 0 {
		return nil
	}
	return &responseCacheIntent{apiKeyID: apiKeyID, key: buildClaudeResponseCacheKey(req, thinking)}
}

// openAICacheIntent is the OpenAI counterpart of claudeCacheIntent.
func (h *Handler) openAICacheIntent(req *OpenAIRequest, thinking bool, apiKeyID string) *responseCacheIntent {
	if h == nil || h.responseCache == nil || apiKeyID == "" {
		return nil
	}
	if !config.GetResponseCacheEnabled() {
		return nil
	}
	if len(req.Tools) > 0 {
		return nil
	}
	return &responseCacheIntent{apiKeyID: apiKeyID, key: buildOpenAIResponseCacheKey(req, thinking)}
}

// lookup returns a cached answer for this intent, or false on miss / nil intent.
func (ci *responseCacheIntent) lookup(cache *responseCache) (cachedResponse, bool) {
	if ci == nil || cache == nil {
		return cachedResponse{}, false
	}
	return cache.Get(ci.apiKeyID, ci.key)
}

// store records an answer for this intent. A truncated answer (finish_reason
// "length") is never stored: it is an incomplete response, and replaying it
// would hand the client a half-finished answer as if it were final.
func (ci *responseCacheIntent) store(cache *responseCache, v cachedResponse) {
	if ci == nil || cache == nil {
		return
	}
	if v.FinishReason == "length" {
		return
	}
	cache.Set(ci.apiKeyID, ci.key, v)
}
