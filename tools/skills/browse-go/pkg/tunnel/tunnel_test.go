package tunnel

import (
	"testing"
)

func TestIsTunnelAllowed(t *testing.T) {
	if !IsTunnelAllowed("goto") {
		t.Error("expected goto allowed")
	}
	if !IsTunnelAllowed("connect") {
		t.Error("expected connect allowed")
	}
	if IsTunnelAllowed("eval") {
		t.Error("expected eval denied")
	}
	if IsTunnelAllowed("js") {
		t.Error("expected js denied")
	}
}

func TestTunnelCommandsCoverage(t *testing.T) {
	// Ensure all expected commands are in the allowlist
	expected := []string{
		"connect", "goto", "back", "forward", "reload",
		"text", "html", "links", "forms", "accessibility",
		"snapshot", "click", "fill", "scroll", "wait",
		"screenshot", "status", "tabs", "tab", "newtab",
		"closetab", "stop",
	}
	for _, cmd := range expected {
		if !IsTunnelAllowed(cmd) {
			t.Errorf("expected %s to be allowed", cmd)
		}
	}
}
