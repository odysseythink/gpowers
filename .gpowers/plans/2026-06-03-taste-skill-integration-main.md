# taste-skill Integration — SKILL.md Main Document

> **Local goal:** Write `core/skills/frontend-design/SKILL.md` — the universal frontend design methodology with Dial system, multi-framework detection, AI-Tells ban, and pre-flight checklist.
>
> **Depends on file:** `2026-06-03-taste-skill-integration-index.md` (global plan)

---

## File Structure (this sub-plan)

| File | Created By | Responsibility |
|------|-----------|---------------|
| `core/skills/frontend-design/SKILL.md` | Tasks 1-4 | Main design methodology document |

---

## Dependency Overview

```
Task 1: Create directory + scaffold (frontmatter, header, brief inference section)
  -> Task 2: Dial system (12 presets) + multi-framework detection + 3 style variants
    -> Task 3: Design system mapping + AI-Tells ban + pre-flight checklist
      -> Task 4: Finalize (cross-reference check, loading rules, trigger table)
```

All four tasks touch the same file (`SKILL.md`) in sequential append order.

---

## Risks & Open Questions

| Risk | Mitigation |
|------|-----------|
| Upstream v2 content (~1200 lines) may exceed comfortable single-file size | Append incrementally across Tasks 1-4; each task writes a coherent section |
| Merged style presets may lose nuance from original standalone skills | Each preset includes explicit visual character description + upstream reference |
| Multi-framework detection logic must be accurate | Detection rules are documented as pseudocode; executor adapts to actual project files |

---

## Tasks

### Task 1: Create Directory + SKILL.md Scaffold

**Depends on:** none

**Files:**
- Create: `core/skills/frontend-design/SKILL.md`

- [ ] **Step 1: Create the directory**

Run: `mkdir -p core/skills/frontend-design/`

- [ ] **Step 2: Write the frontmatter and document header**

Create `core/skills/frontend-design/SKILL.md` with this exact frontmatter and header:

```markdown
---
name: frontend-design
description: Unified anti-slop frontend design methodology for premium interface generation. Brief inference, Dial system, multi-framework detection, design system mapping, AI-Tells ban, pre-flight checklist.
namespace: core
upstream: taste-skill@main
---

# frontend-design: Anti-Slop Frontend Skill

> Landing pages, portfolios, and redesigns. Not dashboards, not data tables, not multi-step product UI.
> Every rule below is **contextual**. None of it fires automatically. First read the brief, then pull only what fits.
>
> **Trigger signals:** `website`, `landing page`, `frontend`, `UI`, `interface`, `portfolio`, `SaaS page`, `web app`
> **Appendices:** `gpt-taste.md` (Awwwards-level), `image-to-code.md` (image-first), `redesign.md` (audit-upgrade), `stitch.md` (DESIGN.md format)
> **Loading rule:** This file loads first. If a sub-signal matches, the corresponding appendix loads additionally. Appendix wins on conflict.
```

- [ ] **Step 3: Write Section 0 — Brief Inference**

Append to `SKILL.md` the complete brief inference section. Include:
- 0.A: Six signal categories (page kind, vibe words, references, audience, brand assets, quiet constraints)
- 0.B: The one-line "Design Read" format with 3 examples
- 0.C: Ambiguity → one clarifying question rule
- 0.D: Anti-Default Discipline (banned defaults list)

Content sourced from upstream v2 Section 0, lines 13-39.

- [ ] **Step 4: Verify file is readable and well-formed**

Run: `head -50 core/skills/frontend-design/SKILL.md`
Expected: Frontmatter renders correctly, header present, Section 0 visible.

- [ ] **Step 5: Commit**

```bash
git add core/skills/frontend-design/SKILL.md
git commit -m "feat(frontend-design): scaffold SKILL.md with frontmatter, header, brief inference"
```

---

### Task 2: Dial System + Multi-Framework Detection + Style Presets

**Depends on:** Task 1

**Files:**
- Modify: `core/skills/frontend-design/SKILL.md` (append)

- [ ] **Step 1: Write Section 1 — The Three Dials**

Append to `SKILL.md`:
- Three dial definitions: `DESIGN_VARIANCE` (1-10), `MOTION_INTENSITY` (1-10), `VISUAL_DENSITY` (1-10)
- Baseline: `8 / 6 / 4`
- 1.A: Dial Inference table (7 rows: minimalist, premium consumer, playful/experimental, landing/portfolio default, trust-first, redesign-preserve, redesign-overhaul)
- 1.B: Use-Case Presets table (9 rows including the 3 merged style variants)
- 1.C: How the Dials Drive Output (global variable usage rule)

**Critical merge from upstream style variants:** Expand the Use-Case Presets table to include the 3 merged presets with full visual character descriptions:

| Use case | VARIANCE | MOTION | DENSITY | Visual Character |
|---|---|---|---|---|
| Landing (SaaS, mainstream) | 7 | 6 | 4 | — |
| Landing (Agency / creative) | 9 | 8 | 3 | — |
| Landing (Premium consumer) | 7 | 6 | 3 | — |
| Portfolio (Designer / studio) | 8 | 7 | 3 | — |
| Portfolio (Developer) | 6 | 5 | 4 | — |
| Editorial / Blog | 6 | 4 | 3 | — |
| Public-sector service | 3 | 2 | 5 | — |
| Redesign - preserve | match | match+1 | match | — |
| Redesign - overhaul | +2 | +2 | match | — |
| **High-end / Soft / Luxury** | **7** | **6** | **3** | Double-Bezel cards (outer shell + inner core), spring physics motion, floating island nav, haptic micro-interactions, premium font rotation |
| **Minimalist / Editorial** | **5** | **3** | **2** | Warm monochrome + muted pastels, 1px borders, massive whitespace, crisp 8-12px radius, editorial serif/sans pairing, quiet sophistication |
| **Brutalist / Industrial** | **9** | **7** | **8** | Swiss grid, aviation red accent, zero border-radius, monospace dominance, CRT scanlines, visible compartmentalization, high data density |

- [ ] **Step 2: Write Section 2 — Brief → Design System Map**

Append:
- 2.A: Real design system table (12 entries: Fluent, Material, Carbon, Polaris, Atlaskit, Primer, GOV.UK, USWDS, Bootstrap, Radix, shadcn/ui, Tailwind v4)
- 2.B: Aesthetic → honest implementation table (8 entries: Glassmorphism, Bento, Brutalism, Editorial, Dark tech, Aurora, Kinetic typography, Apple Liquid Glass with honesty disclaimer)

Include the Honesty rule and One-system-per-project rule.

- [ ] **Step 3: Write Section 3 — Default Architecture & Conventions**

**CRITICAL ADAPTATION:** Rewrite this section from upstream v2 Section 3 to use **Vite + React as default** instead of Next.js. Include multi-framework detection.

Append:
- 3.A Stack:
  - **Framework detection logic** (new — not in upstream):
    ```
    Detection signals (check in order):
    1. package.json dependencies → next / vue / svelte / solid-js
    2. Config files → vite.config.* / next.config.* / nuxt.config.* / svelte.config.*
    3. File patterns → .vue / .svelte / .solid / app/ routes/
    4. Default (no signal) → Vite + React
    ```
  - Branches: No signal → Vite+React; next → Next.js+React; vite+react → Vite+React; vue → Vue 3+Vite; svelte → SvelteKit; solid → Solid+Vite
  - Styling: Tailwind v4 (default), v3 fallback
  - Animation: Motion (`motion/react` import)
  - Fonts: Self-host with `@font-face` + `font-display: swap`. Never Google Fonts `<link>` in production.
- 3.B State: `useState` / `useReducer` / Zustand / Jotai. Motion values for continuous input (NOT `useState`).
- 3.C Icons: Priority order (Phosphor, HugeIcons, Radix, Tabler). Discouraged: Lucide. One family per project.
- 3.D Emoji Policy: Discouraged by default.
- 3.E Responsiveness: Breakpoints, max-width containment, `min-h-[100dvh]`, CSS Grid over flex-math.
- 3.F Dependency Verification: Check `package.json` before importing.

- [ ] **Step 4: Verify the Dial + Framework sections read coherently**

Run: `grep -n "DESIGN_VARIANCE\|Vite + React\|Detection signals" core/skills/frontend-design/SKILL.md`
Expected: All three concepts appear in the file.

- [ ] **Step 5: Commit**

```bash
git add core/skills/frontend-design/SKILL.md
git commit -m "feat(frontend-design): add Dial system, design system map, multi-framework detection"
```

---

### Task 3: Design Engineering Directives + AI-Tells Ban + Pre-Flight Checklist

**Depends on:** Task 2

**Files:**
- Modify: `core/skills/frontend-design/SKILL.md` (append)

- [ ] **Step 1: Write Section 4 — Design Engineering Directives**

Append the complete bias-correction section. Include all subsections from upstream v2 Section 4:
- 4.1 Typography: Display/Body defaults, sans font choices (Geist, Outfit, Cabinet Grotesk, Satoshi), **SERIF DISCIPLINE** (very discouraged as default, explicit override paths, banned defaults: Fraunces, Instrument_Serif), pairings, italic descender clearance
- 4.2 Color Calibration: Max 1 accent, LILA RULE (AI Purple ban), COLOR CONSISTENCY LOCK, **PREMIUM-CONSUMER PALETTE BAN** (banned hex families + 6 alternative families + rotation rule)
- 4.3 Layout Diversification: ANTI-CENTER BIAS
- 4.4 Materiality, Shadows, Cards: SHAPE CONSISTENCY LOCK
- 4.5 Interactive UI States: Loading/Empty/Error/Tactile, BUTTON CONTRAST CHECK, CTA BUTTON WRAP BAN, NO DUPLICATE CTA INTENT, FORM CONTRAST CHECK
- 4.6 Data & Form Patterns: Label above input, no placeholder-as-label
- 4.7 Layout Discipline (hard rules): Hero viewport fit, hero font-scale, HERO TOP PADDING CAP, HERO STACK DISCIPLINE (max 4 elements), logo wall under hero, nav single line, nav height cap, bento rhythm, BENTO CELL COUNT RULE, Section-Layout-Repetition Ban, ZIGZAG ALTERNATION CAP, EYEBROW RESTRAINT (max 1 per 3 sections), SPLIT-HEADER BAN, Bento Background Diversity, Mobile collapse explicit
- 4.8 Image & Visual Asset Strategy: Priority order (gen-tool → real web → tell user), real logos (Simple Icons / devicon / generated SVG), logo-only rule, hand-rolled illustration rules, div-based fake screenshots banned, hero needs real visual
- 4.9 Content Density: Sub-paragraph max 25 words, data table limits, list alternatives

- [ ] **Step 2: Write Sections 5-6 — Context-Aware Proactivity + Performance Guardrails**

Append:
- Section 5: Context-Aware Proactivity
  - 5.A Sticky-Stack canonical skeleton (GSAP ScrollTrigger)
  - 5.B Horizontal-Pan canonical skeleton
  - 5.C Scroll-Reveal Stagger canonical skeleton
  - 5.D Forbidden Animation Patterns
- Section 6: Performance & Accessibility Guardrails
  - 6.A Hardware Acceleration
  - 6.B Reduced Motion (mandatory)
  - 6.C Dark Mode (mandatory for consumer-facing)
  - 6.D Core Web Vitals Targets
  - 6.E DOM Cost
  - 6.F Z-Index Restraint

- [ ] **Step 3: Write Sections 7-8 — Dial Definitions + Dark Mode Protocol**

Append:
- Section 7: Dial Definitions (technical reference for the three dials)
- Section 8: Dark Mode Protocol (token strategy pick-one-stick-to-it, no specific color prescription, default mode, test both modes)

- [ ] **Step 4: Write Section 9 — AI Tells (Forbidden Patterns)**

Append the complete AI-Tells ban section from upstream v2 Section 9:
- 9.A Visual & CSS (5 rules)
- 9.B Typography (3 rules)
- 9.C Layout & Spacing (2 rules)
- 9.D Content & Data — "Jane Doe" Effect (5 rules)
- 9.E External Resources & Components (5 rules)
- 9.F Production-Test Tells (banned outright) — all subcategories: hero & top-of-page, section numbering, separators & dots, em-dashes & typography flourishes, fake product previews, marketing-copy tells, pills/labels/version stamps, decoration text strips, lists/dividers/scoring, locale/time/scroll cues
- 9.G EM-DASH BAN — the single most-violated Tell, with comprehensive replacement rules

- [ ] **Step 5: Write Section 10 — Reference Vocabulary**

Append the pattern names the agent should know (from upstream v2 Section 10).

- [ ] **Step 6: Commit**

```bash
git add core/skills/frontend-design/SKILL.md
git commit -m "feat(frontend-design): add design engineering directives, AI-Tells ban, performance guardrails"
```

---

### Task 4: Finalize SKILL.md — Pre-Flight Checklist + Cross-References + Appendices

**Depends on:** Task 3

**Files:**
- Modify: `core/skills/frontend-design/SKILL.md` (append)

- [ ] **Step 1: Write Section 11 — Redesign Protocol (brief version)**

Append a condensed redesign protocol:
- 11.A Detect the Mode (preserve vs overhaul)
- 11.B Audit Before Touching
- 11.C Preservation Rules
- 11.D Modernisation Levers (priority order)
- 11.E Decision Tree: Targeted Evolution vs Full Redesign
- 11.F What Never Changes Silently

**Note:** Keep this brief — the full redesign workflow lives in `redesign.md` appendix. This section should be a 100-line summary with a pointer: "For the full audit protocol, see `redesign.md`."

- [ ] **Step 2: Write Section 12 — Removed (Block Library)**

Append:
```markdown
## 12. BLOCK LIBRARY

> **Removed in gpowers integration.** Upstream taste-skill v2 defined a schema for a Block Library but provided no implementations. To avoid confusion, this section is omitted. Block components should be built project-specific using the design system and Dial values defined in this document.
```

- [ ] **Step 3: Write Section 13 — Final Pre-Flight Checklist**

Append the complete pre-flight checklist from upstream v2 Section 14, with these **CRITICAL MERGES** from `full-output-enforcement`:

Add these two items to the checklist (merge from full-output-enforcement):
```markdown
- [ ] **Placeholder Pattern Ban**: No `// ...`, `// TODO`, `/* ... */`, or bare `...` truncation patterns in output. Every component must be complete and runnable.
- [ ] **Output Completeness**: If token limit approaches, use the PAUSED mechanism: `[PAUSED — X of Y complete. Send "continue" to resume]`. Never truncate mid-component.
```

The full checklist should have ~45 items total (43 from upstream + 2 merged). Include the introductory text: "THIS IS NOT OPTIONAL. Run every box. If any box fails, the output is not done."

- [ ] **Step 4: Write Section 14 — Appendices (Install Commands + Canonical Sources)**

Append:
- Appendix A: Install Commands per Design System (from upstream v2 Appendix A)
- Appendix B: Canonical Sources (official doc links per design system, from upstream v2 Appendix B)
- Appendix C: Apple Liquid Glass — Honest Web Approximation (the CSS skeleton from upstream v2 Appendix C)

- [ ] **Step 5: Final cross-reference check**

Run: `grep -c "^## " core/skills/frontend-design/SKILL.md`
Expected: 14+ top-level sections (0 through 14 + appendices).

Run: `grep -n "Section [0-9]\|Appendix [ABC]" core/skills/frontend-design/SKILL.md | head -30`
Expected: All referenced sections have corresponding headers.

Run: `grep -n "Pre-Flight\|pre-flight\|TODO\|TBD\|// \.\.\.\|bare \.\.\." core/skills/frontend-design/SKILL.md`
Expected: Only the pre-flight checklist section and the placeholder-ban item reference these terms. No actual TODO or truncation patterns in the document body.

- [ ] **Step 6: Commit**

```bash
git add core/skills/frontend-design/SKILL.md
git commit -m "feat(frontend-design): add pre-flight checklist (merged with full-output-enforcement), redesign protocol, appendices"
```

---

## Self-Review

- [ ] **1. Spec coverage (build the table).**

| Spec section | Task(s) | Status |
|---|---|---|
| §1 Purpose & Scope (directory, 5 files) | Task 1 | covered |
| §2.1 High-Level Structure | Task 1 | covered |
| §2.2 Trigger Routing | Task 1 | covered |
| §3.1 Source-to-Target — SKILL.md content | Tasks 2-4 | covered |
| §3.2 Block Library removal | Task 4 | covered |
| §4.1 Vite+React default | Task 2 | covered |
| §4.2 Multi-framework detection | Task 2 | covered |
| §4.3 Style presets as Dial values | Task 2 | covered |
| §4.4 Output completeness merge | Task 4 | covered |
| §5.2 Frontmatter | Task 1 | covered |
| §8 Error Handling & Degradation | Task 2 | covered |
| §11 Done Criteria 1-2 (directory + frontmatter) | Task 1 | covered |
| §11 Done Criteria 3 (SKILL.md content) | Tasks 2-4 | covered |
| §11 Done Criteria 8 (placeholder ban) | Task 4 | covered |
| §11 Done Criteria 9 (Block Library removed) | Task 4 | covered |

- [ ] **2. Placeholder scan:** No `TODO`/`TBD`/deferred patterns. Every step shows exact content or exact source reference.
- [ ] **3. No phantom tasks:** Every task produces a verifiable file change. No `--allow-empty`. No "already done in Task N."
- [ ] **4. Dependency soundness:** Tasks 1→2→3→4 sequential. All satisfied.
- [ ] **5. Caller & build soundness:** N/A — this is a documentation task, no code signatures.
- [ ] **6. Test-the-risk:** N/A — markdown documentation, no state mutation.
- [ ] **7. Type consistency:** N/A — no types or method signatures in this sub-plan.

