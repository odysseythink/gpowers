package browserskill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupSkillDir(t *testing.T, dir, name string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: ` + name + `
host: example.com
triggers:
  - scrape frontpage
  - click all links
trusted: true
version: 1.0.0
source: human
---

# ` + name + `

This skill does something useful.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// ─── Frontmatter parsing ────────────────────────────────────────

func TestParseFrontmatter(t *testing.T) {
	content := `---
name: test-skill
host: news.ycombinator.com
triggers:
  - scrape hn frontpage
  - click all links
trusted: true
version: 1.2.3
source: agent
---

# Test Skill

Some markdown here.
`
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("parseFrontmatter: %v", err)
	}
	if fm.Name != "test-skill" {
		t.Errorf("name = %q, want test-skill", fm.Name)
	}
	if fm.Host != "news.ycombinator.com" {
		t.Errorf("host = %q", fm.Host)
	}
	if len(fm.Triggers) != 2 {
		t.Errorf("triggers = %v, want 2 items", fm.Triggers)
	}
	if !fm.Trusted {
		t.Error("expected trusted = true")
	}
	if fm.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", fm.Version)
	}
	if fm.Source != "agent" {
		t.Errorf("source = %q, want agent", fm.Source)
	}
	if !strings.Contains(body, "# Test Skill") {
		t.Errorf("body missing expected content: %q", body)
	}
}

func TestParseFrontmatterMissingDelimiter(t *testing.T) {
	_, _, err := parseFrontmatter("no frontmatter here")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseFrontmatterUnterminated(t *testing.T) {
	_, _, err := parseFrontmatter("---\nname: foo\n")
	if err == nil {
		t.Error("expected error for unterminated frontmatter")
	}
}

func TestUnquote(t *testing.T) {
	if unquote(`"hello"`) != "hello" {
		t.Error("unquote failed for double quotes")
	}
	if unquote(`'hello'`) != "hello" {
		t.Error("unquote failed for single quotes")
	}
	if unquote(`hello`) != "hello" {
		t.Error("unquote failed for bare string")
	}
}

// ─── Listing ────────────────────────────────────────────────────

func TestList(t *testing.T) {
	dir := t.TempDir()
	setupSkillDir(t, dir, "skill-a")
	setupSkillDir(t, dir, "skill-b")

	tiers := TierPaths{Project: dir}
	skills := List(tiers)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
	if skills[0].Name != "skill-a" {
		t.Errorf("first skill = %q", skills[0].Name)
	}
	if skills[1].Name != "skill-b" {
		t.Errorf("second skill = %q", skills[1].Name)
	}
}

func TestListIgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "not-a-skill"), []byte("x"), 0o644)

	tiers := TierPaths{Project: dir}
	skills := List(tiers)
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestListSkipsTombstones(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".tombstones", "old-skill"), 0o755)

	tiers := TierPaths{Project: dir}
	skills := List(tiers)
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestListTierPriority(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	// Create same skill in both tiers
	setupSkillDir(t, projectDir, "shared")
	setupSkillDir(t, globalDir, "shared")

	tiers := TierPaths{Project: projectDir, Global: globalDir}
	skills := List(tiers)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill (project shadows global), got %d", len(skills))
	}
	if skills[0].Tier != TierProject {
		t.Errorf("tier = %q, want project", skills[0].Tier)
	}
}

// ─── Read ───────────────────────────────────────────────────────

func TestReadFound(t *testing.T) {
	dir := t.TempDir()
	setupSkillDir(t, dir, "my-skill")

	tiers := TierPaths{Project: dir}
	skill, err := Read("my-skill", tiers)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if skill.Name != "my-skill" {
		t.Errorf("name = %q", skill.Name)
	}
	if skill.Frontmatter.Host != "example.com" {
		t.Errorf("host = %q", skill.Frontmatter.Host)
	}
}

func TestReadNotFound(t *testing.T) {
	dir := t.TempDir()
	tiers := TierPaths{Project: dir}
	_, err := Read("missing", tiers)
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestReadFallsBack(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	setupSkillDir(t, globalDir, "global-only")

	tiers := TierPaths{Project: projectDir, Global: globalDir}
	skill, err := Read("global-only", tiers)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if skill.Tier != TierGlobal {
		t.Errorf("tier = %q, want global", skill.Tier)
	}
}

// ─── Tombstone ──────────────────────────────────────────────────

func TestTombstone(t *testing.T) {
	dir := t.TempDir()
	setupSkillDir(t, dir, "to-delete")

	tiers := TierPaths{Project: dir}
	if err := Tombstone("to-delete", TierProject, tiers); err != nil {
		t.Fatalf("Tombstone: %v", err)
	}

	_, err := Read("to-delete", tiers)
	if err == nil {
		t.Error("expected skill to be tombstoned")
	}

	// Verify tombstone dir exists
	entries, _ := os.ReadDir(filepath.Join(dir, ".tombstones"))
	if len(entries) == 0 {
		t.Error("expected tombstone directory")
	}
}

func TestTombstoneBundledRejection(t *testing.T) {
	dir := t.TempDir()
	tiers := TierPaths{Bundled: dir}
	if err := Tombstone("x", TierBundled, tiers); err == nil {
		t.Error("expected error tombstoning bundled skill")
	}
}

func TestTombstoneMissing(t *testing.T) {
	dir := t.TempDir()
	tiers := TierPaths{Project: dir}
	if err := Tombstone("missing", TierProject, tiers); err == nil {
		t.Error("expected error tombstoning missing skill")
	}
}

// ─── Edge cases ─────────────────────────────────────────────────

func TestReadSkillDirMissingSkillMd(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty-skill"), 0o755)

	tiers := TierPaths{Project: dir}
	_, err := Read("empty-skill", tiers)
	if err == nil {
		t.Error("expected error for skill without SKILL.md")
	}
}

func TestParseFrontmatterEmptyList(t *testing.T) {
	content := `---
name: empty-list
host: example.com
trusted: false
---
body
`
	fm, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Triggers) != 0 {
		t.Errorf("expected 0 triggers, got %d", len(fm.Triggers))
	}
}

func TestDefaultTierPaths(t *testing.T) {
	paths := DefaultTierPaths()
	if paths.Global == "" {
		t.Error("global tier path empty")
	}
	if paths.Project == "" {
		t.Error("project tier path empty")
	}
}

