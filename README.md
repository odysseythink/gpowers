# gpowers

A unified distribution that merges [superpowers](https://github.com/obra/superpowers) (methodology) and [gstack](https://github.com/garrytan/gstack) (role + capability + business automation) into one cross-platform toolkit for AI-coding assistants.

**Supported platforms:** Claude Code, Codex, Gemini, Cursor, OpenCode, Copilot CLI, Kimi.

## Four modules

- **core/** (14 skills) — methodology: TDD, debugging, planning, brainstorming, code review.
- **roles/** (20 skills) — role-based reviews: `/pr-review`, `/cso`, `/plan-ceo-review`, etc.
- **tools/** (28 skills) — capabilities: `/ship`, `/qa`, `/canary`, `/health`, etc.
- **business/** (20 skills, opt-in) — commercial automation: `/money`, `/money-content`, etc.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full model.

## Quickstart

```bash
git clone https://github.com/garrytan/gpowers ~/.gpowers
cd ~/.gpowers
./install
```

The installer auto-detects which AI-coding platforms are present and registers gpowers with each. For per-platform details, see [docs/INSTALL.md](docs/INSTALL.md).

## Try it

After install, in any supported AI-coding platform:

- Type `/pr-review` for a pre-merge review of the current branch.
- Type `/cso` for a security audit.
- Type `/health` for a code-quality score.
- Type `/ship` to push and open a PR.

Core methodology skills (brainstorming, TDD, debugging, planning) load automatically via the session-start hook on platforms that support it.

## Cross-platform browser abstraction

Skills that use a browser do so through a 9-verb abstraction (`browser.open`, `browser.click`, `browser.read`, ...). Two drivers ship out of the box:

- **claude-in-chrome** (MCP, Claude Code native)
- **playwright-cli** (every other platform)

Adding a driver is a `mkdir + 9 scripts` operation. See [docs/DRIVERS.md](docs/DRIVERS.md).

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — modules, drivers, runtime layout
- [Install](docs/INSTALL.md) — per-platform setup
- [Skills](docs/SKILLS.md) — complete index
- [Commands](docs/COMMANDS.md) — slash-command index by scenario
- [Platforms](docs/PLATFORMS.md) — capability matrices for 7 platforms
- [Runtime layout](docs/RUNTIME_LAYOUT.md) — global vs. project directories, env vars, migration
- [Upgrading](docs/UPGRADING.md) — pulling upstream updates
- [Drivers](docs/DRIVERS.md) — 9-verb interface + adding a new driver
- [Contributing](docs/CONTRIBUTING.md) — workflow, TDD discipline, adding skills

## Status

Early development. The brainstorming spec and 12 sub-plans are in `docs/superpowers/`; implementation in progress.

## License

See [LICENSE](LICENSE).
