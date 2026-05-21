---
name: init-deep
description: |
  Generate hierarchical AGENTS.md (root + complexity-scored subdirs). Use when the project
  has grown past a single-file AGENTS.md or when onboarding new contributors needs more
  than a flat overview. Coexists with host built-in /init — this is the hierarchical upgrade.
namespace: tools
upstream: oh-my-opencode@v3.17.10
---

# init-deep — Hierarchical AGENTS.md Generator

## Overview

Hosts that ship a built-in `/init` produce a single-file root `AGENTS.md`. This works for
small projects but degrades as projects grow:

- One file accumulates everything — conventions, anti-patterns, module purposes, gotchas.
- Subdirectory-specific knowledge is either lost or buried.
- Onboarding new agents requires reading the whole file even for a localized task.

`/init-deep` solves this with hierarchical generation: a root file plus
complexity-scored subdirectory files. The scoring matrix and fan-out table are deterministic;
the agent dispatch follows `dispatching-parallel-agents` discipline.

**Output target:** `AGENTS.md` only. `CLAUDE.md` is left untouched even when present.

## When to Use

- Project has >10 source files and >3 distinct subdirectories.
- Existing root `AGENTS.md` exceeds 100 lines.
- Onboarding new agents to localized tasks in a specific module.
- After a major refactor that invalidated the existing flat `AGENTS.md`.

## When NOT to Use

- Tiny projects (<10 files, <3 subdirs) — host built-in `/init` is sufficient.
- Projects already maintaining a well-structured hierarchical `AGENTS.md` tree.
- One-off tasks where you only need a flat overview.

## Modes

| Flag | Behavior |
|---|---|
| (none) | **Update** (default): read existing `AGENTS.md`, merge with fresh findings, surgical edits only. |
| `--create-new` | Read all existing `AGENTS.md` / `CLAUDE.md`, rename to `.bak-<timestamp>`, regenerate from scratch using extracted insights. |
| `--max-depth=N` | Limit subdirectory recursion depth. Default 3. Monorepo auto-bumps to cover package roots. |

## Scoring Matrix

For each non-root directory under `--max-depth`, compute a weighted score (0–10):

| Factor | Weight | High threshold | Source signal |
|---|---|---|---|
| File count | 3× | >20 | `find . -type f -not -path '*/node_modules/*' -not -path '*/.git/*' \| wc -l` |
| Subdir count | 2× | >5 | `find . -type d -maxdepth 2 -not -path '*/node_modules/*' \| wc -l` |
| Code ratio | 2× | >70% | code-extension files ÷ total files |
| Unique patterns | 1× | has own `.eslintrc` / `pyproject.toml` / `package.json` | bash + explore agent |
| Module boundary | 2× | has `index.ts` / `__init__.py` / `mod.rs` | bash |

Max weighted score = 10 (each factor scored 0 or 1, multiplied by weight, summed).

**Decision rules:**

| Score | Action |
|---|---|
| Root (`.`) | ALWAYS create |
| ≥ 7 | Create `AGENTS.md` |
| 4 – 6 | Create only if directory is a distinct domain (justification required) |
| < 4 | Skip — parent covers |

**Default to create when in doubt.** Pruning later is cheap; regenerating lost content is not.

## Dynamic Agent Fan-out

Baseline: 6 fixed explore agents.

| Factor | Threshold | Additional agents |
|---|---|---|
| Total files | >100 | +1 per 100 files |
| Total lines | >10k | +1 per 10k lines |
| Directory depth | ≥4 | +2 for deep exploration |
| Large files (>500 lines) | >10 files | +1 for complexity hotspots |
| Monorepo | detected | +1 per package/workspace |
| Multiple languages | >1 | +1 per language |

**Bash measurement block (run in Phase 1):**

```bash
total_files=$(find . -type f -not -path '*/node_modules/*' -not -path '*/.git/*' | wc -l)
total_lines=$(find . -type f \( -name "*.ts" -o -name "*.py" -o -name "*.go" -o -name "*.js" -o -name "*.rs" \) -not -path '*/node_modules/*' -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
large_files=$(find . -type f \( -name "*.ts" -o -name "*.py" -o -name "*.go" -o -name "*.js" -o -name "*.rs" \) -not -path '*/node_modules/*' -exec wc -l {} + 2>/dev/null | awk '$1 > 500 {count++} END {print count+0}')
max_depth=$(find . -type d -not -path '*/node_modules/*' -not -path '*/.git/*' | awk -F/ '{print NF}' | sort -rn | head -1)
```

**Dynamic agent calculation example:**
500 files / 50k lines / depth-6 / 15 large-files / monorepo-2-packages / 2 languages
→ 6 fixed + 5 (files) + 5 (lines) + 2 (depth) + 1 (large) + 2 (monorepo) + 1 (lang) = **22 agents**

## Root AGENTS.md Template

**Size cap: 50–150 lines.** No generic advice. No obvious info.

````markdown
# PROJECT KNOWLEDGE BASE

**Generated:** {TIMESTAMP}
**Commit:** {SHORT_SHA}
**Branch:** {BRANCH}

## OVERVIEW
{1–2 sentences: what + core stack}

## STRUCTURE
{root}/
├── {dir}/    # {non-obvious purpose only}
└── {entry}

## WHERE TO LOOK
| Task | Location | Notes |

## CONVENTIONS
{ONLY deviations from standard}

## ANTI-PATTERNS (THIS PROJECT)
{Explicitly forbidden here}

## UNIQUE STYLES
{Project-specific}

## COMMANDS
{dev / test / build}

## NOTES
{Gotchas}
````

## Subdir AGENTS.md Template

**Size cap: 30–80 lines.** NEVER repeat parent content.

````markdown
# {DIR_NAME}

## OVERVIEW (1 line)

## STRUCTURE (only if >5 subdirs)

## WHERE TO LOOK

## CONVENTIONS (only if different from root)

## ANTI-PATTERNS (only specific to this dir)
````

## Anti-patterns (avoid these)

- **Static agent count** → MUST scale via the fan-out table.
- **Sequential execution** → MUST parallelize Phase 1 explore agents.
- **Ignoring existing** → ALWAYS read existing `AGENTS.md` / `CLAUDE.md` first, even with `--create-new`.
- **Over-documenting** → Not every dir needs its own `AGENTS.md`.
- **Redundancy** → Child never repeats parent content.
- **Generic content** → Remove anything that applies to all projects.
- **Verbose style** → Telegraphic or die.

## The 4-Phase Workflow

### Phase 1 — Discovery + Analysis (concurrent)

Main session actions:
1. **Bash structural analysis:** compute `total_files`, `total_lines`, `large_files`, `max_depth`.
2. **Find existing docs:** locate every `AGENTS.md` and `CLAUDE.md`.
3. **Read existing docs:** extract key insights → `EXISTING` map.
   - If `--create-new`: read all first, then stage rename to `.bak-<timestamp>`.
4. **Dispatch explore agents** (background, read-only, `load_skills=[]`):
   - Baseline 6: project structure / entry points / conventions / anti-patterns / build-CI / test patterns.
   - Plus dynamic agents from the fan-out table.
   - Each agent returns a short bullet list of findings.

**Hosts without parallel dispatch:** run analysis steps sequentially. Slower but functionally equivalent. Note degraded mode in the final report.

### Phase 2 — Scoring & Decision

For each non-root directory under `--max-depth`:
1. Compute the 5-factor weighted score (0–10).
2. Apply decision rules: root always; ≥7 create; 4–6 conditional; <4 skip.
3. Emit `AGENTS_LOCATIONS` list.

**Monorepo override:** if `package.json` workspaces / `pnpm-workspace.yaml` / `lerna.json` detected, auto-bump `--max-depth` to cover one level past the package root. Note the override in the final report.

### Phase 3 — Generate

- **Root `AGENTS.md`:** main session, full treatment, 50–150 lines.
  - Use **Edit** tool if file exists; **Write** tool only for new files.
- **Subdir `AGENTS.md`:** parallel dispatch, one writer agent per location.
  - Each writer: `load_skills=[]`, short prompt with template + location-specific findings slice.
  - 30–80 lines, NEVER repeat parent.

### Phase 4 — Review

For each generated file:
1. Strip generic advice.
2. Strip parent duplicates.
3. Trim to size caps.
4. Enforce telegraphic style.
5. Scrub AI-slop deny-list.

**AI-slop deny-list:** `comprehensive`, `robust`, `leverages`, `powerful`, `seamlessly`, `elegant`, `battle-tested`, `production-ready`, `enterprise-grade`, `best practices`. Remove or replace with concrete specifics.

## Edge Cases & Error Handling

1. **No code files in project.** If `total_files < 10` or no code-extension files detected: generate root only; skip scoring and fan-out phases. Final report notes "skipped scoring (project too small)".

2. **Host lacks parallel agent dispatch.** Degrade to sequential analysis. One-line warning in final report: `Mode: sequential (host lacks parallel dispatch)`.

3. **Existing AGENTS.md unreadable / corrupt.** Attempt `Read`; on failure log path under `Skipped existing files` in final report and proceed without merging. Never silently lose user content.

4. **`--create-new` mid-run interruption.** Stage move (`AGENTS.md → AGENTS.md.bak-<timestamp>`) rather than `rm`. Restore step documented. If host doesn't support file-rename, fall back to in-memory preservation and warn explicitly before deleting.

5. **Scoring boundary cases (score = 4 or 6).** Default to create when in doubt; Phase 4 reviews each "conditional band" file for justification.

6. **Max-depth conflicts with monorepo packages.** Auto-bump overrides user `--max-depth` when monorepo detected. User flag is treated as floor, not ceiling. Final report notes the override.

7. **AI-slop in generated content.** Phase 4 strips the deny-list. Verify diff before write-back.

8. **CLAUDE.md vs. AGENTS.md collision.** Update mode reads both, merges insights, writes back only to `AGENTS.md`. Do not modify `CLAUDE.md` unless the user explicitly opts in via a future flag.

9. **Concurrent writers racing on the same file.** `AGENTS_LOCATIONS` entries don't overlap by construction. If two dispatches resolve to the same path, abort the second and surface the bug; don't silently overwrite.

10. **Hidden / generated directories.** `node_modules`, `.git`, `dist`, `build`, `venv`, `.venv`, `__pycache__`, `.next`, `.cache` — excluded from both scoring and writing. Any directory matching `.*` or appearing in `.gitignore` is excluded.

11. **`.gitignore`-respecting traversal.** Where available, prefer `rg --files` over `find`. Fallback to `find` with the explicit exclusion list above.

12. **Project with single-file AGENTS.md too large for hierarchical split.** If existing root `AGENTS.md` > 1000 lines: proceed with hierarchical split. After Phase 3, the root file should drop into the 50–150 range; any content that logically belongs to a subdir gets relocated.

## Relationship to Other Skills

| Skill | Relationship |
|---|---|
| `dispatching-parallel-agents` | Phase 1 loads it for explore-agent dispatch discipline. |
| Host built-in `/init` | Coexists. `/init` = flat root file. `/init-deep` = hierarchical upgrade. |
| `writing-skills` | Discipline used when writing this skill itself; not consumed at run-time. |

## Final Report Template

```
=== init-deep complete ===
Mode: update | create-new
Dirs Analyzed: N
Created: N  ·  Updated: N  ·  Skipped: N

Files:
[OK] ./AGENTS.md (root, N lines)
[OK] ./src/hooks/AGENTS.md (N lines)
[--] ./src/utils/AGENTS.md (score 3, parent covers)

Notes:
- Monorepo auto-bump: max-depth 3 → 5
- Degraded mode: sequential (host lacks parallel dispatch)
- Skipped existing files: ./docs/CLAUDE.md (unreadable)
```

## Example: Scoring `src/hooks/`

Hypothetical project: `src/hooks/` has 25 files, 7 subdirs, 85% code ratio, its own `.eslintrc`, and an `index.ts`.

| Factor | Raw | Weighted |
|---|---|---|
| File count | 1 (>20) | 3 |
| Subdir count | 1 (>5) | 2 |
| Code ratio | 1 (>70%) | 2 |
| Unique patterns | 1 (has `.eslintrc`) | 1 |
| Module boundary | 1 (has `index.ts`) | 2 |
| **Total** | | **10** |

Score 10 ≥ 7 → **Create** `src/hooks/AGENTS.md`.
