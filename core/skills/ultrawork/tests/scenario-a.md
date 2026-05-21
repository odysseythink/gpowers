# Scenario A — Happy Path

**Setup:** A trivial task that can be completed in one iteration.

**Task:** "Add a function `double(x)` that returns `x * 2`, write a test for it, and verify both pass."

**Steps:**
1. Invoke the skill on your host (e.g., `/ultrawork` or prompt-based).
2. Worker adds `double` function and test.
3. Worker runs test command, pastes passing output.
4. Worker emits `<promise>DONE</promise>`.
5. Oracle dispatches, re-runs tests, sees pass.
6. Oracle emits `<promise>VERIFIED</promise>`.

**Expected:** Loop exits after 1 iteration. Summary returned to user.

**Evidence to capture:**
- Oracle's evidence block cites the test file and command output.
- No `NOT-VERIFIED` reason appears.