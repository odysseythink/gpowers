package main

import (
	"bufio"
	"errors"
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

	sourceDir, err := resolveSourceDir(opts)
	if err != nil {
		return err
	}

	var platforms []string
	if len(requestedPlatforms) > 0 {
		platforms = requestedPlatforms
	} else {
		platforms = detectPlatforms()
	}

	printPlan(opts, platforms)

	if opts.DryRun {
		return nil
	}

	if opts.WithBusiness && !opts.NonInteractive {
		if err := promptBusiness(&opts, sourceDir); err != nil {
			return err
		}
		modules = opts.Modules()
	}

	if err := os.MkdirAll(opts.Location, 0755); err != nil {
		return fmt.Errorf("create install location: %w", err)
	}

	fmt.Println("[install] staging files...")
	if err := stageFiles(sourceDir, opts.Location, modules, opts.Link); err != nil {
		return err
	}

	if err := createRuntimeDirs(opts.Location); err != nil {
		return err
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
			if err := generatePlatformManifest(p, opts.Location, modules); err != nil {
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

func resolveSourceDir(opts Options) (string, error) {
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
	if sourceDir == "" {
		return "", fmt.Errorf("cannot determine source directory; use --source-dir")
	}
	return sourceDir, nil
}

func printPlan(opts Options, platforms []string) {
	modules := opts.Modules()
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
}

func promptBusiness(opts *Options, sourceDir string) error {
	// Business disclaimer is optional; skip prompt if file missing
	disclaimerPath := filepath.Join(sourceDir, "business", "DISCLAIMER.md")
	data, err := os.ReadFile(disclaimerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read disclaimer: %w", err)
	}
	fmt.Println("============================================================")
	fmt.Print(string(data))
	fmt.Println("============================================================")
	fmt.Print("Activate the business/ module? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	ans, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read prompt: %w", err)
	}
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println("[plan] Skipping business activation.")
		opts.WithBusiness = false
	}
	return nil
}

func createRuntimeDirs(location string) error {
	for _, d := range []string{"config", "state", "cache", "data", "analytics", "tmp", "logs"} {
		if err := os.MkdirAll(filepath.Join(location, d), 0755); err != nil {
			return fmt.Errorf("create runtime dir %s: %w", d, err)
		}
	}
	return nil
}
