# Installing gpowers

## Quickstart

```bash
git clone https://github.com/garrytan/gpowers ~/.gpowers
cd ~/.gpowers
./install
```

The installer auto-detects which AI-coding platforms are present and registers gpowers with each. By default it installs **core/ + roles/ + tools/** and skips the opt-in **business/** module.

## Windows

On Windows, use the provided `install.bat` wrapper instead of running `./install` directly:

```cmd
git clone https://github.com/garrytan/gpowers %USERPROFILE%\.gpowers
cd %USERPROFILE%\.gpowers
install.bat --platforms=kimi
```

`install.bat` will:
1. Auto-detect **Git Bash** (preferred) or **WSL** on your system
2. Delegate to the Unix `install` script
3. After install, fix Kimi skill links by replacing symlinks with **junctions** (directory hardlinks) or copies — Kimi on Windows cannot follow symlinks

If neither Git Bash nor WSL is found, `install.bat` will prompt you to install Git for Windows from https://git-scm.com/download/win.

**Requirements:**
- Git for Windows (recommended) or WSL
- For symlink support without admin rights, enable **Developer Mode** in Windows Settings

**Note:** The Kimi platform uses junctions on Windows instead of symlinks. Other platforms (Claude Code, Cursor, etc.) still use symlinks and may require Developer Mode or an elevated terminal.

## Install flags

| Flag | Effect |
|---|---|
| `--core-only` | Install only `core/` (methodology); skip roles/tools/business. Minimal footprint. |
| `--with-business` | Activate the opt-in `business/` module after showing its DISCLAIMER. |
| `--no-tools` | Skip `tools/` (rare; only useful for offline / methodology-only setups). |
| `--platforms=<list>` | Comma-separated list of platforms to register with. Default: auto-detect all installed. Valid values: `claude-code,codex,gemini,cursor,opencode,copilot,kimi` |
| `--location=<path>` | Install root, default `~/.gpowers/`. |
| `--link` | Symlink modules from this checkout (developer mode) instead of copying. |
| `--dry-run` | Print the plan; make no changes. |
| `--non-interactive` | Auto-accept defaults (used in CI). |
| `--uninstall` | Remove gpowers (see [UPGRADING.md](UPGRADING.md) for nuances). |

## Per-platform notes

### Claude Code

Install link target: `~/.claude/plugins/gpowers`. Full hook support: gpowers loads automatically via SessionStart. Browser driver default: `claude-in-chrome` MCP.

```bash
./install --platforms=claude-code
```

### Codex

Install link target: `~/.codex/plugins/gpowers`. Partial hook support. Browser driver default: `playwright-cli` (install via `bun add -g @playwright/test`).

### Gemini

Install link target: `~/.config/gemini/extensions/gpowers`. No native hooks — installer appends `<!-- gpowers:begin -->` markers to your `~/.gemini/GEMINI.md`. Browser driver default: `playwright-cli`.

### Cursor

Install link target: `~/.cursor/plugins/gpowers`. No native slash commands — installer appends a fragment to your `.cursorrules`. Browser driver default: `playwright-cli`.

### OpenCode

Install link target: `~/.config/opencode/plugins/gpowers`. Full hook support. Browser driver default: `playwright-cli`.

### Copilot CLI

Install link target: `~/.config/copilot-cli/plugins/gpowers`. No SessionStart — using-gpowers is injected at the start of each prompt. Browser driver default: `playwright-cli`.

### Kimi

Install link target: `~/.kimi/skills`. **Adapter generation**: the installer runs `gpowers-platforms gen kimi` to produce `gpowers-<name>/SKILL.md` flat-prefix adapters. Each adapter inlines the using-gpowers preamble.

**Windows note:** Kimi on Windows cannot follow symlinks. The Windows installer (`install.bat`) automatically replaces symlinks with **junctions** (no admin rights needed) or deep copies as a fallback. If installing manually on Windows, use `install.bat` rather than `./install`.

## Verifying

```bash
gpowers-platforms verify all
```

Reports OK or MISSING for each platform you installed.

## Uninstalling

```bash
./uninstall                  # default: keep user data
./uninstall --remove-data    # also delete ~/.gpowers/data/, config/, analytics/
./uninstall --platform=claude-code   # remove from one platform only
```

See [UPGRADING.md](UPGRADING.md) for the full uninstall flag matrix.
