package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPlatforms(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	os.MkdirAll(filepath.Join(tmpHome, ".kimi"), 0755)
	detected := detectPlatforms()
	foundKimi := false
	for _, p := range detected {
		if p == "kimi" {
			foundKimi = true
		}
		if p == "claude-code" {
			t.Errorf("expected claude-code NOT to be detected, got %v", detected)
		}
	}
	if !foundKimi {
		t.Errorf("expected kimi to be detected, got %v", detected)
	}
}

func TestGeneratePlatformManifest(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "platforms"), 0755)
	os.MkdirAll(filepath.Join(tmp, "core", "hooks"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "hooks", "hooks.json"), []byte("{}"), 0644)
	shapes := `{"platforms":{"claude-code":{"manifest_filename":".claude-plugin/plugin.json","command_dir":"commands","command_filename_pattern":"{slash}.md","supports_hooks":true,"namespace_mode":"plugin-scoped","install_link_target":"~/.claude/plugins/gpowers"}}}`
	os.WriteFile(filepath.Join(tmp, "platforms", "_platform-shapes.json"), []byte(shapes), 0644)

	err := generatePlatformManifest("claude-code", tmp, []string{"core", "roles", "tools"})
	if err != nil {
		t.Fatalf("generatePlatformManifest failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "platforms", "claude-code", ".claude-plugin", "plugin.json")); err != nil {
		t.Errorf("plugin.json not generated")
	}
	if _, err := os.Stat(filepath.Join(tmp, "platforms", "claude-code", "hooks.json")); err != nil {
		t.Errorf("hooks.json not copied")
	}
}
