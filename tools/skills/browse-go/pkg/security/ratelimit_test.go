package security

import (
	"os"
	"testing"
	"time"
)

func TestRateLimiterDisabled(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT", "off")
	defer os.Unsetenv("BROWSE_RATE_LIMIT")

	rl := NewRateLimiter()
	if rl.IsEnabled() {
		t.Error("expected rate limiter to be disabled")
	}
	if err := rl.Allow("test"); err != nil {
		t.Errorf("expected no error when disabled: %v", err)
	}
}

func TestRateLimiterAllowWithinBurst(t *testing.T) {
	// Set a high burst so we don't get rate limited during the test
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "100")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "1000")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	rl := NewRateLimiter()
	if !rl.IsEnabled() {
		t.Error("expected rate limiter to be enabled by default")
	}

	// Should allow 10 requests within burst
	for i := 0; i < 10; i++ {
		if err := rl.Allow("test-key"); err != nil {
			t.Fatalf("request %d should be allowed: %v", i+1, err)
		}
	}
}

func TestRateLimiterBlocksOverBurst(t *testing.T) {
	// Very restrictive: 1/sec, burst of 2
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "1")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "2")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	rl := NewRateLimiter()

	// First 2 should succeed (burst capacity)
	if err := rl.Allow("strict-key"); err != nil {
		t.Fatalf("request 1 should be allowed: %v", err)
	}
	if err := rl.Allow("strict-key"); err != nil {
		t.Fatalf("request 2 should be allowed: %v", err)
	}

	// Third should fail (exhausted burst)
	if err := rl.Allow("strict-key"); err == nil {
		t.Fatal("request 3 should be rate limited")
	}

	// Different key should still work
	if err := rl.Allow("other-key"); err != nil {
		t.Fatalf("different key should be allowed: %v", err)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "100") // fast refill
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "2")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	rl := NewRateLimiter()

	// Exhaust burst
	_ = rl.Allow("refill-key")
	_ = rl.Allow("refill-key")
	if err := rl.Allow("refill-key"); err == nil {
		t.Fatal("expected rate limit after exhausting burst")
	}

	// Wait for refill
	time.Sleep(20 * time.Millisecond) // 100 rps = 1 token per 10ms
	if err := rl.Allow("refill-key"); err != nil {
		t.Fatalf("expected refill to allow request: %v", err)
	}
}

func TestRateLimiterAllowScope(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "100")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "1000")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	rl := NewRateLimiter()
	if err := rl.AllowScope("client-1", "read"); err != nil {
		t.Errorf("expected allow: %v", err)
	}
}

func TestRateLimiterStatus(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "5.5")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "15")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	rl := NewRateLimiter()
	status := rl.Status()
	if status["enabled"] != true {
		t.Error("expected enabled=true")
	}
	if status["rps"] != 5.5 {
		t.Errorf("expected rps=5.5, got %v", status["rps"])
	}
	if status["burst"] != 15 {
		t.Errorf("expected burst=15, got %v", status["burst"])
	}
}

func TestRateLimiterKeyForCommand(t *testing.T) {
	if k := KeyForCommand("client-1", "text"); k != "client-1:text" {
		t.Errorf("expected client-1:text, got %s", k)
	}
	if k := KeyForCommand("", "text"); k != "anonymous:text" {
		t.Errorf("expected anonymous:text, got %s", k)
	}
}
