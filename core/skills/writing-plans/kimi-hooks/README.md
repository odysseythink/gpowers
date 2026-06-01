# writing-plans — kimi-code enforcement hooks

The `writing-plans` skill asks the model to generate a large (split) plan **one part
per turn**, so each part is written in a clean context (you `/compact` between parts).
That rule lives in the skill prompt — which `/compact` erases. So a bare `continue`
after a compact can make the model batch-generate every remaining part in one
degrading session.

These two hooks enforce the rule at the **harness level** instead of the prompt level,
using kimi-code's built-in hook system. No source modification — pure config.

## What they do

- **`plan-part-reset.mjs`** (`UserPromptSubmit`): clears the per-turn counter on every
  user message, so each `continue` gets a fresh one-part budget.
- **`plan-part-guard.mjs`** (`PreToolUse`, matcher `Write|Edit`): allows the **first**
  plan sub-plan file written in a turn, then **blocks the second**. The model can't
  make progress on another part, so it ends the turn and hands off.

Net effect: even a bare `continue` (or YOLO mode) yields **exactly one part per turn**.
You still `/compact` between parts for a clean context, but forgetting to no longer
cascades into "it wrote everything."

Only markdown files under a `plans/` directory are considered; `*-index.md` (the
manifest) is always allowed; every other Write/Edit is untouched (fail-open).

## Install

Add to your kimi-code config (TOML). Use absolute paths:

```toml
[[hooks]]
event   = "UserPromptSubmit"
command = "node /Users/ranwei/workspace/go_work/gpowers/core/skills/writing-plans/kimi-hooks/plan-part-reset.mjs"

[[hooks]]
event   = "PreToolUse"
matcher = "Write|Edit"
command = "node /Users/ranwei/workspace/go_work/gpowers/core/skills/writing-plans/kimi-hooks/plan-part-guard.mjs"
timeout = 10
```

## Verify (before trusting it)

Simulate two distinct part writes in one turn — the second must exit 2:

```sh
H=/Users/ranwei/workspace/go_work/gpowers/core/skills/writing-plans/kimi-hooks
# reset the turn
echo '{"session_id":"t1"}' | node "$H/plan-part-reset.mjs"; echo "reset exit=$?"
# first part → allowed (0)
echo '{"session_id":"t1","cwd":"/x","tool_name":"Write","tool_input":{"path":"/x/plans/feat-core.md"}}' | node "$H/plan-part-guard.mjs"; echo "part1 exit=$?"
# index update → allowed (0)
echo '{"session_id":"t1","cwd":"/x","tool_name":"Edit","tool_input":{"path":"/x/plans/feat-index.md"}}' | node "$H/plan-part-guard.mjs"; echo "index exit=$?"
# second DISTINCT part → BLOCKED (2)
echo '{"session_id":"t1","cwd":"/x","tool_name":"Write","tool_input":{"path":"/x/plans/feat-chat.md"}}' | node "$H/plan-part-guard.mjs"; echo "part2 exit=$? (expect 2)"
# a normal code edit → always allowed (0)
echo '{"session_id":"t1","cwd":"/x","tool_name":"Edit","tool_input":{"path":"/x/main.go"}}' | node "$H/plan-part-guard.mjs"; echo "code exit=$?"
```

Expected: reset=0, part1=0, index=0, **part2=2**, code=0.

## Contract (kimi-code 0.6.0, for maintainers)

- Hook command is spawned with `shell:true`; the tool-call payload arrives as **JSON on
  stdin** with **snake_case** keys: `{ hook_event_name, session_id, cwd, tool_name,
  tool_input, tool_call_id }` (`session/hooks/engine.ts` → `toHookInputData` lowercases).
- `tool_input.path` is the file path for both `Write` and `Edit`
  (`tools/builtin/file/{write,edit}.ts` schema field is `path`).
- A hook **blocks** by exiting **2** with the reason on **stderr**
  (`session/hooks/runner.ts` → `resultFromExitCode`). `PreToolUse` block → tool denied,
  reason fed back to the model (`permission/policies/pre-tool-call-hook.ts`).
- `matcher` is a **RegExp** tested against the tool name (`engine.ts` → `matches`), so
  `"Write|Edit"` is valid.

## Limits (honest)

- The guard doesn't force `end_turn`; it denies the second part write. The model may try
  a tool or two before giving up and ending the turn. Acceptable.
- It does NOT `/compact` for you — clean context still requires your manual `/compact`.
  The hook only guarantees the one-part-per-turn cap, regardless of compaction.
