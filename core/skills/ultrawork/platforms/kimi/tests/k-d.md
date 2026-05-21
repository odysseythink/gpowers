# K-D — Oracle Isolation Inspection (Kimi Native)

**Setup:** Run K-A first to produce an Oracle subagent instance.

**Steps:**
1. Run K-A.
2. After completion, inspect the session directory:
   ```bash
   ls ~/.kimi/sessions/<session_id>/subagents/
   ```
3. Locate the Oracle subagent directory.
4. Inspect its contents:
   ```bash
   ls ~/.kimi/sessions/<session_id>/subagents/<oracle_id>/
   cat ~/.kimi/sessions/<session_id>/subagents/<oracle_id>/wire.jsonl
   ```

**Expected:**
- Oracle has its own `wire.jsonl` (separate from Worker's).
- Oracle's wire log contains the verification commands it ran.
- Oracle's context does not contain Worker's earlier chat history.

**Evidence to capture:**
- Listing of subagents directory.
- Snippet of Oracle wire log showing `Agent: Oracle` and `<promise>VERIFIED</promise>`.
