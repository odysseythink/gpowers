# Discipline-Agent Personas — Design (Revised)

**Date:** 2026-05-21  
**Owner:** ranwei693532  
**Status:** Revised — scope reduced after platform-capability audit  
**Source of inspiration:** oh-my-opencode v3.17.10 (`src/agents/{sisyphus,hephaestus,prometheus,oracle,librarian,explore}*`)  
**Roadmap item:** #2 in `2026-05-21-oh-my-opencode-port-roadmap.md`

---

## Architecture Cosplay — What We Learned

The original design assumed gpowers could register *named executor identities* (Sisyphus, Hephaestus, Prometheus, Oracle, Librarian, Explore) as first-class subagent types across all platforms, with hard tool restrictions and user-triggerable persona switching. After auditing the source code of Kimi CLI v1.44.0, Claude Code, Codex, and other hosts, this assumption collapses:

**No mainstream CLI host (except OpenCode, which is open-source) exposes a user-facing API to register custom subagent types.**

| Platform | Custom subagent types? | Tool hard-limit? | User persona trigger? | **Install persona adapter?** |
|---|---|---|---|---|
| **OpenCode** | ✅ Yes (native) | ✅ Yes | `@persona` | **Yes** |
| **Kimi** | ⚠️ Via `--agent-file` only | ✅ `allowed_tools`/`exclude_tools` | ❌ No | **Yes** |
| **Claude Code** | ❌ No (built-in only) | ✅ `tools` field | ❌ No | **No** |
| **Codex** | ❌ No (built-in only) | Unknown | `/plan` only | **No** |
| **Cursor** | ❌ No | ❌ No | ❌ No | **No** |
| **Copilot** | ❌ No | ❌ No | ❌ No | **No** |
| **Gemini** | ❌ No | ❌ No | ❌ No | **No** |
| **Qoder** | ❌ No | ❌ No | ❌ No | **No** |

**Implication:** A 6-persona runtime system cannot be ported faithfully. The design was "architecture cosplay" — mimicking oh-my-opencode's form without its runtime substrate.

**Decision:** Persona adapters are installed **only on Kimi and OpenCode**. All other platforms receive Oracle as a standard skill only (no adapter, no subagent type registration, no persona trigger). This cuts implementation from "6 skills × 8 platforms" to "1 skill + 2 platform adapters."

---

## What Still Has Value

### 1. Oracle — single source of truth for Ultrawork verification

`core/skills/ultrawork/oracle.md` is currently a duplicated, diverging copy of oh-my-opencode's Oracle prompt. Collapsing it to a pointer at `roles/skills/oracle/SKILL.md`:

- Eliminates duplicate maintenance.
- Gives Ultrawork a canonical prompt that evolves in one place.
- The Oracle body dual-modes (standalone advisor vs `<promise>` verifier) — this is genuinely useful behavior, not just identity dressing.

### 2. Methodology composition guide (document, not runtime)

Sisyphus's "load brainstorming → writing-plans → dispatching-parallel-agents → verification" sequence, Hephaestus's 3-strikes recovery, Prometheus's interview-then-plan discipline — these are *workflow patterns*, not runtime identities. A single markdown guide teaching users (and the model) when to chain which skills is more honest and more portable than pretending each is a callable persona.

### 3. Kimi `--agent-file` adapter

Kimi CLI supports custom `agent.yaml` via `--agent-file`. gpowers ships a pre-built `agent.yaml` that registers Oracle as a named subagent type alongside built-in `coder`/`explore`/`plan`. Requires **zero Python code changes** to Kimi — only YAML.

### 4. OpenCode native adapter

OpenCode already runs oh-my-opencode's persona runtime. The gpowers adapter is a thin pointer: `platforms/opencode/adapters/gpowers-oracle/` references `roles/skills/oracle/SKILL.md` as the canonical prompt source. No prompt duplication.

---

## Revised Scope

### Ship

```
roles/skills/
└── oracle/SKILL.md              # standalone advisor + Ultrawork verifier
    ├── ACKNOWLEDGEMENTS.md
    └── tests/scenarios.md       # P-A, P-B, P-L only

docs/methodology/
└── executor-patterns.md         # "When to load which skill" guide
    # Covers: intent routing (ex-Sisyphus), deep-work discipline (ex-Hephaestus),
    #         interview-first planning (ex-Prometheus), external research (ex-Librarian),
    #         codebase exploration (ex-Explore)

core/skills/ultrawork/oracle.md  → 3-line pointer (see below)
```

### Do NOT ship

- `sisyphus/`, `hephaestus/`, `prometheus/` skills — no runtime trigger mechanism.
- `librarian/`, `explore/` standalone skills — Kimi already has `explore`; Librarian is a prompt pattern, not a type.
- 42 PLATFORM-NOTES.md files — no platform collisions to document when personas don't exist.
- `persona-mode:` frontmatter field — Oracle is a normal skill.

### Platform-specific adapters (Kimi + OpenCode only)

Other platforms do **not** install persona adapters — Oracle exists only as a standard skill in the system prompt.

**Kimi:**
```
platforms/kimi/
└── agent.yaml                     # extend: default + oracle subagent
    └── oracle.yaml                # allowed_tools / exclude_tools / system_prompt_args
```
Users opt in via `kimi --agent-file ./platforms/kimi/agent.yaml`. The `_gpowers-gen-kimi.sh` transform writes this file and updates `kimi-skills.json`.

**OpenCode:**
```
platforms/opencode/
└── adapters/gpowers-oracle/
    └── agent.yaml                 # OpenCode-native agent spec referencing roles/skills/oracle/SKILL.md
```
OpenCode is the only host where oh-my-opencode's persona runtime already exists. The adapter is a thin pointer; no duplication of prompt content.

---

## Oracle Skill Spec

### Frontmatter

```yaml
---
name: oracle
description: Read-only strategic advisor. Dispatched as a subagent for complex reasoning, architecture decisions, or independent verification. (Oracle — adapted from oh-my-opencode)
namespace: roles
upstream: oh-my-opencode@v3.17.10
---
```

### Body (~120–180 lines)

1. **Identity** — "You are Oracle, a read-only strategic technical advisor. You do not write code, edit files, or dispatch subagents."
2. **Three-tier response** — Bottom line / Action plan / Effort / Confidence tag.
3. **Scope discipline** — "If asked to implement, decline and redirect."
4. **Tool Discipline** — `forbidden-tools: Edit, Write, Agent`. (Hard-enforced on Kimi/Claude Code/OpenCode via host mechanisms; advisory on others.)
5. **Mode Detection** — If caller's prompt contains `<promise>` on its own line, switch to Ultrawork verification mode: apply promise contract, cite specific evidence, emit exactly one of `<promise>VERIFIED</promise>` or `<promise>NOT-VERIFIED: <reason></promise>`.
6. **Examples** — One correct invocation, one counter-example (Oracle attempting `Edit`).

### Ultrawork pointer

`core/skills/ultrawork/oracle.md` becomes:

```markdown
# Oracle subagent for Ultrawork

The Oracle persona is maintained at `roles/skills/oracle/SKILL.md`.

When dispatching for Ultrawork verification, pass that file's body as the
subagent's system prompt, plus this Ultrawork-specific preamble:

> You are verifying a Worker's <promise>DONE</promise> claim. Apply the
> promise contract from ../protocol.md. Evidence requirements: cite specific
> file paths, test names, command output before emitting your verdict tag.
> Emit exactly one of:
>   Agent: Oracle
>   <promise>VERIFIED</promise>
> or
>   Agent: Oracle
>   <promise>NOT-VERIFIED: <reason></promise>
```

---

## Platform Adapter Reality

**Only Kimi and OpenCode install persona adapters.** All other platforms receive Oracle as a normal skill only.

### Kimi

Kimi CLI v1.44.0 supports `--agent-file` for custom `agent.yaml`. The `subagents` block registers new types into `LaborMarket` at load time.

```yaml
# platforms/kimi/agent.yaml
version: 1
agent:
  extend: default
  subagents:
    coder:
      path: /path/to/kimi/agents/default/coder.yaml
      description: "..."
    explore:
      path: /path/to/kimi/agents/default/explore.yaml
      description: "..."
    plan:
      path: /path/to/kimi/agents/default/plan.yaml
      description: "..."
    oracle:
      path: ./oracle.yaml
      description: "Read-only strategic advisor."
```

Caveat: `subagents` inheritance is **overwrite**, not merge. A custom `agent.yaml` must re-declare all built-in types (`coder`, `explore`, `plan`) to keep them available.

Tool restrictions are hard-enforced via `allowed_tools`/`exclude_tools` in each subagent's YAML — Kimi's `ToolPolicy(mode="allowlist")` blocks forbidden tools at runtime.

### OpenCode

OpenCode ships oh-my-opencode as a native plugin; its persona runtime (Sisyphus, Hephaestus, Oracle, etc.) already exists. The gpowers adapter for Oracle is a thin pointer:

- `platforms/opencode/adapters/gpowers-oracle/agent.yaml` references `roles/skills/oracle/SKILL.md` as the canonical prompt source.
- No prompt duplication; Oracle updates in `roles/skills/oracle/` propagate to OpenCode automatically.
- OpenCode's native runtime handles tool restrictions and persona switching; gpowers does not re-implement.

### All other platforms (Claude Code, Codex, Cursor, Copilot, Gemini, Qoder)

**No persona adapters installed.** Oracle is delivered as a standard skill (`roles/skills/oracle/SKILL.md`) loaded into the system prompt like any other skill. Subagent dispatch uses the host's built-in types with prompt override where available.

---

## Testing (Revised)

### Automated

- `tests/unit/tools/oracle.bats` — frontmatter valid, body contains Mode Detection, AI-slop deny-list respected, examples present.
- `tests/unit/tools/oracle.bats` — Ultrawork pointer file resolves to `roles/skills/oracle/SKILL.md`.

### Manual (3 scenarios, not 12)

| ID | Scenario | Expect |
|---|---|---|
| P-A | Standalone Oracle consult | Three-tier response. Confidence tag. No `Edit`/`Write` calls. |
| P-B | Ultrawork → Oracle verification | Detects `<promise>` mode, applies contract, emits verified/not-verified with evidence. |
| P-L | Ultrawork integration regression | After Oracle extraction, existing K-A..K-F and Scenarios A/B/C still pass. |

### Kimi + OpenCode smoke

- **Kimi:** `kimi --agent-file ./platforms/kimi/agent.yaml` → dispatch `oracle` subagent → verify `exclude_tools` blocks `WriteFile`.
- **OpenCode:** Verify `gpowers-oracle` adapter loads without error and references the correct `SKILL.md` path.

---

## Open Questions / Risks (Revised)

- **Kimi `--agent-file` discoverability.** Users must know to add the flag. A shell alias or wrapper script (`kimi-gpowers`) is documented but not enforced.
- **Oracle prompt drift from upstream.** `roles/upstream-source.json` tracks pinned version; manual refresh per maintenance window.
- **Methodology guide effectiveness.** `docs/methodology/executor-patterns.md` is advisory. Does the model actually follow the routing guidance? No runtime enforcement — same gap as all other platforms.

---

## Out of Scope (Confirmed)

- Sisyphus/Hephaestus/Prometheus as runtime personas — no host support.
- Librarian/Explore as standalone skills — Kimi already has `explore`; Librarian is a prompt pattern.
- Per-platform persona adapters beyond Kimi (`agent.yaml`) and OpenCode (`gpowers-oracle/` pointer).
- Automated persona-prompt regression tests against a real model.
- Category-based model routing — still roadmap #7, unaffected by this revision.
