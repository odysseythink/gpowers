// Package browserskill implements storage helpers for per-task browser scripts.
//
// Three tiers, walked in order project > global > bundled (first-wins):
//   project:  <project>/.gstack/browser-skills/<name>/
//   global:   ~/.gstack/browser-skills/<name>/
//   bundled:  <install>/browser-skills/<name>/ (read-only)
//
// No INDEX.json — listBrowserSkills walks the three directories every call.
// Tombstones move a skill to <tier>/.tombstones/<name>-<ts>/.
package browserskill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"browse-go/pkg/config"
)

// ─── Types ──────────────────────────────────────────────────────

// Tier identifies a skill storage tier.
type Tier string

const (
	TierProject Tier = "project"
	TierGlobal  Tier = "global"
	TierBundled Tier = "bundled"
)

// Frontmatter holds parsed SKILL.md frontmatter fields.
type Frontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Host        string   `yaml:"host"`
	Triggers    []string `yaml:"triggers"`
	Trusted     bool     `yaml:"trusted"`
	Version     string   `yaml:"version"`
	Source      string   `yaml:"source"`
}

// Skill represents a resolved browser skill.
type Skill struct {
	Name        string
	Tier        Tier
	Dir         string
	Frontmatter Frontmatter
	BodyMd      string
}

// TierPaths holds the three directory paths.
type TierPaths struct {
	Project string // may be empty in non-project contexts
	Global  string
	Bundled string
}

// ─── Tier resolution ────────────────────────────────────────────

// DefaultTierPaths resolves tier directories from runtime context.
func DefaultTierPaths() TierPaths {
	projectRoot := config.GitRoot()
	if projectRoot == "" {
		wd, _ := os.Getwd()
		projectRoot = wd
	}
	home := config.Home()
	return TierPaths{
		Project: filepath.Join(projectRoot, ".gstack", "browser-skills"),
		Global:  filepath.Join(home, "browser-skills"),
		Bundled: filepath.Join(home, "..", "browser-skills"), // best-effort
	}
}

// ─── Listing ────────────────────────────────────────────────────

// List returns all skills visible across tiers, project first.
func List(tiers TierPaths) []Skill {
	seen := make(map[string]bool)
	var result []Skill

	for _, tierDir := range []string{tiers.Project, tiers.Global, tiers.Bundled} {
		if tierDir == "" {
			continue
		}
		entries, err := os.ReadDir(tierDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			name := e.Name()
			if seen[name] {
				continue
			}
			dir := filepath.Join(tierDir, name)
			skill, err := readSkillDir(name, dir)
			if err != nil {
				continue
			}
			// Determine tier
			switch {
			case strings.HasPrefix(dir, tiers.Project):
				skill.Tier = TierProject
			case strings.HasPrefix(dir, tiers.Global):
				skill.Tier = TierGlobal
			default:
				skill.Tier = TierBundled
			}
			seen[name] = true
			result = append(result, *skill)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Read returns a single skill by name, walking tiers in order.
func Read(name string, tiers TierPaths) (*Skill, error) {
	for _, tierDir := range []string{tiers.Project, tiers.Global, tiers.Bundled} {
		if tierDir == "" {
			continue
		}
		dir := filepath.Join(tierDir, name)
		skill, err := readSkillDir(name, dir)
		if err == nil {
			switch {
			case strings.HasPrefix(dir, tiers.Project):
				skill.Tier = TierProject
			case strings.HasPrefix(dir, tiers.Global):
				skill.Tier = TierGlobal
			default:
				skill.Tier = TierBundled
			}
			return skill, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found in any tier", name)
}

// readSkillDir parses a skill directory.
func readSkillDir(name, dir string) (*Skill, error) {
	mdPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, err
	}
	fm, body, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, err
	}
	if fm.Name == "" {
		fm.Name = name
	}
	return &Skill{
		Name:        name,
		Dir:         dir,
		Frontmatter: fm,
		BodyMd:      body,
	}, nil
}

// ─── Frontmatter parsing ────────────────────────────────────────

// parseFrontmatter extracts YAML-like frontmatter from SKILL.md.
// Expected format:
//
//	---
//	name: my-skill
//	host: example.com
//	triggers:
//	  - scrape frontpage
//	---
//	<body here>
func parseFrontmatter(content string) (Frontmatter, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return Frontmatter{}, "", fmt.Errorf("SKILL.md missing frontmatter block")
	}
	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return Frontmatter{}, "", fmt.Errorf("SKILL.md frontmatter block not terminated")
	}
	fmText := content[4 : 4+end]
	body := strings.TrimLeft(content[4+end+4:], "\n")
	fm := parseFrontmatterFields(fmText)
	return fm, body, nil
}

func parseFrontmatterFields(text string) Frontmatter {
	var fm Frontmatter
	lines := strings.Split(text, "\n")
	var currentKey string
	var listItems []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// List item: "  - value" or "  - key: value"
		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			listItems = append(listItems, item)
			continue
		}

		// New key — flush previous list if any
		if currentKey != "" && len(listItems) > 0 {
			flushList(currentKey, listItems, &fm)
			listItems = nil
		}

		// Scalar: "key: value"
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			value = unquote(value)
			setScalar(key, value, &fm)
			currentKey = key
			continue
		}
	}

	if currentKey != "" && len(listItems) > 0 {
		flushList(currentKey, listItems, &fm)
	}

	return fm
}

func setScalar(key, value string, fm *Frontmatter) {
	switch key {
	case "name":
		fm.Name = value
	case "description":
		fm.Description = value
	case "host":
		fm.Host = value
	case "version":
		fm.Version = value
	case "source":
		fm.Source = value
	case "trusted":
		fm.Trusted = value == "true" || value == "yes"
	}
}

func flushList(key string, items []string, fm *Frontmatter) {
	if key == "triggers" {
		for _, item := range items {
			fm.Triggers = append(fm.Triggers, unquote(item))
		}
	}
}

func unquote(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// ─── Tombstone ──────────────────────────────────────────────────

// Tombstone moves a skill to <tier>/.tombstones/<name>-<ts>/.
func Tombstone(name string, tier Tier, tiers TierPaths) error {
	var tierDir string
	switch tier {
	case TierProject:
		tierDir = tiers.Project
	case TierGlobal:
		tierDir = tiers.Global
	case TierBundled:
		return fmt.Errorf("cannot tombstone bundled (read-only) skill")
	}
	if tierDir == "" {
		return fmt.Errorf("tier %q directory not resolved", tier)
	}

	src := filepath.Join(tierDir, name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("skill %q not found in %s: %w", name, tier, err)
	}

	tombDir := filepath.Join(tierDir, ".tombstones", fmt.Sprintf("%s-%d", name, time.Now().Unix()))
	if err := os.MkdirAll(filepath.Dir(tombDir), 0o755); err != nil {
		return err
	}
	return os.Rename(src, tombDir)
}
