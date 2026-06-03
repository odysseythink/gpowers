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

## 1. Active Baseline Configuration

- DESIGN_VARIANCE: 8
- VISUAL_DENSITY: 3
- ART_DIRECTION: 8
- IMPLEMENTATION_CLARITY: 9
- IMAGE_USAGE_PRIORITY: 9
- SPACING_GENEROSITY: 9
- ANALYSIS_PRECISION: 10
- IMAGE_GENERATION_EAGERNESS: 10
- UI_SIMPLICITY_DISCIPLINE: 9

Interpretation rules:
- "clean" → reduce density and increase clarity
- "crazy creative" → increase variance and art direction
- "premium SaaS" → keep clarity high and art direction controlled
- "editorial" → allow stronger type and more asymmetry

## 2. Mandatory Image-First Rule

For website design requests where visual quality matters, image generation is mandatory first.

This means:
1. generate the design image or image set yourself first
2. deeply inspect and analyze the generated image(s)
3. extract the design system from them
4. implement the frontend only after that

Do not:
- start with freeform coding
- skip straight to implementation
- describe a website without first generating the visual reference when generation is available

The image is the design source. The code is the translation layer.

## 3. Generate Enough Images Rule

Generate enough images to make the design truly readable and extractable.

Strong rule:
- it is better to generate too many clear images than too few compressed images
- it is better to generate one clear image per section than one unreadable board for the whole site
- it is better to create an extra detail image than to guess details later

Never reduce image count just for convenience if that harms quality.

## 4. Section-Per-Image Rule

In environments supporting image generation, prefer separate large images per section.

Default rule:
- N sections requested → generate N images (when reasonable)

This is preferred because text stays readable, typography becomes analyzable, spacing stays visible, button details stay visible, layout proportions stay visible, extraction quality becomes much better, and implementation becomes more faithful.

Do not default to one giant multi-column collage or one long compressed board with tiny unreadable text.

## 5. Do Not Crop Old Images Rule

When a section needs a dedicated image or a closer detail view, do not simply crop, cut out, zoom into, or slice it from a previously generated larger image.

Instead:
- generate a fresh new image for that section
- generate a fresh new detail image for that section
- keep the same design language, palette, typography mood, and component family
- make the new image specifically optimized for readability and extraction

Reason: cropped images often destroy spacing accuracy, type scale relationships, clean margins, layout proportions, button clarity, and overall implementation fidelity.

## 6. Fresh Re-Generation Rule

If a section or detail is not clear enough, generate it again as a new standalone image.

This standalone regeneration should preserve the same visual language, palette, typography mood, button style, radius logic, and image treatment. But it should also make text larger and more readable, make spacing more visible, make buttons easier to inspect, and make component structure easier to analyze.

This is not a different design. It is a cleaner, more analyzable section-specific render of the same design system.

## 7. Optional Detail / Extraction Image Rule

If a section image still does not expose the necessary detail clearly enough, generate an additional detail image for that same section.

Examples of useful secondary images:
- a closer hero render to read headline, subheadline, CTA, and typography
- a detail image for pricing cards
- a closer render for testimonials
- a closer render for navbar / header treatment
- a closer render for feature cards or UI panels
- a refined variation of the first generated image that makes the section more extractable
- an image focused mainly on typography and spacing instead of the full composition

These additional images exist to improve analysis and extraction quality.

## 8. Clean Analysis Standard

Analyze cleanly and systematically. Do not do vague vibe-only analysis. Do not jump too fast from image to code.

For every generated section image, inspect cleanly:
- what the section is
- what the visual priority is
- what text is readable
- what typography relationships are visible
- what spacing relationships are visible
- what buttons and controls are visible
- what card or block logic is visible
- what colors dominate
- what structural rhythm is visible
- what details are still unclear

If something is unclear, generate another image before coding.

## 9. Deep Image Analysis Requirement

Before implementing anything, deeply analyze the generated image(s). Do not just glance at them. Treat them like a design specification.

Carefully inspect and extract:
- exact visible text where readable
- hero headline wording, subheadline wording, CTA wording, section titles
- typography character, type scale relationships, font mood, line count, line wrapping behavior, alignment logic
- section spacing, internal spacing, padding and gutters
- card dimensions and rhythm, border radius logic, stroke / divider usage
- button shapes, button hierarchy, button padding, hover-implied styling if visually suggested
- color palette, accent colors, background treatment, image treatment, icon treatment
- shadows / depth logic, grid logic, layout structure
- section ordering, section density, visual rhythm, repeated motifs that define the design language

Your goal is to understand exactly why the generated website looks strong. Only after this deep analysis should you implement the frontend.

## 10. Image-First Workflow

Preferred execution order:
1. infer the section count
2. generate section reference images first
3. generate extra detail/extraction images where needed
4. if needed, regenerate unclear sections as fresh standalone images
5. deeply inspect all generated images
6. extract text, typography, spacing, colors, layout, buttons, and component logic
7. implement the website to match the generated design as closely as reasonably possible
8. only invent missing details when the images leave something ambiguous

For visually important frontend tasks, do not begin by freely designing in code. Begin by creating the visual references first whenever image generation is available.

## 11. When to Trigger Image-First

If image generation is available, strongly prefer generating image references first when the request is mainly about visual frontend quality.

Trigger image-first workflow when the user asks for:
- a beautiful hero section
- a premium landing page
- a creative website
- a redesign
- a more modern website
- a more aesthetic interface
- a polished marketing page
- a portfolio site
- a startup site where visual taste matters heavily
- a multi-section website concept
- anything described mainly in visual terms

Direct-code first is more acceptable only when:
- the task is mostly technical
- the user wants a bug fix
- the user already provides a precise design system
- the task is mainly structural rather than visual

## 12. Combinatorial Variation Engine

To avoid repetitive AI-looking output, internally choose a strong combination and commit to it consistently. Do not mash everything into chaos. Pick a coherent visual direction and execute it clearly.

### Theme Paradigm (choose 1)
1. Pristine Light Mode
2. Deep Dark Mode
3. Bold Studio Solid
4. Quiet Premium Neutral

### Background Character (choose 1)
1. subtle technical grid / dotted field
2. pure solid field with soft ambient gradient depth
3. full-bleed cinematic imagery
4. tactile textured surface feel

### Typography Character (choose 1)
1. clean grotesk
2. refined grotesk
3. expressive display
4. compressed statement typography
5. editorial serif + sans
6. Swiss rational hierarchy

### Hero Architecture (choose 1)
1. cinematic centered minimalist
2. asymmetric split hero
3. floating polaroid scatter
4. inline typography behemoth
5. editorial offset composition
6. massive image-first hero with restrained text

### Section System (choose 1)
1. modular bento rhythm
2. alternating editorial blocks
3. poster-like stacked storytelling
4. gallery-led cadence
5. Swiss grid discipline
6. asymmetric premium marketing flow

### Signature Component Set (choose exactly 4 from 11 options)
- diagonal staggered square masonry
- 3D cascading card deck
- hover-accordion slice layout
- pristine gapless bento grid
- infinite brand marquee strip
- turning polaroid arc
- vertical rhythm lines
- off-grid editorial layout
- product UI panel stack
- split testimonial quote wall
- layered image crop frames

### Motion-Implied Language (choose exactly 2 from 6 options)
- scrubbing text reveal energy
- pinned narrative section energy
- staggered float-up energy
- parallax image drift energy
- smooth accordion expansion energy
- cinematic fade-through energy

These are not coding instructions. They are visual-direction cues the design should imply.
