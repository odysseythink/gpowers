package domainskill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"browse-go/pkg/config"
)

func tmpHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	oldHome := os.Getenv("GSTACK_HOME")
	os.Setenv("GSTACK_HOME", dir)
	t.Cleanup(func() { os.Setenv("GSTACK_HOME", oldHome) })
	return dir
}

// ─── Hostname normalization ─────────────────────────────────────

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://EXAMPLE.COM/path", "example.com"},
		{"http://www.example.com", "example.com"},
		{"www.EXAMPLE.COM:8080", "example.com"},
		{"example.com?q=1", "example.com"},
		{"example.com#frag", "example.com"},
		{"  EXAMPLE.COM  ", "example.com"},
		{"sub.example.com", "sub.example.com"},
	}
	for _, tc := range tests {
		got := NormalizeHost(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDeriveHostFromURL(t *testing.T) {
	if _, err := DeriveHostFromURL(""); err == nil {
		t.Error("expected error for empty URL")
	}
	if _, err := DeriveHostFromURL("about:blank"); err == nil {
		t.Error("expected error for about:blank")
	}
	host, err := DeriveHostFromURL("https://example.com/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
}

// ─── Write + Read ───────────────────────────────────────────────

func TestWriteAndRead(t *testing.T) {
	tmpHome(t)

	row, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "# Notes\nSome markdown here.",
		ProjectSlug: "test-project",
		Source:      "agent",
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if row.State != StateQuarantined {
		t.Errorf("new skill state = %q, want quarantined", row.State)
	}
	if row.Version != 1 {
		t.Errorf("version = %d, want 1", row.Version)
	}

	// Read back
	result, err := Read("example.com", "test-project")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result != nil {
		t.Error("quarantined skill should not be returned by Read")
	}
}

func TestReadProjectShadowsGlobal(t *testing.T) {
	tmpHome(t)

	// Create global skill
	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "global body",
		ProjectSlug: "other-project",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Promote to global manually by writing directly to global file
	gr := Row{
		Type: "domain", Host: "example.com", Scope: ScopeGlobal, State: StateGlobal,
		Body: "global body", Version: 1, Source: "agent",
		CreatedTS: "2024-01-01T00:00:00Z", UpdatedTS: "2024-01-01T00:00:00Z",
	}
	_ = appendRow(globalFile(), gr)

	// Create project skill
	_, err = Write(WriteInput{
		Host:        "example.com",
		Body:        "project body",
		ProjectSlug: "this-project",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Project should not shadow global because project skill is quarantined
	result, err := Read("example.com", "this-project")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Source != ScopeGlobal {
		t.Errorf("expected global fallback, got %+v", result)
	}
}

// ─── RecordUse + Auto-promote ───────────────────────────────────

func TestRecordUsePromotesAfterThreshold(t *testing.T) {
	tmpHome(t)

	// Write with classifier score > 0 (required for auto-promote)
	_, err := Write(WriteInput{
		Host:            "example.com",
		Body:            "notes",
		ProjectSlug:     "test",
		Source:          "agent",
		ClassifierScore: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Use 3 times without flags
	for i := 0; i < 3; i++ {
		_, err = RecordUse("example.com", "test", false)
		if err != nil {
			t.Fatalf("RecordUse %d failed: %v", i, err)
		}
	}

	// Should now be active
	result, err := Read("example.com", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected active skill after 3 uses")
	}
	if result.Row.State != StateActive {
		t.Errorf("state = %q, want active", result.Row.State)
	}
}

func TestRecordUseNoPromoteWithFlags(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:            "example.com",
		Body:            "notes",
		ProjectSlug:     "test",
		Source:          "agent",
		ClassifierScore: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Use 3 times but with a flag on second use
	RecordUse("example.com", "test", false)
	RecordUse("example.com", "test", true) // flagged!
	RecordUse("example.com", "test", false)

	result, err := Read("example.com", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("should not promote when flagged")
	}
}

func TestRecordUseNoPromoteWithZeroScore(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:            "example.com",
		Body:            "notes",
		ProjectSlug:     "test",
		Source:          "agent",
		ClassifierScore: 0, // zero score blocks auto-promote
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		RecordUse("example.com", "test", false)
	}

	result, err := Read("example.com", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("should not promote with classifier_score == 0")
	}
}

// ─── PromoteToGlobal ────────────────────────────────────────────

func TestPromoteToGlobal(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:            "example.com",
		Body:            "notes",
		ProjectSlug:     "test",
		Source:          "agent",
		ClassifierScore: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Manually set active
	for i := 0; i < 3; i++ {
		RecordUse("example.com", "test", false)
	}

	globalRow, err := PromoteToGlobal("example.com", "test")
	if err != nil {
		t.Fatalf("PromoteToGlobal failed: %v", err)
	}
	if globalRow.Scope != ScopeGlobal {
		t.Errorf("promoted scope = %q, want global", globalRow.Scope)
	}
	if globalRow.State != StateGlobal {
		t.Errorf("promoted state = %q, want global", globalRow.State)
	}

	// Read still returns project layer first (project shadows global)
	result, err := Read("example.com", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Source != ScopeProject {
		t.Error("expected project skill to shadow global after promotion")
	}
}

func TestPromoteToGlobalRequiresActive(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "notes",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = PromoteToGlobal("example.com", "test")
	if err == nil {
		t.Error("expected error promoting quarantined skill")
	}
}

// ─── Rollback ───────────────────────────────────────────────────

func TestRollback(t *testing.T) {
	tmpHome(t)

	// Write v1
	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "first version",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Write v2
	_, err = Write(WriteInput{
		Host:        "example.com",
		Body:        "second version",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	restored, err := Rollback("example.com", "test", ScopeProject)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if !strings.Contains(restored.Body, "first version") {
		t.Errorf("rollback body = %q, expected first version", restored.Body)
	}
}

func TestRollbackNeedsTwoVersions(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "only version",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Rollback("example.com", "test", ScopeProject)
	if err == nil {
		t.Error("expected error rolling back with only 1 version")
	}
}

// ─── Delete + Tombstone ─────────────────────────────────────────

func TestDeleteTombstones(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "notes",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Delete("example.com", "test", ScopeProject); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, err := List("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Project) != 0 {
		t.Errorf("expected 0 project skills after delete, got %d", len(list.Project))
	}
}

func TestDeleteMissingSkill(t *testing.T) {
	tmpHome(t)
	if err := Delete("nope.com", "test", ScopeProject); err == nil {
		t.Error("expected error deleting nonexistent skill")
	}
}

// ─── List ───────────────────────────────────────────────────────

func TestList(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "a.com",
		Body:        "notes a",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Write(WriteInput{
		Host:        "b.com",
		Body:        "notes b",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := List("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Project) != 2 {
		t.Errorf("expected 2 project skills, got %d", len(list.Project))
	}
}

// ─── Classifier block ───────────────────────────────────────────

func TestWriteBlocksHighScore(t *testing.T) {
	tmpHome(t)
	_, err := Write(WriteInput{
		Host:            "example.com",
		Body:            "notes",
		ProjectSlug:     "test",
		Source:          "agent",
		ClassifierScore: 0.9, // above 0.85 threshold
	})
	if err == nil {
		t.Error("expected error for high classifier score")
	}
}

// ─── Compactor ──────────────────────────────────────────────────

func TestCompactor(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "v1",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Write(WriteInput{
		Host:        "example.com",
		Body:        "v2",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	file := projectFile("test")
	before, _ := os.ReadFile(file)
	beforeLines := strings.Count(string(before), "\n")

	if err := Compactor(file); err != nil {
		t.Fatalf("Compactor failed: %v", err)
	}

	after, _ := os.ReadFile(file)
	afterLines := strings.Count(string(after), "\n")
	if afterLines >= beforeLines {
		t.Errorf("compactor did not reduce lines: before=%d after=%d", beforeLines, afterLines)
	}
}

// ─── Version increments ─────────────────────────────────────────

func TestVersionIncrement(t *testing.T) {
	tmpHome(t)

	r1, _ := Write(WriteInput{Host: "x.com", Body: "a", ProjectSlug: "p", Source: "agent"})
	r2, _ := Write(WriteInput{Host: "x.com", Body: "b", ProjectSlug: "p", Source: "agent"})

	if r1.Version != 1 {
		t.Errorf("v1 = %d, want 1", r1.Version)
	}
	if r2.Version != 2 {
		t.Errorf("v2 = %d, want 2", r2.Version)
	}
}

// ─── config.Home integration ────────────────────────────────────

func TestConfigHome(t *testing.T) {
	dir := tmpHome(t)
	if config.Home() != dir {
		t.Errorf("config.Home() = %q, want %q", config.Home(), dir)
	}
}

// Test file permissions on created files
func TestFilePermissions(t *testing.T) {
	tmpHome(t)
	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "test",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	file := projectFile("test")
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	mode := info.Mode().Perm()
	// File should be readable/writable by owner
	if mode&0o600 != 0o600 {
		t.Errorf("file permissions = %o, expected at least 0o600", mode)
	}
}

// Test empty body rejection
func TestWriteEmptyBody(t *testing.T) {
	tmpHome(t)
	// Empty body is allowed at storage layer (caller validates)
	_, err := Write(WriteInput{
		Host:        "example.com",
		Body:        "",
		ProjectSlug: "test",
		Source:      "agent",
	})
	if err != nil {
		t.Fatalf("empty body should be allowed at storage layer: %v", err)
	}
}

// Test multiple project isolation
func TestProjectIsolation(t *testing.T) {
	tmpHome(t)

	_, err := Write(WriteInput{
		Host:        "shared.com",
		Body:        "project A notes",
		ProjectSlug: "project-a",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Write(WriteInput{
		Host:        "shared.com",
		Body:        "project B notes",
		ProjectSlug: "project-b",
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	listA, _ := List("project-a")
	listB, _ := List("project-b")

	if len(listA.Project) != 1 || listA.Project[0].Body != "project A notes" {
		t.Error("project-a isolation broken")
	}
	if len(listB.Project) != 1 || listB.Project[0].Body != "project B notes" {
		t.Error("project-b isolation broken")
	}
}

// Test SHA256 consistency
func TestSHA256Consistency(t *testing.T) {
	tmpHome(t)

	r1, _ := Write(WriteInput{
		Host:        "example.com",
		Body:        "same body",
		ProjectSlug: "test",
		Source:      "agent",
	})
	r2, _ := Write(WriteInput{
		Host:        "example.com",
		Body:        "same body",
		ProjectSlug: "test",
		Source:      "agent",
	})

	if r1.SHA256 != r2.SHA256 {
		t.Error("SHA256 should be identical for identical bodies")
	}
	if r1.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
}

// Test that the global file path is correct
func TestGlobalFilePath(t *testing.T) {
	dir := tmpHome(t)
	want := filepath.Join(dir, "global-domain-skills.jsonl")
	got := globalFile()
	if got != want {
		t.Errorf("globalFile() = %q, want %q", got, want)
	}
}

// Test Read returns nil for missing skill
func TestReadMissing(t *testing.T) {
	tmpHome(t)
	result, err := Read("nonexistent.com", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("expected nil for missing skill")
	}
}
