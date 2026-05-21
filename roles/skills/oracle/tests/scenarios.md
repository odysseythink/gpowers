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
