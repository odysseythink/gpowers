# Oh-My-OpenCode → gpowers Port Roadmap

**Date:** 2026-05-21
**Source:** oh-my-opencode v3.17.10 (`~/Downloads/oh-my-openagent-3.17.10/`)
**Status:** Survey + sequencing. Each item gets its own spec when picked up.

## Context

oh-my-opencode is a deep runtime harness *inside* OpenCode (a TypeScript plugin). It hooks the agent loop directly: preemptive compaction, model routing, edit-tool replacements, background workers, tmux integration, LSP/AST-grep tools, hash-anchored edits.

gpowers is a portable skill/content layer that sits *above* the host agent runtime (Claude Code, Codex, Gemini, Cursor, OpenCode, Copilot, Kimi). It does not own the agent loop.

Implication: roughly half of oh-my-opencode's features **cannot be ported as-is** (runtime hooks, model routing, LSP-backed tools, hash-anchored edits). What *can* be ported is the **methodology** behind those features, captured as gpowers skills. Where runtime enforcement matters, we degrade to prompt-only protocols with the gap documented.

## What oh-my-opencode genuinely does better

| Area | Their edge |
|---|---|
| Discipline-agent personas | Sisyphus (orchestrator) / Hephaestus (deep worker) / Oracle (verifier) / Librarian (research) / Prometheus (planner) / Explore. Each a *named executor* with model affinity. gpowers roles are mostly reviewers, not executors. |
| Ultrawork loop with Oracle verification | `ultrawork`/`ulw` runs until Oracle independently verifies completion (not just "DONE" tag). gpowers has `verification-before-completion` skill but no enforced loop. |
| Category-based agent routing | Agents declare a *work category* (`visual-engineering`/`deep`/`quick`/`ultrabrain`), harness maps to model. Cleaner than naming models directly. |
| `/init-deep` hierarchical AGENTS.md | Generates AGENTS.md at root + scored subdirs with concrete dynamic-agent-spawn rules (`>100 files → +1 explore agent per 100`). gpowers' Claude Code `/init` is single-file. |
| IntentGate | Pre-flight intent classification step before any action. |
| Comment-checker / remove-AI-slops | A *detector* for AI tells + a strip command. gpowers' `careful` skill guides but doesn't detect. |
| Skill-Embedded MCPs | Skills carry their own MCP servers and load on demand → less context bloat. gpowers MCP support is platform-level, not per-skill. |
| Preemptive compaction | Monitors context degradation and compacts proactively. Runtime-only — methodology can be a skill, mechanism can't. |
| Hash-anchored edits (`LINE#ID`) | Stale-line edit errors → ~0%. Pure runtime — can port discipline only, not the tool. |
| Quantified parallel-agent dispatch | Concrete thresholds for fan-out (depth, file count, language count). gpowers' `dispatching-parallel-agents` is qualitative. |

## What gpowers does better than oh-my-opencode

- True cross-platform (oh-my-opencode is OpenCode-only despite the rename).
- Deeper methodology library (TDD, systematic-debugging, writing-plans, finishing-a-development-branch).
- Reviewer roles (pr-review, cso, plan-ceo-review, devex-review) — oh-my-opencode has nothing equivalent.
- Modular install (core/roles/tools/business).

## Roadmap

Sequenced by ROI and dependency order. Each row gets its own design doc + plan when picked up.

| # | Spec | Module | Status | Dependencies | Notes |
|---|---|---|---|---|---|
| 1 | **Ultrawork loop + Oracle verification** | `core/skills/ultrawork` | ✅ Design approved | — | See `2026-05-21-ultrawork-loop-design.md`. Land first. |
| 2 | **Discipline-agent personas** | `roles/skills/sisyphus,hephaestus,oracle,librarian,prometheus` | Not started | (Oracle persona partially defined by Ultrawork — pull it out into role skill in this pass) | Biggest conceptual shift: gpowers `roles/` gains *executor* roles, not just reviewers. |
| 3 | **`/init-deep` hierarchical AGENTS.md** | `tools/skills/init-deep` | Not started | — | Self-contained. Easy warm-up. Port the scale-based fan-out table verbatim from `src/features/builtin-commands/templates/init-deep.ts`. |
| 4 | **IntentGate pre-flight skill** | `core/skills/intent-gate` | Not started | — | Smallest spec. Complements `brainstorming` for shorter requests. |
| 5 | **AI-slop detector + strip command** | `tools/skills/remove-ai-slops` (+ optional `tools/skills/comment-checker`) | Not started | — | Detector is a heuristic regex set. Strip is a command. PR-hygiene win. |
| 6 | **Skill-Embedded MCPs (schema)** | gpowers manifest / skill loader | Not started | Per-platform loader changes | Schema extension: skills declare `mcp_servers:` in frontmatter, loader installs them on demand. Heavy — touches every platform adapter. |
| 7 | **Category-based model routing (advisory)** | gpowers skill frontmatter | Not started | (6 useful but not required) | Skills declare `category: visual-engineering | deep | quick | ultrabrain`. Hosts that support routing read the field; others ignore. Advisory in v1. |
| 8 | **OpenCode-native Ultrawork adapter** | `platforms/opencode/adapters/gpowers-ultrawork` | Not started | 1 | Closes the assurance gap for OpenCode users (where ultrawork was born). |
| 9 | **Quantified parallel-agent dispatch table** | augment `core/skills/dispatching-parallel-agents` | Not started | — | Pull the threshold table from oh-my-opencode's init-deep into the existing skill. Small. |
| 10 | **Hash-anchored edit discipline** | augment existing edit-related skills | Not started | — | Can't port the tool. Can document: "verify line content before editing, prefer Edit's `old_string` over line-number tools." Methodology only. |
| 11 | **Preemptive compaction discipline** | augment `careful` / `simplify` | Not started | — | Hint inside existing skills: watch for context bloat, summarize proactively. Mechanism stays runtime. |

## Not portable — documented for completeness

The following oh-my-opencode features are runtime-only and have no skill-shaped equivalent:

- Hash-anchored edit tool itself (`LINE#ID`).
- LSP + AST-grep integrated tools.
- Preemptive compaction mechanism.
- Tmux integration as an internal tool.
- Auto-update checker (runtime).
- The whole OpenCode plugin hook system (preemptive-compaction, edit-error-recovery, anthropic-context-window-limit-recovery, model-fallback, etc.).
- Skill-MCP runtime manager (the `mcp_servers:` schema field in item 6 above is the closest gpowers-shaped analogue).

## Operating principle

When porting, prefer:

1. **Skill content over runtime code** — keep gpowers a content layer.
2. **Documenting the assurance gap** when prompt-only enforcement is the only option, rather than pretending parity.
3. **Per-platform adapters only when the value justifies the maintenance cost** — Ultrawork OpenCode adapter (item 8) is the only adapter currently justified.
