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
	if err := os.MkdirAll(filepath.Join(outDir, shape.CommandDir), 0755); err != nil {
		return fmt.Errorf("create command dir: %w", err)
	}

	manifest := map[string]interface{}{
		"$schema":        "https://gpowers.dev/schemas/plugin.json",
		"name":           "gpowers",
		"version":        "1.0.0",
		"namespace_mode": shape.NamespaceMode,
		"description":    "gpowers — unified methodology + roles + tools + business automation",
		"modules":        allModules,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, shape.ManifestFilename), append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	if err := generateSkillsJSONForPlatform(gpowersHome, outDir); err != nil {
		return err
	}

	slashes, err := listSlashes(gpowersHome)
	if err != nil {
		return fmt.Errorf("list slashes: %w", err)
	}
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
		if err := os.WriteFile(cmdFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("write command file %s: %w", cmdFile, err)
		}
	}

	if shape.SupportsHooks == true || shape.SupportsHooks == "true" {
		src := filepath.Join(gpowersHome, "core", "hooks", "hooks.json")
		dst := filepath.Join(outDir, "hooks.json")
		if data, err := os.ReadFile(src); err == nil {
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return fmt.Errorf("copy hooks.json: %w", err)
			}
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
	for _, mod := range allModules {
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
		return fmt.Errorf("marshal skills: %w", err)
	}
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
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("create parent dir %s: %w", parent, err)
	}

	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("remove existing symlink %s: %w", target, err)
			}
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
	if err := os.MkdirAll(kimiSkills, 0755); err != nil {
		return fmt.Errorf("create kimi skills dir: %w", err)
	}

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
				if err := os.RemoveAll(target); err != nil {
					return fmt.Errorf("remove existing dir %s: %w", target, err)
				}
			} else {
				if err := os.Remove(target); err != nil {
					return fmt.Errorf("remove existing file %s: %w", target, err)
				}
			}
		}

		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
			if err := cmd.Run(); err == nil {
				fmt.Printf("[OK] Junction: %s\n", name)
				continue
			}
			if err := copyDir(source, target); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", source, target, err)
			}
			fmt.Printf("[OK] Copied:  %s\n", name)
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
