# `/init-deep` — Hierarchical AGENTS.md Generator — Design

**Date:** 2026-05-21
**Owner:** ranwei693532
**Status:** Approved for plan
**Roadmap row:** #3 in `2026-05-21-oh-my-opencode-port-roadmap.md`
**Source of inspiration:** oh-my-opencode v3.17.10 (`src/features/builtin-commands/templates/init-deep.ts`)

## Problem

Hosts that ship a built-in `/init` (Claude Code, Kimi) produce a single-file root `AGENTS.md`. This works for small projects but degrades as projects grow:

- One file accumulates everything — conventions, anti-patterns, module purposes, gotchas.
- Subdirectory-specific knowledge is either lost or buried.
- Onboarding new agents requires reading the whole file even for a localized task.

oh-my-opencode solves this with `/init-deep`: hierarchical AGENTS.md generation — root file plus complexity-scored subdirectory files. The novel content worth porting:

- A 4-phase workflow (discovery → scoring → generate → review).
- A scoring matrix that decides which subdirs warrant their own AGENTS.md.
- A dynamic fan-out table for explore-agent count based on project scale.
- Telegraphic-output discipline that keeps each file small and concrete.

The implementation in oh-my-opencode is tied to OpenCode runtime tools (LSP queries, `task(subagent_type=...)`, `background_output(...)`). We port the methodology as a gpowers skill, rewriting tool calls in platform-neutral language and dropping the LSP-dependent parts.

## Goal

Ship a gpowers `tools/` skill that:

1. Generates a hierarchical AGENTS.md tree (root + scored subdirs).
2. Decides which subdirs warrant their own file via a deterministic scoring matrix.
3. Scales the explore-agent count to project size via a fan-out table.
4. Enforces telegraphic output discipline (size caps, no AI-slop, no parent-duplication).
5. Coexists with host built-in `/init` rather than shadowing it.

## Non-goals (v1)

- LSP-augmented scoring (symbol density / export count / reference centrality). Cross-platform LSP availability varies; defer to v2 when gpowers has a portable LSP primitive.
- CLAUDE.md regeneration. Cross-platform AGENTS.md is the standard target; CLAUDE.md is left untouched.
- Snapshot tests on generated content. LLM outputs aren't byte-deterministic; scenarios test structure.
- Automated end-to-end runner driving a real model. Manual scenarios for v1.

## Architecture

### File layout

```
tools/skills/init-deep/
├── SKILL.md           # primary skill (single file)
└── tests/             # manual exercise scripts (Scenarios I-A..I-G)
```

Single-file SKILL.md matches the source's monolithic shape. v2 may extract bundled scripts if scoring proves unreliable in practice.

### Placement

- **Path:** `tools/skills/init-deep/SKILL.md`
- **Module:** `tools/` (sibling of `fix-the-roof`, `careful`, etc.)
- **Namespace:** `tools`
- **Upstream:** `oh-my-opencode@v3.17.10`
- **Activation:** registered as a slash command via gpowers' platform adapters. Each host gets `/init-deep` if it supports slash commands; Kimi gets `/skill:init-deep`.
- **Coexistence:** never shadows host built-in `/init`. The hierarchical upgrade is opt-in via the deeper name.

### Frontmatter

```yaml
---
name: init-deep
description: Generate hierarchical AGENTS.md (root + complexity-scored subdirs). Use when the project has grown past a single-file AGENTS.md or when onboarding new contributors needs more than a flat overview.
namespace: tools
upstream: oh-my-opencode@v3.17.10
---
```

### Relationship to existing skills

| Skill | Relationship |
|---|---|
| `core/skills/dispatching-parallel-agents` | init-deep loads it inside Phase 1 — explore-agent dispatch follows that discipline. |
| Host built-in `/init` (Claude Code, Kimi) | Coexists. `/init` = flat root file. `/init-deep` = hierarchical upgrade. |
| `core/skills/writing-skills` | Discipline used when writing the skill itself; not consumed at run-time. |

### Modes

- `/init-deep` — **update** (default): merge existing AGENTS.md content with fresh findings.
- `/init-deep --create-new` — read every existing AGENTS.md/CLAUDE.md, rename to `AGENTS.md.bak-<ts>`, regenerate from scratch using extracted insights.
- `/init-deep --max-depth=N` — limit subdir recursion (default 3; monorepo auto-bumps).

### Output target

- AGENTS.md only — the de facto standard format read by Claude Code, Codex, Cursor, OpenCode, Kimi, Qoder.
- CLAUDE.md left untouched even when present (collision behavior — see edge case 8).

## Components

### Scoring matrix (LSP-trimmed)

| Factor | Weight | High threshold | Source signal |
|---|---|---|---|
| File count | 3× | >20 | `find ... -type f \| wc -l` |
| Subdir count | 2× | >5 | `find ... -type d -maxdepth 2 \| wc -l` |
| Code ratio | 2× | >70% | code-extension files ÷ total files |
| Unique patterns | 1× | has own `.eslintrc` / `pyproject.toml` / `package.json` | bash + explore agent |
| Module boundary | 2× | has `index.ts` / `__init__.py` / `mod.rs` | bash |

Max weighted score = 10 (each factor scored 0 or 1, multiplied by weight, summed).

**Decision rules:**

| Score | Action |
|---|---|
| Root (.) | ALWAYS create |
| ≥ 7 | Create AGENTS.md |
| 4 – 6 | Create only if directory is a distinct domain (justification required) |
| < 4 | Skip — parent covers |

Re-anchored from the source's 8/15 thresholds because the LSP factors were dropped — keeping the old thresholds would over-skip on the lower 0-10 ceiling.

### Dynamic agent fan-out table (verbatim from source)

| Factor | Threshold | Additional agents |
|---|---|---|
| Total files | >100 | +1 per 100 files |
| Total lines | >10k | +1 per 10k lines |
| Directory depth | ≥4 | +2 for deep exploration |
| Large files (>500 lines) | >10 files | +1 for complexity hotspots |
| Monorepo | detected | +1 per package/workspace |
| Multiple languages | >1 | +1 per language |

**Baseline:** 6 fixed explore agents (project structure / entry points / conventions / anti-patterns / build-CI / test patterns). Dynamic table adds on top.

### Bash measurement block (kept as skill content, verbatim from source)

```bash
total_files=$(find . -type f -not -path '*/node_modules/*' -not -path '*/.git/*' | wc -l)
total_lines=$(find . -type f \( -name "*.ts" -o -name "*.py" -o -name "*.go" \) -not -path '*/node_modules/*' -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
large_files=$(find . -type f \( -name "*.ts" -o -name "*.py" -o -name "*.go" \) -not -path '*/node_modules/*' -exec wc -l {} + 2>/dev/null | awk '$1 > 500 {count++} END {print count+0}')
max_depth=$(find . -type d -not -path '*/node_modules/*' -not -path '*/.git/*' | awk -F/ '{print NF}' | sort -rn | head -1)
```

### Root AGENTS.md template

```markdown
# PROJECT KNOWLEDGE BASE

**Generated:** {TIMESTAMP}
**Commit:** {SHORT_SHA}
**Branch:** {BRANCH}

## OVERVIEW
{1-2 sentences: what + core stack}

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
{dev/test/build}

## NOTES
{Gotchas}
```

**Size cap: 50-150 lines.** No generic advice. No obvious info.

### Subdir AGENTS.md template

```markdown
# {DIR_NAME}
## OVERVIEW (1 line)
## STRUCTURE (only if >5 subdirs)
## WHERE TO LOOK
## CONVENTIONS (only if different from root)
## ANTI-PATTERNS (only specific to this dir)
```

**Size cap: 30-80 lines.** NEVER repeat parent content.

### Anti-patterns (verbatim from source)

- Static agent count → MUST scale via fan-out table.
- Sequential execution → MUST parallel.
- Ignoring existing → ALWAYS read existing AGENTS.md / CLAUDE.md first, even with `--create-new`.
- Over-documenting → not every dir needs AGENTS.md.
- Redundancy → child never repeats parent.
- Generic content → remove anything that applies to all projects.
- Verbose style → telegraphic or die.

## Data flow (one run)

```
User: /init-deep                                # update mode (default)
   │
   ▼
[Phase 1 — Discovery + Analysis (concurrent)]
   ├── Main session: bash structural analysis
   │     • directory depth, files-per-dir, code concentration
   │     • find existing AGENTS.md / CLAUDE.md
   │     • compute project-scale variables (total_files, total_lines,
   │       large_files, max_depth)
   ├── Main session: read every existing AGENTS.md / CLAUDE.md
   │     • extract key insights → EXISTING map
   │     • (if --create-new: read all first, then rename to .bak, then regenerate)
   └── Concurrent: dispatch 6 fixed explore agents (background)
         • project structure / entry points / conventions /
           anti-patterns / build-CI / test patterns
         • each: load_skills=[], read-only
         • PLUS dynamic agents from fan-out table
   │
   ▼
[Collect background results]
   • merge bash + existing + explore findings → FINDINGS
   │
   ▼
[Phase 2 — Scoring & Decision]
   • For each non-root directory under --max-depth:
       compute 5-factor weighted score (0-10)
       apply decision rules (root always; ≥7 create; 4-6 conditional; <4 skip)
   • Emit AGENTS_LOCATIONS list
   │
   ▼
[Phase 3 — Generate]
   ├── Root AGENTS.md (main session, full treatment, 50-150 lines)
   │     • Edit existing or Write new (NEVER Write over existing)
   └── Subdir AGENTS.md (parallel dispatch, one writer agent per location)
         • each writer: load_skills=[], short prompt with template +
           location-specific FINDINGS slice
         • 30-80 lines, NEVER repeat parent
   │
   ▼
[Phase 4 — Review]
   • For each generated file: strip generic advice, strip parent duplicates,
     trim to size caps, enforce telegraphic style, scrub AI-slop deny-list
   │
   ▼
[Final Report]
   === init-deep complete ===
   Mode: update | create-new
   Files: [OK] ./AGENTS.md (root, N lines)
          [OK] ./src/hooks/AGENTS.md (N lines)
   Dirs Analyzed: N
   Created: N · Updated: N
```

**State persistence:** none. One-shot. Outputs are AGENTS.md files committed to git.

**Key invariants:**

1. **Always read existing first.** Even in `--create-new`, every existing AGENTS.md / CLAUDE.md is read into EXISTING before any backup or write.
2. **Write tool only for new files; Edit tool for existing.** Phase 3 explicitly directs the host.
3. **Subdir AGENTS.md never repeats parent content.** Phase 4 enforces.
4. **Output discipline = telegraphic.** Phase 4 scrubs AI-slop deny-list.

**Cross-platform behavior:**

- Hosts with native task tool (Claude Code Task, Kimi Agent, OpenCode subagents) execute the fan-out as real concurrent agents.
- Hosts without subagent dispatch fall back to sequential single-session analysis. Skill body says: "if your host doesn't support parallel agent dispatch, run the analysis steps sequentially — slower but functionally equivalent." Final report notes degraded mode.

## Error handling & edge cases

1. **No code files in project.** All scoring factors → 0 except root. Skill body: "if total_files < 10 or no code-extension files detected, generate root only; skip scoring/fan-out phases."

2. **Host lacks parallel agent dispatch.** Degrade to sequential; one-line warning in final report.

3. **Existing AGENTS.md unreadable / corrupt.** Read with `Read`; on failure log path under `Skipped existing files` in final report and proceed without merging. Never silently lose user content.

4. **`--create-new` mid-run interruption.** Skill body mandates staging move (`AGENTS.md → AGENTS.md.bak-<timestamp>`) rather than `rm`. Restore step documented. If host doesn't support file-rename, fall back to in-memory preservation and warn the user explicitly before deleting.

5. **Scoring boundary cases (score = 4 or 6).** Skill body: "default to create when in doubt; pruning later is cheap, regenerating lost content is not."

6. **Max-depth conflicts with monorepo packages.** Default `--max-depth=3` may cut off package roots. Skill body: "if monorepo detected (workspaces / packages / lerna config), automatically bump max-depth to cover one level past the package root, regardless of `--max-depth` flag." Final report notes the override.

7. **AI-slop in generated content.** Phase 4 strip deny-list: `comprehensive`, `robust`, `leverages`, `powerful`, `seamlessly`, `elegant`, `battle-tested`, `production-ready`, `enterprise-grade`, `best practices`. Concrete list embedded in skill body. (Partial preview of roadmap item #5 applied locally.)

8. **CLAUDE.md vs. AGENTS.md collision.** Update mode reads both, merges insights, writes back only to AGENTS.md. CLAUDE.md left untouched. Skill body: "do not modify CLAUDE.md unless the user explicitly opts in via a v2 flag."

9. **Concurrent writers racing on the same file.** AGENTS_LOCATIONS entries don't overlap by construction. Skill body: "if two writer dispatches resolve to the same path, abort the second and surface the bug; don't silently overwrite."

10. **Hidden / generated directories.** `node_modules`, `.git`, `dist`, `build`, `venv`, `.venv`, `__pycache__`, `.next`, `.cache` — excluded from both scoring and writing. Skill body: "any directory matching `.*` or appearing in `.gitignore` is excluded from analysis."

11. **`.gitignore`-respecting traversal.** Where available (Claude Code, Kimi via Shell), prefer `rg --files` over `find`. Fallback to `find` with the explicit exclusion list.

12. **Project with single-file AGENTS.md too large for hierarchical split.** Existing root AGENTS.md > 1000 lines: skill body: "this is the case `/init-deep` was designed for — proceed with hierarchical split; after Phase 3, the root file should drop into the 50-150 range; any content that lived in the root but logically belongs to a subdir gets relocated."

## Testing

Skill ships content only — no executable code. Tests = spec validation + manual exercises + install regression.

### Spec-embedded examples in `SKILL.md`

- Sample 5-factor scoring computation for a fictional `src/hooks/` directory.
- Sample dynamic agent fan-out calculation for a 500-files / 50k-lines / depth-6 / 15-large-files project (expected 6 fixed + 5+5+2+1 = 19 agents).
- Sample root AGENTS.md output (within 50-150 lines, no AI-slop terms).
- Sample subdir AGENTS.md output (30-80 lines, no parent repetition).
- Sample final report block.

### Manual exercise scripts in `tools/skills/init-deep/tests/`

| Scenario | Setup | Expect |
|---|---|---|
| **I-A — small project, root only** | `/init-deep` on a 5-file Python script project | Only `./AGENTS.md`; final report notes "skipped scoring/fan-out (project too small)" |
| **I-B — medium project, root + 2 subdirs** | `/init-deep` on this gpowers repo (`core/`, `tools/`, `roles/`) | `./AGENTS.md` + `./core/AGENTS.md` + `./tools/AGENTS.md` + `./roles/AGENTS.md` |
| **I-C — monorepo, max-depth auto-bump** | `/init-deep --max-depth=2` on a workspace with packages at depth 3 | Auto-bumps to cover package roots; report notes override |
| **I-D — --create-new with existing content** | Existing AGENTS.md with project-specific notes; `/init-deep --create-new` | Existing renamed to `AGENTS.md.bak-<ts>`; new file generated; notes preserved via EXISTING merge |
| **I-E — update mode preserves user edits** | User edited a generated AGENTS.md; rerun `/init-deep` | User edits not blown away — host uses Edit, surgical changes only |
| **I-F — host without parallel dispatch** | Run on a host lacking Task/Agent (e.g., minimal Copilot CLI) | Sequential analysis completes; report notes degraded mode |
| **I-G — AI-slop scrubber** | Inject `comprehensive / robust / leverages` into a draft; run Phase 4 | Deny-list strips them; verify diff before write-back |

### Install regression

- Run gpowers' install/test target; confirm `tools/skills/init-deep/SKILL.md` templates cleanly into every platform adapter (claude-code, codex, gemini, cursor, opencode, copilot, kimi, qoder).
- Confirm slash-command surface: `/init-deep` on hosts that consume slash commands from skills; `/skill:init-deep` on Kimi.
- Confirm coexistence: host built-in `/init` is not shadowed.

### Cross-platform smoke check

Run **I-B** on at least two platforms (one with parallel dispatch, one without). Compare resulting AGENTS.md hierarchy: structure should be identical; runtime differs.

### Not in v1

- Automated runner driving a real model end-to-end.
- Snapshot tests of generated AGENTS.md (LLM outputs not byte-deterministic).
- LSP-augmented scoring (out by design — see Non-goals).
- CLAUDE.md regeneration (out by design — edge case 8).

## Open questions / risks

- **Scoring re-anchor (7 / 4-6 vs. source's 15 / 8-15).** No real-world calibration data yet; we're approximating. Mitigation: Phase 4 reviews each "conditional band" file for justification. v2 may add a `--score-threshold=N` flag if the default proves wrong.
- **AI-slop deny-list maintenance.** Embedded in skill body; drifts from roadmap item #5 if list evolves. Mitigation: item #5 designates the canonical list; this skill references it once #5 lands.
- **Fan-out table calibration.** Source's numbers are oh-my-opencode-tuned. May over- or under-spawn on different project shapes. Adjust based on Scenario I-B runs.
- **Monorepo auto-bump overriding `--max-depth`.** Conscious choice — user-set flag is treated as floor, not ceiling. If a user complains, surface as a `--strict-depth` flag in v2.
