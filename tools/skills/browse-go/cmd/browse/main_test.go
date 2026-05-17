package main

import (
	"os"
	"testing"

	"browse-go/pkg/browser"
)

func TestExtractGlobalFlags(t *testing.T) {
	// Empty args
	flags, err := extractGlobalFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags.Args) != 0 {
		t.Errorf("expected empty args, got %v", flags.Args)
	}
	if flags.ProxyURL != "" {
		t.Error("expected empty proxy")
	}
	if flags.Headed {
		t.Error("expected headed=false")
	}
}

func TestExtractGlobalFlagsProxy(t *testing.T) {
	flags, err := extractGlobalFlags([]string{"--proxy", "socks5://localhost:1080", "goto", "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags.ProxyURL != "socks5://localhost:1080" {
		t.Errorf("proxy: %s", flags.ProxyURL)
	}
	if len(flags.Args) != 2 || flags.Args[0] != "goto" {
		t.Errorf("args: %v", flags.Args)
	}
}

func TestExtractGlobalFlagsHeaded(t *testing.T) {
	flags, err := extractGlobalFlags([]string{"--headed", "status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !flags.Headed {
		t.Error("expected headed=true")
	}
	if len(flags.Args) != 1 || flags.Args[0] != "status" {
		t.Errorf("args: %v", flags.Args)
	}
}

func TestExtractGlobalFlagsProxyEquals(t *testing.T) {
	flags, err := extractGlobalFlags([]string{"--proxy=http://proxy.example.com:8080", "status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags.ProxyURL != "http://proxy.example.com:8080" {
		t.Errorf("proxy: %s", flags.ProxyURL)
	}
}

func TestExtractGlobalFlagsMissingValue(t *testing.T) {
	_, err := extractGlobalFlags([]string{"--proxy"})
	if err == nil {
		t.Error("expected error for missing proxy value")
	}
}

func TestComputeConfigHash(t *testing.T) {
	h1 := computeConfigHash("socks5://localhost:1080", false)
	h2 := computeConfigHash("socks5://localhost:1080", false)
	if h1 != h2 {
		t.Error("expected same hash for same config")
	}
	h3 := computeConfigHash("socks5://localhost:1080", true)
	if h1 == h3 {
		t.Error("expected different hash for different headed")
	}
}

func TestRedactProxyURL(t *testing.T) {
	if redactProxyURL("") != "" {
		t.Error("expected empty")
	}
	if redactProxyURL("http://user:pass@proxy.com:8080") != "http://user:***@proxy.com:8080" {
		t.Errorf("got: %s", redactProxyURL("http://user:pass@proxy.com:8080"))
	}
}

func TestExtractTabID(t *testing.T) {
	id, rest := extractTabID([]string{"--tab-id", "5", "goto", "https://example.com"})
	if id != 5 {
		t.Errorf("tabId: %d", id)
	}
	if len(rest) != 2 || rest[0] != "goto" {
		t.Errorf("rest: %v", rest)
	}
}

func TestExtractTabIDMissing(t *testing.T) {
	id, rest := extractTabID([]string{"goto", "https://example.com"})
	if id != 0 {
		t.Errorf("tabId: %d", id)
	}
	if len(rest) != 2 {
		t.Errorf("rest: %v", rest)
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"--control"}, "--control") {
		t.Error("expected true")
	}
	if hasFlag([]string{"--other"}, "--control") {
		t.Error("expected false")
	}
}

func TestFlagValue(t *testing.T) {
	v := flagValue([]string{"--client-id", "alice", "--control"}, "--client-id")
	if v != "alice" {
		t.Errorf("flagValue: %s", v)
	}
	if flagValue([]string{"--control"}, "--client-id") != "" {
		t.Error("expected empty")
	}
}

// ─── Proxy Setup Tests ────────────────────────────────────

func TestSetupProxyNoEnv(t *testing.T) {
	os.Unsetenv("BROWSE_PROXY_URL")
	bm := browser.NewBrowserManager()
	bridge, err := setupProxy(bm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge != nil {
		t.Error("expected nil bridge when no proxy is set")
	}
}

func TestSetupProxyHTTP(t *testing.T) {
	os.Setenv("BROWSE_PROXY_URL", "http://proxy.example.com:8080")
	defer os.Unsetenv("BROWSE_PROXY_URL")
	bm := browser.NewBrowserManager()
	bridge, err := setupProxy(bm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge != nil {
		t.Error("expected nil bridge for HTTP proxy (no bridge needed)")
	}
	// ProxyConfig is set on bm but not directly observable without reflection.
	// The fact that it doesn't error is the main contract.
}

func TestSetupProxySOCKS5NoAuth(t *testing.T) {
	os.Setenv("BROWSE_PROXY_URL", "socks5://localhost:1080")
	defer os.Unsetenv("BROWSE_PROXY_URL")
	bm := browser.NewBrowserManager()
	bridge, err := setupProxy(bm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge != nil {
		t.Error("expected nil bridge for unauthenticated SOCKS5")
	}
}

func TestSetupProxyInvalidURL(t *testing.T) {
	os.Setenv("BROWSE_PROXY_URL", "://not-a-url")
	defer os.Unsetenv("BROWSE_PROXY_URL")
	bm := browser.NewBrowserManager()
	_, err := setupProxy(bm)
	if err == nil {
		t.Fatal("expected error for invalid proxy URL")
	}
}
