# Discipline-Agent Personas — Design

**Date:** 2026-05-21
**Owner:** ranwei693532
**Status:** Approved for plan
**Source of inspiration:** oh-my-opencode v3.17.10 (`src/agents/{sisyphus,hephaestus,prometheus,oracle,librarian,explore}*`)
**Roadmap item:** #2 in `2026-05-21-oh-my-opencode-port-roadmap.md`

## Problem

gpowers' `roles/` namespace contains only *reviewer* roles (pr-review, cso, devex-review, plan-ceo-review, etc.). The methodology library has skills like `brainstorming`, `writing-plans`, `executing-plans`, `verification-before-completion` — but no *named executor identities* that bind those methodologies together with a voice and a delegation contract.

oh-my-opencode ships six discipline-agent personas — three primaries (orchestrator/worker/planner) and three subagents (advisor/researcher/grep) — that act as the *named executors* of the methodology. Ultrawork's Oracle is one of these, currently duplicated inside `core/skills/ultrawork/oracle.md`.

We want the same persona layer in gpowers, without owning a runtime, without duplicating methodology gpowers already ships, and without per-model prompt forks.

## Goal

Ship six discipline-agent persona skills under `roles/skills/` that:

1. Compose existing gpowers methodology skills (for primaries) rather than re-implementing them.
2. Carry a single canonical, model-agnostic prompt per persona.
3. Declare tool restrictions as a contract that platform adapters enforce where supported and that the persona body honors as a binding rule where not.
4. Let `core/skills/ultrawork/oracle.md` collapse to a pointer at the Oracle role skill (closes the Ultrawork loose end flagged in the roadmap).
5. Coexist with host built-ins (e.g., Codex `/plan`, oh-my-opencode-native OpenCode) — never shadow.

## Non-goals (v1)

- Per-model prompt variants (Claude/GPT/Gemini/Kimi forks). Per-model tuning is the host runtime's job.
- Automated persona-prompt regression tests against a real model. Belongs to a future `benchmark-personas` workstream.
- Auto-sync of upstream personas from oh-my-opencode releases. Manual refresh per maintenance window.
- Replacing existing gpowers methodology skills. Primaries *load* `brainstorming`/`writing-plans`/etc., they don't supersede them.
- Implementing the personas. This doc designs the spec; `writing-plans` produces the implementation plan.

## Architecture

### Layout

```
roles/skills/
├── sisyphus/SKILL.md          # primary — orchestrator
├── hephaestus/SKILL.md        # primary — deep worker
├── prometheus/SKILL.md        # primary — planner
├── oracle/SKILL.md            # subagent — read-only advisor
├── librarian/SKILL.md         # subagent — external-doc researcher
└── explore/SKILL.md           # subagent — internal codebase grep

# Plus one rewrite:
core/skills/ultrawork/oracle.md                          → 3-line pointer
core/skills/ultrawork/platforms/kimi/oracle.yaml         → references role skill body
```

### Two persona shapes

| Shape | Personas | User-invocable? | Skill loading inside |
|---|---|---|---|
| `primary` | Sisyphus, Hephaestus, Prometheus | Yes — via `triggers:` like `@sisyphus` | Yes — loads other gpowers skills |
| `subagent` | Oracle, Librarian, Explore | No — dispatched only via host's Task/Agent | No — clean slate, self-contained prompt |

### Composition principle (primaries are thin shells)

Primary personas DO NOT duplicate `brainstorming` / `writing-plans` / `executing-plans` / `dispatching-parallel-agents` / `verification-before-completion` / `ultrawork`. They are:
- A voice + identity layer
- A trigger declaration
- A delegation map that loads the right gpowers methodology skills at the right phase
- A small persona-specific increment (Sisyphus intent gate, Hephaestus 3-strikes recovery, Prometheus interview-then-plan-only constraint)

This keeps methodology and identity in their respective layers, and lets either layer evolve without forcing changes in the other.

### Tool restrictions: contract, not runtime

Each SKILL ships an explicit `## Tool Discipline` block. Per-platform adapters enforce hard restrictions where the platform supports them:

| Platform | Enforcement | Mechanism |
|---|---|---|
| Claude Code | Hard | subagent `tools:` field |
| Kimi | Hard | agent.yaml `allowed_tools` / `exclude_tools` |
| OpenCode | Hard | agent `permission:` map |
| Codex | Hard where supported | (vendor-specific) |
| Gemini / Cursor / Copilot / Qoder | Advisory | persona body says "you must not use Edit/Write/Agent" — relies on prompt compliance |

The canonical SKILL.md body is identical across platforms; only the wrapping adapter changes.

## Components

### Frontmatter convention

New field `persona-mode: primary | subagent`. Existing fields reused.

```yaml
---
name: oracle
description: Read-only strategic advisor. Dispatched as a subagent for complex reasoning, architecture decisions, or independent verification. (Oracle — adapted from oh-my-opencode)
namespace: roles
persona-mode: subagent
upstream: oh-my-opencode@v3.17.10
triggers:                    # primary only — subagents have empty triggers
  - "@oracle"
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
forbidden-tools:
  - Edit
  - Write
  - Agent
---
```

### Persona contracts

**Primaries** (~120–180 lines of SKILL body each):

| Persona | Identity / voice | Loads (existing gpowers skills) | Persona-specific increment |
|---|---|---|---|
| **Sisyphus** | "Powerful orchestrator. Delegate, verify, ship. No AI slop." | `brainstorming`, `dispatching-parallel-agents`, `verification-before-completion`, `executing-plans` | Intent gate: verbalize surface → intent → routing in one sentence before acting. Delegation table mapping intent → which gpowers skill to load. |
| **Hephaestus** | "Deep worker. Single-task focus, no shortcuts." | `executing-plans`, `verification-before-completion`, optional `ultrawork` for high-stakes work | Failure recovery: after 3 consecutive failed fix attempts → revert → consult Oracle → ask user before proceeding. |
| **Prometheus** | "Planner. Interview-first; write a plan only when the interview is complete." | `brainstorming`, `writing-plans` | Interview mode block (1–3 clarifying questions, then write). Hard rule: refuses implementation requests, redirects to `@hephaestus`. |

**Subagents** (~100–200 lines of SKILL body each):

| Persona | Canonical prompt content |
|---|---|
| **Oracle** | Strategic technical advisor. Three-tier response (Bottom line / Action plan / Effort / Confidence). Scope discipline. No write tools. Adapted from `oracle.ts:ORACLE_DEFAULT_PROMPT`, de-OpenCode-ified. Includes `## Mode Detection` block: if caller's prompt contains `<promise>` tags, applies Ultrawork verification rules; otherwise general advisor. |
| **Librarian** | External library / open-source researcher. Type A/B/C/D classification. Permalink citation requirement. Tool-agnostic prose ("search the codebase", not "use grep_app"). Adapted from `librarian.ts` prompt with vendor tool-names abstracted. Returns `OUT-OF-SCOPE: try @explore instead` for internal-code queries. |
| **Explore** | Internal codebase grep. Find files/code, return actionable results with paths + pattern descriptions. Thoroughness levels (quick / medium / very thorough). Adapted from `explore.ts` prompt verbatim. |

### Ultrawork rewrite

`core/skills/ultrawork/oracle.md` becomes:

```markdown
# Oracle subagent for Ultrawork

The Oracle persona is now maintained at `roles/skills/oracle/SKILL.md`.

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

Kimi addendum: `core/skills/ultrawork/platforms/kimi/oracle.yaml` updates `system_prompt_path: ./oracle.md` → `system_prompt_path: ../../../../roles/skills/oracle/SKILL.md`.

### Acknowledgements

Each persona ships `roles/skills/<persona>/ACKNOWLEDGEMENTS.md` crediting oh-my-opencode v3.17.10 (matches the existing `cso` skill's pattern). `roles/upstream-source.json` gets a `personas:` entry tracking the pinned version. Manual refresh during gpowers maintenance windows — not auto-synced.

## Data flow

### Primary persona invocation (user-facing)

```
User: /sisyphus "add JWT auth to the REST API"
   │
   ▼
[Host loads roles/skills/sisyphus/SKILL.md into system prompt]
   │
   ▼
[Sisyphus — intent gate]
  1. Verbalize routing: "Detect implementation intent — explicit verb 'add' +
     concrete target. Approach: brainstorming → writing-plans → delegate to Hephaestus."
  2. Read gpowers skills it composes (brainstorming, dispatching-parallel-agents, ...)
  3. Hand the task off through gpowers' methodology — NOT by reimplementing it.
   │
   ▼
[Delegation via dispatching-parallel-agents]
  Dispatches sub-tasks via host's Task/Agent tool.
  Subagent prompts CAN reference other personas: subagent_type=oracle|hephaestus|librarian|explore.
   │
   ▼
[Verification gate via verification-before-completion]
  For high-stakes work: suggests /ultrawork (which dispatches Oracle subagent).
```

Same shape for `/hephaestus` (deep-work) and `/prometheus` (plan-only).

### Subagent dispatch (machine-to-machine)

```
[Any caller — Sisyphus, user's plain session, Ultrawork loop, etc.]
   │
   ▼
[Dispatch via host's Task/Agent tool]
   subagent_type = "oracle" | "librarian" | "explore"
   system_prompt = body of roles/skills/<persona>/SKILL.md
   tools         = filtered per allowed-tools + forbidden-tools
   load_skills   = []   # subagents start CLEAN
   prompt        = caller-supplied task
   │
   ▼
[Subagent runs in isolated context]
   Reads/searches per its persona.
   Cannot dispatch nested subagents (Agent in forbidden-tools).
   Cannot write files (Edit/Write in forbidden-tools).
   │
   ▼
[Returns one self-contained response to caller]
```

### Composition with Ultrawork

```
User: /ultrawork "fix the failing auth test"
   ├── Ultrawork dispatches Worker (current host primary persona)
   ├── Worker emits <promise>DONE</promise>
   └── Ultrawork dispatches Oracle subagent
          system_prompt = roles/skills/oracle/SKILL.md   (single source of truth)
                          + Ultrawork-specific preamble (promise contract, evidence rules)
          → Oracle returns VERIFIED or NOT-VERIFIED
```

### State boundaries

- **Persona ≠ session.** Switching from `/sisyphus` to `/hephaestus` re-loads the system prompt; no state crosses unless the user passes it explicitly.
- **Subagent dispatches are stateless by default.** `task_id` continuation (where the host supports it — Claude Code, Kimi, OpenCode) is recommended for multi-turn Oracle/Librarian consults; called out in the SKILL bodies.
- **No on-disk persona state.** Nothing persists to the filesystem from a persona invocation.

## Error handling & edge cases

1. **Persona name collides with host built-in.** Hosts may ship `/plan` or `/explore`. Personas use unambiguous triggers like `@sisyphus` / `@oracle` and never plain words. Per-platform notes captured in `roles/skills/<persona>/PLATFORM-NOTES.md`.

2. **Subagent dispatched on host without Task/Agent tool.** Gemini and Cursor have limited subagent support. Dispatching skill (Ultrawork, `dispatching-parallel-agents`) detects host capability; if no subagent tool, falls back to *in-session impersonation* — load the persona's SKILL.md into the current context, complete the work, restore prior context. Documented as lower-assurance (same-context Oracle is biased — Ultrawork already flags this).

3. **Tool restrictions not enforceable on host X.** SKILL body's `## Tool Discipline` block is a *contract*, not a runtime guard. If a persona uses a forbidden tool on Gemini/Cursor, the dispatching primary catches it on response inspection and re-dispatches with a corrective preamble. Documented in `dispatching-parallel-agents`, not in the persona skill.

4. **Nested subagent dispatch attempt.** Adapters with hard restrictions exclude `Agent` from subagent tools. On hosts without enforcement, the persona body explicitly says "you cannot dispatch further subagents — return what you have." Persona invariant: subagents never call Task/Agent.

5. **Primary persona loaded twice in one session.** User runs `/sisyphus`, then `/hephaestus` later. The two prompts compete. Each primary opens with: "If a different persona is already active in this session, end and report the conflict — do not silently merge personas." User restarts the session to switch.

6. **Persona invokes a gpowers skill that isn't installed.** Sisyphus loads `dispatching-parallel-agents` but install was partial. Graceful degradation — persona checks for the file via `Read`, if missing logs `[persona] skill <name> not found, proceeding without` and continues. No hard fail.

7. **Oracle dispatched without Ultrawork's evidence preamble.** Standalone Oracle calls don't get the promise contract. Oracle's SKILL body has a `## Mode Detection` block — if caller's prompt contains `<promise>` tags, applies Ultrawork verification rules; otherwise operates as a general advisor. Single body handles both.

8. **Librarian asked about internal code.** Librarian is for external libraries / open-source repos, not the current project. SKILL body explicitly says "if the question is about the current project, return `OUT-OF-SCOPE: try @explore instead` and stop." Caller (Sisyphus or user) re-dispatches to Explore.

9. **Explore overlap with `dispatching-parallel-agents`.** That skill teaches the *caller* how to fan out explore calls. New Explore persona is the *callee*. `dispatching-parallel-agents` gets a one-line update referencing the persona by name; Explore body says nothing about parallelism (caller's job).

10. **Primary persona overlaps with host-runtime persona.** User on Sisyphus-vendored OpenCode (oh-my-opencode is already running). SKILL body detects via `Bash` checking for `$OMO_VERSION` or equivalent env var; if detected, primary persona logs `[gpowers] host runtime already provides this persona; gpowers skill stays dormant` and returns control. Prevents double-loading.

11. **Prometheus asked to implement after delivering a plan.** Prometheus identity is plan-only. Hard rule: "if user asks to implement after plan delivered, respond `Plan delivered. Switch to @hephaestus or your default mode to implement.` Do not implement from Prometheus mode."

12. **Acknowledgements drift.** oh-my-opencode evolves upstream. Each skill's `ACKNOWLEDGEMENTS.md` pins the version; `roles/upstream-source.json` gets a `personas:` entry tracking it. Manual refresh, not auto-synced.

## Testing

Skill ships content, not code. Validation = spec-checks + manual exercises.

### Spec-embedded examples

Each SKILL.md ships a `## Examples` block with a correct invocation snippet (frontmatter, call signature) and a counter-example (Oracle attempting `Edit`).

### Manual exercise scenarios

`roles/skills/_personas-tests/` (shared directory, since scenarios reference each other):

| ID | Scenario | Personas | Expect |
|---|---|---|---|
| P-A | Standalone Oracle consult | Oracle | Three-tier response (Bottom line / Action plan / Effort). Confidence tag present. No `Edit`/`Write` calls. |
| P-B | Ultrawork → Oracle re-verification | Oracle (via Ultrawork) | Oracle detects `<promise>` mode, applies promise contract, emits `Agent: Oracle\n<promise>VERIFIED</promise>` with cited evidence. |
| P-C | Librarian on external lib | Librarian | Permalink citation present. Tool-name-agnostic prose. |
| P-D | Librarian asked internal | Librarian | Returns `OUT-OF-SCOPE: try @explore instead`. |
| P-E | Explore on this repo | Explore | Returns file paths + pattern descriptions. No general advice. |
| P-F | Sisyphus orchestrates feature | Sisyphus → Hephaestus → Oracle | Sisyphus verbalizes routing, dispatches Hephaestus for build, dispatches Oracle for review. Each handoff visible in transcript. |
| P-G | Hephaestus 3-strikes recovery | Hephaestus → Oracle | After 3 failed fix attempts: revert → consult Oracle → ask user before proceeding. |
| P-H | Prometheus interview → plan | Prometheus | 1–3 clarifying questions, writes plan, refuses subsequent implementation request. |
| P-I | Persona collision | Sisyphus + Hephaestus same session | Second persona detects active persona, reports conflict, no merge. |
| P-J | Restriction enforcement (Kimi) | Oracle dispatched as Kimi subagent | Kimi's `exclude_tools` blocks `WriteFile`; rejection visible in wire log. |
| P-K | Restriction advisory (Cursor) | Oracle on Cursor | No hard block; persona body's contract holds via prompt compliance. Documented as lower assurance. |
| P-L | Ultrawork integration regression | Oracle | After role-skill extraction, Ultrawork's existing K-A..K-F and Scenarios A/B/C still pass with Oracle prompt resolved from `roles/skills/oracle/SKILL.md`. |

### Cross-platform smoke check

Run P-A on at least three platforms covering the enforcement spectrum:
- Claude Code (hard restriction)
- Kimi (hard restriction via YAML)
- Cursor or Gemini (advisory only)

Each should complete the consult successfully — restrictions vary, persona content does not.

### Install regression

- `./install` picks up the new tree automatically (no new install code).
- Platforms with hard subagent restrictions (Claude Code, Kimi, OpenCode): install transform writes the platform-native agent/subagent declaration file referencing the role skill body.
- `manifest.json` needs no update (existing `roles/` module already covers it).

### Not in v1

- Automated persona-prompt regression tests driving a real model end-to-end (no harness today).
- Per-model prompt-quality benchmarks. Future `benchmark-personas` workstream.
- Auto-sync of upstream personas from oh-my-opencode releases.

## Open questions / risks

- **Re-anchored thresholds vs source.** Source Sisyphus prompt is ~480 lines, Hephaestus ~400. Targeting 120–180-line shells is a strong compression bet. May discover during impl that some content can't be compressed without loss; if so, doc-comment grow back toward source size.
- **Advisory-only restrictions on Gemini/Cursor.** Personas trust the model to honor `forbidden-tools` via prompt. If a model misbehaves, the dispatching primary catches it on response inspection (edge case 3). Open: how aggressively to inspect — full diff-check, or just verb-scan?
- **Oracle dual-mode detection.** `## Mode Detection` block uses presence of `<promise>` tag to switch. If a non-Ultrawork caller happens to pass `<promise>` in unrelated content, Oracle applies promise contract by mistake. Mitigation: require the tag be on its own line, matching the Ultrawork regex. Same mitigation already used in Ultrawork.
- **Drift between role-skill Oracle and Ultrawork's preamble.** As Ultrawork evolves, the preamble it injects when dispatching Oracle may want to override Oracle defaults (e.g., different verdict tags). Open: do we add a `## Caller Override Hooks` section to Oracle, or keep the preamble append-only? v1 picks append-only.
- **Per-platform PLATFORM-NOTES.md proliferation.** 6 personas × 7 platforms = up to 42 notes files. Most will be empty. Open: lazy-create only when a collision/limitation exists, or template all of them? v1 picks lazy-create.

## Out of scope — escalated to roadmap

- **Category-based model routing** (roadmap #7) — Sisyphus delegation table currently maps intent → skill. Source maps intent → *category* → *model*. Defer category layer to roadmap #7.
- **Skill-Embedded MCPs** (roadmap #6) — Librarian wants context7/grep_app MCPs. Today, the canonical body uses tool-agnostic prose. When #6 lands, Librarian gets an `mcp_servers:` frontmatter entry.
- **Quantified parallel-agent dispatch** (roadmap #9) — Sisyphus delegation thresholds (fan-out by file count / depth / language count) come from there.
- **AI-slop detector** (roadmap #5) — Sisyphus identity line says "no AI slop"; enforcement is the detector's job.
