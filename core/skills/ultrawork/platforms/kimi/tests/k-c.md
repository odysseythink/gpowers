# K-C — Hook Fail-Open Documented Gap (Kimi Native)

**Setup:** Hook installed but renamed or made non-executable mid-session.

**Steps:**
1. Start a normal non-Ultrawork conversation.
2. Rename the hook: `mv .kimi/hooks/ultrawork-stop.sh .kimi/hooks/ultrawork-stop.sh.bak`
3. Type bare `<promise>DONE</promise>` in the conversation.

**Expected:**
- Hook does not fire (missing script).
- Kimi's hook runner returns `action=allow` (fail-open).
- DONE is accepted without Oracle verification.
- This documents the known gap.

**Evidence to capture:**
- Confirmation that the message went through without block.
- Reference to README.md "Known Limitations" section.

**Cleanup:** Restore the hook: `mv .kimi/hooks/ultrawork-stop.sh.bak .kimi/hooks/ultrawork-stop.sh`
