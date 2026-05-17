package commands

import (
	"os"
	"path/filepath"
	"testing"

	"browse-go/pkg/browser"
)

func setupBrowserSkillDir(t *testing.T, dir, name string) string {
	t.Helper()
	skillDir := filepath.Join(dir, ".gstack", "browser-skills", name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: ` + name + `
host: example.com
triggers:
  - test trigger
trusted: false
version: 1.0.0
source: human
---

# ` + name + `

Test skill body.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

func TestBrowserSkillUsage(t *testing.T) {
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "skill", []string{"help"})
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if out == "" {
		t.Error("expected usage text")
	}
}

func TestBrowserSkillList(t *testing.T) {
	dir := t.TempDir()
	setupBrowserSkillDir(t, dir, "skill-a")
	setupBrowserSkillDir(t, dir, "skill-b")

	// Override project tier to point to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "skill", []string{"list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !contains(out, "skill-a") || !contains(out, "skill-b") {
		t.Errorf("list output missing skills: %q", out)
	}
}

func TestBrowserSkillListEmpty(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "skill", []string{"list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !contains(out, "No browser-skills") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestBrowserSkillShow(t *testing.T) {
	dir := t.TempDir()
	setupBrowserSkillDir(t, dir, "my-skill")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "skill", []string{"show", "my-skill"})
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !contains(out, "Test skill body") {
		t.Errorf("show output missing body: %q", out)
	}
}

func TestBrowserSkillShowMissing(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "skill", []string{"show", "missing"})
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestBrowserSkillRunMissing(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "skill", []string{"run", "missing"})
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestBrowserSkillRunNoExec(t *testing.T) {
	dir := t.TempDir()
	setupBrowserSkillDir(t, dir, "no-exec")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "skill", []string{"run", "no-exec"})
	if err == nil {
		t.Error("expected error for skill without executable")
	}
}

func TestBrowserSkillTestMissing(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "skill", []string{"test", "missing"})
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestBrowserSkillUnknownSubcommand(t *testing.T) {
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "skill", []string{"nope"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}
