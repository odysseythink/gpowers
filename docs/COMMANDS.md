# gpowers slash command index

All gpowers slash commands, generated from installed skills. Grouped by typical workflow scenario.

## Pre-coding

Use these before writing code:

- `/office-hours` — YC-style product forum (roles)
- `/plan-ceo-review` — product-perspective plan review (roles)
- `/plan-eng-review` — architecture + data-flow review (roles)
- `/plan-design-review` — design review at plan stage (roles)
- `/plan-devex-review` — devex/API plan review (roles)
- `/autoplan` — chained CEO + eng + design + devex review (roles)

## Implementation

Use during coding:

- `/investigate` — root-cause analysis with document output (roles)
- `/codex` — second-opinion via OpenAI Codex CLI (roles)
- `/devex-review` — DX walkthrough after implementation (roles)
- `/design-review` — visual review (roles, requires browser driver)
- `/design-html` — turn mockup into production HTML/CSS (roles)
- `/design-shotgun` — generate multiple AI design variants (roles)
- `/design-consultation` — full design system consultation (roles)
- `/cso` — security audit (roles)

## Pre-merge

- `/pr-review` — pre-merge PR review (roles). *Previously `/review`; that alias is forwarded with a deprecation banner until 2026-11-14.*
- `/qa` — interactive QA against a target URL (tools, browser)
- `/qa-only` — read-only QA (tools, browser)
- `/health` — code quality score (tools)
- `/simplify` — simplification review (tools)

## Ship + post-ship

- `/ship` — create PR and push (tools)
- `/land-and-deploy` — merge + deploy + verify (tools)
- `/landing-report` — ship-queue dashboard (tools)
- `/canary` — post-deploy verification (tools, browser)
- `/benchmark` — performance baseline (tools, browser)

## Memory + retrospective

- `/retro` — weekly retrospective from commit history (roles)
- `/learn` — manage project lessons-learned (roles)
- `/document-release` — sync docs after release (roles)

## Tooling + maintenance

- `/context-save`, `/context-restore` — cross-session context (tools)
- `/careful`, `/freeze`, `/guard`, `/unfreeze` — safety modes (tools)
- `/fix-the-roof` — emergency fix mode (tools)
- `/fewer-permission-prompts` — allow-list configuration (tools)
- `/setup-deploy`, `/setup-browser-cookies`, `/setup-gbrain` — one-time setup (tools)
- `/sync-gbrain` — periodic sync (tools, browser)
- `/make-pdf` — document → PDF (tools)
- `/benchmark-models` — AI model baselines (tools)
- `/open-gstack-browser` — launch managed Chromium (tools, browser)
- `/pair-agent` — cross-agent shared browser (roles, browser; degraded on non-CC)
- `/plan-tune` — adjust sensitivity / developer profile (roles)
- `/gpowers-upgrade` — self-upgrade (tools)

## Generated reference

The complete list, including each command's source skill:

<!-- gpowers:generated:begin kind=commands -->
| Slash | Module | Skill | Notes |
|---|---|---|---|
| `/autoplan` | roles | autoplan |  |
| `/benchmark` | tools | benchmark | requires browser driver |
| `/benchmark-models` | tools | benchmark-models |  |
| `/browse` | tools | browse | requires browser driver |
| `/canary` | tools | canary | requires browser driver |
| `/careful` | tools | careful |  |
| `/codex` | roles | codex |  |
| `/context-restore` | tools | context-restore |  |
| `/context-save` | tools | context-save |  |
| `/cso` | roles | cso |  |
| `/design-consultation` | roles | design-consultation |  |
| `/design-html` | roles | design-html |  |
| `/design-review` | roles | design-review | requires browser driver |
| `/design-shotgun` | roles | design-shotgun |  |
| `/devex-review` | roles | devex-review |  |
| `/document-release` | roles | document-release |  |
| `/fewer-permission-prompts` | tools | fewer-permission-prompts |  |
| `/fix-the-roof` | tools | fix-the-roof |  |
| `/freeze` | tools | freeze |  |
| `/gpowers-upgrade` | tools | gpowers-upgrade |  |
| `/guard` | tools | guard |  |
| `/health` | tools | health |  |
| `/investigate` | roles | investigate |  |
| `/land-and-deploy` | tools | land-and-deploy |  |
| `/landing-report` | tools | landing-report |  |
| `/learn` | roles | learn |  |
| `/make-pdf` | tools | make-pdf |  |
| `/office-hours` | roles | office-hours |  |
| `/open-gstack-browser` | tools | open-gstack-browser | requires browser driver |
| `/pair-agent` | roles | pair-agent |  |
| `/plan-ceo-review` | roles | plan-ceo-review |  |
| `/plan-design-review` | roles | plan-design-review |  |
| `/plan-devex-review` | roles | plan-devex-review |  |
| `/plan-eng-review` | roles | plan-eng-review |  |
| `/plan-tune` | roles | plan-tune |  |
| `/pr-review` | roles | pr-review |  |
| `/qa` | tools | qa | requires browser driver |
| `/qa-only` | tools | qa-only | requires browser driver |
| `/retro` | roles | retro |  |
| `/setup-browser-cookies` | tools | setup-browser-cookies | requires browser driver |
| `/setup-deploy` | tools | setup-deploy |  |
| `/setup-gbrain` | tools | setup-gbrain | requires browser driver |
| `/ship` | tools | ship |  |
| `/simplify` | tools | simplify |  |
| `/sync-gbrain` | tools | sync-gbrain | requires browser driver |
| `/unfreeze` | tools | unfreeze |  |
<!-- gpowers:generated:end -->
