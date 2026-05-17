package commands

import (
	"os"
	"testing"

	"browse-go/pkg/browser"
	"browse-go/pkg/config"
	"browse-go/pkg/domainskill"
)

func tmpDomainHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("GSTACK_HOME", dir)
	t.Cleanup(func() {
		os.Unsetenv("GSTACK_HOME")
	})
	return dir
}

func TestDomainSkillUsage(t *testing.T) {
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"help"})
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if out == "" {
		t.Error("expected usage text")
	}
}

func TestDomainSkillListEmpty(t *testing.T) {
	tmpDomainHome(t)
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestDomainSkillSaveAndList(t *testing.T) {
	tmpDomainHome(t)

	// Seed storage layer directly
	_, err := domainskill.Write(domainskill.WriteInput{
		Host:        "example.com",
		Body:        "test notes",
		ProjectSlug: config.RemoteSlug(),
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"list"})
	if err != nil {
		t.Fatalf("list after save: %v", err)
	}
	if !contains(out, "example.com") {
		t.Errorf("list output missing example.com: %q", out)
	}
}

func TestDomainSkillShow(t *testing.T) {
	tmpDomainHome(t)

	// Write then auto-promote via RecordUse (requires classifier_score > 0)
	_, err := domainskill.Write(domainskill.WriteInput{
		Host:            "show-test.com",
		Body:            "show me",
		ProjectSlug:     config.RemoteSlug(),
		Source:          "agent",
		ClassifierScore: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_, _ = domainskill.RecordUse("show-test.com", config.RemoteSlug(), false)
	}

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"show", "show-test.com"})
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !contains(out, "show me") {
		t.Errorf("show output missing body: %q", out)
	}
}

func TestDomainSkillShowMissing(t *testing.T) {
	tmpDomainHome(t)
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"show", "nope.com"})
	if err != nil {
		t.Fatalf("show missing: %v", err)
	}
	if !contains(out, "No active skill") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestDomainSkillRm(t *testing.T) {
	tmpDomainHome(t)

	_, err := domainskill.Write(domainskill.WriteInput{
		Host:        "delete-me.com",
		Body:        "temp",
		ProjectSlug: config.RemoteSlug(),
		Source:      "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	bm := browser.NewBrowserManager()
	out, err := r.Execute(bm, "domain-skill", []string{"rm", "delete-me.com"})
	if err != nil {
		t.Fatalf("rm: %v", err)
	}
	if !contains(out, "Tombstoned") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestDomainSkillUnknownSubcommand(t *testing.T) {
	r := NewRegistry()
	bm := browser.NewBrowserManager()
	_, err := r.Execute(bm, "domain-skill", []string{"nope"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
