# Ultrawork Protocol Contract

## Tag Shape (verbatim from oh-my-opencode)

- **Worker completion:** `<promise>DONE</promise>`
- **Oracle pass:** `Agent: Oracle` on one line, then `<promise>VERIFIED</promise>`
- **Oracle fail:** `Agent: Oracle` on one line, then `<promise>NOT-VERIFIED: <reason></promise>`
- **Match regex (case-insensitive):** `<promise>\s*([^<]+?)\s*</promise>`

## Rules

1. The `<promise>` tag must appear **bare** — outside code fences, on its own line.
2. Malformed tags (extra spaces inside, case mismatch, fenced code) are ignored by the regex and treated as "no promise emitted".
3. Oracle must cite **specific evidence** (file paths, test names, command-output snippets) before the promise tag. Missing evidence = invalid verdict = treated as `NOT-VERIFIED`.

## Exit Conditions

| # | Condition | Result |
|---|---|---|
| 1 | Oracle emits `VERIFIED` | Success — return summary to user |
| 2 | Iteration count reaches **100** | Fail-loud — print table of every iteration's Oracle reason + diff stats |
| 3 | User cancels | Host-dependent — see `platforms.md` |

## Iteration Counter

- Default cap: **100** (matches oh-my-opencode normal mode).
- Each time Oracle returns `NOT-VERIFIED`, increment counter by 1 and feed the reason into the next Worker iteration.
- No cross-session persistence in v1.

## Self-Verify Fallback (hosts without subagent tool)

When the host does not expose a Task/Agent subagent tool, the skill falls back to in-session protocol:
- Worker completes work, runs verification, emits `<promise>DONE</promise>`.
- Worker then re-reads its own context via `gpowers:context-restore` and performs a separate "Oracle pass" as a self-check.
- Same tag contract applies, but without true isolation.
- This mode is **explicitly documented as lower-assurance** in `SKILL.md`.
