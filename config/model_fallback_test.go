package config

import "testing"

func TestGetModelFallback(t *testing.T) {
	cfgLock.Lock()
	old := cfg
	cfg = &Config{ModelFallback: map[string][]ModelFallbackTarget{
		"claude-opus-4.8": {{Model: "grok-4.5"}, {Model: "gemini-2.5-pro"}, {Model: ""}},
	}}
	cfgLock.Unlock()
	defer func() {
		cfgLock.Lock()
		cfg = old
		cfgLock.Unlock()
	}()

	got := GetModelFallback("Claude-Opus-4.8")
	if len(got) != 2 || got[0].Model != "grok-4.5" || got[1].Model != "gemini-2.5-pro" {
		t.Fatalf("unexpected fallback: %+v", got)
	}

	// A model with no explicit chain falls back to the built-in default chain
	// (Claude 4.6 first, then Gemini 3.1 Pro), not nil.
	def := GetModelFallback("nope")
	if len(def) != len(defaultModelFallbackChain) {
		t.Fatalf("expected default chain for unconfigured model, got %+v", def)
	}
	for i, want := range defaultModelFallbackChain {
		if def[i].Model != want.Model {
			t.Fatalf("default chain[%d] = %q, want %q", i, def[i].Model, want.Model)
		}
	}

	// An empty model id still returns nil (no request to route).
	if GetModelFallback("") != nil {
		t.Fatalf("expected nil for empty model")
	}
}
