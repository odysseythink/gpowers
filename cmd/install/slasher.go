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
		line = strings.TrimSuffix(line, "\r")
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

func readSkillBody(content string) (string, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("no frontmatter")
	}
	return strings.TrimSpace(parts[2]), nil
}
