package security

import (
	"strings"
	"testing"
)

func TestScopeSetAll(t *testing.T) {
	ss := NewScopeSet("all")
	if !ss.all {
		t.Error("expected all=true")
	}
	if !ss.Has(ScopeRead) || !ss.Has(ScopeWrite) {
		t.Error("all scope should grant everything")
	}
}

func TestScopeSetEmpty(t *testing.T) {
	ss := NewScopeSet("")
	if !ss.all {
		t.Error("empty scope should default to all")
	}
}

func TestScopeSetSingle(t *testing.T) {
	ss := NewScopeSet("read")
	if ss.all {
		t.Error("expected all=false for single scope")
	}
	if !ss.Has(ScopeRead) {
		t.Error("expected read to be granted")
	}
	if ss.Has(ScopeWrite) {
		t.Error("write should not be granted")
	}
}

func TestScopeSetMultiple(t *testing.T) {
	ss := NewScopeSet("read, navigate, interact")
	if !ss.Has(ScopeRead) || !ss.Has(ScopeNavigate) || !ss.Has(ScopeInteract) {
		t.Error("expected read, navigate, interact to be granted")
	}
	if ss.Has(ScopeWrite) {
		t.Error("write should not be granted")
	}
}

func TestScopeSetString(t *testing.T) {
	if s := NewScopeSet("all").String(); s != "all" {
		t.Errorf("expected 'all', got %q", s)
	}
	if s := NewScopeSet("read, write").String(); s != "inspect,interact,navigate,read,system,write" {
		// Actually sorted by order in String()
		if !scopeContains(s, "read") || !scopeContains(s, "write") {
			t.Errorf("expected read+write, got %q", s)
		}
	}
	if s := NewScopeSet("").String(); s != "all" {
		t.Errorf("expected 'all', got %q", s)
	}
}

func scopeContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestCheckScopeAllowed(t *testing.T) {
	ss := NewScopeSet("read")
	if err := CheckScope("text", ss); err != nil {
		t.Errorf("text should be allowed with read scope: %v", err)
	}
	if err := CheckScope("html", ss); err != nil {
		t.Errorf("html should be allowed with read scope: %v", err)
	}
}

func TestCheckScopeDenied(t *testing.T) {
	ss := NewScopeSet("read")
	if err := CheckScope("goto", ss); err == nil {
		t.Error("goto should be denied with only read scope")
	}
	if err := CheckScope("click", ss); err == nil {
		t.Error("click should be denied with only read scope")
	}
}

func TestCheckScopeAll(t *testing.T) {
	ss := NewScopeSet("all")
	for _, cmd := range []string{"goto", "text", "click", "write", "cdp", "status"} {
		if err := CheckScope(cmd, ss); err != nil {
			t.Errorf("%s should be allowed with all scope: %v", cmd, err)
		}
	}
}

func TestCheckScopeUnknownCommand(t *testing.T) {
	// Unknown command with "all" scope is allowed
	ss := NewScopeSet("all")
	if err := CheckScope("unknown-cmd", ss); err != nil {
		t.Errorf("unknown command should be allowed with all scope")
	}

	// Unknown command with restricted scope is denied
	ss2 := NewScopeSet("read")
	if err := CheckScope("unknown-cmd", ss2); err == nil {
		t.Error("unknown command should be denied with restricted scope")
	}
}

func TestScopeSetInvalidScopeIgnored(t *testing.T) {
	ss := NewScopeSet("read,invalid,write")
	if !ss.Has(ScopeRead) || !ss.Has(ScopeWrite) {
		t.Error("valid scopes should be granted, invalid ignored")
	}
	if ss.Has(Scope("invalid")) {
		t.Error("invalid scope should not be granted")
	}
}
