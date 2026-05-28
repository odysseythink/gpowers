---
name: gpowers-ultrawork
description: "Cross-platform verify-then-exit loop with independent Oracle verification — prevents premature \"done\" claims by requiring a fresh subagent to re-run verification and emit VERIFIED before exit (gpowers adapter for Kimi Code)"
gpowers-source: core/skills/ultrawork/SKILL.md
gpowers-module: core
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

<!-- SOURCE: $GPOWERS_HOME/core/skills/ultrawork/SKILL.md -->


# Ultrawork Loop + Oracle Verification

## Overview

Never trust your own "done". Ultrawork wraps any task in a verify-then-exit loop:

1. **Worker** does the work, runs verification, emits `<promise>DONE</promise>`.
2. **Oracle** (independent subagent) re-runs verification from scratch.
3. Only `VERIFIED` ends the loop. `NOT-VERIFIED` feeds the reason back for another iteration.

**Assurance levels by platform:**
- **High** — Claude Code (`/loop` + `task()`), OpenCode (`task(subagent_type=...)`), Kimi (`/flow:ultrawork` + Stop hook + `Agent` subagent)
- **Medium** — Codex / Cursor / Gemini / Copilot (prompt protocol; self-verify if no subagent tool)

## When to Use

- Any non-trivial implementation task where "done" is ambiguous.
- Before claiming tests pass, build succeeds, or a bug is fixed.
- When you want a second pair of eyes without involving a human.

## When NOT to Use

- Trivial one-liner fixes (overhead exceeds value).
- Tasks where verification is impossible to automate ("make the UI prettier").
- Already-verified work (use `verification-before-completion` standalone instead).

## Entry by Platform

| Platform | Command |
|---|---|
| Claude Code | `/ultrawork "<task>"` |
| Kimi (installed) | `/flow:ultrawork "<task>"` |
| Others | "Use ultrawork mode to: <task>" |

## The Protocol

### Worker Rules

1. **Load verification-before-completion** at the start of every iteration.
2. Do the work (edits, commands, tools).
3. Run verification commands and paste **full output** in the transcript.
4. Only when all verification passes → emit exactly:
   ```
   <promise>DONE</promise>
   ```
   (bare, on its own line, outside code fences)
5. If verification fails → keep working. Do not emit the tag.

### Oracle Rules

1. Dispatched as a **fresh subagent** with clean context (`load_skills=[]`).
2. Re-reads the original task.
3. Inspects changes and **re-runs verification commands itself**.
4. Cites specific evidence before the verdict tag.
5. Emits exactly:
   ```
   Agent: Oracle
   Evidence:
   - <file>: <what was checked>
   - <command>: <output snippet>
   <promise>VERIFIED</promise>
   ```
   or:
   ```
   Agent: Oracle
   Evidence:
   - <file>: <what failed>
   <promise>NOT-VERIFIED: <single-line reason></promise>
   ```

### Loop Exit

| Condition | Action |
|---|---|
| Oracle `VERIFIED` | Exit loop, return summary |
| Oracle `NOT-VERIFIED` | Increment iteration counter, feed reason to next Worker iteration |
| Iteration count == 100 | Fail-loud — print table of all Oracle reasons + diff stats |

## Edge Cases

1. **No subagent tool** → fall back to self-verify mode (lower assurance, documented).
2. **Malformed promise tag** → ignored by regex; Worker keeps working.
3. **Worker emits DONE without verification output in transcript** → Oracle rejects with `NOT-VERIFIED: no verification evidence in transcript`.
4. **Oracle hallucinates VERIFIED without evidence** → missing evidence block = invalid = treated as `NOT-VERIFIED`.
5. **Verification commands unknown** → Worker asks user **before** starting the loop.
6. **Iteration cap hit** → fail-loud summary table, never silent exit.
7. **User cancels** → host-dependent; next user message reverts agent to normal behavior.
8. **Same-context Oracle** (fallback hosts) → bias risk; documented as assurance downgrade.

## Relationship to Other Skills

| Skill | How Ultrawork Uses It |
|---|---|
| `verification-before-completion` | Loaded **inside every Worker iteration** — verification commands & evidence pasting come from here. |
| `executing-plans` | Still owns stepwise execution. Ultrawork wraps execution in an outer verify-loop. |
| `requesting-code-review` | Adjacent concept (independent review). Ultrawork differs: iterative + programmatic contract. |

## Kimi Native Path

Kimi users with kimi-cli 1.43.0+ can install runtime enforcement:

```bash
cd $(gpowers-path config)/core/skills/ultrawork/platforms/kimi
./install.sh        # project-scoped
# or
./install.sh --user # user-scoped
```

This enables `/flow:ultrawork "<task>"` with:
- Stop hook blocking unverified DONE
- Isolated Oracle subagent via `Agent` tool
- Flow runner programmatic iteration dispatch

See `platforms/kimi/README.md` for details.
