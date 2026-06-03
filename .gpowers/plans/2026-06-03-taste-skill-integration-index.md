# taste-skill Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create `core/skills/frontend-design/` as a unified 5-file frontend design methodology skill, integrating battle-tested anti-slop directives from the taste-skill ecosystem into gpowers.

**Architecture:** One main skill file (`SKILL.md`) establishes universal design methodology with the Dial system, multi-framework detection, and AI-Tells bans. Four appendix files (`gpt-taste.md`, `image-to-code.md`, `redesign.md`, `stitch.md`) provide specialized modes loaded on sub-signal match. Appendix content overrides main doc when conflicts occur.

**Tech Stack:** Pure Markdown skill files — no compilation or runtime dependencies. Content covers Vite+React (default), Next.js, Vue, Svelte, Solid frameworks.

---

## File Structure

All files created under `core/skills/frontend-design/`:

| File | Responsibility | Source |
|------|---------------|--------|
| `SKILL.md` | Universal design methodology: brief inference, Dial system (12 presets + 3 style variants), multi-framework detection, design system mapping, AI-Tells ban, pre-flight checklist (merged from full-output-enforcement) | `design-taste-frontend` v2 + 3 style variants |
| `gpt-taste.md` | Aggressive creative mode: Python RNG randomization, AIDA forced structure, 2-line hero rule, gapless bento grid, mandatory GSAP | `gpt-taste` |
| `image-to-code.md` | Image-to-code workflow: image-first mandatory rule, generate-enough-images rule, deep image analysis spec, combinatorial variation engine | `image-to-code` |
| `redesign.md` | Existing project upgrade: scan-diagnose-fix workflow, full design audit checklist, upgrade techniques, fix priority order | `redesign-existing-projects` |
| `stitch.md` | Stitch-compatible design system: DESIGN.md output format, semantic design system encoding, Stitch anti-patterns | `stitch-design-taste` |

---

## Dependency Overview

```
Phase A: SKILL.md Main Document (Tasks 1-4)
  Task 1: Create directory + scaffold (frontmatter, header, brief inference)
    -> Task 2: Dial system + multi-framework detection
      -> Task 3: Design system mapping + AI-Tells + pre-flight checklist
        -> Task 4: Finalize SKILL.md (cross-reference check)

Phase B: Appendix Files (Tasks 5-8) — independent of each other after Phase A
  Task 5: gpt-taste.md   [parallel with Tasks 6-8]
  Task 6: image-to-code.md [parallel with Tasks 5,7,8]
  Task 7: redesign.md    [parallel with Tasks 5-6,8]
  Task 8: stitch.md      [parallel with Tasks 5-7]

Phase C: Verification + Commit (Tasks 9-10)
  Task 9: Content integrity scan (no placeholders, frontmatter, truncation check)
    -> Task 10: Git commit
```

**Parallelism note:** Tasks 5-8 can run in any order or in parallel once Task 1 (directory creation) is done. They have no cross-dependencies.

---

## Risks & Open Questions

| Risk | Assumption | Impact if Wrong |
|------|-----------|-----------------|
| Context window size | 5 skill files + main context fit within Kimi Code CLI context window | If not, content needs further truncation; appendix pattern already mitigates this |
| Kimi Code CLI trigger mechanism | Kimi Code CLI loads core skills automatically based on trigger keywords | If not, skills won't activate automatically; test after deployment |
| Upstream content accuracy | Content rewritten from upstream memory is accurate and complete | If gaps found, reference upstream files at `~/Downloads/taste-skill-main/` during execution |
| Block Library removal | Removing placeholder Block Library does not break any downstream references | Confirmed in design doc — upstream only has schema, no implementations |

---

## Spec Coverage

| Design Doc Section | Task(s) | Status |
|---|---|---|
| §1 Purpose & Scope (directory, 5 files) | Task 1 | covered |
| §2.1 High-Level Structure | Task 1 | covered |
| §2.2 Trigger Routing | Task 1 | covered |
| §3.1 Source-to-Target Mapping — SKILL.md content | Tasks 2-4 | covered |
| §3.1 Source-to-Target Mapping — gpt-taste | Task 5 | covered |
| §3.1 Source-to-Target Mapping — image-to-code | Task 6 | covered |
| §3.1 Source-to-Target Mapping — redesign | Task 7 | covered |
| §3.1 Source-to-Target Mapping — stitch | Task 8 | covered |
| §3.2 Block Library removal | Task 3 | covered |
| §4.1 Vite+React default | Task 2 | covered |
| §4.2 Multi-framework detection | Task 2 | covered |
| §4.3 Style presets as Dial values | Task 2 | covered |
| §4.4 Output completeness (full-output-enforcement merge) | Task 3 | covered |
| §5.1 Skill Loading Interface | Task 1 | no-op (gpowers convention, no code) |
| §5.2 Frontmatter | Tasks 1,5-8 | covered |
| §8 Error Handling & Degradation | Task 2 | covered |
| §11 Done Criteria 1-2 (directory + frontmatter) | Tasks 1,5-8 | covered |
| §11 Done Criteria 3 (SKILL.md content) | Tasks 2-4 | covered |
| §11 Done Criteria 4 (gpt-taste content) | Task 5 | covered |
| §11 Done Criteria 5 (image-to-code content) | Task 6 | covered |
| §11 Done Criteria 6 (redesign content) | Task 7 | covered |
| §11 Done Criteria 7 (stitch content) | Task 8 | covered |
| §11 Done Criteria 8 (placeholder ban merge) | Task 3 | covered |
| §11 Done Criteria 9 (Block Library removed) | Task 3 | covered |
| §11 Done Criteria 10 (no truncation patterns) | Task 9 | covered |
| §11 Done Criteria 11 (git commit) | Task 10 | covered |

---

## Parts (generate one per invocation, in order)

> ▶ To generate the next `pending` part: run `/compact`, then re-invoke the `/writing-plans` slash command. Do NOT type "continue" — it skips the rule reload and batch-generates everything.

| # | File | Scope | Status |
|---|---|---|---|
| 1 | taste-skill-integration-main.md | SKILL.md main document (Tasks 1-4) | done |
| 2 | taste-skill-integration-appendices.md | 4 appendix files (Tasks 5-8) | done |
| 3 | taste-skill-integration-verify.md | Verification + git commit (Tasks 9-10) | done |
