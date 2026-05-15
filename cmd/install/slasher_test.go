package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "basic",
			content: "---\nname: foo\ndescription: \"bar baz\"\nslash: /test\n---\nbody here\n",
			want:    map[string]string{"name": "foo", "description": "bar baz", "slash": "/test"},
		},
		{
			name:    "single-quotes",
			content: "---\ndescription: 'single quoted'\n---\nbody",
			want:    map[string]string{"description": "single quoted"},
		},
		{
			name:    "no-frontmatter",
			content: "just body\n",
			want:    map[string]string{},
		},
		{
			name:    "value-with-colon",
			content: "---\nurl: https://example.com\n---\nbody",
			want:    map[string]string{"url": "https://example.com"},
		},
		{
			name:    "crlf",
			content: "---\r\nname: foo\r\nslash: /test\r\n---\r\nbody",
			want:    map[string]string{"name": "foo", "slash": "/test"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFrontmatter(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseFrontmatter() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestListSlashes(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "test-skill"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "test-skill", "SKILL.md"), []byte(
		"---\nname: test\nslash: /test\nrequires-driver: playwright-cli\n---\nbody"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	slashes, err := listSlashes(tmp)
	if err != nil {
		t.Fatalf("listSlashes failed: %v", err)
	}
	if len(slashes) != 1 {
		t.Fatalf("expected 1 slash, got %d", len(slashes))
	}
	if slashes[0].Slash != "/test" {
		t.Errorf("slash = %q, want /test", slashes[0].Slash)
	}
	if slashes[0].RequiresDriver != "playwright-cli" {
		t.Errorf("driver = %q, want playwright-cli", slashes[0].RequiresDriver)
	}
}

func TestListSlashesEmpty(t *testing.T) {
	tmp := t.TempDir()
	// no skills with slash field
	if err := os.MkdirAll(filepath.Join(tmp, "core", "skills", "no-slash"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "core", "skills", "no-slash", "SKILL.md"), []byte(
		"---\nname: no-slash\n---\nbody"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	slashes, err := listSlashes(tmp)
	if err != nil {
		t.Fatalf("listSlashes failed: %v", err)
	}
	if len(slashes) != 0 {
		t.Fatalf("expected 0 slashes, got %d", len(slashes))
	}
}

func TestReadSkillBody(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "skill.md")
	if err := os.WriteFile(path, []byte("---\nname: x\n---\nbody content\nmore"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	body, err := readSkillBody(path)
	if err != nil {
		t.Fatalf("readSkillBody failed: %v", err)
	}
	if body != "body content\nmore" {
		t.Errorf("body = %q, want 'body content\\nmore'", body)
	}
}

func TestReadSkillBodyMissingFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "missing.md")
	_, err := readSkillBody(path)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadSkillBodyNoFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "skill.md")
	if err := os.WriteFile(path, []byte("no frontmatter here\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := readSkillBody(path)
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}
}
