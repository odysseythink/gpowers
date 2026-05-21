# Oracle Test Scenarios

These are manual exercise scenarios. Run each by dispatching Oracle (via the host's Task/Agent tool, or via Ultrawork) and verifying the expected outcome.

## P-A — Standalone Oracle Consult

**Setup:** Any project. Dispatch Oracle with a real architecture question, e.g.:

> "Should we add a Redis cache in front of our user lookups? Currently 50 RPS hitting Postgres directly."

**Expect:**
- Three-tier response: Bottom line + Action plan + Effort + Confidence.
- No `Edit` / `Write` tool calls in the transcript.
- No AI-slop words (`comprehensive`, `robust`, etc.) in the response.
- Confidence tag present.
- Response is self-contained (no "let me think about this more...").

## P-B — Ultrawork → Oracle Verification

**Setup:** Run `/ultrawork "add password length validation to the login flow"` on a project with a `login` module + tests.

**Expect:**
- Worker performs the edit, runs tests, emits `<promise>DONE</promise>`.
- Ultrawork dispatches Oracle with the verification preamble.
- Oracle detects `<promise>` tag → switches to Verifier mode.
- Oracle re-runs the test command itself; does NOT trust pasted output.
- Oracle emits `Agent: Oracle\nEvidence:\n- ...\n<promise>VERIFIED</promise>` with at least 2 evidence lines (file + test).

## P-L — Ultrawork Integration Regression

**Setup:** Re-run the full Ultrawork test suite (`core/skills/ultrawork/tests/`) after this change lands.

**Expect:**
- Scenarios A/B/C from `ultrawork/tests/` still pass.
- Kimi scenarios K-A..K-F from `ultrawork/platforms/kimi/tests/` still pass.
- Oracle prompt resolves from `roles/skills/oracle/SKILL.md` via the rewritten pointer.
- No regression in `VERIFIED` / `NOT-VERIFIED` ratios across the 3 base scenarios.

---

**Verification status:**

- P-A Claude Code: ⚠️ tested — content quality high, format compliance ~30% (Bottom line present but position inconsistent, Effort/Confidence often missing, Expanded tier overproduced)
- P-A Kimi (--agent-file): ⚠️ tested — content quality high, format compliance ~60% (Bottom line present, Action plan present, Effort/Confidence still missing despite prompt tightening, response length excessive)
- P-B Ultrawork → Oracle: ⚠️ tested — Kimi skill mechanism loads ultrawork as text-only skill, **not as a runtime protocol**. Worker did NOT emit `<promise>DONE</promise>`; main agent manually dispatched Oracle. Oracle re-ran tests independently, but verdict format was filtered/summarized by main agent, not emitted as raw `Agent: Oracle\nEvidence:\n<promise>VERIFIED</promise>`.
- P-L Ultrawork regression: ⏳ pending — requires true Ultrawork runtime (OpenCode only)

**Key finding:** Kimi supports custom subagent types (`--agent-file`) but does NOT support custom runtime protocols (promise-tag detection, automatic Oracle dispatch, iterative Worker→Oracle loops). Ultrawork on Kimi is advisory-only; true Ultrawork verification requires OpenCode.

All automated install-regression checks (bats) pass. Manual smoke tests recorded as above.
