---
name: ultrawork
description: Cross-platform verify-then-exit loop with independent Oracle verification — prevents premature "done" claims by requiring a fresh subagent to re-run verification and emit VERIFIED before exit
namespace: core
upstream: gpowers@local
---

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