# gpowers platforms

gpowers supports seven AI-coding platforms. The single source of truth in `~/.gpowers/` is exposed to each platform via a thin adapter layer under `~/.gpowers/platforms/<platform>/`.

## Native capability matrix

| Platform | Slash commands | Auto hooks | Skill loading | Namespace mode |
|---|---|---|---|---|
| Claude Code | native `/cmd` | SessionStart hook | `Skill` tool | plugin-scoped |
| Codex | native `/cmd` | partial hook | `skill` tool | plugin-scoped |
| Gemini | native `/cmd` | via `GEMINI.md` injection | `activate_skill` | extension |
| Cursor | via `.cursorrules` injection | via session injection | context inject | flat-prefix |
| OpenCode | native `/cmd` | hooks supported | native | plugin-scoped |
| Copilot CLI | native `/cmd` | via prompt injection | `skill` tool | plugin-scoped |
| Kimi | via skill-name reference | inlined preamble per skill | skill-name prefix | flat-prefix (`gpowers-*`) |

Kimi uses **flat-prefix namespace** (`gpowers-brainstorming`, `gpowers-pr-review`, ...) because it cannot follow symlinks and has no SessionStart hook. The using-gpowers preamble is inlined into every Kimi adapter.

## roles/ availability per platform

| Skill class | claude-code | codex | gemini | cursor | opencode | copilot | kimi |
|---|---|---|---|---|---|---|---|
| All roles except design-review and pair-agent | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| design-review | ✓ (MCP) | playwright | playwright | playwright | playwright | playwright | playwright |
| pair-agent | ✓ | degraded | degraded | degraded | degraded | degraded | degraded |

## tools/ availability per platform

| Skill | claude-code | codex | gemini | cursor | opencode | copilot | kimi |
|---|---|---|---|---|---|---|---|
| Non-browser skills (ship, land-and-deploy, health, etc.) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Browser skills (browse, qa, canary, benchmark, setup-browser-cookies) | ✓ (MCP) | playwright | playwright | playwright | playwright | playwright | playwright |
| open-gstack-browser, aidesigner, aidesigner-frontend, setup-gbrain, sync-gbrain | ✓ | degraded | degraded | degraded | degraded | degraded | degraded |

"degraded" — the skill is installed and invocable but its full functionality assumes Claude Code's MCP. On other platforms, behavior falls back to playwright-cli, which lacks some browser-skill features (interactive auth state, native cookie store integration, etc.).

## Browser driver selection per platform

| Platform | Default driver | Override |
|---|---|---|
| Claude Code | claude-in-chrome (MCP) | `GPOWERS_BROWSER_DRIVER=playwright-cli` |
| Codex / Gemini / Cursor / OpenCode / Copilot / Kimi | playwright-cli | `GPOWERS_BROWSER_DRIVER=claude-in-chrome` (if MCP server reachable) |

## business/ availability per platform

All 7 platforms support 100% of `business/` skills. business/ has no browser dependency.

## Install link targets

See `manifest.json` for the authoritative list. Examples:

| Platform | Install link target |
|---|---|
| Claude Code | `~/.claude/plugins/gpowers` |
| Codex | `~/.codex/plugins/gpowers` |
| Gemini | `~/.config/gemini/extensions/gpowers` |
| Cursor | `~/.cursor/plugins/gpowers` |
| OpenCode | `~/.config/opencode/plugins/gpowers` |
| Copilot CLI | `~/.config/copilot-cli/plugins/gpowers` |
| Kimi | `~/.kimi/skills` (adapter dir, not symlink) |

## Known limitations per platform

- **Cursor**: no native slash commands → users invoke skills via context injection in `.cursorrules`. The agent recognizes skill names by prefix.
- **Copilot CLI**: no SessionStart hook → using-gpowers preamble is injected at the start of each prompt rather than the session.
- **Kimi**: skill names must be prefixed `gpowers-` and each adapter inlines the entire using-gpowers preamble. Some token overhead per skill load.

For migration from gstack or superpowers, see [RUNTIME_LAYOUT.md#migration](RUNTIME_LAYOUT.md#migration).
