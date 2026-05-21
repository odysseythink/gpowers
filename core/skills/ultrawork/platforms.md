# Ultrawork — Per-Host Integration Notes

## Claude Code (Primary / High Assurance)

**Entry:** `/ultrawork "<task>"` or `/ulw "<task>"`

**Loop driver:** Claude Code's built-in `/loop` skill.

**Flow:**
1. User types `/ultrawork "fix the auth test"`.
2. Agent loads `SKILL.md` + `protocol.md` + `oracle.md`.
3. Agent invokes `/loop /ultrawork-continue "fix the auth test"`.
4. Each `/ultrawork-continue` iteration:
   - Worker does work, runs verification, emits `<promise>DONE</promise>`.
   - `/loop` detects `DONE` and ends the iteration.
   - Next `/loop` iteration dispatches a fresh `task()` subagent with Oracle prompt + original task + recent transcript.
   - Oracle returns verdict.
   - If `VERIFIED` → `/loop` exits.
   - If `NOT-VERIFIED` → next `/ultrawork-continue` iteration with reason.

**Iteration cap:** `/loop` max-iter parameter or agent-enforced counter.

**User cancel:** Stop the `/loop` with the host's cancel gesture (Ctrl+C / Esc).

## OpenCode (High Assurance)

**Entry:** Same as Claude Code if OpenCode exposes `/loop`; otherwise prompt-based.

**Subagent tool:** OpenCode's `task(subagent_type=...)` provides true context isolation.

**Notes:** Same flow as Claude Code. OpenCode parity is on the roadmap for a dedicated adapter.

## Kimi (High Assurance — Native Path)

**Entry:** `/flow:ultrawork "<task>"`

**Loop driver:** Kimi Soul executing the Mermaid flowchart in `platforms/kimi/flow.md`.

**Enforcement:** Server-side `Stop` hook (`platforms/kimi/stop-hook.sh`) blocks bare `<promise>DONE</promise>` unless Oracle `VERIFIED` is present. See `platforms/kimi/README.md` for install/uninstall.

**Oracle dispatch:** `Agent` tool with `subagent_type="oracle"`, registered via `oracle.yaml` in user's `agent.yaml`.

**Installation (opt-in):**
```bash
cd core/skills/ultrawork/platforms/kimi
./install.sh          # project-scoped (./.kimi/)
./install.sh --user   # user-scoped (~/.kimi/)
```

## Codex / Cursor / Gemini / Copilot (Medium Assurance — Protocol Fallback)

**Entry:** Prompt-based — "Use ultrawork mode to: <task>"

**Loop driver:** Worker self-loops through the protocol section of `SKILL.md`.

**Subagent:** If host exposes a subagent tool, use it for Oracle isolation. Otherwise self-verify.

**Assurance:** Lower — relies on model compliance with prompt instructions. Gap is explicitly documented in `SKILL.md`.

## Fallback Self-Verify Mode (All Hosts)

If no subagent tool is available:
1. Worker completes work and emits `<promise>DONE</promise>`.
2. Worker then performs a separate self-check pass, re-reading its context via `gpowers:context-restore`.
3. Worker issues its own Oracle-style verdict using the same tag format.
4. This is a single-agent simulation of the two-agent loop. Assurance is lower but still better than no verification.