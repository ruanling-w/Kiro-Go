package proxy

import (
	"kiro-go/config"
	"sync"
	"time"

	"github.com/google/uuid"
)

const grokCLISessionMax = 5000

var grokCLITurns = struct {
	sync.Mutex
	items map[string]grokCLITurnState
}{items: map[string]grokCLITurnState{}}

type grokCLITurnState struct {
	turn int
	used time.Time
}

func grokCLISessionID(account *config.Account, payload *KiroPayload) string {
	if payload != nil && payload.ConversationState.ConversationID != "" {
		return payload.ConversationState.ConversationID
	}
	if account != nil {
		if account.MachineId != "" {
			return account.MachineId
		}
		if account.ID != "" {
			return account.ID
		}
	}
	return uuid.New().String()
}

func countGrokCLIUserTurns(input []map[string]interface{}) int {
	n := 0
	for _, item := range input {
		typ, _ := item["type"].(string)
		if item["role"] == "user" && (typ == "" || typ == "message") {
			n++
		}
	}
	if n < 1 {
		return 1
	}
	return n
}

func resolveGrokCLITurn(session string, input []map[string]interface{}) int {
	fromInput := countGrokCLIUserTurns(input)
	now := time.Now()
	grokCLITurns.Lock()
	defer grokCLITurns.Unlock()
	for id, state := range grokCLITurns.items {
		if now.Sub(state.used) > time.Hour {
			delete(grokCLITurns.items, id)
		}
	}
	previous := grokCLITurns.items[session].turn
	turn := fromInput
	if previous >= turn {
		turn = previous + 1
	}
	if len(grokCLITurns.items) >= grokCLISessionMax {
		for id := range grokCLITurns.items {
			delete(grokCLITurns.items, id)
			break
		}
	}
	grokCLITurns.items[session] = grokCLITurnState{turn: turn, used: now}
	return turn
}
