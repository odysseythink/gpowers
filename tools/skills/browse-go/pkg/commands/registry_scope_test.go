package commands

import (
	"os"
	"testing"

	"browse-go/pkg/browser"
	"browse-go/pkg/security"
)

func TestExecuteWithOptsScopeAllowed(t *testing.T) {
	bm := browser.NewBrowserManager()
	r := NewRegistry()

	// "status" requires "system" scope
	ss := security.NewScopeSet("system")
	_, err := r.ExecuteWithOpts(bm, "status", nil, &ExecuteOpts{ScopeSet: ss})
	if err != nil {
		t.Fatalf("status should be allowed with system scope: %v", err)
	}
}

func TestExecuteWithOptsScopeDenied(t *testing.T) {
	bm := browser.NewBrowserManager()
	r := NewRegistry()

	// "goto" requires "navigate" scope
	ss := security.NewScopeSet("read")
	_, err := r.ExecuteWithOpts(bm, "goto", []string{"https://example.com"}, &ExecuteOpts{ScopeSet: ss})
	if err == nil {
		t.Fatal("goto should be denied with only read scope")
	}
}

func TestExecuteWithOptsRateLimit(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "1")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "1")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	bm := browser.NewBrowserManager()
	r := NewRegistry()

	ss := security.NewScopeSet("system")
	opts := &ExecuteOpts{ScopeSet: ss, ClientID: "test-client"}

	// First request should succeed
	_, err := r.ExecuteWithOpts(bm, "status", nil, opts)
	if err != nil {
		t.Fatalf("first request should succeed: %v", err)
	}

	// Second request should be rate limited
	_, err = r.ExecuteWithOpts(bm, "status", nil, opts)
	if err == nil {
		t.Fatal("second request should be rate limited")
	}
}

func TestExecuteWithOptsAllScope(t *testing.T) {
	bm := browser.NewBrowserManager()
	r := NewRegistry()

	ss := security.NewScopeSet("all")
	// Should allow any command (that doesn't need a session)
	for _, cmd := range []string{"status", "help", "tabs"} {
		_, err := r.ExecuteWithOpts(bm, cmd, nil, &ExecuteOpts{ScopeSet: ss})
		if err != nil {
			t.Fatalf("%s should be allowed with all scope: %v", cmd, err)
		}
	}
}

func TestExecuteWithOptsDefaultScope(t *testing.T) {
	bm := browser.NewBrowserManager()
	r := NewRegistry()

	// nil opts or nil scope should default to "all"
	_, err := r.ExecuteWithOpts(bm, "status", nil, nil)
	if err != nil {
		t.Fatalf("nil opts should default to all scope: %v", err)
	}

	_, err = r.ExecuteWithOpts(bm, "status", nil, &ExecuteOpts{})
	if err != nil {
		t.Fatalf("empty opts should default to all scope: %v", err)
	}
}

func TestRateLimitStatus(t *testing.T) {
	r := NewRegistry()
	status := r.RateLimitStatus()
	if status == nil {
		t.Fatal("expected rate limit status")
	}
	if _, ok := status["enabled"]; !ok {
		t.Fatal("expected enabled field in status")
	}
}
