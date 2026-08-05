package proxy

import (
	"kiro-go/config"
	"strings"
)

// providerLabel returns a short stable name for the upstream that will serve an
// account. Used for admin-only request logs so operators can see which provider
// answered (kiro / grok / codex / antigravity / remotekiro).
func providerLabel(account *config.Account) string {
	if account == nil {
		return ""
	}
	switch {
	case isAntigravityAccount(account):
		return "antigravity"
	case isCodexAccount(account):
		return "codex"
	case isGrokAccount(account):
		return "grok"
	case isRemoteKiroAccount(account):
		return "remotekiro"
	case isKiroAPIKeyAccount(account):
		return "kiro-apikey"
	default:
		if p := strings.TrimSpace(account.Provider); p != "" {
			return strings.ToLower(p)
		}
		if p := strings.TrimSpace(account.AuthMethod); p != "" {
			return strings.ToLower(p)
		}
		return "kiro"
	}
}

func accountLabel(account *config.Account) string {
	if account == nil {
		return ""
	}
	if account.Email != "" {
		return account.Email
	}
	if account.Nickname != "" {
		return account.Nickname
	}
	return account.ID
}
