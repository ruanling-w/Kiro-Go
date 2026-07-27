package proxy

import (
	"path/filepath"
	"testing"
	"time"

	"kiro-go/config"
	accountpool "kiro-go/pool"
)

func TestOverridePayloadModel(t *testing.T) {
	p := &KiroPayload{}
	p.SourceClaude = &ClaudeRequest{Model: "claude-opus-4.8"}
	p.SourceOpenAI = &OpenAIRequest{Model: "claude-opus-4.8"}
	p.ConversationState.CurrentMessage.UserInputMessage.ModelID = "claude-opus-4.8"

	overridePayloadModel(p, "grok-4.5")
	if p.ConversationState.CurrentMessage.UserInputMessage.ModelID != "grok-4.5" {
		t.Fatalf("payload ModelID not overridden")
	}
	if p.SourceClaude.Model != "grok-4.5" || p.SourceOpenAI.Model != "grok-4.5" {
		t.Fatalf("source models not overridden")
	}
}

func TestProviderLabel(t *testing.T) {
	if providerLabel(nil) != "" {
		t.Fatal("nil account")
	}
	if providerLabel(&config.Account{Provider: "grok"}) != "grok" {
		t.Fatal("grok")
	}
	if providerLabel(&config.Account{Provider: "codex"}) != "codex" {
		t.Fatal("codex")
	}
	if providerLabel(&config.Account{AuthMethod: "antigravity"}) != "antigravity" {
		t.Fatal("antigravity")
	}
	if providerLabel(&config.Account{}) != "kiro" {
		t.Fatal("default kiro")
	}
}

// When the native pool for the requested model is exhausted, the built-in default
// chain must spill over to an Antigravity account and rewrite only the payload
// ModelID (Claude 4.6 first), leaving the client-facing model name untouched.
func TestNextAccountDefaultSpilloverToAntigravity(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	if err := config.Init(cfgFile); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	// Only an Antigravity account is in the pool; there is no native account for the
	// requested client model, so phase 1 finds nothing and phase 2 must route here.
	if err := config.AddAccount(config.Account{
		ID:          "ag",
		Email:       "ag@example.com",
		AccessToken: "ag-token",
		AuthMethod:  "antigravity",
		Provider:    "Antigravity",
		ExpiresAt:   time.Now().Unix() + 3600,
		Enabled:     true,
	}); err != nil {
		t.Fatalf("AddAccount(ag): %v", err)
	}

	p := accountpool.GetPool()
	p.Reload()
	// Register the AG model catalog so the pool routes fallback targets here.
	p.SetModelList("ag", antigravityModelIDs())

	const clientModel = "claude-opus-4.8"
	payload := &KiroPayload{SourceClaude: &ClaudeRequest{Model: clientModel}}
	payload.ConversationState.CurrentMessage.UserInputMessage.ModelID = clientModel

	excluded := map[string]bool{}
	nativeDone := false
	nativeAttempts := 0
	fallbackIdx := 0
	fallbackAttempts := 0

	acc, usedModel := nextAccountForAttempt(p, clientModel, payload, excluded,
		&nativeDone, &nativeAttempts, &fallbackIdx, &fallbackAttempts)

	if acc == nil {
		t.Fatal("expected spillover to the Antigravity account, got nil")
	}
	if acc.ID != "ag" {
		t.Fatalf("expected account ag, got %q", acc.ID)
	}
	if usedModel != "claude-sonnet-4-6" {
		t.Fatalf("expected first default target claude-sonnet-4-6, got %q", usedModel)
	}
	if got := payload.ConversationState.CurrentMessage.UserInputMessage.ModelID; got != "claude-sonnet-4-6" {
		t.Fatalf("payload ModelID should be overridden to the fallback target, got %q", got)
	}
	if payload.SourceClaude.Model != "claude-sonnet-4-6" {
		t.Fatalf("source model should be overridden for the upstream, got %q", payload.SourceClaude.Model)
	}
}
