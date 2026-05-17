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

	installName := "install"
	if runtime.GOOS == "windows" {
		installName = "install.exe"
	}

	for _, entry := range entries {
		src := filepath.Join(sourceDir, entry)
		dst := filepath.Join(targetDir, entry)

		if entry == "install" {
			src = filepath.Join(sourceDir, installName)
			dst = filepath.Join(targetDir, installName)
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

	// Stage the browse Go binary into bin/ if present alongside the installer
	if err := stageBrowseBinary(sourceDir, targetDir, linkMode); err != nil {
		return err
	}

	return nil
}

func stageBrowseBinary(sourceDir, targetDir string, linkMode bool) error {
	browseName := "browse"
	if runtime.GOOS == "windows" {
		browseName = "browse.exe"
	}

	src := filepath.Join(sourceDir, browseName)
	dst := filepath.Join(targetDir, "bin", browseName)

	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// browse binary is optional (may be missing in dev mode or source builds)
			return nil
		}
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
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
		return nil
	}

	mode := info.Mode() | 0111 // ensure executable
	if err := copyFile(src, dst, mode); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	fmt.Printf("[install] staged browse binary: %s\n", dst)
	return nil
}
