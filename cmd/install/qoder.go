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

func generateQoderAdapters(gpowersHome string) error {
	adaptersDir := filepath.Join(gpowersHome, "platforms", "qoder", "adapters")
	if err := os.MkdirAll(filepath.Join(adaptersDir, "gpowers"), 0755); err != nil {
		return fmt.Errorf("create gpowers adapter dir: %w", err)
	}

	usingPath := filepath.Join(gpowersHome, "core", "skills", "using-gpowers", "SKILL.md")
	usingData, err := os.ReadFile(usingPath)
	if err != nil {
		return fmt.Errorf("using-gpowers missing: %w", err)
	}
	preamble, err := readSkillBody(string(usingData))
	if err != nil {
		return fmt.Errorf("using-gpowers missing: %w", err)
	}

	routerPath := filepath.Join(adaptersDir, "gpowers", "SKILL.md")
	routerContent := fmt.Sprintf(`---
name: gpowers
description: gpowers entry — three-module model (core / roles / tools)
gpowers-source: core/skills/using-gpowers/SKILL.md
---

%s
`, preamble)
	if err := os.WriteFile(routerPath, []byte(routerContent), 0644); err != nil {
		return fmt.Errorf("write router adapter: %w", err)
	}

	var adapterNames []string

	for _, mod := range allModules {
		dir := filepath.Join(gpowersHome, mod, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read dir %s: %w", dir, err)
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
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("read skill %s: %w", file, err)
			}

			fm := parseFrontmatter(string(data))
			body, err := readSkillBody(string(data))
			if err != nil {
				continue
			}

			adapterName := "gpowers-" + orig
			if strings.HasPrefix(orig, "gpowers-") {
				adapterName = orig
			}

			adapterDir := filepath.Join(adaptersDir, adapterName)
			if err := os.MkdirAll(adapterDir, 0755); err != nil {
				return fmt.Errorf("create adapter dir %s: %w", adapterDir, err)
			}

			desc := fm["description"]
			if desc == "" {
				desc = orig
			}

			adapterContent := fmt.Sprintf(`---
name: %s
description: "%s (gpowers adapter for Qoder)"
gpowers-source: %s/skills/%s/SKILL.md
gpowers-module: %s
---

<!-- gpowers preamble (auto, three-module model) -->

%s

<!-- SOURCE: $GPOWERS_HOME/%s/skills/%s/SKILL.md -->

%s
`, adapterName, desc, mod, orig, mod, preamble, mod, orig, body)
			if err := os.WriteFile(filepath.Join(adapterDir, "SKILL.md"), []byte(adapterContent), 0644); err != nil {
				return fmt.Errorf("write adapter %s: %w", adapterName, err)
			}
			adapterNames = append(adapterNames, adapterName)
		}
	}

	out, err := json.MarshalIndent(map[string]interface{}{"adapters": adapterNames}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal adapters: %w", err)
	}
	return os.WriteFile(filepath.Join(gpowersHome, "platforms", "qoder", "qoder-skills.json"), append(out, '\n'), 0644)
}

func registerQoder(gpowersHome string) error {
	adaptersDir := filepath.Join(gpowersHome, "platforms", "qoder", "adapters")
	qoderSkills := expandPath("~/.qoder/skills")
	if err := os.MkdirAll(qoderSkills, 0755); err != nil {
		return fmt.Errorf("create qoder skills dir: %w", err)
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
		target := filepath.Join(qoderSkills, name)

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
	fmt.Printf("[install] qoder skills registered in: %s\n", qoderSkills)
	return nil
}
