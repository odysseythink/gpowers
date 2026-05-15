package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateManifest(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	initial := `{"version":"0.0.1","installed_modules":null,"install_location":null,"installed_at":null}`
	if err := os.WriteFile(manifestPath, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err := updateManifest(manifestPath, "/tmp/gpowers", []string{"core", "roles"})
	if err != nil {
		t.Fatalf("updateManifest failed: %v", err)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m["install_location"] != "/tmp/gpowers" {
		t.Errorf("install_location = %q, want %q", m["install_location"], "/tmp/gpowers")
	}
	mods, ok := m["installed_modules"].([]interface{})
	if !ok || len(mods) != 2 {
		t.Errorf("installed_modules = %v, want 2 items", m["installed_modules"])
	}
	if m["installed_at"] == "" {
		t.Errorf("installed_at = %q, want non-empty string", m["installed_at"])
	}
	found := map[string]bool{"core": false, "roles": false}
	for _, mod := range mods {
		if s, ok := mod.(string); ok {
			found[s] = true
		}
	}
	for name, present := range found {
		if !present {
			t.Errorf("installed_modules missing %q", name)
		}
	}
}

func TestUpdateManifestMissingFile(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "nonexistent.json")
	err := updateManifest(manifestPath, "/tmp/gpowers", []string{"core"})
	if err == nil {
		t.Fatal("expected error for missing manifest file, got nil")
	}
}

func TestGenerateSkillsJSON(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "test-skill"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "test-skill", "SKILL.md"), []byte(
		"---\nname: test-skill\ndescription: A test skill\n---\nbody"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err := generateSkillsJSON(tmp)
	if err != nil {
		t.Fatalf("generateSkillsJSON failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "skills.json"))
	if err != nil {
		t.Fatalf("ReadFile skills.json failed: %v", err)
	}
	var result struct {
		Skills []struct {
			Name        string `json:"name"`
			Module      string `json:"module"`
			Description string `json:"description"`
			Path        string `json:"path"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "test-skill" {
		t.Errorf("name = %q, want %q", result.Skills[0].Name, "test-skill")
	}
	if result.Skills[0].Module != "core" {
		t.Errorf("module = %q, want %q", result.Skills[0].Module, "core")
	}
	if result.Skills[0].Description != "A test skill" {
		t.Errorf("description = %q, want %q", result.Skills[0].Description, "A test skill")
	}
	wantPath := "core/skills/test-skill/SKILL.md"
	if result.Skills[0].Path != wantPath {
		t.Errorf("path = %q, want %q", result.Skills[0].Path, wantPath)
	}
}

func TestGenerateSkillsJSONEmptyHome(t *testing.T) {
	tmp := t.TempDir()
	err := generateSkillsJSON(tmp)
	if err != nil {
		t.Fatalf("generateSkillsJSON failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "skills.json"))
	if err != nil {
		t.Fatalf("ReadFile skills.json failed: %v", err)
	}
	var result struct {
		Skills []interface{} `json:"skills"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(result.Skills))
	}
}
