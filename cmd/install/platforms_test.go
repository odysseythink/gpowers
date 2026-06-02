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
	os.MkdirAll(filepath.Join(tmpHome, ".kimi-code"), 0755)
	os.MkdirAll(filepath.Join(tmpHome, ".qoder"), 0755)
	detected := detectPlatforms()
	foundKimi := false
	foundKimiCode := false
	foundQoder := false
	for _, p := range detected {
		if p == "kimi" {
			foundKimi = true
		}
		if p == "kimi-code" {
			foundKimiCode = true
		}
		if p == "qoder" {
			foundQoder = true
		}
		if p == "claude-code" {
			t.Errorf("expected claude-code NOT to be detected, got %v", detected)
		}
	}
	if !foundKimi {
		t.Errorf("expected kimi to be detected, got %v", detected)
	}
	if !foundKimiCode {
		t.Errorf("expected kimi-code to be detected, got %v", detected)
	}
	if !foundQoder {
		t.Errorf("expected qoder to be detected, got %v", detected)
	}
}

func TestDetectKimiCodeDistinctFromKimi(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Only ~/.kimi-code present, not ~/.kimi: the two must not be conflated.
	os.MkdirAll(filepath.Join(tmpHome, ".kimi-code"), 0755)
	detected := detectPlatforms()
	foundKimi := false
	foundKimiCode := false
	for _, p := range detected {
		if p == "kimi" {
			foundKimi = true
		}
		if p == "kimi-code" {
			foundKimiCode = true
		}
	}
	if foundKimi {
		t.Errorf("expected kimi NOT to be detected when only ~/.kimi-code exists, got %v", detected)
	}
	if !foundKimiCode {
		t.Errorf("expected kimi-code to be detected, got %v", detected)
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

func TestGenerateQoderAdapters(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "core", "skills", "using-gpowers"), 0755)
	os.MkdirAll(filepath.Join(tmp, "core", "hooks"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "hooks", "hooks.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmp, "core", "skills", "using-gpowers", "SKILL.md"), []byte("---\nname: using-gpowers\ndescription: entry\n---\nUsing gpowers body\n"), 0644)

	os.MkdirAll(filepath.Join(tmp, "core", "skills", "brainstorming"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "skills", "brainstorming", "SKILL.md"), []byte("---\nname: brainstorming\ndescription: brainstorm skill\n---\nBrainstorm body\n"), 0644)

	err := generateQoderAdapters(tmp)
	if err != nil {
		t.Fatalf("generateQoderAdapters failed: %v", err)
	}

	routerPath := filepath.Join(tmp, "platforms", "qoder", "adapters", "gpowers", "SKILL.md")
	if _, err := os.Stat(routerPath); err != nil {
		t.Errorf("gpowers router adapter not generated")
	}

	adapterPath := filepath.Join(tmp, "platforms", "qoder", "adapters", "gpowers-brainstorming", "SKILL.md")
	if _, err := os.Stat(adapterPath); err != nil {
		t.Errorf("gpowers-brainstorming adapter not generated")
	}

	manifestPath := filepath.Join(tmp, "platforms", "qoder", "qoder-skills.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("qoder-skills.json not generated")
	}
}
