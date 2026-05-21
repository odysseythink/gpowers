# K-F — ACP Mode Smoke (Kimi Native)

**Setup:** Kimi running as an ACP server for an IDE (e.g., Zed, JetBrains).

**Steps:**
1. Start Kimi in ACP mode: `kimi acp`
2. Connect IDE to the ACP server.
3. In the IDE's AI chat, send: `/flow:ultrawork "add a function double(x) + test and verify"`
4. Observe the flow execution.

**Expected:**
- Same behavior as terminal mode: flow runs, hook fires, Oracle dispatches, VERIFIED ends loop.
- Stop hook works (ACP uses the same `events.py:stop` path).
- No UI-specific differences.

**Evidence to capture:**
- IDE chat transcript showing flow execution.
- Confirmation that Oracle dispatched and returned verdict.

**Note:** If IDE does not support slash commands, use the prompt-based fallback: "Use ultrawork mode to add a function double(x) + test and verify."
