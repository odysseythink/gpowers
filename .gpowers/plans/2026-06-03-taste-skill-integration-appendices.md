# taste-skill Integration — Appendix Files

> **Local goal:** Write 4 appendix files (`gpt-taste.md`, `image-to-code.md`, `redesign.md`, `stitch.md`) that extend `frontend-design/SKILL.md` with specialized modes.
>
> **Depends on file:** `2026-06-03-taste-skill-integration-index.md` (global plan), `taste-skill-integration-main.md` (Task 1 for directory)

---

## File Structure (this sub-plan)

| File | Created By | Responsibility | Trigger Signal |
|------|-----------|---------------|----------------|
| `core/skills/frontend-design/gpt-taste.md` | Task 5 | Aggressive Awwwards-level creative mode | `Awwwards`, `award-winning`, `very creative`, `cinematic`, `high-end showcase` |
| `core/skills/frontend-design/image-to-code.md` | Task 6 | Image-first design-to-code workflow | `implement from image`, `image to code`, `code from screenshot` |
| `core/skills/frontend-design/redesign.md` | Task 7 | Existing project audit and upgrade protocol | `redesign`, `upgrade existing`, `revamp`, `optimize existing UI` |
| `core/skills/frontend-design/stitch.md` | Task 8 | Stitch-compatible DESIGN.md generation | `Stitch`, `DESIGN.md`, `design system document` |

---

## Dependency Overview

```
Task 5: gpt-taste.md     [parallel with Tasks 6, 7, 8]
Task 6: image-to-code.md [parallel with Tasks 5, 7, 8]
Task 7: redesign.md      [parallel with Tasks 5, 6, 8]
Task 8: stitch.md        [parallel with Tasks 5, 6, 7]
```

All four tasks depend on Task 1 (directory `core/skills/frontend-design/` already exists). They are fully parallel — no cross-dependencies between appendix files.

---

## Risks & Open Questions

| Risk | Mitigation |
|------|-----------|
| `image-to-code.md` references Codex-specific behavior (section-per-image rule) | Adapt language to be environment-agnostic: "In environments supporting image generation, prefer..." |
| `stitch.md` references Google Stitch (labs.google.com/stitch) which may change | Include Stitch version note; document that DESIGN.md format is stable even if Stitch evolves |
| Appendix files may conflict with main SKILL.md on overlapping rules | Each appendix header explicitly states "Appendix wins on conflict" and only documents overrides/extensions |

---

## Tasks

### Task 5: gpt-taste.md — Aggressive Creative Mode

**Depends on:** Task 1 (directory exists)

**Files:**
- Create: `core/skills/frontend-design/gpt-taste.md`

- [ ] **Step 1: Write frontmatter and appendix header**

Create `core/skills/frontend-design/gpt-taste.md` with:

```markdown
---
name: frontend-design
description: Aggressive Awwwards-level creative mode appendix for frontend-design. Python RNG, AIDA structure, 2-line hero rule, gapless bento, mandatory GSAP.
namespace: core
upstream: taste-skill@main
---

# gpt-taste: Aggressive Creative Mode

> This appendix extends `frontend-design/SKILL.md`. General design principles, Dial system, and AI-Tells bans are defined in the main skill. This document only records rules that **override or extend** the main skill.
>
> **Trigger:** `Awwwards`, `award-winning`, `very creative`, `cinematic`, `high-end showcase`, `agency-level`
```

- [ ] **Step 2: Write Section 1 — Python-Driven True Randomization**

Append the complete Python RNG randomization section from upstream `gpt-tasteskill` Section 1:
- Simulate a Python script execution in `<design_plan>` before writing UI code
- Deterministic seed (prompt character count modulo math)
- Select: 1 Hero Architecture, 1 Typography Stack (Satoshi, Cabinet Grotesk, Outfit, Geist — NEVER Inter), 3 Unique Component Architectures, 2 Advanced GSAP Paradigms
- Forbidden from defaulting to same UI twice

- [ ] **Step 3: Write Section 2 — AIDA Structure & Spacing**

Append:
- AIDA framework: Attention (Hero), Interest (Features/Bento), Desire (GSAP Scroll/Media), Action (Footer/Pricing)
- SPACING RULE: `py-32 md:py-48` between major sections
- Premium Navigation Bar requirement (floating glass pill or minimal split nav)

- [ ] **Step 4: Write Section 3 — Hero Architecture & The 2-Line Iron Rule**

Append:
- Container Width Fix: ultra-wide containers for H1 (`max-w-5xl`, `max-w-6xl`, `w-full`)
- Line Limit: H1 MUST NEVER exceed 2-3 lines. Use `clamp(3rem, 5vw, 5.5rem)` and wider containers.
- 3 Hero Layout Options (randomly assigned): Cinematic Center, Artistic Asymmetry, Editorial Split
- Button Contrast rule
- BANNED IN HERO: floating stamp/badge icons, pill-tags under hero, raw data/stats in hero

- [ ] **Step 5: Write Section 4 — The Gapless Bento Grid**

Append:
- `grid-flow-dense` (`grid-auto-flow: dense`) mandatory on every Bento Grid
- Mathematical verification of `col-span` and `row-span` interlocking
- Card Restraint: 3-5 highly intentional cards, mix of imagery/typography/CSS effects

- [ ] **Step 6: Write Section 5 — Advanced GSAP Motion & Hover Physics**

Append:
- Mandatory GSAP (`@gsap/react`, `ScrollTrigger`) — static interfaces forbidden
- Hover Physics: `group-hover:scale-105 transition-transform duration-700 ease-out` in `overflow-hidden`
- Scroll Pinning (GSAP Split): pin title left, gallery scrolls right
- Image Scale & Fade Scroll: `scale: 0.8` → `1.0`, fade to `opacity: 0.2`
- Scrubbing Text Reveals: opacity 0.1 → 1.0 sequentially
- Card Stacking: cards overlap and stack dynamically from bottom

- [ ] **Step 7: Write Section 6 — Component Arsenal**

Append the 4 component types:
- Inline Typography Images: embed pill-shaped images inside massive headings
- Horizontal Accordions: vertical slices expanding horizontally on hover
- Infinite Marquee (Trusted Partners): smooth scrolling rows
- Feedback/Testimonial Carousel: overlapping portraits + minimalist quotes

- [ ] **Step 8: Write Section 7 — Content, Assets & Strict Bans**

Append:
- Meta-Label Ban: "SECTION 01", "QUESTION 05", "ABOUT US" — banned forever
- Image Context & Style: `picsum.photos/seed/{keyword}/1920/1080` + CSS filters (`grayscale`, `mix-blend-luminosity`, `opacity-90`, `contrast-125`)
- Creative Backgrounds: radial blurs, grainy mesh gradients, shifting dark overlays
- Horizontal Scroll Bug prevention: `<main className="overflow-x-hidden w-full max-w-full">`

- [ ] **Step 9: Write Section 8 — Mandatory Pre-Flight `<design_plan>`**

Append the 5-step pre-flight design plan:
1. Python RNG Execution (3-line mock Python output)
2. AIDA Check
3. Hero Math Verification (`max-w` class, no stamps/spam tags)
4. Bento Density Verification (zero empty spaces, `grid-flow-dense`)
5. Label Sweep & Button Check (no meta-labels, perfect contrast)

Only output UI code after this verification.

- [ ] **Step 10: Verify file structure**

Run: `wc -l core/skills/frontend-design/gpt-taste.md`
Expected: ~120-150 lines.

Run: `grep -c "^## " core/skills/frontend-design/gpt-taste.md`
Expected: 8 sections (Sections 1-8).

- [ ] **Step 11: Commit**

```bash
git add core/skills/frontend-design/gpt-taste.md
git commit -m "feat(frontend-design): add gpt-taste appendix — aggressive creative mode"
```

---

### Task 6: image-to-code.md — Image-First Workflow

**Depends on:** Task 1 (directory exists)

**Files:**
- Create: `core/skills/frontend-design/image-to-code.md`

- [ ] **Step 1: Write frontmatter and appendix header**

Create `core/skills/frontend-design/image-to-code.md` with:

```markdown
---
name: frontend-design
description: Image-first design-to-code workflow appendix for frontend-design. Generate images first, deeply analyze, then implement. Combinatorial variation engine.
namespace: core
upstream: taste-skill@main
---

# image-to-code: Image-First Design-to-Code Workflow

> This appendix extends `frontend-design/SKILL.md`. When the user provides a reference image or requests "implement from image," this appendix overrides the default workflow.
>
> **Trigger:** `implement from image`, `image to code`, `code from screenshot`, `reference image implementation`
```

- [ ] **Step 2: Write Section 1 — Active Baseline Configuration**

Append the 8 dial values from upstream `image-to-code-skill` Section 1:
- DESIGN_VARIANCE: 8
- VISUAL_DENSITY: 3
- ART_DIRECTION: 8
- IMPLEMENTATION_CLARITY: 9
- IMAGE_USAGE_PRIORITY: 9
- SPACING_GENEROUSITY: 9
- ANALYSIS_PRECISION: 10
- IMAGE_GENERATION_EAGERNESS: 10
- UI_SIMPLICITY_DISCIPLINE: 9

Include interpretation rules: "clean" → reduce density/increase clarity; "crazy creative" → increase variance/art direction; "premium SaaS" → controlled art direction.

- [ ] **Step 3: Write Section 2 — Mandatory Image-First Rule**

Append:
- Workflow: image generation first → deep analysis second → implementation third
- The image is the design source. The code is the translation layer.
- Do not start with freeform coding. Do not skip image generation when available.

- [ ] **Step 4: Write Section 3 — Generate Enough Images Rule**

Append:
- Strong rule: better too many clear images than too few compressed images
- Better one clear image per section than one unreadable board for the whole site
- Better extra detail image than guessing details later
- Never reduce image count for convenience if quality suffers

- [ ] **Step 5: Write Section 4 — Section-Per-Image Rule**

Append:
- In environments supporting image generation, prefer separate large images per section
- Default: N sections requested → generate N images (when reasonable)
- Why: readable text, analyzable typography, visible spacing, visible buttons, visible layout proportions
- Do not default to one giant multi-column collage or one long compressed board

- [ ] **Step 6: Write Section 5 — Do Not Crop Old Images Rule**

Append:
- When a section needs dedicated image or detail view, do NOT crop from previously generated larger image
- Instead: generate fresh new image for that section, preserving design language/palette/typography
- Reason: cropped images destroy spacing accuracy, type scale, clean margins, layout proportions

- [ ] **Step 7: Write Section 6 — Fresh Re-Generation Rule**

Append:
- If section/detail not clear enough, regenerate as new standalone image
- Preserve: visual language, palette, typography mood, button style, radius logic, image treatment
- Improve: larger text, more visible spacing, easier button inspection, clearer component structure

- [ ] **Step 8: Write Section 7 — Optional Detail / Extraction Image Rule**

Append:
- Generate additional detail images when first image too broad
- Examples: closer hero render, pricing card detail, testimonial detail, navbar detail, feature card detail
- Use for: readable text, clearer button states, tighter spacing analysis, card inspection, color extraction

- [ ] **Step 9: Write Section 8 — Clean Analysis Standard**

Append:
- Analyze cleanly and systematically — no vague vibe-only analysis
- For every generated section image, inspect: section purpose, visual priority, readable text, typography relationships, spacing relationships, buttons/controls, card/block logic, dominant colors, structural rhythm, unclear details
- If unclear, generate another image before coding

- [ ] **Step 10: Write Section 9 — Deep Image Analysis Requirement**

Append the complete deep analysis checklist from upstream Section 9:
- Extract: exact visible text, hero headline, subheadline, CTA wording, section titles, typography character, type scale, line count, alignment, spacing, padding, card dimensions, border radius, buttons, color palette, accent colors, background treatment, image treatment, icon treatment, shadows, grid logic, layout structure, section ordering, density, visual rhythm, repeated motifs

- [ ] **Step 11: Write Section 10 — Image-First Workflow**

Append the 8-step preferred execution order:
1. infer section count
2. generate section reference images first
3. generate extra detail/extraction images where needed
4. if needed, regenerate unclear sections as fresh standalone images
5. deeply inspect all generated images
6. extract text, typography, spacing, colors, layout, buttons, component logic
7. implement to match as closely as reasonably possible
8. only invent missing details when images leave something ambiguous

- [ ] **Step 12: Write Section 11 — When to Trigger Image-First**

Append trigger conditions:
- Trigger: beautiful hero, premium landing page, creative website, redesign, modern website, aesthetic interface, polished marketing page, portfolio site, startup site, multi-section website, visually-described tasks
- Direct-code first acceptable: technical tasks, bug fixes, precise design system provided, structural tasks

- [ ] **Step 13: Write Section 12 — Combinatorial Variation Engine**

Append the complete variation engine from upstream Section 12:
- Theme Paradigm (choose 1): Pristine Light, Deep Dark, Bold Studio Solid, Quiet Premium Neutral
- Background Character (choose 1): technical grid, pure solid + ambient gradient, full-bleed cinematic, tactile textured
- Typography Character (choose 1): clean grotesk, refined grotesk, expressive display, compressed statement, editorial serif+sans, Swiss rational
- Hero Architecture (choose 1): cinematic centered, asymmetric split, floating polaroid scatter, inline typography behemoth, editorial offset, massive image-first
- Section System (choose 1): modular bento, alternating editorial, poster-like stacked, gallery-led, Swiss grid, asymmetric premium
- Signature Component Set (choose exactly 4 from 11 options)
- Motion-Implied Language (choose exactly 2 from 6 options)

- [ ] **Step 14: Verify file structure**

Run: `wc -l core/skills/frontend-design/image-to-code.md`
Expected: ~250-300 lines.

Run: `grep -c "^## " core/skills/frontend-design/image-to-code.md`
Expected: 12 sections.

- [ ] **Step 15: Commit**

```bash
git add core/skills/frontend-design/image-to-code.md
git commit -m "feat(frontend-design): add image-to-code appendix — image-first workflow"
```

---

### Task 7: redesign.md — Redesign Audit Protocol

**Depends on:** Task 1 (directory exists)

**Files:**
- Create: `core/skills/frontend-design/redesign.md`

- [ ] **Step 1: Write frontmatter and appendix header**

Create `core/skills/frontend-design/redesign.md` with:

```markdown
---
name: frontend-design
description: Existing project audit and upgrade protocol appendix for frontend-design. Scan-diagnose-fix workflow, full design audit checklist, upgrade techniques.
namespace: core
upstream: taste-skill@main
---

# redesign: Existing Project Audit & Upgrade Protocol

> This appendix extends `frontend-design/SKILL.md`. When the user asks to redesign, upgrade, or revamp an existing project, this appendix loads in addition to the main skill.
>
> **Trigger:** `redesign`, `upgrade existing`, `revamp existing`, `optimize existing UI`, `modernize`
```

- [ ] **Step 2: Write Section 1 — How This Works**

Append the scan-diagnose-fix sequence:
1. **Scan** — Read codebase. Identify framework, styling method, current design patterns.
2. **Diagnose** — Run through audit. List every generic pattern, weak point, missing state.
3. **Fix** — Apply targeted upgrades working with existing stack. Do not rewrite from scratch.

- [ ] **Step 3: Write Section 2 — Design Audit**

Append the complete audit checklist from upstream `redesign-skill`, organized by category:
- **Typography** (9 checks): default fonts, headline presence, body width, weight variety, numbers in proportional font, letter-spacing, all-caps subheaders, orphaned words
- **Color and Surfaces** (12 checks): pure black bg, oversaturated accents, multiple accents, warm/cool gray mix, AI gradient aesthetic, generic box-shadow, flat design, even gradients, inconsistent lighting, random dark sections, empty flat sections
- **Layout** (16 checks): centered symmetry, 3 equal cards, `100vh`, flexbox math, no max-width, equal height cards, uniform radius, no overlap, symmetrical padding, dashboard sidebar, missing whitespace, misaligned buttons, misaligned feature lists, inconsistent vertical rhythm, optical alignment
- **Interactivity and States** (11 checks): no hover, no active, instant transitions, missing focus, no loading, no empty, no error, dead links, no active nav, scroll jumping, layout-property animations
- **Content** (12 checks): generic names, fake numbers, placeholder companies, AI clichés, exclamation marks, "Oops!", passive voice, identical dates, same avatars, Lorem Ipsum, Title Case headers
- **Component Patterns** (13 checks): generic card, filled+ghost button pair, pill badges, accordion FAQ, 3-card carousel, 3 pricing towers, modals, avatar circles, light/dark toggle, footer link farm
- **Iconography** (5 checks): Lucide exclusively, cliché metaphors, inconsistent stroke, missing favicon, stock team photos
- **Code Quality** (8 checks): div soup, inline styles, hardcoded pixels, missing alt text, arbitrary z-index, dead code, import hallucinations, missing meta tags
- **Strategic Omissions** (6 checks): no legal links, no back nav, no 404, no form validation, no skip link, no cookie consent

- [ ] **Step 4: Write Section 3 — Upgrade Techniques**

Append 4 upgrade categories from upstream:
- **Typography Upgrades** (3): variable font animation, outlined-to-fill transitions, text mask reveals
- **Layout Upgrades** (4): broken grid/asymmetry, whitespace maximization, parallax card stacks, split-screen scroll
- **Motion Upgrades** (4): smooth scroll with inertia, staggered entry, spring physics, scroll-driven reveals
- **Surface Upgrades** (4): true glassmorphism, spotlight borders, grain/noise overlays, colored tinted shadows

- [ ] **Step 5: Write Section 4 — Fix Priority**

Append the 7-step priority order:
1. Font swap
2. Color palette cleanup
3. Hover and active states
4. Layout and spacing
5. Replace generic components
6. Add loading, empty, and error states
7. Polish typography scale and spacing

- [ ] **Step 6: Write Section 5 — Rules**

Append:
- Work with existing tech stack. Do not migrate frameworks.
- Do not break existing functionality. Test after every change.
- Check dependency file before importing new libraries.
- Check Tailwind version (v3 vs v4) before modifying config.
- If no framework, use vanilla CSS.
- Keep changes reviewable and focused. Small targeted improvements over big rewrites.

- [ ] **Step 7: Verify file structure**

Run: `wc -l core/skills/frontend-design/redesign.md`
Expected: ~180-220 lines.

Run: `grep -c "^## " core/skills/frontend-design/redesign.md`
Expected: 5 sections.

- [ ] **Step 8: Commit**

```bash
git add core/skills/frontend-design/redesign.md
git commit -m "feat(frontend-design): add redesign appendix — audit and upgrade protocol"
```

---

### Task 8: stitch.md — Stitch-Compatible DESIGN.md

**Depends on:** Task 1 (directory exists)

**Files:**
- Create: `core/skills/frontend-design/stitch.md`

- [ ] **Step 1: Write frontmatter and appendix header**

Create `core/skills/frontend-design/stitch.md` with:

```markdown
---
name: frontend-design
description: Stitch-compatible DESIGN.md generation appendix for frontend-design. Semantic design system encoding for Google Stitch screen generation.
namespace: core
upstream: taste-skill@main
---

# stitch: Stitch-Compatible DESIGN.md Generation

> This appendix extends `frontend-design/SKILL.md`. When the user requests a DESIGN.md or mentions Stitch, this appendix provides the output format and semantic encoding rules.
>
> **Trigger:** `Stitch`, `DESIGN.md`, `generate design system document`, `semantic design spec`
```

- [ ] **Step 2: Write Section 1 — Overview**

Append:
- Purpose: Generate `DESIGN.md` files optimized for Google Stitch screen generation
- Translates anti-slop directives into Stitch's native semantic design language
- DESIGN.md serves as single source of truth for Stitch prompting
- Stitch interprets design through "Visual Descriptions" + specific values

- [ ] **Step 3: Write Section 2 — The Goal**

Append the 7 encoding goals:
1. Visual atmosphere — mood, density, design philosophy
2. Color calibration — neutrals, accents, banned patterns with hex codes
3. Typographic architecture — font stacks, scale hierarchy, anti-patterns
4. Component behaviors — buttons, cards, inputs with interaction states
5. Layout principles — grid systems, spacing philosophy, responsive strategy
6. Motion philosophy — animation engine specs, spring physics, perpetual micro-interactions
7. Anti-patterns — explicit list of banned AI design clichés

- [ ] **Step 4: Write Section 3 — Analysis & Synthesis Instructions**

Append the 9 synthesis steps from upstream `stitch-skill`:
- 3.1 Define the Atmosphere: Density, Variance, Motion scales (1-10) with evocative adjectives
- 3.2 Map the Color Palette: Descriptive Name + Hex Code + Functional Role for each color. Mandatory constraints: max 1 accent, saturation < 80%, AI Purple ban, one palette, no pure black
- 3.3 Establish Typography Rules: Display/Headlines, Body, Font Selection (Inter BANNED for premium), Serif Ban (generic serifs BANNED, distinctive modern serifs only if needed), Dashboard Constraint (sans only), High-Density Override (monospace for numbers)
- 3.4 Define the Hero Section: Inline Image Typography (signature creative technique), No Overlapping, No Filler Text, Asymmetric Structure (centered banned when variance > 4), CTA Restraint (max one primary)
- 3.5 Describe Component Stylings: Buttons (tactile push, no neon glow), Cards (only when elevation communicates hierarchy), Inputs (label above, error below), Loading States (skeletal), Empty States (composed), Error States (inline)
- 3.6 Define Layout Principles: No overlapping, centered hero banned when variance > 4, 3 equal cards banned, CSS Grid over flexbox math, max-width containment, `min-h-[100dvh]` never `h-screen`
- 3.7 Define Responsive Rules: Mobile-first collapse <768px, no horizontal scroll, `clamp()` for headlines, 44px touch targets, inline images stack below on mobile, nav collapses to clean menu, spacing reduces proportionally
- 3.8 Encode Motion Philosophy: Spring Physics default (`stiffness: 100, damping: 20`), Perpetual Micro-Interactions, Staggered Orchestration, Performance (transform/opacity only)
- 3.9 List Anti-Patterns (AI Tells): 14 explicit NEVER DO rules (emojis, Inter, generic serifs, pure black, neon glows, oversaturated accents, gradient text on large headers, custom cursors, overlapping elements, 3-column equal cards, generic names, fake numbers, AI clichés, filler UI text, broken Unsplash links, centered hero for high variance)

- [ ] **Step 5: Write Section 4 — Output Format (DESIGN.md Structure)**

Append the exact output format template from upstream:
```markdown
# Design System: [Project Title]

## 1. Visual Theme & Atmosphere
(Evocative description...)

## 2. Color Palette & Roles
- **Canvas White** (#F9FAFB) — Primary background surface
...

## 3. Typography Rules
...

## 4. Component Stylings
...

## 5. Layout Principles
...

## 6. Motion & Interaction
...

## 7. Anti-Patterns (Banned)
...
```

- [ ] **Step 6: Write Section 5 — Best Practices**

Append: Be Descriptive, Be Functional, Be Consistent, Be Precise, Be Opinionated.

- [ ] **Step 7: Write Section 6 — Tips for Success**

Append 5 tips: start with atmosphere, look for patterns, think semantically, consider hierarchy, encode the bans.

- [ ] **Step 8: Write Section 7 — Common Pitfalls to Avoid**

Append 6 pitfalls: technical jargon without translation, omitting hex codes, forgetting functional roles, vague atmosphere descriptions, ignoring anti-pattern list, defaulting to generic safe designs.

- [ ] **Step 9: Verify file structure**

Run: `wc -l core/skills/frontend-design/stitch.md`
Expected: ~180-220 lines.

Run: `grep -c "^## " core/skills/frontend-design/stitch.md`
Expected: 7 sections.

- [ ] **Step 10: Commit**

```bash
git add core/skills/frontend-design/stitch.md
git commit -m "feat(frontend-design): add stitch appendix — Stitch-compatible DESIGN.md generation"
```

---

## Self-Review

- [ ] **1. Spec coverage (build the table).**

| Spec section | Task(s) | Status |
|---|---|---|
| §3.1 Source-to-Target — gpt-taste | Task 5 | covered |
| §3.1 Source-to-Target — image-to-code | Task 6 | covered |
| §3.1 Source-to-Target — redesign | Task 7 | covered |
| §3.1 Source-to-Target — stitch | Task 8 | covered |
| §5.2 Frontmatter (appendices) | Tasks 5-8 | covered |
| §11 Done Criteria 4 (gpt-taste content) | Task 5 | covered |
| §11 Done Criteria 5 (image-to-code content) | Task 6 | covered |
| §11 Done Criteria 6 (redesign content) | Task 7 | covered |
| §11 Done Criteria 7 (stitch content) | Task 8 | covered |

- [ ] **2. Placeholder scan:** No `TODO`/`TBD`/deferred patterns. Every step shows exact content or exact upstream source reference (section number + file).
- [ ] **3. No phantom tasks:** Every task creates one complete appendix file. No `--allow-empty`. No "already done in Task N."
- [ ] **4. Dependency soundness:** Tasks 5-8 all depend on Task 1 (directory). They are parallel with each other. All dependencies satisfied by earlier tasks in the global plan.
- [ ] **5. Caller & build soundness:** N/A — documentation task, no code signatures.
- [ ] **6. Test-the-risk:** N/A — markdown documentation, no state mutation.
- [ ] **7. Type consistency:** N/A — no types or method signatures in this sub-plan.
