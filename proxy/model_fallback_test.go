package proxy

import (
	"testing"

	"kiro-go/config"
)

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
	if providerLabel(&config.Account{AuthMethod: "remotekiro", Provider: "remotekiro"}) != "remotekiro" {
		t.Fatal("remotekiro")
	}
}
