package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	"qoder":       "~/.qoder",
}

func detectPlatforms() []string {
	var detected []string
	for platform, marker := range platformMarkers {
		if _, err := os.Stat(expandPath(marker)); err == nil {
			detected = append(detected, platform)
		}
	}
	sort.Strings(detected)
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

func generatePlatformManifest(platform, gpowersHome string, modules []string) error {
	shapes, err := loadPlatformShapes(gpowersHome)
	if err != nil {
		return err
	}
	shape, ok := shapes[platform]
	if !ok {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	outDir := filepath.Join(gpowersHome, "platforms", platform)
	cmdDir := filepath.Join(outDir, shape.CommandDir)
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		return fmt.Errorf("create command dir: %w", err)
	}
	// Clear stale command files staged from source repo so that only
	// current skills are emitted.
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, ent := range entries {
			if err := os.RemoveAll(filepath.Join(cmdDir, ent.Name())); err != nil {
				return fmt.Errorf("clear stale command file %s: %w", ent.Name(), err)
			}
		}
	}

	manifest := map[string]interface{}{
		"$schema":     "https://gpowers.dev/schemas/plugin.json",
		"name":        "gpowers",
		"version":     "1.0.0",
		"description": "gpowers — unified methodology + roles + tools automation",
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	manifestPath := filepath.Join(outDir, shape.ManifestFilename)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0644); err != nil {
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
		desc := s.Description
		if desc == "" {
			desc = fmt.Sprintf("Invoke the %s skill", s.SkillDir)
		}
		// Flatten multi-line descriptions to a single line for frontmatter
		desc = strings.ReplaceAll(desc, "\n", " ")
		content := fmt.Sprintf(`---
description: %q
---

<!-- SOURCE: $GPOWERS_HOME/%s/skills/%s/SKILL.md -->

This command invokes the gpowers skill **%s** (%s).

Refer to the source SKILL.md (above) for the full workflow. The platform's skill mechanism will load it on demand.
`, desc, s.Module, s.SkillDir, s.SkillDir, s.Module)
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

func registerPlatform(platform, gpowersHome string, nonInteractive bool) error {
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
	if platform == "qoder" {
		return registerQoder(gpowersHome)
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

	if platform == "claude-code" {
		if err := setupClaudeCodeAutoLoad(nonInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "[install] warn: Claude Code auto-load setup: %v\n", err)
		}
	}
	return nil
}

func setupClaudeCodeAutoLoad(nonInteractive bool) error {
	shell := os.Getenv("SHELL")
	var rcFile string
	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = expandPath("~/.zshrc")
	case strings.Contains(shell, "bash"):
		rcFile = expandPath("~/.bashrc")
	default:
		fmt.Println("[install] NOTE: Claude Code does not auto-load plugins.")
		fmt.Println("[install]       Add the following to your shell rc file:")
		fmt.Println("[install]       alias claude='claude --plugin-dir ~/.claude/plugins/gpowers'")
		return nil
	}

	markerBegin := "# >>> gpowers claude-code plugin autoload <<<"
	markerEnd := "# <<< gpowers claude-code plugin autoload <<<"

	// Check if already installed
	if data, err := os.ReadFile(rcFile); err == nil {
		if strings.Contains(string(data), markerBegin) {
			fmt.Printf("[install] Claude Code auto-load already configured in %s\n", rcFile)
			return nil
		}
	}

	wrapper := fmt.Sprintf(`
%s
claude() {
    local plugin_dirs=()
    for manifest in "$HOME/.claude/plugins"/*/.claude-plugin/plugin.json; do
        if [[ -f "$manifest" ]]; then
            plugin_dirs+=(--plugin-dir "$(dirname "$(dirname "$manifest")")")
        fi
    done
    command claude "${plugin_dirs[@]}" "$@"
}
%s
`, markerBegin, markerEnd)

	if !nonInteractive {
		fmt.Printf("[install] Add Claude Code auto-load wrapper to %s? [Y/n] ", rcFile)
		reader := bufio.NewReader(os.Stdin)
		ans, err := reader.ReadString('\n')
		if err != nil {
			// stdin not available (e.g. piped), skip silently
			fmt.Println("")
			fmt.Println("[install] Skipped Claude Code auto-load setup (non-tty).")
			fmt.Println("[install] To manually enable, start Claude Code with:")
			fmt.Println("[install]   claude --plugin-dir ~/.claude/plugins/gpowers")
			return nil
		}
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			fmt.Println("[install] Skipped Claude Code auto-load setup.")
			fmt.Println("[install] To manually enable, start Claude Code with:")
			fmt.Println("[install]   claude --plugin-dir ~/.claude/plugins/gpowers")
			return nil
		}
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", rcFile, err)
	}
	defer f.Close()

	if _, err := f.WriteString(wrapper); err != nil {
		return fmt.Errorf("write %s: %w", rcFile, err)
	}

	fmt.Printf("[install] Claude Code auto-load configured in %s\n", rcFile)
	fmt.Println("[install] Reload your shell or run: source " + rcFile)
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
