package proxy

import "testing"

// The three image-model classifiers must be mutually exclusive so a request
// routes to exactly one upstream (see handleImageGenerations). Codex is the
// catch-all "-image" matcher, so it must explicitly exclude Grok/Antigravity.
func TestImageModelClassifiersMutuallyExclusive(t *testing.T) {
	cases := []struct {
		model string
		grok  bool
		ag    bool
		codex bool
	}{
		{"grok-2-image-1212", true, false, false},
		{"gemini-3.1-flash-image", false, true, false},
		{"gpt-5.5-image", false, false, true},
		{"gpt-5.6-image", false, false, true},
		// Non-image models must not match any image classifier.
		{"grok-4.5", false, false, false},
		{"gemini-pro-agent", false, false, false},
		{"claude-opus-4.8", false, false, false},
		// Grok video must not be treated as image.
		{"grok-imagine-video", false, false, false},
	}

	for _, c := range cases {
		if got := isGrokImageModel(c.model); got != c.grok {
			t.Errorf("isGrokImageModel(%q) = %v, want %v", c.model, got, c.grok)
		}
		if got := isAntigravityImageModel(c.model); got != c.ag {
			t.Errorf("isAntigravityImageModel(%q) = %v, want %v", c.model, got, c.ag)
		}
		if got := isCodexImageModel(c.model); got != c.codex {
			t.Errorf("isCodexImageModel(%q) = %v, want %v", c.model, got, c.codex)
		}

		// At most one classifier may match a given model.
		matches := 0
		if isGrokImageModel(c.model) {
			matches++
		}
		if isAntigravityImageModel(c.model) {
			matches++
		}
		if isCodexImageModel(c.model) {
			matches++
		}
		if matches > 1 {
			t.Errorf("model %q matched %d image classifiers, want at most 1", c.model, matches)
		}
	}
}

func TestSizeToAspectRatio(t *testing.T) {
	cases := map[string]string{
		"1024x1024": "1:1",
		"1792x1024": "16:9",
		"1024x1792": "9:16",
		"":          "1:1",
		"garbage":   "1:1",
	}
	for size, want := range cases {
		if got := sizeToAspectRatio(size); got != want {
			t.Errorf("sizeToAspectRatio(%q) = %q, want %q", size, got, want)
		}
	}
}
