package commands

import (
	"testing"
)

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"goto", "goto"},
		{"GOTO", "goto"},
		{"  Goto  ", "goto"},
		{"setcontent", "load-html"},
		{"set_content", "load-html"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		got := Canonicalize(tc.in)
		if got != tc.want {
			t.Errorf("Canonicalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRegistryBasics(t *testing.T) {
	r := NewRegistry()

	// Check that common commands are registered
	for _, name := range []string{"goto", "text", "click", "screenshot", "tabs", "status", "console", "js"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected command %q to be registered", name)
		}
	}

	// Check descriptions exist
	for _, name := range []string{"goto", "text", "click"} {
		if _, ok := r.GetDesc(name); !ok {
			t.Errorf("expected description for %q", name)
		}
	}

	// Unknown command
	if _, ok := r.Get("not-a-command"); ok {
		t.Error("expected not-a-command to be missing")
	}
}

func TestRegistryPhase3aCommands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"is", "data", "media", "inspect"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 3a command %q to be registered", name)
		}
	}
}

func TestRegistryPhase3bCommands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"dialog-accept", "dialog-dismiss", "cookie", "header", "style", "cleanup", "upload"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 3b command %q to be registered", name)
		}
	}
}

func TestRegistryPhase3cCommands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"prettyscreenshot", "state", "tab-each"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 3c command %q to be registered", name)
		}
	}
}

func TestRegistryPhase3dCommands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"archive", "restart", "responsive", "resume", "connect", "disconnect", "focus"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 3d command %q to be registered", name)
		}
	}
}

func TestRegistryPhase4Commands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"download", "scrape", "ux-audit", "watch"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 4 command %q to be registered", name)
		}
	}
}

func TestRegistryPhase5Commands(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"inbox", "cdp"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected Phase 5 command %q to be registered", name)
		}
	}
}

func TestRegistryExecuteUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(nil, "not-a-command", nil)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
