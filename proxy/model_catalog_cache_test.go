package proxy

import (
	"testing"
	"time"
)

func TestModelCatalogCacheGetSetRespectsTTL(t *testing.T) {
	c := newModelCatalogCache(50 * time.Millisecond)
	models := []ModelInfo{{ModelId: "claude-opus-4-8"}}

	if _, ok := c.Get("acc1"); ok {
		t.Fatalf("expected miss on empty cache")
	}

	c.Set("acc1", models)
	got, ok := c.Get("acc1")
	if !ok {
		t.Fatalf("expected hit after Set")
	}
	if len(got) != 1 || got[0].ModelId != "claude-opus-4-8" {
		t.Fatalf("unexpected cached models: %#v", got)
	}

	time.Sleep(70 * time.Millisecond)
	if _, ok := c.Get("acc1"); ok {
		t.Fatalf("expected miss after TTL expiry")
	}
}

func TestModelCatalogCacheGetStaleIgnoresExpiry(t *testing.T) {
	c := newModelCatalogCache(1 * time.Millisecond)
	c.Set("acc1", []ModelInfo{{ModelId: "m1"}})
	time.Sleep(5 * time.Millisecond)

	if _, ok := c.Get("acc1"); ok {
		t.Fatalf("expected fresh Get to miss after expiry")
	}
	stale, ok := c.GetStale("acc1")
	if !ok || len(stale) != 1 || stale[0].ModelId != "m1" {
		t.Fatalf("expected stale entry to survive expiry, got %#v ok=%v", stale, ok)
	}
}

func TestModelCatalogCacheGetStaleEmptyIsMiss(t *testing.T) {
	c := newModelCatalogCache(time.Minute)
	if _, ok := c.GetStale("acc1"); ok {
		t.Fatalf("expected stale miss when nothing was ever cached")
	}
	// An explicitly empty catalog is treated as no fallback.
	c.Set("acc1", []ModelInfo{})
	if _, ok := c.GetStale("acc1"); ok {
		t.Fatalf("expected empty catalog to be treated as no stale fallback")
	}
}

func TestModelCatalogCacheInvalidate(t *testing.T) {
	c := newModelCatalogCache(time.Minute)
	c.Set("acc1", []ModelInfo{{ModelId: "m1"}})
	c.Invalidate("acc1")
	if _, ok := c.Get("acc1"); ok {
		t.Fatalf("expected miss after Invalidate")
	}
	if _, ok := c.GetStale("acc1"); ok {
		t.Fatalf("expected no stale entry after Invalidate")
	}
}

func TestModelCatalogCacheEmptyIDIsNoOp(t *testing.T) {
	c := newModelCatalogCache(time.Minute)
	c.Set("", []ModelInfo{{ModelId: "m1"}})
	if _, ok := c.Get(""); ok {
		t.Fatalf("expected empty id to never cache")
	}
}
