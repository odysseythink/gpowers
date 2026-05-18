package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var allModules = []string{"core", "roles", "tools"}

// updateManifest updates the manifest file at manifestPath with the given
// install location and modules, setting installed_at to the current time.
func updateManifest(manifestPath, location string, modules []string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	m["installed_at"] = time.Now().UTC().Format(time.RFC3339)
	m["install_location"] = location
	m["installed_modules"] = modules

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// generateSkillsJSON scans the gpowersHome directory for skills in each module
// and writes a skills.json file containing the discovered skills.
func generateSkillsJSON(gpowersHome string) error {
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
			if os.IsNotExist(err) {
				continue
			}
			return err
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
