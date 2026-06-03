# Design: Integrate taste-skill Frontend Design System into gpowers

**Date:** 2026-06-03
**Topic:** taste-skill integration into gpowers core/skills
**Audit Strategy:** Deep

---

## 1. Purpose & Scope

### 1.1 What We're Building

[C:USER] A unified frontend design methodology skill for gpowers that integrates the battle-tested anti-slop directives from the taste-skill ecosystem. This skill guides AI agents to produce premium, non-generic frontend interfaces across multiple frameworks.

### 1.2 In Scope

[C:USER]

- **5 integrated skill files** under `core/skills/frontend-design/`:
  - `SKILL.md` — unified general-purpose frontend design methodology
  - `gpt-taste.md` — aggressive/Awwwards-level creative mode
  - `image-to-code.md` — image-first design-to-code workflow
  - `redesign.md` — existing project audit and upgrade protocol
  - `stitch.md` — Google Stitch-compatible DESIGN.md generation

- **Multi-framework detection and adaptation**: Vite+React (default), Next.js, Vue, Svelte, Solid
- **3 style presets merged** from upstream variants: high-end/soft, minimalist/editorial, brutalist/industrial
- **Output completeness enforcement** integrated into pre-flight checklist (from full-output-enforcement)

### 1.3 Out of Scope (Deferred)

[C:USER]

| Item | Reason |
|------|--------|
| `design-taste-frontend-v1` | v2 supersedes v1 completely; no backward-compatibility need in gpowers |
| `imagegen-frontend-web` | Requires text-to-image capabilities not available in Kimi Code CLI |
| `imagegen-frontend-mobile` | Requires text-to-image capabilities not available in Kimi Code CLI |
| `imagegen-brandkit` | Requires text-to-image capabilities not available in Kimi Code CLI |
| `high-end-visual-design` (standalone) | Merged into v2 as Use-Case Preset |
| `minimalist-ui` (standalone) | Merged into v2 as Use-Case Preset |
| `industrial-brutalist-ui` (standalone) | Merged into v2 as Use-Case Preset |
| `full-output-enforcement` (standalone) | Content merged into pre-flight checklist; standalone skill not needed |

---

## 2. Architecture

### 2.1 High-Level Structure

[C:USER]

```
core/skills/frontend-design/
├── SKILL.md              # Layer 1: Universal design methodology
├── gpt-taste.md          # Layer 2: Aggressive creative mode
├── image-to-code.md      # Layer 2: Image-to-code workflow
├── redesign.md           # Layer 2: Redesign audit protocol
└── stitch.md             # Layer 2: Stitch-compatible DESIGN.md
```

**Loading Rule:** [C:USER]

1. Detect any frontend/design trigger signal → Load `SKILL.md` to establish universal context
2. Detect specific sub-signal → Additionally load corresponding appendix
3. Appendix conflicts with main doc → **Appendix wins** (explicit override)

### 2.2 Trigger Routing

[C:USER]

| Skill File | Trigger Signals |
|------------|-----------------|
| `SKILL.md` (always loaded first) | website, landing page, frontend, UI, interface, portfolio, SaaS page, web app |
| `gpt-taste.md` | Awwwards, award-winning, very creative, cinematic, high-end showcase, agency-level |
| `image-to-code.md` | implement from image, image to code, code from screenshot, reference image implementation |
| `redesign.md` | redesign, upgrade existing, revamp existing, optimize existing UI, modernize |
| `stitch.md` | Stitch, DESIGN.md, generate design system document, semantic design spec |

---

## 3. Content Reorganization Strategy

### 3.1 Source-to-Target Mapping

[C:USER]

| Upstream Source | Absorbed into `SKILL.md` | Kept in Appendix |
|-----------------|--------------------------|------------------|
| `design-taste-frontend` (v2, 1206 lines) | Brief inference, Dial system, Design system mapping, Architecture conventions, Design engineering directives, Context-aware proactivity, Performance guardrails, AI-Tells ban, Pre-flight checklist | — |
| `high-end-visual-design` (98 lines) | Double-Bezel → component spec, Floating island nav → nav pattern, Spring physics → motion preset | — |
| `minimalist-ui` (85 lines) | Warm monochrome + muted pastels → palette preset, 1px borders → component spec, Editorial style → Use-Case Preset | — |
| `industrial-brutalist-ui` (92 lines) | Swiss grid → layout preset, Aviation red → palette preset, CRT scanlines → motion preset, Monospace dominance → typography preset | — |
| `gpt-taste` (74 lines) | — | Python RNG randomization, AIDA forced structure, 2-line hero iron rule, Gapless bento grid, Mandatory GSAP |
| `image-to-code` (1228 lines) | — | Image-first mandatory rule, Generate-enough-images rule, Deep image analysis spec, Combinatorial variation engine |
| `redesign-existing-projects` (178 lines) | — | Scan → Diagnose → Fix workflow, Full design audit checklist, Upgrade techniques, Fix priority order |
| `stitch-design-taste` (184 lines) | — | DESIGN.md output format, Semantic design system encoding, Anti-pattern list for Stitch |

### 3.2 Content Removed

[C:USER]

- **Block Library placeholder** (v2 Section 12): Schema defined but no implementations exist; removed to avoid confusion
- **React/Next.js-only assumptions**: Replaced with multi-framework detection logic

---

## 4. Key Design Decisions

### 4.1 Default Tech Stack: Vite + React

[C:USER] The default recommended stack is **Vite + React + Tailwind + Motion** instead of upstream's Next.js default.

**Rationale:**
- Vite builds produce static files deployable on any HTTP server (nginx, httpd, CDN)
- Lower build dependency footprint
- Faster cold start and HMR
- Next.js mode still available when `next` is detected in project dependencies

### 4.2 Multi-Framework Detection

[C:INFERRED]

```
Detection signals:
  - package.json dependencies
  - vite.config.* / next.config.* / nuxt.config.* / svelte.config.*
  - Existing file patterns in project

Branches:
  No signal / new project  → Vite + React (default)
  next detected            → Next.js + React
  vite + react detected    → Vite + React
  vue detected             → Vue 3 + Vite
  svelte detected          → SvelteKit
  solid detected           → Solid + Vite
```

### 4.3 Style Presets as Dial Values

[C:USER]

The 3 upstream style-variant skills are merged as **Use-Case Presets** in the Dial system:

| Preset | VARIANCE | MOTION | DENSITY | Visual Character |
|--------|----------|--------|---------|-----------------|
| High-end / Soft / Luxury | 7 | 6 | 3 | Double-Bezel cards, spring physics, floating nav |
| Minimalist / Editorial | 5 | 3 | 2 | Warm monochrome, muted pastels, 1px borders, massive whitespace |
| Brutalist / Industrial | 9 | 7 | 8 | Swiss grid, aviation red, zero border-radius, monospace, CRT effects |

### 4.4 Output Completeness Integration

[C:USER]

The `full-output-enforcement` skill is not imported as a standalone skill. Its two unique mechanisms are merged into the pre-flight checklist:

1. **Placeholder pattern ban**: `// ...`, `// TODO`, `/* ... */`, bare `...` are added to forbidden output patterns
2. **PAUSED continuation mechanism**: `[PAUSED — X of Y complete. Send "continue" to resume]` is documented as the standard token-limit handling strategy

---

## 5. Interfaces

### 5.1 Skill Loading Interface

[C:INFERRED]

The skill follows gpowers core skill auto-trigger convention:

```
Trigger: user message matches any frontend/design keyword
Action:  load core/skills/frontend-design/SKILL.md into context
Then:    if message matches sub-signal, load corresponding appendix
```

### 5.2 Frontmatter Specification

[C:USER]

Minimal frontmatter for all 5 files:

```yaml
---
name: frontend-design
description: Unified anti-slop frontend design methodology for premium interface generation
namespace: core
upstream: taste-skill@main
---
```

Appendix files additionally include a header indicating their relationship to the main skill:

```markdown
# gpt-taste: Aggressive Creative Mode

> This appendix extends `frontend-design/SKILL.md`. General design principles, Dial system, and AI-Tells bans are defined in the main skill. This document only records rules that **override or extend** the main skill.
```

---

## 6. Data Flow

### 6.1 Task Execution Flow

[C:USER]

```
User Request
    │
    ▼
[Trigger Detection]
    │
    ├── No match → proceed normally (no skill loaded)
    │
    └── Match frontend/design signal
            │
            ▼
    [Load SKILL.md]
            │
            ▼
    [Detect sub-signal?]
            │
            ├── No → execute with SKILL.md only
            │
            └── Yes → load corresponding appendix
                            │
                            ├── gpt-taste signal → load gpt-taste.md
                            ├── image-to-code signal → load image-to-code.md
                            ├── redesign signal → load redesign.md
                            └── stitch signal → load stitch.md
                                        │
                                        ▼
                            [Execute with main + appendix]
                                        │
                                        ▼
                            [Pre-flight Checklist]
                                        │
                                        ▼
                            [Output]
```

---

## 7. Configuration

### 7.1 No External Configuration

[C:INFERRED]

This skill requires no external configuration files, environment variables, or feature toggles. All behavior is encoded in the SKILL.md documents themselves.

### 7.2 Per-Project Override

[C:INFERRED]

Users can override Dial values conversationally (as in upstream). No file-based configuration needed.

---

## 8. Error Handling & Degradation

### 8.1 Framework Detection Failure

[C:INFERRED]

| Scenario | Handling |
|----------|----------|
| Cannot read package.json | Fall back to Vite + React default |
| package.json exists but no recognizable framework | Fall back to Vite + React default |
| Multiple frameworks detected (e.g., Vue + React in monorepo) | Use the framework with the most source files |

### 8.2 Skill Load Failure

[C:INFERRED]

| Scenario | Handling |
|----------|----------|
| Main SKILL.md missing | Skill does not load; proceed without design methodology |
| Appendix missing but signal matched | Load main SKILL.md only; warn that specific mode unavailable |

---

## 9. Observability

[C:INFERRED]

No logging, metrics, or telemetry required. This is a static methodology skill.

---

## 10. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| 1 | Upstream taste-skill releases updates; gpowers copy becomes stale | Medium | Medium | Document upstream version in `upstream` frontmatter field; periodic manual sync |
| 2 | Merged style presets lose nuanced distinctions from original standalone skills | Low | Medium | Each preset includes explicit visual character description; appendix references upstream source |
| 3 | Multi-framework detection produces false positives in monorepos | Medium | Low | Detection uses "most source files" heuristic; user can correct conversationally |
| 4 | Content volume too large for context window (v2 alone is 1200+ lines) | Medium | High | Remove Block Library placeholder; appendix pattern ensures only relevant content loads |

---

## 11. Done Criteria

[C:USER]

1. [ ] `core/skills/frontend-design/` directory exists with 5 files
2. [ ] Each file has correct gpowers frontmatter (name, description, namespace: core, upstream: taste-skill@main)
3. [ ] `SKILL.md` contains: brief inference, Dial system with 12 presets, multi-framework detection, design system mapping, AI-Tells ban, pre-flight checklist
4. [ ] `gpt-taste.md` contains: Python RNG, AIDA, 2-line rule, gapless bento, mandatory GSAP
5. [ ] `image-to-code.md` contains: image-first rule, generate-enough rule, deep analysis spec, variation engine
6. [ ] `redesign.md` contains: scan-diagnose-fix workflow, audit checklist, upgrade techniques
7. [ ] `stitch.md` contains: DESIGN.md format, semantic design system, Stitch anti-patterns
8. [ ] All placeholder patterns from full-output-enforcement merged into pre-flight checklist
9. [ ] Block Library placeholder removed
10. [ ] No `// ...` or `...` truncation patterns in any file
11. [ ] Git commit with clear message

---

## 12. Test Plan

[C:INFERRED]

| Test | Assertion |
|------|-----------|
| Load test | All 5 files are readable markdown with valid frontmatter |
| Trigger test | Each trigger signal correctly identifies its appendix |
| Framework detection | Vite+React default; Next.js/Vue/Svelte detected correctly |
| Content integrity | No placeholder truncation patterns (`// ...`, bare `...`) exist |
| Cross-reference | Each appendix correctly references main SKILL.md |

---

## 13. Assumptions & Unverified Items

| # | Assumption | Confidence | Impact if Wrong | How to Verify |
|---|-----------|-----------|-----------------|---------------|
| 1 | Kimi Code CLI loads core skills automatically based on trigger keywords | Medium | If not, skills won't activate | Test with a frontend task prompt after deployment |
| 2 | 5 skill files + main context fit within Kimi Code CLI context window | Medium | If not, content needs further truncation | Test with a realistic prompt after deployment |
| 3 | Users prefer Vite + React over Next.js as default | Medium | If not, default stack misaligned with user expectation | User feedback after first week of use |
| 4 | Merging 3 style variants into Dial presets does not lose critical distinctions | Medium | If presets too vague, output quality degrades | Manual review of outputs using each preset |
| 5 | `verification-before-completion` skill remains unchanged; no merge conflicts | High | If changed, output completeness checks may diverge | Review verification-before-completion before final commit |

---

## 14. Open Questions / Resolved Decisions

### Resolved Decisions

| # | Decision | Source | Dimension |
|---|----------|--------|-----------|
| 1 | Import 5 skills only (not all 13 upstream) | [C:USER] | Scope |
| 2 | Place in `core/skills/` | [C:USER] | Module |
| 3 | Use upstream install names as directory names | [C:USER] | Naming |
| 4 | Use unified directory `frontend-design/` with 5 files | [C:USER] | Architecture |
| 5 | Block Library placeholder removed | [C:USER] | Content |
| 6 | Minimal frontmatter (name + description + namespace + upstream) | [C:USER] | Format |
| 7 | Vite + React as default stack (was Next.js) | [C:USER] | Tech Stack |
| 8 | Multi-framework detection with branches | [C:USER] | Integration |
| 9 | 3 style variants merged as Dial Use-Case Presets | [C:USER] | Content |
| 10 | full-output-enforcement merged into pre-flight checklist | [C:USER] | Integration |
| 11 | gpt-taste kept as separate appendix with task-signal routing | [C:USER] | Scope |
| 12 | 5 independent trigger signals map to main + appendix loading | [C:USER] | Integration |

### Deferred to V2+

| # | Item | Reason |
|---|------|--------|
| 1 | Block Library implementation | Upstream only has schema, no actual blocks |
| 2 | Image generation skills | No text-to-image capability in current environment |
| 3 | Automated upstream sync | Manual sync sufficient for initial release |
| 4 | Per-project configuration file | Conversation-level override sufficient |

---

*Design document written per gpowers brainstorming (core) methodology.*
*Audit Strategy: Deep. All [C:INFERRED] assumptions surfaced for review.*
