# Ultrawork — Kimi Native Path

Runtime-enforced verify-then-exit loop for Kimi using Stop hooks, flow skills, and isolated YAML subagents.

## Prerequisites

- kimi-cli 1.43.0+ (Stop hook, subagent YAML spec, flow skills)
- gpowers ultrawork base skill (auto-templated with Kimi adapter)

## Install

```bash
cd core/skills/ultrawork/platforms/kimi
./install.sh          # project-scoped → ./.kimi/
./install.sh --user   # user-scoped → ~/.kimi/
```

**Idempotent.** Re-run safely after upstream changes.

**Requires restart:** Kimi must be restarted after `config.toml` changes.

## Uninstall

Remove the idempotent blocks manually:

1. Delete hook block from `.kimi/config.toml` (between `# >>> gpowers:ultrawork >>>` and `# <<< gpowers:ultrawork <<<`).
2. Delete `oracle:` entry from `.kimi/agent.yaml` under `subagents:`.
3. Delete `.kimi/hooks/ultrawork-stop.sh`.
4. Delete `.kimi/agents/oracle/`.
5. Delete `.kimi/skills/ultrawork/`.

Session state files (`.ultrawork-iter`, `.ultrawork-flow-active`, `.ultrawork-hook.log`) are not cleaned up from old session directories.

## Usage

```
/flow:ultrawork "fix the failing auth test and verify"
```

The flow runner drives iterations. The Stop hook blocks bare `<promise>DONE</promise>` unless Oracle `VERIFIED` is present.

## Architecture

```
User → /flow:ultrawork
         │
         ▼
    [Flow runner] ←── Mermaid flowchart in flow.md
         │
         ├── loads protocol + verification-before-completion
         ├── touches .ultrawork-flow-active (arms Stop hook)
         ├── Worker iteration (edit → verify → emit DONE)
         ├── Stop hook blocks unverified DONE → injects reason
         ├── dispatches Oracle via Agent(subagent_type="oracle")
         ├── reads Oracle verdict
         └── VERIFIED → END / NOT-VERIFIED → loop
```

## Configuration Samples

### config.toml hook entry

```toml
# >>> gpowers:ultrawork >>>
[hooks.stop]
command = "/path/to/.kimi/hooks/ultrawork-stop.sh"
# <<< gpowers:ultrawork <<<
```

### agent.yaml subagent entry

```yaml
subagents:
  oracle:
    spec: ./agents/oracle/oracle.yaml
```

### wire.jsonl excerpt (what the hook parses)

```json
{"event":"TurnBegin","turn":7,"role":"assistant"}
{"event":"MessageDelta","content":"..."}
{"event":"MessageDelta","content":"<promise>DONE</promise>\n"}
{"event":"TurnEnd","turn":7}
```

## Known Limitations

1. **Hook fail-open.** Kimi's hook runner allows on crash/timeout. Mitigation: flow runner second line + heartbeat log.
2. **Hook dormancy outside flows.** `.ultrawork-flow-active` absent → hook exits 0. Out-of-flow `<promise>DONE</promise>` is not blocked.
3. **config.toml requires restart.** Kimi does not hot-reload hook config.
4. **extend: path absolutization.** Moving the project requires re-running the installer.
5. **Session state not cleaned on uninstall.** Old `.ultrawork-*` files remain in session dirs.

## Assurance Gap vs. oh-my-opencode

| Feature | oh-my-opencode | gpowers Ultrawork on Kimi |
|---|---|---|
| Hook type | Plugin (TypeScript, aborts on crash) | Shell (fail-open) |
| Subagent isolation | Full context reset | YAML spec + SubagentStore |
| Loop driver | Plugin-controlled | Soul flow runner |
| Verdict parsing | Plugin regex | Hook regex + flow runner |

The Kimi path matches oh-my-opencode's runtime primitives but with fail-open semantics. The flow runner provides a second line of defense.

## Testing

Run the manual exercise scripts in `tests/`:

| Scenario | What it tests |
|---|---|
| K-A | Happy path |
| K-B | Premature DONE blocked by hook |
| K-C | Hook fail-open documented gap |
| K-D | Oracle isolation (own Context, wire log) |
| K-E | Install/uninstall idempotency |
| K-F | ACP mode compatibility |
