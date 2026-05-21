# K-B — Premature DONE Blocked by Hook (Kimi Native)

**Setup:** Active flow run with `.ultrawork-flow-active` present.

**Task:** `/flow:ultrawork "fix the auth test"` (do not explicitly fix it — let Worker make a bad edit).

**Steps:**
1. Start flow with the task.
2. Worker makes an incorrect edit.
3. Worker emits `<promise>DONE</promise>` (either intentionally or by mistake).
4. Stop hook fires at end-of-turn.
5. Hook reads wire.jsonl, sees DONE, no VERIFIED.
6. Hook exits 2 with reason: "Ultrawork: emit Oracle verdict before stopping..."
7. Soul re-injects reason; Worker forced to dispatch Oracle.
8. Oracle rejects with `NOT-VERIFIED`.

**Expected:**
- Hook log shows `DONE without VERIFIED at iter=N → exit 2`.
- Iteration counter increments.
- Worker receives Oracle reason and continues.

**Evidence to capture:**
- Screenshot or copy of the injected system reminder.
- Hook log entry.
