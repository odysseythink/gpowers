# Ultrawork Loop + Oracle Verification — Design

**Date:** 2026-05-21
**Owner:** ranwei693532
**Status:** Approved for plan
**Source of inspiration:** oh-my-opencode v3.17.10 (`src/hooks/ralph-loop/`, `src/features/builtin-commands/templates/ralph-loop.ts`)

## Problem

When the host agent claims a task is "done", that claim is unverified. The agent may have:
- Skipped verification commands.
- Misread test output.
- Stopped halfway and reported partial success.

oh-my-opencode solves this with an *ultrawork loop*: the worker emits a completion promise, an independent Oracle subagent re-verifies, and the loop exits only after Oracle issues `VERIFIED`. The mechanism is implemented as an OpenCode plugin hook — runtime-level, OpenCode-only.

gpowers is cross-platform and content-only (no runtime). We want the same guarantee — premature "done" gets caught — without owning a runtime.

## Goal

Ship a gpowers core skill that:

1. Defines a portable completion-promise contract any host can follow.
2. Enforces independent verification via a fresh Oracle subagent dispatched through the host's existing Task/Agent tool.
3. Drives iteration through Claude Code's built-in `/loop` skill on the primary host, with a graceful protocol-only fallback for other hosts.
4. Fails loud at an iteration cap rather than silently exiting partial work.

## Non-goals (v1)

- Cross-session state persistence (host crash mid-loop = work lost). Defer to v2.
- Model routing per role (Worker on Claude, Oracle on GPT). That's the discipline-agent personas spec.
- Parallel Oracles. Complexity outweighs gain at v1.
- Automated end-to-end runner. Reuses gpowers' existing manual scenario scripts.

## Architecture

- **New skill:** `core/skills/ultrawork/`
- **Module:** `core/` (methodology — siblings of `verification-before-completion`, `executing-plans`).
- **No new runtime code.** Pure markdown content distributed through gpowers' existing platform-templating flow.
- **No new install surface** — `./install` picks it up automatically.

### Relationship to existing skills

| Skill | Relationship |
|---|---|
| `verification-before-completion` | Ultrawork loads it inside each Worker iteration; verification commands & evidence pasting come from there. |
| `executing-plans` | Still owns stepwise execution. Ultrawork wraps the execution in an outer verify-loop. |
| `requesting-code-review` | Conceptually adjacent (independent review). Ultrawork differs: iterative + bound to a programmatic contract. |
| Claude Code `/loop` (host built-in) | Loop driver on Claude Code. Ultrawork delegates iteration to it. |

### Cross-platform behavior

- **Claude Code:** skill instructs the agent to invoke `/loop /ultrawork-continue "<task>"`. `/ultrawork-continue` enforces the verify-then-exit contract each iteration.
- **OpenCode / Codex / Gemini / Cursor / Copilot / Kimi:** skill degrades to in-session protocol. Worker self-monitors and dispatches Oracle if the host exposes a subagent tool; otherwise switches to self-verify mode (lower assurance, called out explicitly in the skill).

## Components

### Files produced

```
core/skills/ultrawork/
├── SKILL.md           # primary skill (entry point)
├── oracle.md          # Oracle subagent prompt template
├── protocol.md        # tag contract, exit conditions, max iterations
├── platforms.md       # per-host integration notes
└── tests/             # manual exercise scripts (Scenarios A/B/C below)
```

### Roles

**Worker — the host agent**

- Receives the user's task plus the ultrawork preamble.
- Loads `verification-before-completion` before emitting any promise.
- Runs verification commands (tests, lint, build) and pastes passing output in the transcript.
- Emits exactly `<promise>DONE</promise>` only after the above.

**Oracle — a fresh subagent**

- Dispatched via the host's Task/Agent tool with `load_skills=[]` (clean context).
- Receives:
  - Original task description (verbatim).
  - The recent transcript or diff (host-dependent payload).
  - Oracle prompt template from `oracle.md`.
- Re-reads task, inspects changes, re-runs verification commands itself (does **not** trust Worker's pasted output).
- Emits exactly `Agent: Oracle\n<promise>VERIFIED</promise>` on pass, or `Agent: Oracle\n<promise>NOT-VERIFIED: <reason></promise>` on fail.
- Must cite specific evidence (file paths, test names, command output) before the promise tag. Missing evidence = invalid verdict = treated as `NOT-VERIFIED`.

**Loop driver**

- Claude Code: `/loop` re-invokes `/ultrawork-continue` until the verify-exit condition fires.
- Other hosts: Worker self-loops through the protocol section of `SKILL.md`.

### Contract (verbatim tag shape — identical to oh-my-opencode)

- Worker completion: `<promise>DONE</promise>`
- Oracle pass: `Agent: Oracle\n<promise>VERIFIED</promise>`
- Oracle fail: `Agent: Oracle\n<promise>NOT-VERIFIED: <reason></promise>`
- Match regex: `<promise>\s*([^<]+?)\s*</promise>` (case-insensitive)

### Iteration cap

- Default: **100** (matches oh-my-opencode's normal mode).
- Future flag `--max-iter=N` lifts to 500 to match oh-my-opencode's ultrawork mode. Not in v1.

### Exit conditions

1. Oracle emits `VERIFIED` → success exit, return summary.
2. Iteration count reaches cap → fail-loud exit with table of every iteration's Oracle reason + diff stats.
3. User cancels (host-dependent — see `platforms.md`).

## Data flow (one run)

```
User: /ultrawork "fix the failing auth test and verify"
   │
   ▼
[Worker iteration N]
  1. Read SKILL.md → load protocol + verification-before-completion
  2. Do the work (edits, tools, etc.)
  3. Run verification commands → paste output in transcript
  4. If all green → emit <promise>DONE</promise>
     Else → continue working
   │
   ▼
[Promise detected in transcript]
  5. Dispatch Oracle subagent via Task/Agent tool:
        prompt = oracle.md + original task + recent transcript/diff
        load_skills = []
   │
   ▼
[Oracle subagent]
  6. Re-read the original task
  7. Inspect changes (read files, re-run verification commands)
  8. Decide: VERIFIED or NOT-VERIFIED + reason
  9. Emit:  Agent: Oracle
            <promise>VERIFIED</promise>   (or NOT-VERIFIED: <reason>)
   │
   ▼
[Result branch]
  VERIFIED       → loop exits, summary to user
  NOT-VERIFIED   → reason becomes next iteration's input,
                   iteration count +1, back to Worker iteration N+1
  iter == cap    → fail-loud exit
```

**State between iterations is conversational, not file-based.** Only Oracle's `NOT-VERIFIED: <reason>` crosses the boundary. No state file on disk in v1.

**Oracle re-runs verification itself.** Core "stop premature done" guarantee — Worker can lie, Oracle re-checks from a clean slate.

## Error handling & edge cases

1. **Host doesn't expose a Task/Agent tool.** Skill detects at start of `/ultrawork`. Falls back to self-verify mode (same protocol, Worker re-loads its own context via `gpowers:context-restore` and verifies as a separate pass). Documented as lower-assurance.

2. **Malformed promise tag** (`<promise> DONE </promise>`, case mismatch, code-fence wrapping). Protocol section gives the exact case-insensitive regex matching the oh-my-opencode shape. Skill explicitly instructs Worker to emit *bare* tags outside code fences.

3. **Worker emits promise without running verification.** Oracle's first check: "did the transcript contain verification command output?" If not → `NOT-VERIFIED: no verification evidence in transcript`.

4. **Oracle hallucinates VERIFIED.** Oracle prompt requires citing *specific evidence* (file paths, test names, command-output snippets) **before** the promise tag. No evidence cited → invalid verdict → loop treats as `NOT-VERIFIED`.

5. **Verification commands unknown.** Worker reads `verification-before-completion`, then `AGENTS.md`/`CLAUDE.md` in the project. If still unclear, Worker asks the user **before** starting the loop (not mid-loop).

6. **Iteration cap hit.** Fail-loud: summary table of each iteration's Oracle reason + Worker diff stats. Never silent exit.

7. **User cancels.** Documented per host in `platforms.md`. Claude Code: stop the `/loop`. Other hosts: stop responding to the protocol — agent reverts to normal behavior on next user message.

8. **Same-context Oracle drift.** On hosts without true subagent isolation, the Oracle runs in the Worker's context and is biased. Skill explicitly calls this out as the reason Claude Code is the recommended primary path; lowered assurance is documented for fallback mode.

## Testing

Skill ships content, not code. "Tests" = spec validation + manual exercises.

1. **Spec-embedded examples in `SKILL.md`:**
   - Valid `<promise>DONE</promise>` placement (verification output above it).
   - Malformed promise that Oracle should reject.
   - Oracle response with evidence citations.

2. **Manual exercise scripts in `core/skills/ultrawork/tests/`:**
   - **Scenario A — happy path:** "add a function that doubles a number and write its test." Expect: 1 Worker iter → Oracle VERIFIED → exit.
   - **Scenario B — premature DONE caught:** Worker tries to emit `DONE` while verification commands fail (induce via misleading hint). Expect: Oracle `NOT-VERIFIED` citing the failing command; loop continues.
   - **Scenario C — iteration cap fail-loud:** impossible task (contradictory test). Expect: cap hit, summary table printed.

3. **Cross-platform smoke check:** run Scenario A on at least one fallback host (Codex or Cursor). Confirm in-session protocol completes without subagent dispatch.

4. **Install regression:** run gpowers' install/test target; confirm `core/skills/ultrawork/` templates cleanly into every platform adapter directory.

**Not in v1:** automated runner driving a real model end-to-end (belongs to a `benchmark-models` extension), regression test for transcript regex (rely on copy-paste correctness against the oh-my-opencode shape).

## Open questions / risks

- **Fallback assurance gap.** Pure-protocol mode on non-Claude hosts gives the same guarantee oh-my-opencode does *only* if the model complies with prompt instructions. We document the gap; OpenCode parity (a per-platform runner) is on the roadmap (see roadmap doc).
- **Tag clashes.** `<promise>DONE</promise>` is rare in normal transcripts but not impossible. Mitigation: regex must match the *exact bare-line* pattern on its own line.
- **Subagent payload size.** Sending the recent transcript to Oracle can be large on long tasks. v1 accepts this; v2 may switch to diff-only payloads.

## Out of scope — escalated to roadmap

See `2026-05-21-oh-my-opencode-port-roadmap.md` for the broader list of features being ported from oh-my-opencode (discipline-agent personas, `/init-deep`, IntentGate, AI-slop detector, skill-embedded MCPs).
