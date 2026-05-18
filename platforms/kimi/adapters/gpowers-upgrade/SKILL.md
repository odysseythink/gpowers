---
name: gpowers-upgrade
description: "Pull upstream changes for any gpowers module (core / roles / tools / business) — git subtree mechanics, transform re-application, test re-run, platform manifest refresh. (gpowers adapter for Kimi)"
gpowers-source: tools/skills/gpowers-upgrade/SKILL.md
gpowers-module: tools
---

<!-- gpowers preamble (auto, four-module model) -->


# Using gpowers

You have gpowers — a unified methodology + role + tools automation distribution. There are three modules, two trigger tracks, and one naming convention you must follow.

## The three modules

- **core/** — methodology skills (TDD, debugging, planning, brainstorming, code review, etc.). Apply these automatically when they fit the task. Tag `(core)` when you reference them in replies.
- **roles/** — role-based slash commands (`/pr-review`, `/cso`, `/plan-ceo-review`, `/investigate`, ...). Do NOT invoke these yourself. **Suggest** them to the user when their input matches a role's trigger. Tag `(roles)` when you reference them.
- **tools/** — capability skills (`/ship`, `/qa`, `/canary`, `/health`, ...). Call them on demand when the task requires that capability. Tag `(tools)`.


## Dual-track triggering

- **Auto track** — `core/` only. The session-start hook injected this skill; from here, apply core methodology skills automatically when they apply. Example: bug report → invoke systematic-debugging (core). Implementation request → invoke writing-plans (core) before coding.
- **Explicit track** — `roles/`, `tools/`. Wait for the user to type the slash command. You may *suggest* one when a trigger phrase appears: "preparing to ship" → suggest `/pr-review` + `/cso` + `/qa` before `/ship`.

## Namespace tags in replies

When you reference a gpowers skill in user-facing text, append the module tag in parentheses so the user knows where it lives:

- "I'll use brainstorming (core) to walk this through."
- "Consider `/cso` (roles) for a security review."
- "I'll run /qa (tools) against the staging URL."


## Language consistency

When communicating with the user — asking questions, presenting options, explaining trade-offs, or reporting results — **output in the same language the user is writing in**. If the user writes in Chinese, reply in Chinese. If the user writes in English, reply in English. This reduces comprehension friction and ensures the user can fully understand proposals and make informed decisions.

## Skill priority

When multiple skills could apply, follow this order:
1. **Process skills first** (brainstorming, systematic-debugging, executing-plans)
2. **Implementation skills next** (writing-plans, TDD)
3. **Role / tool skills only when user-invoked** or suggested with explicit user confirmation

## Routing for overlapping skills

Three pairs are intentionally similar but serve distinct purposes. Use this table to decide:

### Debugging / investigation

| Situation | Use |
|---|---|
| Any bug, test failure, unexpected behavior — needs fixing | `systematic-debugging` (core) — auto-triggered, no output doc |
| Root-cause analysis that needs a written investigation report, or when user explicitly wants `/investigate` | `/investigate` (roles) — user-invoked, writes `$(gpowers-path project investigate)/<slug>.md` |

"Iron Law: no fixes without root cause" applies to both. The difference is outputs and invocation: `systematic-debugging` runs silently in the background of any coding session; `/investigate` is a deliberate role-based ceremony with a persisted artifact.

### Brainstorming / ideation

| Situation | Use |
|---|---|
| "I have a feature idea / how should I build X" — design-first workflow | `brainstorming` (core) — auto-triggered, leads to spec + writing-plans |
| "Is this worth building?", "validate my idea", "startup thinking", "office hours" | `/office-hours` (roles) — user-invoked, YC-style six forcing questions + Builder mode |

`brainstorming` always ends in a spec and a plan. `/office-hours` may conclude that an idea is *not* worth building — that's a valid outcome. If `office-hours` results in "yes, build it", transition to `brainstorming` to write the spec.

### Code review

| Situation | Use |
|---|---|
| After completing a task or major feature — dispatch a fresh reviewer subagent | `requesting-code-review` (core) — auto-triggered, subagent reviews your work |
| Pre-merge: comprehensive PR audit against checklist before `/ship` | `/pr-review` (roles) — user-invoked, runs full review with specialist passes |
| After receiving review feedback — deciding what to act on | `receiving-code-review` (core) — auto-triggered, structures your response to feedback |

The typical flow: code → `requesting-code-review` (core, catches issues early) → `/pr-review` (roles, gate before merge) → `receiving-code-review` (core, if reviewer pushes back).

## Reading the rest

Use the `Skill` tool (Claude Code / Codex / OpenCode), `activate_skill` (Gemini), or skill-name reference (Kimi) to load any specific skill. Skill files live under `$GPOWERS_HOME/<module>/skills/<name>/SKILL.md` — never read them by absolute path; use the platform's skill mechanism so per-platform adaptations apply.

Path queries go through `gpowers-path` (`gpowers-path config`, `gpowers-path project plans`, ...) — never concatenate `~/.gpowers/` directly in skills.

<!-- SOURCE: $GPOWERS_HOME/tools/skills/gpowers-upgrade/SKILL.md -->


# gpowers-upgrade

When the user wants to refresh gpowers from upstream:

## Decide scope first

- **All four modules**: `gpowers upgrade` (no argument)
- **One module**: `gpowers upgrade core` (or `roles`, `tools`, `business`)
- **Just check what's new**: `gpowers upgrade --check` (read-only, no merge)

## Recommend a check before pulling

Suggest the user run `gpowers upgrade --check` first. It prints a table of
remote SHAs versus locally recorded SHAs and labels each row "up-to-date" or
"new version available". Use this to decide which modules actually need
pulling.

## Pull workflow

```bash
gpowers upgrade core            # pulls from github.com/obra/superpowers
gpowers upgrade tools           # pulls from github.com/garrytan/gstack
gpowers upgrade                 # all four
```

For each pulled module the runner:

1. Verifies `~/$(gpowers-path project)/` working tree is clean (git subtree requirement).
2. Runs `git subtree pull --squash` from the upstream listed in
   `~/$(gpowers-path project)/upstream-sources.json`.
3. Captures the new SHA and runs the module's `_upgrade-transform.sh` —
   re-applies `namespace:` and `upstream:` frontmatter, `~/$(gpowers-path project)/` path
   rewrites, `superpowers:` → `gpowers:` reference rewrites, and (for browser
   skills) the abstract-verb rewriter.
4. Regenerates all 7 platform manifests via `gpowers-platforms gen all`.
5. Runs the module's bats tests under `tests/unit/<module>/` and
   `tests/integration/<module>/`.
6. Bumps the SHA in `<module>/upstream-source.json`.
7. Commits the transformed state.

## Conflicts

`git subtree pull` may produce a merge conflict if you've made local edits
inside `~/$(gpowers-path project)/<module>/`. The runner stops, prints `git status`, and
exits non-zero. Guide the user through:

```bash
cd ~/.gpowers
# Resolve conflicts in the listed files
git add <resolved-files>
gpowers upgrade --resume
```

`--resume` finishes the merge commit, runs the transform, regenerates
manifests, runs tests, and bumps the SHA — picking up where the conflict
interrupted things.

## Dry run

`gpowers upgrade --dry-run` prints the plan without acting. Use this to show
the user what would happen before they commit to a pull.

## Why each module has its own transform

The transform encodes how gpowers normalizes upstream content. Each module
ships a `_upgrade-transform.sh` that wraps the import helper used at first
install. Changing the normalization in one place (the helper) auto-applies
to upgrades — no separate code path to maintain.

## Telemetry

Upgrade events are recorded under `$(gpowers-path analytics)/upgrade.jsonl`.
Disable with `GPOWERS_ANALYTICS=off`.
