package main

import (
	"errors"
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

		info, err := os.Stat(src)
		// Optional files may be missing; skip silently
		if errors.Is(err, os.ErrNotExist) {
			// For install binary, try platform-specific fallbacks
			if entry == "install" {
				fallbacks := []string{"install_linux", "install_darwin", "install_darwin_arm64"}
				for _, fb := range fallbacks {
					src = filepath.Join(sourceDir, fb)
					info, err = os.Stat(src)
					if err == nil {
						break
					}
				}
				if err != nil {
					continue
				}
			} else {
				continue
			}
		}
		if err != nil {
			return err
		}

		if entry == "install" && sameFile(src, dst) {
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("remove existing %s: %w", dst, err)
			}
		}

		if linkMode {
			if err := os.Symlink(src, dst); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", src, dst, err)
			}
			continue
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
