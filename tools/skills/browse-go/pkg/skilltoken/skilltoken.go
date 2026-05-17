// Package skilltoken implements per-spawn scoped tokens for browser skill execution.
//
// When $B skill run <name> spawns a script, the script needs a token to call
// back into the daemon. It MUST NOT receive the daemon root token. This package
// mints short-lived tokens bound to a restricted scope set (read+navigate+
// interact+inspect+system, which covers the browser-driving surface). The token
// expires automatically after the spawn timeout + 30s slack.
package skilltoken

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Token holds a minted skill token and its metadata.
type Token struct {
	Token      string
	ClientID   string
	SkillName  string
	SpawnID    string
	Scopes     string // comma-separated scope list
	ExpiresAt  time.Time
}

// DefaultScopes covers the browser-driving command surface (TS read+write equivalent).
const DefaultScopes = "read,navigate,interact,inspect,system"

// TTL slack past the spawn timeout.
var ttlSlack = 30 * time.Second

var (
	registry = make(map[string]*Token) // keyed by token string
	mu       sync.RWMutex
)

// Mint creates a new scoped token for a skill spawn.
// The returned token string is what the child receives via GSTACK_SKILL_TOKEN.
func Mint(skillName, spawnID string, spawnTimeout time.Duration, scopes string) *Token {
	if scopes == "" {
		scopes = DefaultScopes
	}
	tok := make([]byte, 24)
	_, _ = rand.Read(tok)
	tokenStr := "sk_" + hex.EncodeToString(tok)

	clientID := fmt.Sprintf("skill:%s:%s", skillName, spawnID)
	t := &Token{
		Token:     tokenStr,
		ClientID:  clientID,
		SkillName: skillName,
		SpawnID:   spawnID,
		Scopes:    scopes,
		ExpiresAt: time.Now().Add(spawnTimeout + ttlSlack),
	}

	mu.Lock()
	registry[tokenStr] = t
	mu.Unlock()
	return t
}

// Revoke invalidates a token by skill name + spawn ID. Idempotent.
func Revoke(skillName, spawnID string) bool {
	clientID := fmt.Sprintf("skill:%s:%s", skillName, spawnID)
	mu.Lock()
	defer mu.Unlock()
	for k, v := range registry {
		if v.ClientID == clientID {
			delete(registry, k)
			return true
		}
	}
	return false
}

// Validate checks whether a token is valid and returns its metadata.
func Validate(token string) (*Token, bool) {
	mu.RLock()
	t, ok := registry[token]
	mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(t.ExpiresAt) {
		mu.Lock()
		delete(registry, token)
		mu.Unlock()
		return nil, false
	}
	return t, true
}

// ValidateString is a convenience wrapper returning scopes and ok.
func ValidateString(token string) (clientID, scopes string, ok bool) {
	t, ok := Validate(token)
	if !ok {
		return "", "", false
	}
	return t.ClientID, t.Scopes, true
}

// PurgeExpired removes all expired tokens.
func PurgeExpired() {
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()
	for k, v := range registry {
		if now.After(v.ExpiresAt) {
			delete(registry, k)
		}
	}
}

// Count returns the number of active tokens.
func Count() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(registry)
}

// Reset clears the entire registry (for tests).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]*Token)
}

// IsSkillToken returns true if the string looks like a skill token.
func IsSkillToken(s string) bool {
	return strings.HasPrefix(s, "sk_")
}
