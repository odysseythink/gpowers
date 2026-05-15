# gpowers architecture

gpowers is a unified distribution that merges the *methodology* skills of [superpowers](https://github.com/obra/superpowers) with the *role + capability + business* skills of [gstack](https://github.com/garrytan/gstack) into one cross-platform toolkit.

## Four modules

gpowers ships four modules. Each lives in its own directory under the install root:

| Module | What it is | Trigger | Frontmatter |
|---|---|---|---|
| **core/** | Methodology skills — TDD, debugging, planning, brainstorming, code review | Auto, via SessionStart hook | `namespace: core` |
| **roles/** | Role-based reviews — `/pr-review`, `/cso`, `/plan-ceo-review`, etc. (20 skills) | Explicit (slash command) | `namespace: roles` |
| **tools/** | Capability skills — `/ship`, `/qa`, `/health`, `/canary`, etc. (28 skills) | On demand by agent or user | `namespace: tools` |
| **business/** | Commercial automation — `/money`, `/money-content`, etc. (20 skills, opt-in) | Explicit, requires `--with-business` | `namespace: business` |

The four-module model is taught to the agent at session start via the `using-gpowers` skill (the entry point that replaces `using-superpowers`).

## Dual-track triggering

gpowers has two trigger tracks. Knowing which applies to a given skill avoids both under-use (forgetting useful skills) and over-reach (auto-invoking role reviews the user didn't ask for).

- **Auto track — `core/` only.** The session-start hook injects `using-gpowers` content into each new session on platforms that support it (Claude Code, Codex, OpenCode have full hook support; Gemini and Cursor get fragment injection; Copilot gets prompt injection; Kimi gets the preamble inlined into each adapted skill). Agents apply core skills automatically when they fit the task.
- **Explicit track — `roles/`, `tools/`, `business/`.** The user invokes these by typing a slash command. The agent may *suggest* one in conversation but must not invoke it without an explicit user signal.

A typical session uses many core skills automatically (brainstorming → writing-plans → TDD → systematic-debugging) and zero, one, or two explicit roles/tools.

## Browser driver abstraction

Spec §4 introduces a 9-verb browser-driver interface so that skills can use a browser without binding to any specific MCP server or CLI:

```
browser.open · browser.click · browser.type · browser.read
browser.screenshot · browser.wait · browser.eval · browser.cookies · browser.close
```

Two drivers ship out of the box:
- **claude-in-chrome** — translates verbs to Anthropic's `claude-in-chrome` MCP tools (Claude Code native).
- **playwright-cli** — translates verbs to Playwright JS API over a long-lived Node runner (every other platform).

Skills call `gpowers-browser <verb>` and read `$GPOWERS_BROWSER_DRIVER` (exported by `tools/drivers/browser/select-driver.sh`). Adding a new driver = `mkdir tools/drivers/browser/<new>/ + cp template/*.sh`. See [DRIVERS.md](DRIVERS.md) for the full spec.

## Runtime layout (two-layer)

gpowers splits runtime data between a *global* layer and a *per-project* layer:

- **Global** — `~/.gpowers/` holds config, state, cache, telemetry, and the install itself. Cross-project, machine-wide, single source of truth for skill content.
- **Per-project** — `<repo>/.gpowers/` holds project-scoped artifacts: plans, designs, retros, learnings, investigations, canary history, etc. These commit with the project; the team shares decision memory.

`gpowers-path` is the single resolver — skills never concatenate `~/.gpowers/` directly. Detailed mapping and environment variables are in [RUNTIME_LAYOUT.md](RUNTIME_LAYOUT.md).

## Cross-platform exposure

A single source of truth in `~/.gpowers/` is *exposed* to each platform via lightweight platform-specific shims under `~/.gpowers/platforms/<platform>/`. Six platforms get symlinks into their plugin directories; Kimi (which can't follow symlinks) gets generated `gpowers-<name>/` adapters. See [PLATFORMS.md](PLATFORMS.md) for the matrix.

## Upgrade and migration

`gpowers upgrade [<module>]` pulls upstream updates per module via `git subtree` (see [UPGRADING.md](UPGRADING.md)). `gpowers migrate` ports existing gstack and superpowers users into the unified layout (see [RUNTIME_LAYOUT.md#migration](RUNTIME_LAYOUT.md#migration)).
