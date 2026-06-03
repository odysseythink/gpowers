# taste-skill Integration — Verification + Git Commit

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Verify content integrity of all 5 skill files and commit the completed `core/skills/frontend-design/` directory.

**Architecture:** Two-phase verification — first a programmatic + manual integrity scan across all files, then a single git commit with a descriptive message.

**Tech Stack:** Shell (grep, wc), git. No compilation or test dependencies.

**Depends on file:** `taste-skill-integration-main.md` (Tasks 1-4), `taste-skill-integration-appendices.md` (Tasks 5-8)

---

## File Structure

All files verified under `core/skills/frontend-design/`:

| File | Verification Focus |
|------|-------------------|
| `SKILL.md` | Frontmatter, line count ≤1200, no truncation patterns, cross-ref to appendices |
| `gpt-taste.md` | Frontmatter, override header, no truncation patterns |
| `image-to-code.md` | Frontmatter, override header, no truncation patterns |
| `redesign.md` | Frontmatter, override header, no truncation patterns |
| `stitch.md` | Frontmatter, override header, no truncation patterns |

---

## Dependency Overview

```
Tasks 1-8 (SKILL.md + 4 appendices written)
  -> Task 9: Content integrity scan
    -> Task 10: Git commit
```

---

## Risks & Open Questions

| Risk | Assumption | Impact if Wrong |
|------|-----------|-----------------|
| Files may contain truncation artifacts from upstream | Content was rewritten, not copied, but human error possible | Scan catches it; fix inline if found |
| Line count may exceed 1200 for SKILL.md | Upstream v2 was 1206 lines; content was condensed during rewrite | Scan catches it; trim if needed |

---

### Task 9: Content Integrity Verification

**Depends on:** Tasks 1-8 (all 5 skill files must exist)

**Files:**
- Read: `core/skills/frontend-design/SKILL.md`
- Read: `core/skills/frontend-design/gpt-taste.md`
- Read: `core/skills/frontend-design/image-to-code.md`
- Read: `core/skills/frontend-design/redesign.md`
- Read: `core/skills/frontend-design/stitch.md`

- [ ] **Step 1: Scan for truncation / placeholder patterns**

Run:
```bash
grep -rn -E '(\.\.\.|// \.\.\.|/\* \.\.\. \*/|\[PAUSED|\bTODO\b|\bTBD\b|\bFIXME\b)' core/skills/frontend-design/
```
Expected: No matches. Any match is a plan failure — fix the file before continuing.

- [ ] **Step 2: Verify line counts**

Run:
```bash
wc -l core/skills/frontend-design/*.md
```
Expected: Every file ≤ 1200 lines. `SKILL.md` should be the largest but still under 1200.

- [ ] **Step 3: Verify frontmatter on all 5 files**

Run:
```bash
for f in core/skills/frontend-design/*.md; do
  echo "=== $f ==="
  head -n 6 "$f"
done
```
Expected: Each file starts with exactly:
```yaml
---
name: frontend-design
description: Unified anti-slop frontend design methodology for premium interface generation
namespace: core
upstream: taste-skill@main
---
```
No extra fields, no missing fields.

- [ ] **Step 4: Verify appendix override headers**

Run:
```bash
for f in core/skills/frontend-design/gpt-taste.md \
         core/skills/frontend-design/image-to-code.md \
         core/skills/frontend-design/redesign.md \
         core/skills/frontend-design/stitch.md; do
  echo "=== $f ==="
  head -n 10 "$f" | grep -i "appendix\|extends\|override\|SKILL.md"
done
```
Expected: Each appendix contains a header block (within first 10 lines after frontmatter) that explicitly states it extends `frontend-design/SKILL.md` and that appendix rules override the main skill when conflicts occur.

- [ ] **Step 5: Verify SKILL.md contains all required sections**

Run:
```bash
grep -n "^## " core/skills/frontend-design/SKILL.md
```
Expected sections (order may vary):
- `## Brief Inference`
- `## Dial System` (or `## The Dial`)
- `## Multi-Framework Detection`
- `## Design System Mapping`
- `## AI-Tells Ban`
- `## Pre-flight Checklist`

If any section is missing, the corresponding task from `taste-skill-integration-main.md` was not completed correctly.

- [ ] **Step 6: Verify appendix-specific content requirements**

Run these one per appendix:
```bash
# gpt-taste.md
grep -c "Python\|RNG\|AIDA\|2-line hero\|bento\|GSAP" core/skills/frontend-design/gpt-taste.md
# Expected: ≥3 matches

# image-to-code.md
grep -c "image-first\|generate-enough\|deep analysis\|variation" core/skills/frontend-design/image-to-code.md
# Expected: ≥3 matches

# redesign.md
grep -c "scan\|diagnose\|fix\|audit\|upgrade" core/skills/frontend-design/redesign.md
# Expected: ≥3 matches

# stitch.md
grep -c "DESIGN.md\|semantic\|anti-pattern\|Stitch" core/skills/frontend-design/stitch.md
# Expected: ≥3 matches
```

- [ ] **Step 7: Record verification results**

If all steps pass, create a brief verification log:
```bash
cat > core/skills/frontend-design/.verify-log << 'EOF'
Verification completed: all checks passed.
Line counts: [paste from Step 2]
No truncation patterns found.
All frontmatter valid.
All appendix headers valid.
All required sections present.
EOF
git add core/skills/frontend-design/.verify-log
```

**Note:** The `.verify-log` file is optional but useful for audit trails. Do NOT commit if any check failed — fix first.

---

### Task 10: Git Commit

**Depends on:** Task 9 (all verification checks pass)

**Files:**
- Modify: `core/skills/frontend-design/` (all 5 `.md` files + optional `.verify-log`)

- [ ] **Step 1: Stage all new skill files**

Run:
```bash
git add core/skills/frontend-design/
```

- [ ] **Step 2: Review the staged diff**

Run:
```bash
git diff --staged --stat
```
Expected: Exactly 5 files (or 6 with `.verify-log`) under `core/skills/frontend-design/`, all new (`A`). No unintended changes outside this directory.

- [ ] **Step 3: Commit with descriptive message**

Run:
```bash
git commit -m "feat(core): add frontend-design skill with taste-skill methodology

Integrate 5 battle-tested anti-slop frontend design skills from taste-skill:
- SKILL.md: universal design methodology (Dial system, multi-framework,
  AI-Tells ban, pre-flight checklist)
- gpt-taste.md: aggressive creative mode appendix
- image-to-code.md: image-first design-to-code workflow appendix
- redesign.md: existing project audit + upgrade protocol appendix
- stitch.md: Stitch-compatible DESIGN.md generation appendix

Key decisions:
- Vite+React default (was Next.js)
- 3 style variants merged into Dial Use-Case Presets
- full-output-enforcement merged into pre-flight checklist
- Block Library placeholder removed (upstream has no implementations)

Upstream: taste-skill@main"
```

- [ ] **Step 4: Verify clean working tree**

Run:
```bash
git status
```
Expected: `nothing to commit, working tree clean`

- [ ] **Step 5: Show commit summary**

Run:
```bash
git log -1 --stat
```
Expected: Commit hash + the 5 skill files listed.

---

## Self-Review

- [ ] **1. Spec coverage (build the table).**

| Spec section | Task(s) | Status |
|---|---|---|
| §1 Purpose & Scope (directory, 5 files) | Tasks 1-8 | covered |
| §2.1 High-Level Structure | Tasks 1-8 | covered |
| §2.2 Trigger Routing | Tasks 1,5-8 | covered |
| §3.1 Source-to-Target Mapping — all 5 files | Tasks 2-8 | covered |
| §3.2 Block Library removal | Task 3 | covered |
| §4.1 Vite+React default | Task 2 | covered |
| §4.2 Multi-framework detection | Task 2 | covered |
| §4.3 Style presets as Dial values | Task 2 | covered |
| §4.4 Output completeness (full-output-enforcement merge) | Task 3 | covered |
| §5.1 Skill Loading Interface | Task 1 | no-op (gpowers convention) |
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

- [ ] **2. Placeholder scan:** No `TODO`/`TBD`/deferred-by-dependency excuses in Tasks 9-10. Every step has an exact command and expected output.

- [ ] **3. No phantom tasks (binary):** Tasks 9 and 10 both produce verifiable file-system and git changes. No `--allow-empty`, no "already done in Task N" bodies.

- [ ] **4. Dependency soundness:** Task 9 depends on Tasks 1-8 (files must exist to scan). Task 10 depends on Task 9 (must pass before commit). No forward references.

- [ ] **5. Caller & build soundness:** No shared signatures changed in Tasks 9-10 — these are verification and git tasks, not code tasks. Whole-tree verification in Task 9 uses `grep` and `wc`, not `go build`.

- [ ] **6. Test-the-risk:** Task 9's Step 1 (truncation pattern scan) is the risk — it asserts content integrity via `grep` with expected "no matches". Task 9's Step 5 (section check) asserts SKILL.md completeness. Both are behavioral assertions, not just "file exists".

- [ ] **7. Type consistency:** N/A for verification + git tasks — no types, signatures, or property names are introduced in Tasks 9-10.
