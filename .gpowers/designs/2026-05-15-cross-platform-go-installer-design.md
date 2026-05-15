# Cross-Platform Go Installer Design

## Overview

Replace the existing Windows (`install.bat`) and Unix (`install`) bash-based installers with a single Go program that is fully self-contained, cross-platform, and has zero external dependencies.

The Go installer becomes the sole installation entry point for gpowers. The existing `install.bat` and `install` scripts will be removed from the repository.

## Goals

- Single binary (`install` on Unix, `install.exe` on Windows) with no runtime dependencies.
- Behavior identical to the existing bash installers: same CLI flags, same output format, same file layout.
- Cross-platform file operations: symlinks on Unix, symlinks/junctions/deep-copy on Windows.
- Self-contained JSON and Markdown frontmatter parsing (no `jq`, no YAML library).

## Non-Goals

- Do not migrate other gpowers bash utilities (e.g., `bin/_gpowers-lib.sh`, `bin/gpowers-detect-platforms`) beyond what the installer itself needs.
- Do not change the installation directory layout or manifest schema.

## Directory Structure

```
cmd/install/
├── main.go          # Entry point and orchestration
├── flags.go         # CLI flag parsing and Options struct
├── staging.go       # File copy, symlink, and junction logic
├── manifest.go      # manifest.json and skills.json read/write
├── platforms.go     # Platform detection, manifest generation, registration
├── kimi.go          # Kimi adapter generation and kimi-skills.json
├── slasher.go       # Slash command discovery across skill modules
└── utils.go         # Home directory, path expansion, recursive deep-copy helpers
```

Build artifacts are placed in the repository root (replacing the old scripts):
- `install` (Linux/macOS)
- `install.exe` (Windows)

## Build Flow

```bash
# Windows
go build -o install.exe ./cmd/install

# Linux / macOS
go build -o install ./cmd/install
```

A GitHub Actions workflow (or local build script) compiles matrix targets:
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`

Precompiled binaries are committed to the repository root so users can `git clone` and run immediately.

## CLI Flags

Fully backward-compatible with the existing bash installers:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--core-only` | bool | false | Install only `core/` |
| `--with-business` | bool | false | Include `business/` module (shows DISCLAIMER) |
| `--no-tools` | bool | false | Skip `tools/` module |
| `--no-roles` | bool | false | Skip `roles/` module |
| `--platforms` | string | `""` (auto-detect) | Comma-separated platform list |
| `--location` | string | `~/.gpowers` | Install root |
| `--link` | bool | false | Symlink source repo (dev mode) instead of copying |
| `--dry-run` | bool | false | Print plan, make no changes |
| `--non-interactive` | bool | false | Auto-accept defaults (CI mode) |
| `--source-dir` | string | auto | Override source directory detection |
| `--help` | bool | false | Show usage |

## Core Modules

### `flags.go`

Defines `Options` struct and `ParseFlags()`. Validates flag combinations (e.g., `--core-only` implies `--no-tools` and `--no-roles`).

### `main.go`

Orchestrates the install pipeline in order:

1. Parse CLI flags.
2. Resolve module list (`computeModules`).
3. Detect/filter platforms (`resolvePlatforms`).
4. Print plan to stdout (`[plan] ...`).
5. If `--dry-run`, exit 0.
6. If `--with-business` and not `--non-interactive`, show DISCLAIMER and prompt for confirmation.
7. Create install location.
8. Stage source files (copy or symlink).
9. Create runtime directories (`config`, `state`, `cache`, `data`, `analytics`, `tmp`, `logs`).
10. Update `manifest.json`.
11. Generate platform manifests / Kimi adapters.
12. Register platforms (create symlinks / junctions).
13. Print success message.

Output format matches existing bash scripts exactly:
- `[plan] ...`
- `[install] ...`
- `[install] warn: ...`
- `[install] error: ...`

### `staging.go`

**Entry list** (identical to bash installer):
```
core, roles, tools, platforms, lib, bin, templates,
manifest.json, upstream-sources.json, install, uninstall, README.md, LICENSE
```
Optionally append `business`.

Note: `install` refers to the Go installer binary itself (`install` on Unix, `install.exe` on Windows). It is staged into the target directory so the user can re-run or upgrade from the installed location.

**Copy mode (default)**:
- Recursively copy each entry from `sourceDir` to `targetDir`.
- If target exists, remove then re-copy.
- Self-file protection: if the source file is the currently running executable and source == target path, skip.

**Link mode (`--link`)**:
- Unix: `os.Symlink(source, target)`.
- Windows: attempt `os.Symlink(source, target)`. If that fails and the target is a Kimi adapter directory, fallback to junction. Otherwise fail.

**Deep Copy helper**:
Recursive `filepath.Walk` + `os.MkdirAll` + `copyFile` (~30 lines), since Go stdlib has no `cp -R`.

### `manifest.go`

- Read `manifest.json` into `map[string]interface{}` (or a small struct) using `encoding/json`.
- Update fields: `installed_at` (UTC ISO-8601), `install_location`, `installed_modules`.
- Write back with indentation.
- Generate `skills.json` and `kimi-skills.json` by marshaling small structs directly.

No `jq` dependency.

### `platforms.go`

**Platform Detection**:
Check marker directories for each platform:
- `claude-code`: `~/.claude`
- `codex`: `~/.codex`
- `gemini`: `~/.config/gemini`
- `cursor`: `~/.cursor`
- `opencode`: `~/.config/opencode`
- `copilot`: `~/.config/copilot-cli`
- `kimi`: `~/.kimi`

Cross-platform home directory resolution:
- Windows: prefer `%USERPROFILE%`
- Unix: `$HOME`

**Generic Platform Manifest Generation** (non-Kimi):
1. Read `platforms/_platform-shapes.json`.
2. Per platform, write:
   - `<manifest>` (e.g., `plugin.json`, `extension.json`)
   - `skills.json` (all skills index)
   - `commands/<slash>.md` per slash command
   - `hooks.json` (if `supports_hooks == true`)

**Platform Registration**:
- Non-Kimi: create symlink from platform plugin dir to `~/.gpowers/platforms/<platform>`.
- Kimi: create symlinks/junctions from `~/.kimi/skills/<adapter>` to `~/.gpowers/platforms/kimi/adapters/<adapter>`.

### `kimi.go`

**Adapter Generation** (equivalent to `_gpowers-gen-kimi.sh`):
1. Read `core/skills/using-gpowers/SKILL.md`, extract body after frontmatter as `PREAMBLE`.
2. Create `gpowers` router adapter (frontmatter + preamble only).
3. Iterate `core/roles/tools/business/skills/*`:
   - Skip `using-gpowers`.
   - Parse frontmatter for `name`, `description`.
   - Generate adapter named `gpowers-<name>`.
   - Output: new frontmatter (`name`, `description` with suffix, `gpowers-source`, `gpowers-module`) + preamble + source body.
4. Write `platforms/kimi/kimi-skills.json` (adapter name list).

### `slasher.go`

Equivalent to `_gpowers-list-slashes.sh`:
- Walk `core/roles/tools/business/skills/*/SKILL.md`.
- Extract `slash:` and `requires-driver:` from frontmatter.
- Return structured data for manifest generation.

### `utils.go`

- `homeDir()` — cross-platform home directory.
- `expandPath(p string)` — expand `~` to home directory.
- `copyFile(src, dst string, mode os.FileMode) error`.
- `copyDir(src, dst string) error`.

## Cross-Platform File Operations

| OS | Operation | Implementation |
|---|---|---|
| Unix | Symlink | `os.Symlink` |
| Windows | Symlink | `os.Symlink` (requires Developer Mode or admin) |
| Windows | Junction | `exec.Command("cmd", "/c", "mklink", "/J", target, source)` |
| All | Deep Copy | Recursive `filepath.Walk` helper |

**Kimi on Windows** (critical path):
1. Remove existing target (file, directory, or junction).
2. Attempt `mklink /J`.
3. If junction fails, deep copy.
4. Log outcome: `[OK] Junction: <name>`, `[OK] Copied: <name>`, or `[ERR] Failed: <name>`.

**Non-Kimi on Windows**:
- Attempt `os.Symlink`.
- If that fails, do **not** fallback to junction (per existing docs, these platforms require Developer Mode or elevated terminal for symlinks).

## Markdown Frontmatter Parsing

No external YAML library. A small (~40 line) parser handles the fixed subset used by gpowers:

1. Split file content by `---` delimiters.
2. Extract key-value pairs from lines between delimiters.
3. `description` special handling:
   - Quoted string: strip outer matching quotes.
   - Multiline (`|` or `>`): read first content line.
   - Plain string: use as-is.

This covers all current gpowers skill frontmatters.

## Source Directory Detection

The installer needs to locate the gpowers repository root to copy files from:

1. If `--source-dir` is provided, use it.
2. Try `os.Executable()` to find the installer's own path, then `filepath.Dir()`.
3. If that fails (e.g., `go run` mode), fallback to `os.Getwd()`.

## Error Handling

- Fatal errors (file system failures, manifest corruption) return error and terminate installation.
- Platform registration failures log a warning (`[install] warn: ...`) and continue, matching existing `|| true` behavior in bash.
- All errors include context so the user knows which step failed.

## Testing Strategy

Since the installer is a single `package main` CLI, tests are primarily:
- Unit tests for `flags.go`, `utils.go`, and frontmatter parsing.
- Integration tests that run the installer in a temporary directory and verify the output file tree, manifest content, and symlink targets.
- Platform-smoke tests can be added later for CI matrix builds.

## Backward Compatibility

- All existing flags are supported.
- Output format is identical (`[plan]`, `[install]`, `[warn]`, `[err]` prefixes).
- Installed directory layout is unchanged.
- `manifest.json` schema is unchanged.

## Migration Plan

1. Implement Go installer in `cmd/install/`.
2. Build and commit `install` and `install.exe` to repository root.
3. Remove `install.bat` and `install` (bash).
4. Update `docs/INSTALL.md` to reference the new Go binary.
5. Verify on Windows (junction behavior) and Unix (symlink behavior).
