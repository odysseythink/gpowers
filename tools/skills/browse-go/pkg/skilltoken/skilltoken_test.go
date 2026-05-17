package skilltoken

import (
	"strings"
	"testing"
	"time"
)

func TestMintAndValidate(t *testing.T) {
	Reset()

	tok := Mint("my-skill", "spawn-1", 60*time.Second, "")
	if tok == nil {
		t.Fatal("expected token, got nil")
	}
	if !strings.HasPrefix(tok.Token, "sk_") {
		t.Errorf("token should start with sk_: %s", tok.Token)
	}
	if tok.ClientID != "skill:my-skill:spawn-1" {
		t.Errorf("clientID mismatch: %s", tok.ClientID)
	}
	if tok.Scopes != DefaultScopes {
		t.Errorf("scopes should default to %s, got %s", DefaultScopes, tok.Scopes)
	}

	// Validate by token string
	v, ok := Validate(tok.Token)
	if !ok {
		t.Fatal("expected token to validate")
	}
	if v.ClientID != tok.ClientID {
		t.Error("validated clientID mismatch")
	}

	cid, scopes, ok := ValidateString(tok.Token)
	if !ok {
		t.Fatal("ValidateString should succeed")
	}
	if cid != tok.ClientID {
		t.Error("ValidateString clientID mismatch")
	}
	if scopes != DefaultScopes {
		t.Error("ValidateString scopes mismatch")
	}
}

func TestRevoke(t *testing.T) {
	Reset()

	tok := Mint("my-skill", "spawn-1", 60*time.Second, "")
	if _, ok := Validate(tok.Token); !ok {
		t.Fatal("token should be valid before revoke")
	}

	Revoke("my-skill", "spawn-1")
	if _, ok := Validate(tok.Token); ok {
		t.Error("token should be invalid after revoke")
	}
}

func TestExpiry(t *testing.T) {
	Reset()
	oldSlack := ttlSlack
	ttlSlack = 1 * time.Millisecond
	defer func() { ttlSlack = oldSlack }()

	tok := Mint("my-skill", "spawn-1", 1*time.Millisecond, "")
	time.Sleep(5 * time.Millisecond)

	if _, ok := Validate(tok.Token); ok {
		t.Error("token should be expired")
	}
}

func TestIsSkillToken(t *testing.T) {
	if !IsSkillToken("sk_abc123") {
		t.Error("sk_ prefix should be recognized")
	}
	if IsSkillToken("root_abc123") {
		t.Error("non-sk prefix should be rejected")
	}
}

func TestPurgeExpired(t *testing.T) {
	Reset()
	oldSlack := ttlSlack
	ttlSlack = 1 * time.Millisecond
	defer func() { ttlSlack = oldSlack }()

	Mint("a", "1", 1*time.Millisecond, "")
	Mint("b", "2", 1*time.Hour, "")

	time.Sleep(5 * time.Millisecond)
	PurgeExpired()

	if Count() != 1 {
		t.Errorf("expected 1 token after purge, got %d", Count())
	}
}
