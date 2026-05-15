# Installing gpowers

## Quickstart

```bash
git clone https://github.com/garrytan/gpowers ~/.gpowers
cd ~/.gpowers
./install
```

The installer auto-detects which AI-coding platforms are present and registers gpowers with each. By default it installs **core/ + roles/ + tools/** and skips the opt-in **business/** module.

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

Install link target: `~/.kimi/skills`. **Adapter generation**: the installer runs `gpowers-platforms gen kimi` to produce `gpowers-<name>/SKILL.md` flat-prefix adapters. Each adapter inlines the using-gpowers preamble. No symlinks (Kimi cannot follow them).

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
