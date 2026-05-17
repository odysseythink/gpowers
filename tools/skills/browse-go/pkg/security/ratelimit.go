package security

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter provides per-key token-bucket rate limiting.
type RateLimiter struct {
	enabled    bool
	rate       float64 // tokens per second
	burst      int     // max bucket capacity
	mu         sync.RWMutex
	buckets    map[string]*bucket
	lastClean  time.Time
}

type bucket struct {
	tokens    float64
	lastRefill time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a rate limiter from environment.
// BROWSE_RATE_LIMIT=off disables rate limiting (default: on).
// BROWSE_RATE_LIMIT_RPS sets tokens per second (default: 10).
// BROWSE_RATE_LIMIT_BURST sets burst capacity (default: 20).
func NewRateLimiter() *RateLimiter {
	enabled := strings.ToLower(os.Getenv("BROWSE_RATE_LIMIT")) != "off"
	rate := 10.0
	burst := 20

	if v := os.Getenv("BROWSE_RATE_LIMIT_RPS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			rate = f
		}
	}
	if v := os.Getenv("BROWSE_RATE_LIMIT_BURST"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			burst = i
		}
	}

	return &RateLimiter{
		enabled:   enabled,
		rate:      rate,
		burst:     burst,
		buckets:   make(map[string]*bucket),
		lastClean: time.Now(),
	}
}

// Allow checks whether a request from the given key is allowed.
// Returns nil if allowed, error if rate limited.
func (rl *RateLimiter) Allow(key string) error {
	if !rl.enabled {
		return nil
	}

	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(rl.burst), lastRefill: time.Now()}
		rl.buckets[key] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastRefill = now

	if b.tokens < 1.0 {
		return fmt.Errorf("rate limited: try again in %.1fs (limit: %.1f/sec, burst: %d)",
			(1.0-b.tokens)/rl.rate, rl.rate, rl.burst)
	}

	b.tokens--
	return nil
}

// AllowScope is a convenience that builds a key from client + scope.
func (rl *RateLimiter) AllowScope(clientID, scope string) error {
	if !rl.enabled {
		return nil
	}
	key := clientID + ":" + scope
	return rl.Allow(key)
}

// Status returns the current rate limit configuration.
func (rl *RateLimiter) Status() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return map[string]interface{}{
		"enabled": rl.enabled,
		"rps":     rl.rate,
		"burst":   rl.burst,
		"buckets": len(rl.buckets),
	}
}

// IsEnabled returns whether rate limiting is active.
func (rl *RateLimiter) IsEnabled() bool {
	return rl.enabled
}

// KeyForCommand builds a rate-limit key for a command invocation.
// Uses clientID + command name as the key.
func KeyForCommand(clientID, command string) string {
	if clientID == "" {
		clientID = "anonymous"
	}
	return clientID + ":" + command
}
