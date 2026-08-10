package proxy

import "testing"

func TestUpdateTokenUsageFromEventCacheAliases(t *testing.T) {
	event := map[string]interface{}{
		"usage": map[string]interface{}{
			"input_tokens":  float64(120),
			"output_tokens": float64(30),
			"input_tokens_details": map[string]interface{}{
				"cached_tokens": float64(80),
			},
			"cache_creation_input_tokens": float64(10),
		},
	}

	usage := updateTokenUsageFromEvent(event, tokenUsage{})
	if usage.Input != 120 || usage.Output != 30 {
		t.Fatalf("unexpected input/output usage: %+v", usage)
	}
	if usage.CacheRead != 80 || usage.CacheCreation != 10 {
		t.Fatalf("unexpected cache usage: %+v", usage)
	}
}

func TestUpdateTokenUsageFromEventMergesSnapshots(t *testing.T) {
	usage := updateTokenUsageFromEvent(map[string]interface{}{
		"usage": map[string]interface{}{
			"input_tokens":          float64(200),
			"cacheReadInputTokens":  float64(150),
			"cacheWriteInputTokens": float64(20),
		},
	}, tokenUsage{})
	usage = updateTokenUsageFromEvent(map[string]interface{}{
		"usage": map[string]interface{}{"output_tokens": float64(40)},
	}, usage)

	if usage.Input != 200 || usage.Output != 40 || usage.CacheRead != 150 || usage.CacheCreation != 20 {
		t.Fatalf("stream fragments were not merged: %+v", usage)
	}
}
