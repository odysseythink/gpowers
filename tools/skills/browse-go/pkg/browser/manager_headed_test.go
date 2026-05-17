package browser

import (
	"os"
	"path/filepath"
	"testing"

	"browse-go/pkg/config"
)

func TestWriteAuthJSON(t *testing.T) {
	home := config.Home()
	authFile := filepath.Join(home, ".auth.json")
	// Clean up before and after
	_ = os.Remove(authFile)
	defer os.Remove(authFile)

	writeAuthJSON("test-token-123", 34567)

	data, err := os.ReadFile(authFile)
	if err != nil {
		t.Fatalf("failed to read auth file: %v", err)
	}
	content := string(data)
	if !contains(content, "test-token-123") {
		t.Fatalf("expected token in auth file, got: %s", content)
	}
	if !contains(content, "34567") {
		t.Fatalf("expected port in auth file, got: %s", content)
	}
}

func TestGetConnectionModeDefault(t *testing.T) {
	bm := NewBrowserManager()
	if bm.GetConnectionMode() != "launched" {
		t.Fatalf("expected default mode 'launched', got %s", bm.GetConnectionMode())
	}
}

func TestSetOnDisconnect(t *testing.T) {
	bm := NewBrowserManager()
	called := false
	bm.SetOnDisconnect(func() {
		called = true
	})
	fn := bm.OnDisconnect()
	if fn == nil {
		t.Fatal("expected onDisconnect callback to be set")
	}
	fn()
	if !called {
		t.Fatal("expected onDisconnect callback to be called")
	}
}

func TestIsCustomChromium(t *testing.T) {
	t.Setenv("GSTACK_CHROMIUM_KIND", "")
	t.Setenv("GSTACK_CHROMIUM_PATH", "")
	if IsCustomChromium() {
		t.Fatal("expected false when no env set")
	}

	t.Setenv("GSTACK_CHROMIUM_KIND", "custom-extension-baked")
	if !IsCustomChromium() {
		t.Fatal("expected true for custom-extension-baked")
	}
}

func TestFindExtensionPathEmpty(t *testing.T) {
	t.Setenv("BROWSE_EXTENSIONS_DIR", "")
	// This may or may not find an extension depending on the system.
	// We just verify it doesn't panic.
	_ = FindExtensionPath()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
