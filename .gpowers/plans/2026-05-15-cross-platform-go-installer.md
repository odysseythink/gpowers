# Cross-Platform Go Installer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `install.bat` and `install` bash scripts with a single Go program that is fully cross-platform, self-contained, and produces identical installation behavior.

**Architecture:** A single `package main` CLI under `cmd/install/` split into focused files. Zero external dependencies. Built-in JSON/Markdown processing. Cross-platform symlinks/junctions/deep-copy.

**Tech Stack:** Go 1.22+ (standard library only)

---

## File Structure

| File | Responsibility |
|---|---|
| `cmd/install/main.go` | Entry point, orchestration, business disclaimer prompt |
| `cmd/install/flags.go` | CLI flag parsing, `Options` struct, module/platform list helpers |
| `cmd/install/utils.go` | Home directory, path expansion (`~`), recursive deep-copy, self-file detection |
| `cmd/install/staging.go` | Copy or symlink source files into install location |
| `cmd/install/manifest.go` | Read/write `manifest.json`, generate `skills.json` |
| `cmd/install/slasher.go` | Parse Markdown frontmatter, discover slash commands |
| `cmd/install/platforms.go` | Auto-detect platforms, generate generic platform manifests, register platforms |
| `cmd/install/kimi.go` | Generate Kimi adapters and `kimi-skills.json` |

---

### Task 1: Initialize Go Module and Utilities

**Files:**
- Create: `go.mod`
- Create: `cmd/install/utils.go`
- Create: `cmd/install/utils_test.go`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
go mod init github.com/odysseythink/gpowers
```

Expected: `go.mod` created in repository root.

- [ ] **Step 2: Write `cmd/install/utils.go`**

```go
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func homeDir() string {
	if runtime.GOOS == "windows" {
		if hd := os.Getenv("USERPROFILE"); hd != "" {
			return hd
		}
	}
	if hd := os.Getenv("HOME"); hd != "" {
		return hd
	}
	return "."
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		p = filepath.Join(homeDir(), p[1:])
	}
	return filepath.Clean(p)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Write `cmd/install/utils_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home := homeDir()
	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"/abs/path", filepath.Clean("/abs/path")},
	}
	for _, tc := range tests {
		got := expandPath(tc.input)
		if got != tc.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestCopyDir(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	os.MkdirAll(filepath.Join(src, "sub"), 0755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0644)

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dst, "a.txt"))
	if string(data) != "hello" {
		t.Errorf("a.txt content mismatch")
	}
	data, _ = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if string(data) != "world" {
		t.Errorf("sub/b.txt content mismatch")
	}
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
go test ./cmd/install -v -run "TestExpandPath|TestCopyDir"
```

Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/install/utils.go cmd/install/utils_test.go
git commit -m "feat(install): initialize Go module and utility functions"
```

---

### Task 2: CLI Flag Parsing

**Files:**
- Create: `cmd/install/flags.go`
- Create: `cmd/install/flags_test.go`

- [ ] **Step 1: Write `cmd/install/flags.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Options struct {
	CoreOnly       bool
	WithBusiness   bool
	NoTools        bool
	NoRoles        bool
	Platforms      string
	Location       string
	Link           bool
	DryRun         bool
	NonInteractive bool
	SourceDir      string
}

func ParseFlags() Options {
	var opts Options
	flag.BoolVar(&opts.CoreOnly, "core-only", false, "install only core/")
	flag.BoolVar(&opts.WithBusiness, "with-business", false, "include business/ module")
	flag.BoolVar(&opts.NoTools, "no-tools", false, "skip tools/ module")
	flag.BoolVar(&opts.NoRoles, "no-roles", false, "skip roles/ module")
	flag.StringVar(&opts.Platforms, "platforms", "", "comma-separated platform list (default: auto-detect)")
	flag.StringVar(&opts.Location, "location", expandPath("~/.gpowers"), "install location")
	flag.BoolVar(&opts.Link, "link", false, "symlink source repo (dev mode)")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "print plan, do not execute")
	flag.BoolVar(&opts.NonInteractive, "non-interactive", false, "skip prompts (CI mode)")
	flag.StringVar(&opts.SourceDir, "source-dir", "", "override source directory")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: install [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	return opts
}

func (o Options) Modules() []string {
	if o.CoreOnly {
		return []string{"core"}
	}
	mods := []string{"core"}
	if !o.NoRoles {
		mods = append(mods, "roles")
	}
	if !o.NoTools {
		mods = append(mods, "tools")
	}
	if o.WithBusiness {
		mods = append(mods, "business")
	}
	return mods
}

func (o Options) PlatformList() []string {
	if o.Platforms == "" {
		return nil
	}
	parts := strings.Split(o.Platforms, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 2: Write `cmd/install/flags_test.go`**

```go
package main

import "testing"

func TestOptionsModules(t *testing.T) {
	tests := []struct {
		opts Options
		want []string
	}{
		{Options{}, []string{"core", "roles", "tools"}},
		{Options{CoreOnly: true}, []string{"core"}},
		{Options{NoTools: true}, []string{"core", "roles"}},
		{Options{WithBusiness: true}, []string{"core", "roles", "tools", "business"}},
	}
	for _, tc := range tests {
		got := tc.opts.Modules()
		if len(got) != len(tc.want) {
			t.Errorf("Modules() = %v, want %v", got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("Modules()[%d] = %q, want %q", i, got[i], tc.want[i])
			}
		}
	}
}

func TestOptionsPlatformList(t *testing.T) {
	opts := Options{Platforms: "kimi, claude-code,cursor"}
	got := opts.PlatformList()
	want := []string{"kimi", "claude-code", "cursor"}
	if len(got) != len(want) {
		t.Fatalf("PlatformList() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("PlatformList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestOptions"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/flags.go cmd/install/flags_test.go
git commit -m "feat(install): add CLI flag parsing"
```

---

### Task 3: Markdown Frontmatter Parser and Slash Discovery

**Files:**
- Create: `cmd/install/slasher.go`
- Create: `cmd/install/slasher_test.go`

- [ ] **Step 1: Write `cmd/install/slasher.go`**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SlashInfo struct {
	Slash          string
	Module         string
	SkillDir       string
	RequiresDriver string
}

func parseFrontmatter(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return result
	}
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			break
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		result[key] = val
	}
	return result
}

func listSlashes(gpowersHome string) ([]SlashInfo, error) {
	var slashes []SlashInfo
	for _, mod := range []string{"core", "roles", "tools", "business"} {
		dir := filepath.Join(gpowersHome, mod, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			skillDir := ent.Name()
			file := filepath.Join(dir, skillDir, "SKILL.md")
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			fm := parseFrontmatter(string(data))
			slash := fm["slash"]
			if slash == "" {
				continue
			}
			driver := fm["requires-driver"]
			if driver == "" {
				driver = "none"
			}
			slashes = append(slashes, SlashInfo{
				Slash:          slash,
				Module:         mod,
				SkillDir:       skillDir,
				RequiresDriver: driver,
			})
		}
	}
	return slashes, nil
}

func readSkillBody(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("no frontmatter in %s", path)
	}
	return strings.TrimPrefix(parts[2], "\n"), nil
}
```

- [ ] **Step 2: Write `cmd/install/slasher_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	content := `---
name: foo
description: "bar baz"
slash: /test
---
body here
`
	fm := parseFrontmatter(content)
	if fm["name"] != "foo" {
		t.Errorf("name = %q, want foo", fm["name"])
	}
	if fm["description"] != "bar baz" {
		t.Errorf("description = %q, want 'bar baz'", fm["description"])
	}
	if fm["slash"] != "/test" {
		t.Errorf("slash = %q, want /test", fm["slash"])
	}
}

func TestListSlashes(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "core", "skills", "test-skill"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "skills", "test-skill", "SKILL.md"), []byte(
		"---\nname: test\nslash: /test\nrequires-driver: playwright-cli\n---\nbody"), 0644)

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
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestParseFrontmatter|TestListSlashes"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/slasher.go cmd/install/slasher_test.go
git commit -m "feat(install): add markdown frontmatter parser and slash discovery"
```

---

### Task 4: Manifest and Skills JSON Handling

**Files:**
- Create: `cmd/install/manifest.go`
- Create: `cmd/install/manifest_test.go`

- [ ] **Step 1: Write `cmd/install/manifest.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func updateManifest(manifestPath, location string, modules []string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	m["installed_at"] = time.Now().UTC().Format(time.RFC3339)
	m["install_location"] = location
	m["installed_modules"] = modules

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, append(out, '\n'), 0644)
}

func generateSkillsJSON(gpowersHome string) error {
	type SkillInfo struct {
		Name        string `json:"name"`
		Module      string `json:"module"`
		Description string `json:"description"`
		Path        string `json:"path"`
	}
	var skills []SkillInfo
	for _, mod := range []string{"core", "roles", "tools", "business"} {
		dir := filepath.Join(gpowersHome, mod, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			file := filepath.Join(dir, name, "SKILL.md")
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			fm := parseFrontmatter(string(data))
			skills = append(skills, SkillInfo{
				Name:        name,
				Module:      mod,
				Description: fm["description"],
				Path:        fmt.Sprintf("%s/skills/%s/SKILL.md", mod, name),
			})
		}
	}
	out, err := json.MarshalIndent(map[string]interface{}{"skills": skills}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gpowersHome, "skills.json"), append(out, '\n'), 0644)
}
```

- [ ] **Step 2: Write `cmd/install/manifest_test.go`**

```go
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
	os.WriteFile(manifestPath, []byte(initial), 0644)

	err := updateManifest(manifestPath, "/tmp/gpowers", []string{"core", "roles"})
	if err != nil {
		t.Fatalf("updateManifest failed: %v", err)
	}

	data, _ := os.ReadFile(manifestPath)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["install_location"] != "/tmp/gpowers" {
		t.Errorf("install_location = %v", m["install_location"])
	}
	mods, ok := m["installed_modules"].([]interface{})
	if !ok || len(mods) != 2 {
		t.Errorf("installed_modules = %v", m["installed_modules"])
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestUpdateManifest"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/manifest.go cmd/install/manifest_test.go
git commit -m "feat(install): add manifest and skills JSON handling"
```

---

### Task 5: File Staging (Copy and Symlink)

**Files:**
- Create: `cmd/install/staging.go`
- Create: `cmd/install/staging_test.go`

- [ ] **Step 1: Write `cmd/install/staging.go`**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func stageFiles(sourceDir, targetDir string, modules []string, linkMode bool) error {
	entries := []string{
		"core", "roles", "tools", "platforms", "lib", "bin", "templates",
		"manifest.json", "upstream-sources.json", "install", "uninstall",
		"README.md", "LICENSE",
	}
	if contains(modules, "business") {
		entries = append(entries, "business")
	}

	binaryName := "install"
	if runtime.GOOS == "windows" {
		binaryName = "install.exe"
	}

	for _, entry := range entries {
		src := filepath.Join(sourceDir, entry)
		dst := filepath.Join(targetDir, entry)

		if entry == "install" {
			src = filepath.Join(sourceDir, binaryName)
			dst = filepath.Join(targetDir, binaryName)
		}

		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}

		if entry == binaryName && sameFile(src, dst) {
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			os.RemoveAll(dst)
		}

		if linkMode {
			if err := os.Symlink(src, dst); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", src, dst, err)
			}
			continue
		}

		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
			}
		} else {
			if err := copyFile(src, dst, info.Mode()); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: Write `cmd/install/staging_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageFilesCopy(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	os.MkdirAll(filepath.Join(src, "core", "skills"), 0755)
	os.WriteFile(filepath.Join(src, "manifest.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(src, "core", "skills", "a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(src, "install.exe"), []byte("binary"), 0644)

	err := stageFiles(src, dst, []string{"core"}, false)
	if err != nil {
		t.Fatalf("stageFiles failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dst, "core", "skills", "a.md"))
	if string(data) != "a" {
		t.Errorf("staged file mismatch")
	}
	data, _ = os.ReadFile(filepath.Join(dst, "install.exe"))
	if string(data) != "binary" {
		t.Errorf("install binary not staged")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestStageFilesCopy"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/staging.go cmd/install/staging_test.go
git commit -m "feat(install): add file staging with copy and symlink support"
```

---

### Task 6: Platform Detection, Manifest Generation, and Registration

**Files:**
- Create: `cmd/install/platforms.go`
- Create: `cmd/install/platforms_test.go`

- [ ] **Step 1: Write `cmd/install/platforms.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var platformMarkers = map[string]string{
	"claude-code": "~/.claude",
	"codex":       "~/.codex",
	"gemini":      "~/.config/gemini",
	"cursor":      "~/.cursor",
	"opencode":    "~/.config/opencode",
	"copilot":     "~/.config/copilot-cli",
	"kimi":        "~/.kimi",
}

func detectPlatforms() []string {
	var detected []string
	for platform, marker := range platformMarkers {
		if _, err := os.Stat(expandPath(marker)); err == nil {
			detected = append(detected, platform)
		}
	}
	return detected
}

type PlatformShape struct {
	ManifestFilename       string      `json:"manifest_filename"`
	CommandDir             string      `json:"command_dir"`
	CommandFilenamePattern string      `json:"command_filename_pattern"`
	SupportsHooks          interface{} `json:"supports_hooks"`
	NamespaceMode          string      `json:"namespace_mode"`
	InstallLinkTarget      string      `json:"install_link_target"`
}

func loadPlatformShapes(gpowersHome string) (map[string]PlatformShape, error) {
	path := filepath.Join(gpowersHome, "platforms", "_platform-shapes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Platforms map[string]PlatformShape `json:"platforms"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Platforms, nil
}

func generatePlatformManifest(platform, gpowersHome string) error {
	shapes, err := loadPlatformShapes(gpowersHome)
	if err != nil {
		return err
	}
	shape, ok := shapes[platform]
	if !ok {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	outDir := filepath.Join(gpowersHome, "platforms", platform)
	os.MkdirAll(filepath.Join(outDir, shape.CommandDir), 0755)

	manifest := map[string]interface{}{
		"$schema":        "https://gpowers.dev/schemas/plugin.json",
		"name":           "gpowers",
		"version":        "1.0.0",
		"namespace_mode": shape.NamespaceMode,
		"description":    "gpowers — unified methodology + roles + tools + business automation",
		"modules":        []string{"core", "roles", "tools", "business"},
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(outDir, shape.ManifestFilename), append(data, '\n'), 0644)

	if err := generateSkillsJSONForPlatform(gpowersHome, outDir); err != nil {
		return err
	}

	slashes, _ := listSlashes(gpowersHome)
	for _, s := range slashes {
		cmdName := strings.TrimPrefix(s.Slash, "/")
		cmdFile := filepath.Join(outDir, shape.CommandDir, cmdName+".md")
		content := fmt.Sprintf(`---
slash: %s
module: %s
skill: %s
requires_driver: %s
---

<!-- SOURCE: $GPOWERS_HOME/%s/skills/%s/SKILL.md -->

This command invokes the gpowers skill **%s** (%s).

Refer to the source SKILL.md (above) for the full workflow. The platform's skill mechanism will load it on demand.
`, s.Slash, s.Module, s.SkillDir, s.RequiresDriver, s.Module, s.SkillDir, s.SkillDir, s.Module)
		os.WriteFile(cmdFile, []byte(content), 0644)
	}

	if shape.SupportsHooks == true || shape.SupportsHooks == "true" {
		src := filepath.Join(gpowersHome, "core", "hooks", "hooks.json")
		dst := filepath.Join(outDir, "hooks.json")
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(dst, data, 0644)
		}
	}

	return nil
}

func generateSkillsJSONForPlatform(gpowersHome, outDir string) error {
	type SkillInfo struct {
		Name        string `json:"name"`
		Module      string `json:"module"`
		Description string `json:"description"`
		Path        string `json:"path"`
	}
	var skills []SkillInfo
	for _, mod := range []string{"core", "roles", "tools", "business"} {
		dir := filepath.Join(gpowersHome, mod, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			file := filepath.Join(dir, name, "SKILL.md")
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			fm := parseFrontmatter(string(data))
			skills = append(skills, SkillInfo{
				Name:        name,
				Module:      mod,
				Description: fm["description"],
				Path:        fmt.Sprintf("%s/skills/%s/SKILL.md", mod, name),
			})
		}
	}
	out, _ := json.MarshalIndent(map[string]interface{}{"skills": skills}, "", "  ")
	return os.WriteFile(filepath.Join(outDir, "skills.json"), append(out, '\n'), 0644)
}

func registerPlatform(platform, gpowersHome string) error {
	shapes, err := loadPlatformShapes(gpowersHome)
	if err != nil {
		return err
	}
	shape, ok := shapes[platform]
	if !ok {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	if platform == "kimi" {
		return registerKimi(gpowersHome)
	}

	target := expandPath(shape.InstallLinkTarget)
	source := filepath.Join(gpowersHome, "platforms", platform)

	parent := filepath.Dir(target)
	os.MkdirAll(parent, 0755)

	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(target)
		} else {
			return fmt.Errorf("%s exists and is not a symlink; skipping", target)
		}
	}

	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", source, target, err)
	}
	fmt.Printf("[install] linked %s -> %s\n", target, source)
	return nil
}

func registerKimi(gpowersHome string) error {
	adaptersDir := filepath.Join(gpowersHome, "platforms", "kimi", "adapters")
	kimiSkills := expandPath("~/.kimi/skills")
	os.MkdirAll(kimiSkills, 0755)

	entries, err := os.ReadDir(adaptersDir)
	if err != nil {
		return err
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		source := filepath.Join(adaptersDir, name)
		target := filepath.Join(kimiSkills, name)

		if info, err := os.Stat(target); err == nil {
			if info.IsDir() {
				os.RemoveAll(target)
			} else {
				os.Remove(target)
			}
		}

		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
			if err := cmd.Run(); err == nil {
				fmt.Printf("[OK] Junction: %s\n", name)
				continue
			}
			if err := copyDir(source, target); err == nil {
				fmt.Printf("[OK] Copied:  %s\n", name)
			} else {
				fmt.Printf("[ERR] Failed:  %s\n", name)
			}
		} else {
			if err := os.Symlink(source, target); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", source, target, err)
			}
			fmt.Printf("[install] linked %s -> %s\n", target, source)
		}
	}
	fmt.Printf("[install] kimi skills registered in: %s\n", kimiSkills)
	return nil
}
```

- [ ] **Step 2: Write `cmd/install/platforms_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPlatforms(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	os.MkdirAll(filepath.Join(tmpHome, ".kimi"), 0755)
	detected := detectPlatforms()
	found := false
	for _, p := range detected {
		if p == "kimi" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected kimi to be detected, got %v", detected)
	}
}

func TestGeneratePlatformManifest(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "platforms"), 0755)
	os.MkdirAll(filepath.Join(tmp, "core", "hooks"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "hooks", "hooks.json"), []byte("{}"), 0644)
	shapes := `{"platforms":{"claude-code":{"manifest_filename":"plugin.json","command_dir":"commands","command_filename_pattern":"{slash}.md","supports_hooks":true,"namespace_mode":"plugin-scoped","install_link_target":"~/.claude/plugins/gpowers"}}}`
	os.WriteFile(filepath.Join(tmp, "platforms", "_platform-shapes.json"), []byte(shapes), 0644)

	err := generatePlatformManifest("claude-code", tmp)
	if err != nil {
		t.Fatalf("generatePlatformManifest failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "platforms", "claude-code", "plugin.json")); err != nil {
		t.Errorf("plugin.json not generated")
	}
	if _, err := os.Stat(filepath.Join(tmp, "platforms", "claude-code", "hooks.json")); err != nil {
		t.Errorf("hooks.json not copied")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestDetectPlatforms|TestGeneratePlatformManifest"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/platforms.go cmd/install/platforms_test.go
git commit -m "feat(install): add platform detection, manifest generation, and registration"
```

---

### Task 7: Kimi Adapter Generation

**Files:**
- Create: `cmd/install/kimi.go`
- Create: `cmd/install/kimi_test.go`

- [ ] **Step 1: Write `cmd/install/kimi.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func generateKimiAdapters(gpowersHome string) error {
	adaptersDir := filepath.Join(gpowersHome, "platforms", "kimi", "adapters")
	os.MkdirAll(filepath.Join(adaptersDir, "gpowers"), 0755)

	usingPath := filepath.Join(gpowersHome, "core", "skills", "using-gpowers", "SKILL.md")
	preamble, err := readSkillBody(usingPath)
	if err != nil {
		return fmt.Errorf("using-gpowers missing: %w", err)
	}

	routerPath := filepath.Join(adaptersDir, "gpowers", "SKILL.md")
	routerContent := fmt.Sprintf(`---
name: gpowers
description: gpowers entry — four-module model (core / roles / tools / business)
gpowers-source: core/skills/using-gpowers/SKILL.md
---

%s
`, preamble)
	os.WriteFile(routerPath, []byte(routerContent), 0644)

	var adapterNames []string

	for _, mod := range []string{"core", "roles", "tools", "business"} {
		dir := filepath.Join(gpowersHome, mod, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			orig := ent.Name()
			if orig == "using-gpowers" {
				continue
			}
			file := filepath.Join(dir, orig, "SKILL.md")
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			fm := parseFrontmatter(string(data))
			body, err := readSkillBody(file)
			if err != nil {
				continue
			}

			adapterName := "gpowers-" + orig
			if strings.HasPrefix(orig, "gpowers-") {
				adapterName = orig
			}

			adapterDir := filepath.Join(adaptersDir, adapterName)
			os.MkdirAll(adapterDir, 0755)

			desc := fm["description"]
			if desc == "" {
				desc = orig
			}

			adapterContent := fmt.Sprintf(`---
name: %s
description: "%s (gpowers adapter for Kimi)"
gpowers-source: %s/skills/%s/SKILL.md
gpowers-module: %s
---

<!-- gpowers preamble (auto, four-module model) -->

%s

<!-- SOURCE: $GPOWERS_HOME/%s/skills/%s/SKILL.md -->

%s
`, adapterName, desc, mod, orig, mod, preamble, mod, orig, body)
			os.WriteFile(filepath.Join(adapterDir, "SKILL.md"), []byte(adapterContent), 0644)
			adapterNames = append(adapterNames, adapterName)
		}
	}

	out, _ := json.MarshalIndent(map[string]interface{}{"adapters": adapterNames}, "", "  ")
	return os.WriteFile(filepath.Join(gpowersHome, "platforms", "kimi", "kimi-skills.json"), append(out, '\n'), 0644)
}
```

- [ ] **Step 2: Write `cmd/install/kimi_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKimiAdapters(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "core", "skills", "using-gpowers"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "skills", "using-gpowers", "SKILL.md"), []byte(
		"---\nname: using-gpowers\n---\nThis is the preamble."), 0644)

	os.MkdirAll(filepath.Join(tmp, "core", "skills", "brainstorming"), 0755)
	os.WriteFile(filepath.Join(tmp, "core", "skills", "brainstorming", "SKILL.md"), []byte(
		"---\nname: brainstorming\ndescription: Explore ideas\n---\nBody here."), 0644)

	err := generateKimiAdapters(tmp)
	if err != nil {
		t.Fatalf("generateKimiAdapters failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers", "SKILL.md")); err != nil {
		t.Errorf("gpowers router adapter missing")
	}
	if _, err := os.Stat(filepath.Join(tmp, "platforms", "kimi", "adapters", "gpowers-brainstorming", "SKILL.md")); err != nil {
		t.Errorf("gpowers-brainstorming adapter missing")
	}
	if _, err := os.Stat(filepath.Join(tmp, "platforms", "kimi", "kimi-skills.json")); err != nil {
		t.Errorf("kimi-skills.json missing")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/install -v -run "TestGenerateKimiAdapters"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/install/kimi.go cmd/install/kimi_test.go
git commit -m "feat(install): add Kimi adapter generation"
```

---

### Task 8: Main Orchestration and Integration

**Files:**
- Create: `cmd/install/main.go`
- Create: `cmd/install/main_test.go` (subprocess integration test)

- [ ] **Step 1: Write `cmd/install/main.go`**

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[install] error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	opts := ParseFlags()

	modules := opts.Modules()
	requestedPlatforms := opts.PlatformList()

	sourceDir := opts.SourceDir
	if sourceDir == "" {
		if exe, err := os.Executable(); err == nil {
			sourceDir = filepath.Dir(exe)
		} else {
			if wd, err := os.Getwd(); err == nil {
				sourceDir = wd
			}
		}
	}

	var platforms []string
	if len(requestedPlatforms) > 0 {
		platforms = requestedPlatforms
	} else {
		platforms = detectPlatforms()
	}

	fmt.Printf("[plan] install location: %s\n", opts.Location)
	fmt.Printf("[plan] modules: %s\n", strings.Join(modules, " "))
	if len(platforms) == 0 {
		fmt.Printf("[plan] platforms: <none detected>\n")
	} else {
		fmt.Printf("[plan] platforms: %s\n", strings.Join(platforms, " "))
	}
	if opts.Link {
		fmt.Printf("[plan] mode: symlink (dev)\n")
	}

	if opts.DryRun {
		return nil
	}

	if opts.WithBusiness && !opts.NonInteractive {
		disclaimerPath := filepath.Join(sourceDir, "business", "DISCLAIMER.md")
		if data, err := os.ReadFile(disclaimerPath); err == nil {
			fmt.Println("============================================================")
			fmt.Print(string(data))
			fmt.Println("============================================================")
			fmt.Print("Activate the business/ module? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("[plan] Skipping business activation.")
				opts.WithBusiness = false
				modules = opts.Modules()
			}
		}
	}

	os.MkdirAll(opts.Location, 0755)

	fmt.Println("[install] staging files...")
	if err := stageFiles(sourceDir, opts.Location, modules, opts.Link); err != nil {
		return err
	}

	for _, d := range []string{"config", "state", "cache", "data", "analytics", "tmp", "logs"} {
		os.MkdirAll(filepath.Join(opts.Location, d), 0755)
	}

	manifestPath := filepath.Join(opts.Location, "manifest.json")
	if err := updateManifest(manifestPath, opts.Location, modules); err != nil {
		return fmt.Errorf("update manifest: %w", err)
	}

	for _, p := range platforms {
		if p == "kimi" {
			if err := generateKimiAdapters(opts.Location); err != nil {
				return fmt.Errorf("generate kimi adapters: %w", err)
			}
		} else {
			if err := generatePlatformManifest(p, opts.Location); err != nil {
				fmt.Fprintf(os.Stderr, "[install] warn: generate platform manifest for %s: %v\n", p, err)
			}
		}
	}

	for _, p := range platforms {
		if err := registerPlatform(p, opts.Location); err != nil {
			fmt.Fprintf(os.Stderr, "[install] warn: register platform %s: %v\n", p, err)
		}
	}

	fmt.Printf("[install] done. location: %s\n", opts.Location)
	return nil
}
```

- [ ] **Step 2: Build and run a dry-run smoke test**

```bash
go build -o install_test ./cmd/install
mkdir -p /tmp/gpowers_smoke/core/skills
mkdir -p /tmp/gpowers_smoke/platforms
echo '{"version":"0.0.1"}' > /tmp/gpowers_smoke/manifest.json
echo '# uninstall' > /tmp/gpowers_smoke/uninstall
echo '# README' > /tmp/gpowers_smoke/README.md
echo 'MIT' > /tmp/gpowers_smoke/LICENSE
./install_test --dry-run --source-dir /tmp/gpowers_smoke --location /tmp/gpowers_installed
```

Expected output:
```
[plan] install location: /tmp/gpowers_installed
[plan] modules: core roles tools
[plan] platforms: <none detected>
```

- [ ] **Step 3: Commit**

```bash
git add cmd/install/main.go cmd/install/main_test.go
git commit -m "feat(install): add main orchestration loop"
```

---

### Task 9: Build Binaries, Remove Legacy Scripts, Update Docs

**Files:**
- Modify: `docs/INSTALL.md`
- Delete: `install.bat`
- Delete: `install` (bash)
- Create: `install` (Go binary, Unix)
- Create: `install.exe` (Go binary, Windows)

- [ ] **Step 1: Build all platform binaries**

```bash
# Local platform binary (Windows in this case)
go build -o install.exe ./cmd/install

# Cross-compile for reference
GOOS=linux GOARCH=amd64 go build -o install_linux ./cmd/install
GOOS=darwin GOARCH=amd64 go build -o install_darwin ./cmd/install
GOOS=darwin GOARCH=arm64 go build -o install_darwin_arm64 ./cmd/install
```

- [ ] **Step 2: Run full unit test suite**

```bash
go test ./cmd/install/...
```

Expected: ALL PASS.

- [ ] **Step 3: Remove legacy bash/batch installers**

```bash
git rm install.bat install
```

- [ ] **Step 4: Update `docs/INSTALL.md`**

Replace the Windows section to remove references to `install.bat` and bash delegation. Replace the Quickstart to reference the Go binary.

Example diff:
```markdown
## Quickstart

```bash
git clone https://github.com/odysseythink/gpowers ~/.gpowers
cd ~/.gpowers
./install
```

## Windows

On Windows, run the provided `install.exe`:

```cmd
git clone https://github.com/odysseythink/gpowers %USERPROFILE%\.gpowers
cd %USERPROFILE%\.gpowers
install.exe --platforms=kimi
```

`install.exe` is a self-contained Go program. It does not require Git Bash or WSL.
```

- [ ] **Step 5: Commit**

```bash
git add install.exe install_linux install_darwin install_darwin_arm64
git add docs/INSTALL.md
git commit -m "feat(install): replace bash installers with cross-platform Go binary"
```

---

## Self-Review

**1. Spec coverage:**
- ✅ CLI flags — Task 2
- ✅ Cross-platform home dir — Task 1
- ✅ File staging (copy/symlink) — Task 5
- ✅ Manifest JSON handling — Task 4
- ✅ Platform detection — Task 6
- ✅ Generic platform manifest generation — Task 6
- ✅ Kimi adapter generation — Task 7
- ✅ Platform registration (symlink/junction/copy) — Task 6
- ✅ Business disclaimer — Task 8
- ✅ Output format compatibility — Task 8
- ✅ Source directory self-detection — Task 8
- ✅ Build and commit binaries — Task 9
- ✅ Remove legacy scripts — Task 9

**2. Placeholder scan:**
- ✅ No TBD/TODO/fill-in-details found.
- ✅ All steps contain concrete code or exact commands.

**3. Type consistency:**
- ✅ `Options.Modules()` and `Options.PlatformList()` used consistently.
- ✅ `parseFrontmatter` returns `map[string]string` throughout.
- ✅ `SlashInfo` fields match usage in `platforms.go` and `kimi.go`.
