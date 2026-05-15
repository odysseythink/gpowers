# Contributing to gpowers

gpowers is built with its own methodology: brainstorming → spec → plan → TDD. Contributions follow the same flow.

## Where to start

- **A new skill** — go to "Adding a skill" below.
- **A new browser driver** — go to "Adding a driver".
- **A new platform** — see [PLATFORMS.md](PLATFORMS.md) and `platforms/_platform-shapes.json`; one new entry there + a `_gpowers-gen-platform.sh` smoke test usually does it.
- **A bug fix** — open an issue first if behavior change is significant; otherwise PR with a failing test.

## Workflow

1. Open an issue describing the change. For new skills, propose the slash command name and module placement.
2. **Brainstorm + spec.** Non-trivial changes go through `docs/superpowers/specs/`. Reuse the `brainstorming` core skill.
3. **Write a plan** in `docs/superpowers/plans/` listing tasks with TDD steps. The pattern is "failing test → minimal code → passing test → commit" per task.
4. **Implement.** Each commit pairs a test with the code it tests. No "fix tests later" commits.
5. **Open a PR.** The CI runs unit + integration + lint + a best-effort smoke for claude-code.

## Adding a skill

1. Decide the module: `core/` (methodology), `roles/` (slash review), `tools/` (capability), `business/` (commercial, opt-in).
2. Create the directory:
   ```bash
   mkdir -p <module>/skills/<name>
   ```
3. Write `<module>/skills/<name>/SKILL.md` with the frontmatter:
   ```yaml
   ---
   name: <name>
   description: One-line description (under 150 chars)
   namespace: <module>        # required
   slash: /<command>          # required for roles/tools/business; omit for core
   upstream: gpowers-native   # for new skills (no upstream)
   requires-driver: browser   # only if the skill uses the browser
   ---
   ```
4. Write the body in the second-person Markdown style superpowers uses ("When the user…", "First check…", "Run this command…"). Reference other skills with their slug — `[[brainstorming]]`.
5. For browser-using skills: source `tools/drivers/browser/select-driver.sh` in a Preamble block, and use only `gpowers-browser <verb>` — never reference `mcp__claude-in-chrome__*` or `playwright` literally. See [DRIVERS.md](DRIVERS.md).
6. Add a slash-command file: when the installer runs `gpowers-platforms gen all` it auto-creates `platforms/<platform>/commands/<name>.md`. You only edit it if you want platform-specific tweaks.
7. Run tests:
   ```bash
   ./tests/run.sh unit <module>
   ```

## Adding a driver

See [DRIVERS.md](DRIVERS.md) for the full procedure. Short version:

1. `mkdir -p tools/drivers/browser/<your-driver>` and add 9 verb scripts.
2. Add `capabilities.json`.
3. Add a row to `tests/integration/drivers/parity.bats`.
4. Update `tools/drivers/browser/select-driver.sh` if the new driver should ever be the default.

## TDD discipline

Every commit pairs a failing test that documents the change with the minimal code to make it pass. This is encoded in the writing-plans skill's task structure:

- Step 1: write the failing test
- Step 2: run it; assert it fails
- Step 3: minimal implementation
- Step 4: run; assert it passes
- Step 5: commit

Don't batch. Don't commit "framework only" PRs without tests. The reviewer will ask for a test.

## Code style

- Bash: shellcheck-clean at `-S warning`. Strict mode (`set -euo pipefail`) on all new scripts.
- Markdown: present-tense, second-person voice for skills.
- Test naming: `<feature>.bats` for unit, mirror directory structure under `tests/`.
- Commit messages: imperative ("feat(tools): add /simplify skill"), one logical change per commit.

## Pull requests

- One feature per PR.
- Link the issue.
- Run `./tests/run.sh all` locally before pushing.
- Squash isn't required, but each commit must be passing on its own (CI runs against each commit on push, not just the PR head).
