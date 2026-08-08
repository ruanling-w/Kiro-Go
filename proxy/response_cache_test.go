package proxy

import (
	"testing"
	"time"
)

func TestResponseCacheGetSetRoundTrip(t *testing.T) {
	c := newResponseCache(time.Minute, 8)
	var key [32]byte
	key[0] = 1

	if _, ok := c.Get("k1", key); ok {
		t.Fatalf("expected miss on empty cache")
	}
	c.Set("k1", key, cachedResponse{Content: "hello", InputTokens: 3, OutputTokens: 5})
	hit, ok := c.Get("k1", key)
	if !ok {
		t.Fatalf("expected hit after Set")
	}
	if hit.Content != "hello" || hit.InputTokens != 3 || hit.OutputTokens != 5 {
		t.Fatalf("unexpected cached value: %#v", hit)
	}
}

func TestResponseCacheScopedByApiKey(t *testing.T) {
	c := newResponseCache(time.Minute, 8)
	var key [32]byte
	c.Set("owner", key, cachedResponse{Content: "secret"})

	if _, ok := c.Get("intruder", key); ok {
		t.Fatalf("cache entry leaked across api keys")
	}
	if _, ok := c.Get("owner", key); !ok {
		t.Fatalf("owner should still hit its own entry")
	}
}

func TestResponseCacheAnonymousNeverCached(t *testing.T) {
	c := newResponseCache(time.Minute, 8)
	var key [32]byte
	c.Set("", key, cachedResponse{Content: "x"})
	if _, ok := c.Get("", key); ok {
		t.Fatalf("anonymous (empty apiKeyID) requests must never cache")
	}
}

func TestResponseCacheExpiry(t *testing.T) {
	c := newResponseCache(20*time.Millisecond, 8)
	var key [32]byte
	c.Set("k1", key, cachedResponse{Content: "temp"})
	if _, ok := c.Get("k1", key); !ok {
		t.Fatalf("expected hit before TTL")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get("k1", key); ok {
		t.Fatalf("expected miss after TTL")
	}
}

func TestResponseCacheEvictsOldestOverCap(t *testing.T) {
	c := newResponseCache(time.Minute, 2)
	mk := func(b byte) [32]byte { var k [32]byte; k[0] = b; return k }

	c.Set("k1", mk(1), cachedResponse{Content: "first"})
	time.Sleep(2 * time.Millisecond)
	c.Set("k1", mk(2), cachedResponse{Content: "second"})
	time.Sleep(2 * time.Millisecond)
	c.Set("k1", mk(3), cachedResponse{Content: "third"}) // over cap → evict oldest (mk(1))

	if _, ok := c.Get("k1", mk(1)); ok {
		t.Fatalf("expected oldest entry to be evicted")
	}
	if _, ok := c.Get("k1", mk(2)); !ok {
		t.Fatalf("expected second entry to survive")
	}
	if _, ok := c.Get("k1", mk(3)); !ok {
		t.Fatalf("expected newest entry to survive")
	}
}

func TestClaudeResponseCacheKeyStability(t *testing.T) {
	req := &ClaudeRequest{
		Model:       "claude-sonnet-4.5",
		MaxTokens:   1000,
		Temperature: 0.7,
		System:      "you are helpful",
		Messages:    []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	a := buildClaudeResponseCacheKey(req, false)
	b := buildClaudeResponseCacheKey(req, false)
	if a != b {
		t.Fatalf("identical requests must produce identical keys")
	}
}

func TestClaudeResponseCacheKeyDiffersOnParams(t *testing.T) {
	base := &ClaudeRequest{
		Model:     "claude-sonnet-4.5",
		MaxTokens: 1000,
		System:    "sys",
		Messages:  []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	baseKey := buildClaudeResponseCacheKey(base, false)

	// Thinking flag must change the key (model string has suffix stripped).
	if buildClaudeResponseCacheKey(base, true) == baseKey {
		t.Fatalf("thinking on/off must produce different keys")
	}

	changes := []func(*ClaudeRequest){
		func(r *ClaudeRequest) { r.Model = "claude-opus-4.8" },
		func(r *ClaudeRequest) { r.MaxTokens = 2000 },
		func(r *ClaudeRequest) { r.Temperature = 0.9 },
		func(r *ClaudeRequest) { r.TopP = 0.5 },
		func(r *ClaudeRequest) { r.System = "other" },
		func(r *ClaudeRequest) { r.Messages = []ClaudeMessage{{Role: "user", Content: "bye"}} },
	}
	for i, mutate := range changes {
		clone := *base
		clone.Messages = append([]ClaudeMessage(nil), base.Messages...)
		mutate(&clone)
		if buildClaudeResponseCacheKey(&clone, false) == baseKey {
			t.Fatalf("change #%d should have altered the cache key", i)
		}
	}
}

func TestOpenAIResponseCacheKeyDiffersOnParams(t *testing.T) {
	base := &OpenAIRequest{
		Model:     "gpt-4o",
		MaxTokens: 500,
		Messages:  []OpenAIMessage{{Role: "user", Content: "hi"}},
	}
	baseKey := buildOpenAIResponseCacheKey(base, false)

	changes := []func(*OpenAIRequest){
		func(r *OpenAIRequest) { r.Model = "gpt-4" },
		func(r *OpenAIRequest) { r.MaxTokens = 900 },
		func(r *OpenAIRequest) { r.Temperature = 0.3 },
		func(r *OpenAIRequest) { r.Stop = []string{"END"} },
		func(r *OpenAIRequest) { r.Messages = []OpenAIMessage{{Role: "user", Content: "different"}} },
	}
	for i, mutate := range changes {
		clone := *base
		clone.Messages = append([]OpenAIMessage(nil), base.Messages...)
		mutate(&clone)
		if buildOpenAIResponseCacheKey(&clone, false) == baseKey {
			t.Fatalf("change #%d should have altered the OpenAI cache key", i)
		}
	}
}

func TestResponseCacheIntentStoreSkipsLength(t *testing.T) {
	c := newResponseCache(time.Minute, 8)
	var key [32]byte
	ci := &responseCacheIntent{apiKeyID: "k1", key: key}

	// A truncated (max_tokens → "length") answer must not be stored.
	ci.store(c, cachedResponse{Content: "half", FinishReason: "length"})
	if _, ok := ci.lookup(c); ok {
		t.Fatalf("truncated (length) answer must not be cached")
	}

	// A complete answer stores and replays.
	ci.store(c, cachedResponse{Content: "full", FinishReason: "stop"})
	hit, ok := ci.lookup(c)
	if !ok || hit.Content != "full" {
		t.Fatalf("complete answer should be cached, got ok=%v hit=%#v", ok, hit)
	}
}

func TestResponseCacheIntentNilLookupIsMiss(t *testing.T) {
	c := newResponseCache(time.Minute, 8)
	var ci *responseCacheIntent
	if _, ok := ci.lookup(c); ok {
		t.Fatalf("nil intent must always miss")
	}
	ci.store(c, cachedResponse{Content: "x"}) // must not panic
}
