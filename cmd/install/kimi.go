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
description: "%s (gpowers adapter for Kimi)"
gpowers-source: %s/skills/%s/SKILL.md
gpowers-module: %s
---

<!-- gpowers preamble (auto, four-module model) -->

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
	return os.WriteFile(filepath.Join(gpowersHome, "platforms", "kimi", "kimi-skills.json"), append(out, '\n'), 0644)
}
