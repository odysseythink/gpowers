package tokenregistry

import (
	"strings"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	r := New()
	if err := r.Init("root123"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if r.Root() != "root123" {
		t.Errorf("expected root123, got %s", r.Root())
	}
	// Idempotent
	if err := r.Init("root123"); err != nil {
		t.Fatalf("Idempotent Init failed: %v", err)
	}
	// Mismatch
	if err := r.Init("different"); err == nil {
		t.Error("expected error for mismatched root token")
	}
}

func TestIsRoot(t *testing.T) {
	r := New()
	_ = r.Init("secret-root")
	if !r.IsRoot("secret-root") {
		t.Error("expected IsRoot true")
	}
	if r.IsRoot("wrong") {
		t.Error("expected IsRoot false")
	}
	// Timing-safe: different length
	if r.IsRoot("secret-root-longer") {
		t.Error("expected IsRoot false for different length")
	}
}

func TestCreateToken(t *testing.T) {
	r := New()
	_ = r.Init("root")

	twentyFour := 86400
	tok, err := r.CreateToken(CreateTokenOptions{
		ClientID:       "alice",
		Scopes:         []ScopeCategory{ScopeRead, ScopeWrite},
		ExpiresSeconds: &twentyFour,
	})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if tok.ClientID != "alice" {
		t.Errorf("clientId: %s", tok.ClientID)
	}
	if !strings.HasPrefix(tok.Token, "gsk_sess_") {
		t.Errorf("bad token prefix: %s", tok.Token)
	}
	if tok.ExpiresAt == nil {
		t.Error("expected expiresAt")
	}
}

func TestCreateTokenDefaults(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, err := r.CreateToken(CreateTokenOptions{ClientID: "bob"})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if tok.TabPolicy != "own-only" {
		t.Errorf("default tabPolicy: %s", tok.TabPolicy)
	}
	if tok.RateLimit != 10 {
		t.Errorf("default rateLimit: %d", tok.RateLimit)
	}
}

func TestValidate(t *testing.T) {
	r := New()
	_ = r.Init("root")
	rootInfo := r.Validate("root")
	if rootInfo == nil || rootInfo.ClientID != "root" {
		t.Error("expected root info")
	}

	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "eve"})
	info := r.Validate(tok.Token)
	if info == nil || info.ClientID != "eve" {
		t.Error("expected valid token")
	}
	if r.Validate("nope") != nil {
		t.Error("expected invalid token")
	}
}

func TestValidateExpired(t *testing.T) {
	r := New()
	_ = r.Init("root")
	minusOne := -1
	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "exp", ExpiresSeconds: &minusOne})
	if r.Validate(tok.Token) != nil {
		t.Error("expected expired token to be invalid")
	}
}

func TestCheckScope(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{
		ClientID: "scoped",
		Scopes:   []ScopeCategory{ScopeRead},
	})
	if !r.CheckScope(tok, "text") {
		t.Error("expected text allowed")
	}
	if r.CheckScope(tok, "goto") {
		t.Error("expected goto denied")
	}
	// chain with meta scope
	tokMeta, _ := r.CreateToken(CreateTokenOptions{
		ClientID: "meta",
		Scopes:   []ScopeCategory{ScopeMeta},
	})
	if !r.CheckScope(tokMeta, "chain") {
		t.Error("expected chain allowed with meta")
	}
}

func TestCheckDomain(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{
		ClientID: "dom",
		Scopes:   []ScopeCategory{ScopeRead},
		Domains:  []string{"*.example.com", "google.com"},
	})
	if !r.CheckDomain(tok, "https://sub.example.com/path") {
		t.Error("expected *.example.com match")
	}
	if !r.CheckDomain(tok, "https://google.com") {
		t.Error("expected google.com match")
	}
	if r.CheckDomain(tok, "https://evil.com") {
		t.Error("expected evil.com denied")
	}
	// No restrictions = allow all
	tok2, _ := r.CreateToken(CreateTokenOptions{ClientID: "open", Scopes: []ScopeCategory{ScopeRead}})
	if !r.CheckDomain(tok2, "https://anything.com") {
		t.Error("expected no restriction = allow all")
	}
}

func TestCheckRate(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "ratelimit", Scopes: []ScopeCategory{ScopeRead}, RateLimit: 2})
	for i := 0; i < 2; i++ {
		allowed, _ := r.CheckRate(tok)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	allowed, retry := r.CheckRate(tok)
	if allowed {
		t.Error("expected rate limited")
	}
	if retry <= 0 {
		t.Error("expected positive retryAfter")
	}
}

func TestRevoke(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "revoke-me"})
	if r.Validate(tok.Token) == nil {
		t.Fatal("token should be valid before revoke")
	}
	if !r.Revoke("revoke-me") {
		t.Error("expected revoke success")
	}
	if r.Revoke("revoke-me") {
		t.Error("expected revoke false for already-revoked")
	}
	if r.Validate(tok.Token) != nil {
		t.Error("token should be invalid after revoke")
	}
}

func TestRotateRoot(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "alice"})
	newRoot := r.RotateRoot()
	if newRoot == "root" {
		t.Error("expected new root token")
	}
	if r.Validate(tok.Token) != nil {
		t.Error("expected scoped tokens invalidated after rotate")
	}
}

func TestSetupKey(t *testing.T) {
	r := New()
	_ = r.Init("root")
	setup, err := r.CreateSetupKey(CreateTokenOptions{ClientID: "remote1", Scopes: []ScopeCategory{ScopeRead}})
	if err != nil {
		t.Fatalf("CreateSetupKey failed: %v", err)
	}
	if setup.Type != "setup" {
		t.Errorf("type: %s", setup.Type)
	}
	if !strings.HasPrefix(setup.Token, "gsk_setup_") {
		t.Errorf("bad prefix: %s", setup.Token)
	}

	// Exchange
	session, err := r.ExchangeSetupKey(setup.Token, nil)
	if err != nil {
		t.Fatalf("ExchangeSetupKey failed: %v", err)
	}
	if session.Type != "session" {
		t.Errorf("expected session type, got %s", session.Type)
	}

	// Idempotent: same key, 0 commands → same session
	session2, err := r.ExchangeSetupKey(setup.Token, nil)
	if err != nil {
		t.Fatalf("Idempotent exchange failed: %v", err)
	}
	if session2.Token != session.Token {
		t.Error("expected same session token on idempotent exchange")
	}

	// After command consumed, no longer idempotent
	r.RecordCommand(session.Token)
	_, err = r.ExchangeSetupKey(setup.Token, nil)
	if err == nil {
		t.Error("expected failure after command consumed")
	}
}

func TestSetupKeyExpired(t *testing.T) {
	r := New()
	_ = r.Init("root")
	setup, _ := r.CreateSetupKey(CreateTokenOptions{ClientID: "old"})
	// Manually expire
	past := time.Now().Add(-10 * time.Minute)
	setup.ExpiresAt = &past
	_, err := r.ExchangeSetupKey(setup.Token, nil)
	if err == nil {
		t.Error("expected expired setup key to fail")
	}
}

func TestListTokens(t *testing.T) {
	r := New()
	_ = r.Init("root")
	r.CreateToken(CreateTokenOptions{ClientID: "a"})
	r.CreateToken(CreateTokenOptions{ClientID: "b"})
	list := r.ListTokens()
	if len(list) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(list))
	}
}

func TestSerializeRestore(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{
		ClientID: "persist",
		Scopes:   []ScopeCategory{ScopeRead},
		Domains:  []string{"example.com"},
	})
	state := r.Serialize()
	if len(state.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(state.Agents))
	}

	r2 := New()
	_ = r2.Init("root2")
	r2.Restore(state)
	restored := r2.Validate(tok.Token)
	if restored == nil {
		t.Fatal("expected restored token to be valid")
	}
	if restored.ClientID != "persist" {
		t.Errorf("clientId: %s", restored.ClientID)
	}
	if len(restored.Domains) != 1 || restored.Domains[0] != "example.com" {
		t.Errorf("domains: %v", restored.Domains)
	}
}

func TestConnectRateLimit(t *testing.T) {
	r := New()
	for i := 0; i < 300; i++ {
		if !r.CheckConnectRateLimit() {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if r.CheckConnectRateLimit() {
		t.Error("expected 301st request to be rate limited")
	}
	r.ResetConnectRateLimit()
	if !r.CheckConnectRateLimit() {
		t.Error("expected request after reset to be allowed")
	}
}

func TestRecordCommand(t *testing.T) {
	r := New()
	_ = r.Init("root")
	tok, _ := r.CreateToken(CreateTokenOptions{ClientID: "count"})
	r.RecordCommand(tok.Token)
	r.RecordCommand(tok.Token)
	info := r.Validate(tok.Token)
	if info.CommandCount != 2 {
		t.Errorf("commandCount: %d", info.CommandCount)
	}
}
