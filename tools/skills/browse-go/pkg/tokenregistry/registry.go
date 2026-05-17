// Package tokenregistry implements per-agent scoped tokens for multi-agent
// browser access, ported from TypeScript token-registry.ts.
//
// Architecture:
//   Root token → POST /token → scoped sub-tokens
//   POST /pair → setup key → POST /connect → session token
//
// Security invariants:
//   1. Only root token can mint sub-tokens
//   2. admin scope denied by default
//   3. chain command scope-checks each subcommand individually
//   4. Root token never in connection strings
package tokenregistry

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ScopeCategory is a TS-aligned permission category.
type ScopeCategory string

const (
	ScopeRead    ScopeCategory = "read"
	ScopeWrite   ScopeCategory = "write"
	ScopeAdmin   ScopeCategory = "admin"
	ScopeMeta    ScopeCategory = "meta"
	ScopeControl ScopeCategory = "control"
)

// SCOPE_READ defines commands safe for read-only agents.
var SCOPE_READ = map[string]bool{
	"snapshot": true, "text": true, "html": true, "links": true,
	"forms": true, "accessibility": true, "console": true,
	"network": true, "perf": true, "dialog": true, "is": true,
	"inspect": true, "url": true, "tabs": true, "status": true,
	"screenshot": true, "pdf": true, "css": true, "attrs": true,
	"media": true, "data": true,
}

// SCOPE_WRITE defines commands that modify page state or navigate.
var SCOPE_WRITE = map[string]bool{
	"goto": true, "back": true, "forward": true, "reload": true,
	"load-html": true,
	"click": true, "fill": true, "select": true, "hover": true,
	"type": true, "press": true, "scroll": true, "wait": true,
	"upload": true, "viewport": true, "newtab": true, "closetab": true,
	"dialog-accept": true, "dialog-dismiss": true,
	"download": true, "scrape": true, "archive": true,
}

// SCOPE_ADMIN defines page-level power tools.
var SCOPE_ADMIN = map[string]bool{
	"eval": true, "js": true, "cookies": true, "storage": true,
	"cookie": true, "cookie-import": true, "cookie-import-browser": true,
	"header": true, "useragent": true,
	"style": true, "cleanup": true, "prettyscreenshot": true,
}

// SCOPE_CONTROL defines browser-wide destructive commands.
var SCOPE_CONTROL = map[string]bool{
	"state": true, "handoff": true, "resume": true, "stop": true,
	"restart": true, "connect": true, "disconnect": true,
}

// SCOPE_META defines meta commands.
var SCOPE_META = map[string]bool{
	"tab": true, "diff": true, "frame": true, "responsive": true,
	"snapshot": true, "watch": true, "inbox": true, "focus": true,
}

var scopeMap = map[ScopeCategory]map[string]bool{
	ScopeRead:    SCOPE_READ,
	ScopeWrite:   SCOPE_WRITE,
	ScopeAdmin:   SCOPE_ADMIN,
	ScopeControl: SCOPE_CONTROL,
	ScopeMeta:    SCOPE_META,
}

var validScopes = []ScopeCategory{ScopeRead, ScopeWrite, ScopeAdmin, ScopeMeta, ScopeControl}

// TokenInfo holds metadata for a scoped token.
type TokenInfo struct {
	Token              string            `json:"token"`
	ClientID           string            `json:"clientId"`
	Type               string            `json:"type"` // "session" or "setup"
	Scopes             []ScopeCategory   `json:"scopes"`
	Domains            []string          `json:"domains,omitempty"` // glob patterns
	TabPolicy          string            `json:"tabPolicy"`         // "own-only" or "shared"
	RateLimit          int               `json:"rateLimit"`         // req/s (0 = unlimited)
	ExpiresAt          *time.Time        `json:"expiresAt,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
	UsesRemaining      *int              `json:"usesRemaining,omitempty"` // for setup keys
	IssuedSessionToken string            `json:"issuedSessionToken,omitempty"`
	CommandCount       int               `json:"commandCount"`
}

// CreateTokenOptions controls how a session token is minted.
type CreateTokenOptions struct {
	ClientID       string
	Scopes         []ScopeCategory
	Domains        []string
	TabPolicy      string
	RateLimit      int
	ExpiresSeconds *int // nil = default 86400, use negative for never
}

// RegistryState is the serialized form for persistence.
type RegistryState struct {
	Agents map[string]TokenInfo `json:"agents"` // key = clientId
}

// Registry holds all tokens and rate-limit buckets.
type Registry struct {
	mu        sync.RWMutex
	tokens    map[string]*TokenInfo
	rootToken string

	rateBuckets      map[string]*rateBucket
	rateBucketsMu    sync.Mutex

	connectAttempts  []connectAttempt
	connectMu        sync.Mutex
}

type rateBucket struct {
	count       int
	windowStart time.Time
}

type connectAttempt struct {
	ts time.Time
}

// New creates a Registry. Call Init before use.
func New() *Registry {
	return &Registry{
		tokens:      make(map[string]*TokenInfo),
		rateBuckets: make(map[string]*rateBucket),
	}
}

// Init sets the root token. Idempotent for same token; errors on mismatch.
func (r *Registry) Init(root string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rootToken != "" && !constantTimeEqual(r.rootToken, root) {
		return fmt.Errorf("token-registry already initialized with a different token")
	}
	r.rootToken = root
	return nil
}

// Root returns the root token.
func (r *Registry) Root() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rootToken
}

// IsRoot checks whether a token is the root token (constant-time).
func (r *Registry) IsRoot(token string) bool {
	r.mu.RLock()
	root := r.rootToken
	r.mu.RUnlock()
	if root == "" {
		return false
	}
	return constantTimeEqual(root, token)
}

// constantTimeEqual does a constant-time string comparison.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CreateToken mints a scoped session token (root-only).
func (r *Registry) CreateToken(opts CreateTokenOptions) (*TokenInfo, error) {
	// Validate scopes
	for _, s := range opts.Scopes {
		valid := false
		for _, vs := range validScopes {
			if s == vs {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid scope: %q", s)
		}
	}
	if opts.RateLimit < 0 {
		return nil, fmt.Errorf("rateLimit must be >= 0")
	}

	var expires *time.Time
	if opts.ExpiresSeconds != nil {
		if *opts.ExpiresSeconds >= 0 {
			t := time.Now().Add(time.Duration(*opts.ExpiresSeconds) * time.Second)
			expires = &t
		} else {
			// negative = already expired
			t := time.Now().Add(-1 * time.Second)
			expires = &t
		}
	} else {
		// default 24h
		t := time.Now().Add(24 * time.Hour)
		expires = &t
	}

	tokenStr := generateToken("gsk_sess_")
	now := time.Now()

	info := &TokenInfo{
		Token:      tokenStr,
		ClientID:   opts.ClientID,
		Type:       "session",
		Scopes:     opts.Scopes,
		Domains:    opts.Domains,
		TabPolicy:  firstNonEmpty(opts.TabPolicy, "own-only"),
		RateLimit:  firstNonEmptyInt(opts.RateLimit, 10),
		ExpiresAt:  expires,
		CreatedAt:  now,
		CommandCount: 0,
	}

	r.mu.Lock()
	// Overwrite existing session for same clientId
	for t, existing := range r.tokens {
		if existing.ClientID == opts.ClientID && existing.Type == "session" {
			delete(r.tokens, t)
			break
		}
	}
	r.tokens[tokenStr] = info
	r.mu.Unlock()

	return info, nil
}

// CreateSetupKey creates a one-time setup key for the pair-agent ceremony.
func (r *Registry) CreateSetupKey(opts CreateTokenOptions) (*TokenInfo, error) {
	tokenStr := generateToken("gsk_setup_")
	now := time.Now()
	expires := now.Add(5 * time.Minute)
	uses := 1

	info := &TokenInfo{
		Token:         tokenStr,
		ClientID:      firstNonEmpty(opts.ClientID, fmt.Sprintf("remote-%d", now.UnixMilli())),
		Type:          "setup",
		Scopes:        opts.Scopes,
		Domains:       opts.Domains,
		TabPolicy:     firstNonEmpty(opts.TabPolicy, "own-only"),
		RateLimit:     firstNonEmptyInt(opts.RateLimit, 10),
		ExpiresAt:     &expires,
		CreatedAt:     now,
		UsesRemaining: &uses,
		CommandCount:  0,
	}

	r.mu.Lock()
	r.tokens[tokenStr] = info
	r.mu.Unlock()

	return info, nil
}

// ExchangeSetupKey converts a setup key into a session token.
// Idempotent: if the same key is presented again and the prior session
// has 0 commands, returns the same session token.
func (r *Registry) ExchangeSetupKey(setupKey string, sessionExpiresSeconds *int) (*TokenInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	setup, ok := r.tokens[setupKey]
	if !ok || setup.Type != "setup" {
		return nil, fmt.Errorf("invalid or unknown setup key")
	}

	// Check expiry
	if setup.ExpiresAt != nil && setup.ExpiresAt.Before(time.Now()) {
		delete(r.tokens, setupKey)
		return nil, fmt.Errorf("setup key expired")
	}

	// Idempotent: already exchanged but session has 0 commands
	if setup.UsesRemaining != nil && *setup.UsesRemaining == 0 {
		if setup.IssuedSessionToken != "" {
			existing, ok := r.tokens[setup.IssuedSessionToken]
			if ok && existing.CommandCount == 0 {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("setup key already used")
	}

	// Consume setup key
	zero := 0
	setup.UsesRemaining = &zero

	// Create session token
	session, err := r.createTokenLocked(CreateTokenOptions{
		ClientID:       setup.ClientID,
		Scopes:         setup.Scopes,
		Domains:        setup.Domains,
		TabPolicy:      setup.TabPolicy,
		RateLimit:      setup.RateLimit,
		ExpiresSeconds: sessionExpiresSeconds,
	})
	if err != nil {
		return nil, err
	}

	setup.IssuedSessionToken = session.Token
	return session, nil
}

// createTokenLocked is the locked variant used internally.
func (r *Registry) createTokenLocked(opts CreateTokenOptions) (*TokenInfo, error) {
	var expires *time.Time
	if opts.ExpiresSeconds != nil {
		if *opts.ExpiresSeconds >= 0 {
			t := time.Now().Add(time.Duration(*opts.ExpiresSeconds) * time.Second)
			expires = &t
		}
	} else {
		t := time.Now().Add(24 * time.Hour)
		expires = &t
	}

	tokenStr := generateToken("gsk_sess_")
	now := time.Now()

	info := &TokenInfo{
		Token:      tokenStr,
		ClientID:   opts.ClientID,
		Type:       "session",
		Scopes:     opts.Scopes,
		Domains:    opts.Domains,
		TabPolicy:  firstNonEmpty(opts.TabPolicy, "own-only"),
		RateLimit:  firstNonEmptyInt(opts.RateLimit, 10),
		ExpiresAt:  expires,
		CreatedAt:  now,
		CommandCount: 0,
	}

	for t, existing := range r.tokens {
		if existing.ClientID == opts.ClientID && existing.Type == "session" {
			delete(r.tokens, t)
			break
		}
	}
	r.tokens[tokenStr] = info
	return info, nil
}

// Validate checks a token and returns its info if valid.
// Returns nil for expired, revoked, or unknown tokens.
func (r *Registry) Validate(token string) *TokenInfo {
	if r.IsRoot(token) {
		return &TokenInfo{
			Token:     token,
			ClientID:  "root",
			Type:      "session",
			Scopes:    []ScopeCategory{ScopeRead, ScopeWrite, ScopeAdmin, ScopeMeta, ScopeControl},
			TabPolicy: "shared",
			RateLimit: 0,
		}
	}

	r.mu.RLock()
	info, ok := r.tokens[token]
	r.mu.RUnlock()
	if !ok {
		return nil
	}

	// Check expiry
	if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
		r.mu.Lock()
		delete(r.tokens, token)
		r.mu.Unlock()
		return nil
	}

	return info
}

// CheckScope returns whether a command is allowed by the token's scopes.
func (r *Registry) CheckScope(info *TokenInfo, command string) bool {
	if info.ClientID == "root" {
		return true
	}
	// chain is special: allowed if meta scope, subcommands checked at dispatch
	if command == "chain" {
		for _, s := range info.Scopes {
			if s == ScopeMeta {
				return true
			}
		}
		return false
	}
	for _, s := range info.Scopes {
		if m, ok := scopeMap[s]; ok && m[command] {
			return true
		}
	}
	return false
}

// CheckDomain returns whether a URL is allowed by domain restrictions.
func (r *Registry) CheckDomain(info *TokenInfo, rawURL string) bool {
	if info.ClientID == "root" {
		return true
	}
	if len(info.Domains) == 0 {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	for _, pattern := range info.Domains {
		if matchDomainGlob(hostname, pattern) {
			return true
		}
	}
	return false
}

func matchDomainGlob(hostname, pattern string) bool {
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // .example.com
		return strings.HasSuffix(hostname, suffix) || hostname == pattern[2:]
	}
	return hostname == pattern
}

// CheckRate returns whether the client is within rate limit.
func (r *Registry) CheckRate(info *TokenInfo) (allowed bool, retryAfter time.Duration) {
	if info.ClientID == "root" || info.RateLimit <= 0 {
		return true, 0
	}

	r.rateBucketsMu.Lock()
	defer r.rateBucketsMu.Unlock()

	now := time.Now()
	bucket, ok := r.rateBuckets[info.ClientID]
	if !ok || now.Sub(bucket.windowStart) >= time.Second {
		r.rateBuckets[info.ClientID] = &rateBucket{count: 1, windowStart: now}
		return true, 0
	}
	if bucket.count >= info.RateLimit {
		retry := time.Second - now.Sub(bucket.windowStart)
		if retry < 100*time.Millisecond {
			retry = 100 * time.Millisecond
		}
		return false, retry
	}
	bucket.count++
	return true, 0
}

// RecordCommand increments the command count for a token.
func (r *Registry) RecordCommand(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.tokens[token]; ok {
		info.CommandCount++
	}
}

// Revoke removes all tokens for a client ID. Returns true if found.
func (r *Registry) Revoke(clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for t, info := range r.tokens {
		if info.ClientID == clientID {
			delete(r.tokens, t)
			r.rateBucketsMu.Lock()
			delete(r.rateBuckets, clientID)
			r.rateBucketsMu.Unlock()
			return true
		}
	}
	return false
}

// RotateRoot generates a new root token and invalidates all scoped tokens.
func (r *Registry) RotateRoot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rootToken = generateToken("")
	r.tokens = make(map[string]*TokenInfo)
	r.rateBucketsMu.Lock()
	r.rateBuckets = make(map[string]*rateBucket)
	r.rateBucketsMu.Unlock()
	return r.rootToken
}

// ListTokens returns all active session tokens.
func (r *Registry) ListTokens() []*TokenInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var result []*TokenInfo
	for token, info := range r.tokens {
		if info.ExpiresAt != nil && info.ExpiresAt.Before(now) {
			delete(r.tokens, token)
			continue
		}
		if info.Type == "session" {
			result = append(result, info)
		}
	}
	return result
}

// Serialize returns the registry state for persistence.
func (r *Registry) Serialize() RegistryState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make(map[string]TokenInfo)
	for _, info := range r.tokens {
		if info.Type != "session" {
			continue
		}
		// strip commandCount
		copyInfo := *info
		copyInfo.CommandCount = 0
		agents[info.ClientID] = copyInfo
	}
	return RegistryState{Agents: agents}
}

// Restore loads tokens from persisted state.
func (r *Registry) Restore(state RegistryState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for clientID, data := range state.Agents {
		if data.ExpiresAt != nil && data.ExpiresAt.Before(now) {
			continue
		}
		data.ClientID = clientID
		data.CommandCount = 0
		r.tokens[data.Token] = &data
	}
}

// CheckConnectRateLimit is a global rate limiter for /connect flood protection.
func (r *Registry) CheckConnectRateLimit() bool {
	r.connectMu.Lock()
	defer r.connectMu.Unlock()

	now := time.Now()
	windowStart := now.Add(-60 * time.Second)
	// Filter old attempts
	var keep []connectAttempt
	for _, a := range r.connectAttempts {
		if a.ts.After(windowStart) {
			keep = append(keep, a)
		}
	}
	r.connectAttempts = keep

	if len(r.connectAttempts) >= 300 {
		return false
	}
	r.connectAttempts = append(r.connectAttempts, connectAttempt{ts: now})
	return true
}

// ResetConnectRateLimit clears the connect rate limiter (for tests).
func (r *Registry) ResetConnectRateLimit() {
	r.connectMu.Lock()
	defer r.connectMu.Unlock()
	r.connectAttempts = nil
}

// Reset clears the entire registry (for tests).
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rootToken = ""
	r.tokens = make(map[string]*TokenInfo)
	r.rateBucketsMu.Lock()
	r.rateBuckets = make(map[string]*rateBucket)
	r.rateBucketsMu.Unlock()
}

// ToScopeString converts TokenInfo scopes to a comma-separated string.
func (info *TokenInfo) ToScopeString() string {
	if info.ClientID == "root" {
		return "all"
	}
	var parts []string
	for _, s := range info.Scopes {
		parts = append(parts, string(s))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}

// MarshalJSON for TokenInfo handles nil ExpiresAt cleanly.
func (info TokenInfo) MarshalJSON() ([]byte, error) {
	type Alias TokenInfo
	aux := &struct {
		ExpiresAt *string `json:"expiresAt,omitempty"`
	}{
		ExpiresAt: nil,
	}
	if info.ExpiresAt != nil {
		s := info.ExpiresAt.Format(time.RFC3339Nano)
		aux.ExpiresAt = &s
	}
	// Use custom struct to avoid infinite recursion
	return json.Marshal(struct {
		Token              string          `json:"token"`
		ClientID           string          `json:"clientId"`
		Type               string          `json:"type"`
		Scopes             []ScopeCategory `json:"scopes"`
		Domains            []string        `json:"domains,omitempty"`
		TabPolicy          string          `json:"tabPolicy"`
		RateLimit          int             `json:"rateLimit"`
		ExpiresAt          *string         `json:"expiresAt,omitempty"`
		CreatedAt          time.Time       `json:"createdAt"`
		UsesRemaining      *int            `json:"usesRemaining,omitempty"`
		IssuedSessionToken string          `json:"issuedSessionToken,omitempty"`
		CommandCount       int             `json:"commandCount"`
	}{
		Token:              info.Token,
		ClientID:           info.ClientID,
		Type:               info.Type,
		Scopes:             info.Scopes,
		Domains:            info.Domains,
		TabPolicy:          info.TabPolicy,
		RateLimit:          info.RateLimit,
		ExpiresAt:          aux.ExpiresAt,
		CreatedAt:          info.CreatedAt,
		UsesRemaining:      info.UsesRemaining,
		IssuedSessionToken: info.IssuedSessionToken,
		CommandCount:       info.CommandCount,
	})
}

func generateToken(prefix string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func firstNonEmptyInt(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}
