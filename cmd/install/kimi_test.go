package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKimiAdapters(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "using-gpowers"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "using-gpowers", "SKILL.md"), []byte(
		"---\nname: using-gpowers\n---\nThis is the preamble."), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "brainstorming"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "brainstorming", "SKILL.md"), []byte(
		"---\nname: brainstorming\ndescription: Explore ideas\n---\nBody here."), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err := generateKimiAdapters(tmp)
	if err != nil {
		t.Fatalf("generateKimiAdapters failed: %v", err)
	}

	routerData, err := os.ReadFile(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers", "SKILL.md"))
	if err != nil {
		t.Fatalf("gpowers router adapter missing: %v", err)
	}
	if !strings.Contains(string(routerData), "This is the preamble.") {
		t.Errorf("gpowers router adapter missing preamble")
	}

	brainData, err := os.ReadFile(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers-brainstorming", "SKILL.md"))
	if err != nil {
		t.Fatalf("gpowers-brainstorming adapter missing: %v", err)
	}
	if !strings.Contains(string(brainData), "Explore ideas (gpowers adapter for Kimi)") {
		t.Errorf("gpowers-brainstorming adapter missing expected description")
	}

	skillsData, err := os.ReadFile(filepath.Join(tmp, "platforms", "kimi", "kimi-skills.json"))
	if err != nil {
		t.Fatalf("kimi-skills.json missing: %v", err)
	}
	if !strings.Contains(string(skillsData), `"gpowers-brainstorming"`) {
		t.Errorf("kimi-skills.json missing gpowers-brainstorming adapter")
	}
}

func TestGenerateKimiAdaptersMissingUsingGpowers(t *testing.T) {
	tmp := t.TempDir()
	err := generateKimiAdapters(tmp)
	if err == nil {
		t.Fatal("expected error when using-gpowers is missing, got nil")
	}
}

func TestGenerateKimiAdaptersGpowersPrefix(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "using-gpowers"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "using-gpowers", "SKILL.md"), []byte(
		"---\nname: using-gpowers\n---\nPreamble."), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "gpowers-test"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "gpowers-test", "SKILL.md"), []byte(
		"---\nname: gpowers-test\ndescription: Test skill\n---\nBody."), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err := generateKimiAdapters(tmp)
	if err != nil {
		t.Fatalf("generateKimiAdapters failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers-test", "SKILL.md")); err != nil {
		t.Fatalf("gpowers-test adapter missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers-gpowers-test", "SKILL.md")); err == nil {
		t.Fatal("unexpected gpowers-gpowers-test adapter created")
	}

	skillsData, err := os.ReadFile(filepath.Join(tmp, "platforms", "kimi", "kimi-skills.json"))
	if err != nil {
		t.Fatalf("kimi-skills.json missing: %v", err)
	}
	if !strings.Contains(string(skillsData), `"gpowers-test"`) {
		t.Errorf("kimi-skills.json missing gpowers-test adapter")
	}
}
