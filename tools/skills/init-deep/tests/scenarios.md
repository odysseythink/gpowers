# init-deep Test Scenarios

These are manual exercise scenarios. Run each by invoking `/init-deep` (or `/skill:init-deep` on Kimi) in the described context and verifying the expected outcome.

## I-A — Small Project, Root Only

**Setup:** A 5-file Python script project with no subdirectories.
**Command:** `/init-deep`
**Expect:** Only `./AGENTS.md` created. Final report notes: "skipped scoring/fan-out (project too small)".

## I-B — Medium Project, Root + Subdirs

**Setup:** This gpowers repo (`core/`, `tools/`, `roles/`).
**Command:** `/init-deep`
**Expect:** `./AGENTS.md` + `./core/AGENTS.md` + `./tools/AGENTS.md` + `./roles/AGENTS.md`. Each subdir file 30–80 lines, no parent duplication.

## I-C — Monorepo, Max-depth Auto-bump

**Setup:** A workspace with packages at depth 3. Run with `--max-depth=2`.
**Command:** `/init-deep --max-depth=2`
**Expect:** Auto-bumps depth to cover package roots. Final report notes the override.

## I-D — Create-new with Existing Content

**Setup:** Existing `AGENTS.md` with project-specific notes.
**Command:** `/init-deep --create-new`
**Expect:** Existing renamed to `AGENTS.md.bak-<timestamp>`. New file generated. Original notes preserved via `EXISTING` merge.

## I-E — Update Mode Preserves User Edits

**Setup:** User edited a previously generated `AGENTS.md` (added a custom NOTE section).
**Command:** `/init-deep` (update mode, no flags)
**Expect:** User edits not blown away — surgical edits only. Custom NOTE section remains.

## I-F — Host Without Parallel Dispatch

**Setup:** A host lacking Task/Agent subagent tools (e.g., minimal Copilot CLI).
**Command:** `/init-deep`
**Expect:** Sequential analysis completes. Final report notes: `Mode: sequential (host lacks parallel dispatch)`.

## I-G — AI-slop Scrubber

**Setup:** Inject `comprehensive / robust / leverages` into a draft `AGENTS.md`. Run Phase 4 review.
**Command:** `/init-deep` on a project where the LLM previously generated verbose output.
**Expect:** Deny-list strips AI-slop terms. Verify diff before write-back shows removals.
