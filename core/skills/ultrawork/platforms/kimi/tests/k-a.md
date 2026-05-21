# K-A — Happy Path (Kimi Native)

**Setup:** Fresh install of Ultrawork Kimi native path in a test project.

**Task:** `/flow:ultrawork "add a function double(x) that returns x*2, write a test, and verify"`

**Steps:**
1. Ensure installer ran successfully: `./install.sh`
2. Restart Kimi.
3. Run `/flow:ultrawork "add a function double(x) that returns x*2, write a test, and verify"`.
4. Worker adds function and test.
5. Worker runs tests, pastes output.
6. Worker emits `<promise>DONE</promise>`.
7. Stop hook sees DONE without VERIFIED → blocks, injects reason.
8. Worker dispatches Oracle via `Agent(subagent_type="oracle")`.
9. Oracle re-runs tests, cites evidence, emits `<promise>VERIFIED</promise>`.
10. Flow runner sees VERIFIED → END(success).

**Expected:**
- Loop exits after 1 iteration.
- `${KIMI_SESSION_DIR}/.ultrawork-iter` contains `1`.
- `${KIMI_SESSION_DIR}/.ultrawork-flow-active` is removed.
- User sees success summary.

**Evidence to capture:**
- Oracle's evidence block cites test file and command.
- Hook log shows `VERIFIED found → exit 0`.
