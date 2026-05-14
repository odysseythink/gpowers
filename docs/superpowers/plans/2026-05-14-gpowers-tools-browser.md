# gpowers tools/ Browser-Dependent Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the 11 browser-dependent tools/ skills from gstack, refactoring every direct `mcp__claude-in-chrome__*` or `playwright` reference in skill bodies into calls to the `gpowers-browser` 9-verb shim (built in Plan #3). After this, **no skill body anywhere in gpowers** contains a literal MCP tool name or playwright command — skills know only the abstract verbs.

**Architecture:** Migration is "import + refactor". The import side is the same generic helper from Plan #4 (`_gpowers-import-tool.sh`). The refactor side is a *body-rewriter*: a Python or AWK pass that recognizes patterns like "use MCP tool `mcp__claude-in-chrome__navigate`" or "run `npx playwright …`" and replaces them with the corresponding `gpowers-browser <verb>` invocation. Because the rewrite is mechanical and pattern-based, we ship it as a separate tool (`bin/_gpowers-rewrite-browser.py`) with unit tests on the regex/AST level before running it on real skill bodies. The shim sets `GPOWERS_BROWSER_DRIVER` once at skill preamble; verbs flow through `gpowers-browser` and end at whichever driver Plan #3 selected.

**Tech Stack:** Python 3.9+ (for the rewriter — text munging is more reliable here than awk), bats-core, the 9-verb interface from Plan #3, `jq` for argument JSON construction shown in skill examples.

**Depends on:** Plan #1 (foundation), Plan #3 (driver abstraction must exist), Plan #4 (`_gpowers-import-tool.sh` and the rewrite patterns for `~/.gstack/` paths). Can be implemented in parallel with Plan #6/#7.

---

## Skill Inventory (11 browser-dependent tools)

| Slash command | Skill dir | Browser usage | Driver verbs used |
|---|---|---|---|
| `/browse` | `browse` | navigate + read | open, wait, read, close |
| `/qa` | `qa` | navigate + interact + read | open, click, type, wait, read, screenshot, close |
| `/qa-only` | `qa-only` | navigate + read only | open, read, screenshot, close |
| `/canary` | `canary` | post-deploy probe | open, wait, eval, close |
| `/benchmark` | `benchmark` | perf timing | open, eval (Performance API), close |
| `/setup-browser-cookies` | `setup-browser-cookies` | cookie import | open, cookies, close |
| `/setup-gbrain` | `setup-gbrain` | external account onboard | open, type, click, close |
| `/sync-gbrain` | `sync-gbrain` | periodic sync | open, eval, close |
| `/open-gstack-browser` | `open-gstack-browser` | launches managed Chromium | open (with persistent context) |
| `/aidesigner` | `aidesigner` | design ingest | open, screenshot, read, close |
| `/aidesigner-frontend` | `aidesigner-frontend` | end-to-end design+ship | open, click, type, screenshot, eval, close |

Total: 11 skills.

---

## File Structure

```
tools/skills/
├── browse/SKILL.md
├── qa/SKILL.md
├── qa-only/SKILL.md
├── canary/SKILL.md
├── benchmark/SKILL.md
├── setup-browser-cookies/SKILL.md
├── setup-gbrain/SKILL.md
├── sync-gbrain/SKILL.md
├── open-gstack-browser/SKILL.md          NOTE: skill keeps name (project name on disk),
│                                            but slash command may stay /open-gstack-browser
│                                            during deprecation; documented in skill body.
├── aidesigner/SKILL.md
└── aidesigner-frontend/SKILL.md
bin/
└── _gpowers-rewrite-browser.py            Pattern-based MCP/playwright → gpowers-browser rewriter
tests/fixtures/tools-browser/
├── fake-gstack-checkout/skills/<name>/SKILL.md       × 11 stubs (with MCP & playwright refs)
└── rewriter-snippets/                                 small known-input / known-output pairs
tests/unit/tools-browser/
├── rewriter-patterns.bats                            individual regex pairs
├── rewriter-roundtrip.bats                           full SKILL.md before/after
├── no-mcp-refs.bats                                  no `mcp__claude-in-chrome__` in skill bodies
├── no-playwright-refs.bats                           no literal `playwright` CLI in skill bodies
└── preamble-sets-driver.bats                          each skill sources select-driver.sh
tests/integration/tools-browser/
└── browse-skill-smoke.bats                            sources select-driver, invokes verbs (mocked)
```

---

## Task 1: Stage browser-tool fixtures with realistic MCP / playwright references

**Files:**
- Create: `tests/fixtures/tools-browser/fake-gstack-checkout/skills/<name>/SKILL.md` × 11

Each stub contains the patterns we want the rewriter to catch. Realistic enough to test all regexes, small enough to read.

- [ ] **Step 1: Generate stubs**

```bash
mkdir -p tests/fixtures/tools-browser/fake-gstack-checkout/skills

write_stub() {
  local name="$1" body="$2"
  local dir="tests/fixtures/tools-browser/fake-gstack-checkout/skills/$name"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<EOF
---
name: $name
description: stub fixture for $name
slash: /$name
---

# $name

$body
EOF
}

write_stub browse '
1. Use MCP tool `mcp__claude-in-chrome__tabs_create_mcp` with URL.
2. Call `mcp__claude-in-chrome__read_page` for the text content.
3. Close with `mcp__claude-in-chrome__tabs_close_mcp`.
On non-Claude-Code platforms, fall back: `npx playwright open https://example.com`.
'

write_stub qa '
- Navigate: mcp__claude-in-chrome__navigate
- Type: mcp__claude-in-chrome__form_input
- Click: mcp__claude-in-chrome__find then click
- Screenshot: mcp__claude-in-chrome__computer (action: screenshot)
- Console: mcp__claude-in-chrome__read_console_messages
Non-CC: use `npx playwright test --headed` with custom script.
'

write_stub qa-only 'Use mcp__claude-in-chrome__read_page and mcp__claude-in-chrome__computer for screenshots only. No interaction. Non-CC: `playwright screenshot URL out.png`.'

write_stub canary 'Post-deploy: mcp__claude-in-chrome__navigate to canary URL, then mcp__claude-in-chrome__javascript_tool to check `window.__version`. Non-CC: `npx playwright test canary.spec.js`.'

write_stub benchmark 'mcp__claude-in-chrome__navigate, then mcp__claude-in-chrome__javascript_tool with `JSON.stringify(performance.getEntriesByType("navigation"))`. Non-CC: playwright eval.'

write_stub setup-browser-cookies 'mcp__claude-in-chrome__navigate, run mcp__claude-in-chrome__javascript_tool to set document.cookie. Non-CC: `playwright … --save-storage`.'

write_stub setup-gbrain 'mcp__claude-in-chrome__navigate to gbrain.app/onboard, mcp__claude-in-chrome__form_input for email, mcp__claude-in-chrome__find + click submit. Non-CC: playwright.'

write_stub sync-gbrain 'Periodic: mcp__claude-in-chrome__navigate, mcp__claude-in-chrome__javascript_tool window.__sync(). Non-CC: playwright headless eval.'

write_stub open-gstack-browser 'Starts persistent Chromium via mcp__claude-in-chrome__tabs_create_mcp with profile dir ~/.gstack/cache/chromium-profile. Non-CC: `playwright launch --persistent-context`.'

write_stub aidesigner 'mcp__claude-in-chrome__navigate to AIDesigner, mcp__claude-in-chrome__computer screenshot, mcp__claude-in-chrome__read_page DOM. Non-CC: playwright.'

write_stub aidesigner-frontend 'Full design+ship pipeline: navigate, form_input prompt, find+click generate, computer screenshot, javascript_tool eval result, close. Non-CC: playwright equivalent.'
```

- [ ] **Step 2: Commit**

```bash
git add tests/fixtures/tools-browser/
git commit -m "test(tools-browser): 11 fixture skills with realistic MCP+playwright refs"
```

---

## Task 2: Write rewriter pattern tests

**Files:**
- Create: `tests/unit/tools-browser/rewriter-patterns.bats`
- Create: `tests/fixtures/tools-browser/rewriter-snippets/{input,expected}/<case>.md` × several

Each `<case>` is one regex transformation. Bats loads input, runs `_gpowers-rewrite-browser.py` on it, and diffs against expected.

- [ ] **Step 1: Create snippet pairs**

```bash
mkdir -p tests/fixtures/tools-browser/rewriter-snippets/{input,expected}

# Case 1: tabs_create_mcp + URL → browser.open
cat > tests/fixtures/tools-browser/rewriter-snippets/input/01-tabs_create.md <<'EOF'
Use `mcp__claude-in-chrome__tabs_create_mcp` with URL https://example.com.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/01-tabs_create.md <<'EOF'
Use `gpowers-browser open` with `{"url":"https://example.com"}` on stdin.
EOF

# Case 2: navigate → browser.open (alias)
cat > tests/fixtures/tools-browser/rewriter-snippets/input/02-navigate.md <<'EOF'
Call `mcp__claude-in-chrome__navigate` then `mcp__claude-in-chrome__read_page` for text.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/02-navigate.md <<'EOF'
Call `gpowers-browser open` then `gpowers-browser read` (mode: text).
EOF

# Case 3: form_input → browser.type
cat > tests/fixtures/tools-browser/rewriter-snippets/input/03-form_input.md <<'EOF'
mcp__claude-in-chrome__form_input fills the email field.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/03-form_input.md <<'EOF'
gpowers-browser type fills the email field.
EOF

# Case 4: find then click → browser.click
cat > tests/fixtures/tools-browser/rewriter-snippets/input/04-find-click.md <<'EOF'
Use `mcp__claude-in-chrome__find` to locate, then click it.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/04-find-click.md <<'EOF'
Use `gpowers-browser wait` (condition: selector:<css>) to locate, then `gpowers-browser click`.
EOF

# Case 5: computer screenshot → browser.screenshot
cat > tests/fixtures/tools-browser/rewriter-snippets/input/05-screenshot.md <<'EOF'
Take a `mcp__claude-in-chrome__computer` action screenshot.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/05-screenshot.md <<'EOF'
Take a `gpowers-browser screenshot`.
EOF

# Case 6: javascript_tool → browser.eval
cat > tests/fixtures/tools-browser/rewriter-snippets/input/06-eval.md <<'EOF'
Run `mcp__claude-in-chrome__javascript_tool` with `window.__version`.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/06-eval.md <<'EOF'
Run `gpowers-browser eval` with code `window.__version`.
EOF

# Case 7: read_console_messages → browser.read mode console
cat > tests/fixtures/tools-browser/rewriter-snippets/input/07-console.md <<'EOF'
Use `mcp__claude-in-chrome__read_console_messages` after the action.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/07-console.md <<'EOF'
Use `gpowers-browser read` (mode: console) after the action.
EOF

# Case 8: tabs_close_mcp → browser.close
cat > tests/fixtures/tools-browser/rewriter-snippets/input/08-close.md <<'EOF'
Close with `mcp__claude-in-chrome__tabs_close_mcp`.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/08-close.md <<'EOF'
Close with `gpowers-browser close`.
EOF

# Case 9: playwright CLI line → "use the abstract driver" comment
cat > tests/fixtures/tools-browser/rewriter-snippets/input/09-playwright-fallback.md <<'EOF'
Non-CC: `npx playwright open https://example.com`.
EOF
cat > tests/fixtures/tools-browser/rewriter-snippets/expected/09-playwright-fallback.md <<'EOF'
The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
EOF
```

- [ ] **Step 2: Write the bats test**

```bash
cat > tests/unit/tools-browser/rewriter-patterns.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  REWRITER="$GPOWERS_REPO/bin/_gpowers-rewrite-browser.py"
  FIX="$GPOWERS_REPO/tests/fixtures/tools-browser/rewriter-snippets"
}

@test "rewriter exists and is executable" {
  [ -x "$REWRITER" ]
}

@test "case 01: tabs_create_mcp + URL → browser.open" {
  out=$("$REWRITER" < "$FIX/input/01-tabs_create.md")
  diff <(echo "$out") "$FIX/expected/01-tabs_create.md"
}

@test "case 02: navigate + read_page → open + read" {
  out=$("$REWRITER" < "$FIX/input/02-navigate.md")
  diff <(echo "$out") "$FIX/expected/02-navigate.md"
}

@test "case 03: form_input → type" {
  out=$("$REWRITER" < "$FIX/input/03-form_input.md")
  diff <(echo "$out") "$FIX/expected/03-form_input.md"
}

@test "case 04: find + click → wait + click" {
  out=$("$REWRITER" < "$FIX/input/04-find-click.md")
  diff <(echo "$out") "$FIX/expected/04-find-click.md"
}

@test "case 05: computer screenshot → screenshot" {
  out=$("$REWRITER" < "$FIX/input/05-screenshot.md")
  diff <(echo "$out") "$FIX/expected/05-screenshot.md"
}

@test "case 06: javascript_tool → eval" {
  out=$("$REWRITER" < "$FIX/input/06-eval.md")
  diff <(echo "$out") "$FIX/expected/06-eval.md"
}

@test "case 07: read_console_messages → read mode console" {
  out=$("$REWRITER" < "$FIX/input/07-console.md")
  diff <(echo "$out") "$FIX/expected/07-console.md"
}

@test "case 08: tabs_close_mcp → close" {
  out=$("$REWRITER" < "$FIX/input/08-close.md")
  diff <(echo "$out") "$FIX/expected/08-close.md"
}

@test "case 09: playwright CLI line → abstract driver reference" {
  out=$("$REWRITER" < "$FIX/input/09-playwright-fallback.md")
  diff <(echo "$out") "$FIX/expected/09-playwright-fallback.md"
}
EOF
```

Run: expect FAIL — rewriter not implemented yet.

- [ ] **Step 3: Commit failing tests + fixtures**

```bash
git add tests/fixtures/tools-browser/rewriter-snippets/ tests/unit/tools-browser/rewriter-patterns.bats
git commit -m "test(tools-browser): 9 pattern-rewrite cases for browser refactor"
```

---

## Task 3: Implement `_gpowers-rewrite-browser.py`

**Files:**
- Create: `bin/_gpowers-rewrite-browser.py`

Python 3 standard library only (no third-party deps). Reads stdin, writes stdout. Applies an ordered list of regex substitutions. Order matters because some patterns subsume others (longest first).

- [ ] **Step 1: Write the rewriter**

```bash
cat > bin/_gpowers-rewrite-browser.py <<'EOF'
#!/usr/bin/env python3
"""
Rewrites browser-MCP and playwright-CLI references in a SKILL.md body to the
abstract `gpowers-browser <verb>` interface defined in tools/drivers/browser/.
Stdin → stdout. No external deps.
"""
import re
import sys

# Order matters: longest / most specific patterns first.
RULES = [
    # 1. tabs_create_mcp + bare URL in the same sentence → open
    (re.compile(
        r"`mcp__claude-in-chrome__tabs_create_mcp`\s+with\s+URL\s+(\S+?)\.",
        re.IGNORECASE),
     r'`gpowers-browser open` with `{"url":"\1"}` on stdin.'),

    # 2. navigate then read_page (combined sentence)
    (re.compile(
        r"`mcp__claude-in-chrome__navigate`\s+then\s+`mcp__claude-in-chrome__read_page`\s+for\s+(\w+)",
        re.IGNORECASE),
     r'`gpowers-browser open` then `gpowers-browser read` (mode: \1)'),

    # 3. find then click (combined sentence)
    (re.compile(
        r"`mcp__claude-in-chrome__find`\s+to\s+locate,?\s+then\s+click(\s+it)?",
        re.IGNORECASE),
     r'`gpowers-browser wait` (condition: selector:<css>) to locate, then `gpowers-browser click`'),

    # 4. computer action screenshot → screenshot
    (re.compile(
        r"`mcp__claude-in-chrome__computer`\s+action\s+screenshot",
        re.IGNORECASE),
     r'`gpowers-browser screenshot`'),

    # 5. javascript_tool with `<expr>` → eval with code `<expr>`
    (re.compile(
        r"`mcp__claude-in-chrome__javascript_tool`\s+with\s+`([^`]+)`"),
     r'`gpowers-browser eval` with code `\1`'),

    # 6. read_console_messages → read mode console
    (re.compile(
        r"`mcp__claude-in-chrome__read_console_messages`"),
     r'`gpowers-browser read` (mode: console)'),

    # 7. tabs_close_mcp → close
    (re.compile(
        r"`mcp__claude-in-chrome__tabs_close_mcp`"),
     r'`gpowers-browser close`'),

    # 8. form_input → type (no surrounding context)
    (re.compile(
        r"`mcp__claude-in-chrome__form_input`"),
     r'`gpowers-browser type`'),
    (re.compile(r"\bmcp__claude-in-chrome__form_input\b"),
     r'gpowers-browser type'),

    # 9. navigate (standalone)
    (re.compile(r"`mcp__claude-in-chrome__navigate`"),
     r'`gpowers-browser open`'),
    (re.compile(r"\bmcp__claude-in-chrome__navigate\b"),
     r'gpowers-browser open'),

    # 10. read_page (standalone)
    (re.compile(r"`mcp__claude-in-chrome__read_page`"),
     r'`gpowers-browser read`'),

    # 11. find (standalone) — rare on its own; map to wait selector
    (re.compile(r"`mcp__claude-in-chrome__find`"),
     r'`gpowers-browser wait` (condition: selector:<css>)'),

    # 12. computer (standalone)
    (re.compile(r"`mcp__claude-in-chrome__computer`"),
     r'`gpowers-browser screenshot`'),

    # 13. tabs_create_mcp (standalone)
    (re.compile(r"`mcp__claude-in-chrome__tabs_create_mcp`"),
     r'`gpowers-browser open`'),

    # 14. javascript_tool (standalone)
    (re.compile(r"`mcp__claude-in-chrome__javascript_tool`"),
     r'`gpowers-browser eval`'),

    # 15. Generic playwright fallback lines — replace whole line
    (re.compile(
        r"^(Non-CC:|On non[- ]Claude[- ]Code\b[^\n]*?:)\s+`?npx\s+playwright[^`\n]*`?\.?$",
        re.MULTILINE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
    (re.compile(
        r"^(Non-CC:|On non[- ]Claude[- ]Code\b[^\n]*?:)\s+`?playwright[^`\n]*`?\.?$",
        re.MULTILINE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
    # Fallback prefix: "fall back: <command>" → strip
    (re.compile(
        r"On non-Claude-Code platforms, fall back:\s+`npx playwright[^`]*`\.",
        re.IGNORECASE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
]


def rewrite(text: str) -> str:
    for pat, repl in RULES:
        text = pat.sub(repl, text)
    return text


def main() -> int:
    src = sys.stdin.read()
    sys.stdout.write(rewrite(src))
    if not src.endswith("\n"):
        sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
EOF
chmod +x bin/_gpowers-rewrite-browser.py
```

- [ ] **Step 2: Run the pattern test**

Run: `bats tests/unit/tools-browser/rewriter-patterns.bats`
Expected: PASS (all 10 tests including the existence check).

Note: if Case 02's expected output ("...for text.") and the rewriter produce slightly different spacing or trailing period, fix in this step — the test must pass deterministically. If diff fails on whitespace, normalize both sides with `tr -s ' '` in the test.

- [ ] **Step 3: Commit**

```bash
git add bin/_gpowers-rewrite-browser.py
git commit -m "feat(tools-browser): pattern-based MCP/playwright rewriter"
```

---

## Task 4: Roundtrip test — full SKILL.md

**Files:**
- Create: `tests/unit/tools-browser/rewriter-roundtrip.bats`

After rewriting any fixture stub, the output must contain zero MCP refs and zero literal `playwright` words (the prose `playwright-cli driver` from descriptions is allowed; we ban only the *command* references).

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/tools-browser/rewriter-roundtrip.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  REWRITER="$GPOWERS_REPO/bin/_gpowers-rewrite-browser.py"
  FIX_BASE="$GPOWERS_REPO/tests/fixtures/tools-browser/fake-gstack-checkout/skills"
}

@test "every stub produces zero mcp__ refs after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    if echo "$out" | grep -q "mcp__claude-in-chrome"; then
      echo "$name retained mcp__claude-in-chrome ref after rewrite"
      return 1
    fi
  done
}

@test "every stub produces zero literal playwright command refs after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    if echo "$out" | grep -qE '`(npx )?playwright[^`]*`'; then
      echo "$name retained playwright CLI ref after rewrite"
      return 1
    fi
  done
}

@test "every stub gains at least one gpowers-browser ref after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    echo "$out" | grep -q "gpowers-browser" || {
      echo "$name has no gpowers-browser ref after rewrite"; return 1
    }
  done
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/unit/tools-browser/rewriter-roundtrip.bats`
Expected: PASS (3 tests × 11 stubs).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/tools-browser/rewriter-roundtrip.bats
git commit -m "test(tools-browser): roundtrip — every stub free of MCP/playwright refs"
```

---

## Task 5: Migrate all 11 browser skills

**Files:**
- Create: `tools/skills/{browse,qa,qa-only,canary,benchmark,setup-browser-cookies,setup-gbrain,sync-gbrain,open-gstack-browser,aidesigner,aidesigner-frontend}/SKILL.md`

Pipeline: copy via `_gpowers-import-tool.sh` (Plan #4's importer — adds namespace+upstream, rewrites `~/.gstack/` paths and `gstack-` CLI refs) → pipe body through `_gpowers-rewrite-browser.py` → write to `tools/skills/<name>/SKILL.md`.

- [ ] **Step 1: Write a small composer**

```bash
mkdir -p tools/skills
SRC_BASE="tests/fixtures/tools-browser/fake-gstack-checkout/skills"
DST_BASE="tools/skills"

for name in browse qa qa-only canary benchmark setup-browser-cookies setup-gbrain \
            sync-gbrain open-gstack-browser aidesigner aidesigner-frontend; do
  # Step A: import (adds frontmatter, rewrites gstack paths)
  ./bin/_gpowers-import-tool.sh "$SRC_BASE/$name" "$DST_BASE/$name"
  # Step B: rewrite browser references in body
  awk 'BEGIN{fm=0} /^---$/{fm++; print; next} fm<2{print; next} {print}' \
      "$DST_BASE/$name/SKILL.md" > /tmp/skillbody.md
  # Splice frontmatter | rewritten body
  fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$DST_BASE/$name/SKILL.md")
  head -n "$fm_end" "$DST_BASE/$name/SKILL.md" > /tmp/fm.md
  tail -n +$((fm_end+1)) "$DST_BASE/$name/SKILL.md" \
    | ./bin/_gpowers-rewrite-browser.py > /tmp/body.md
  cat /tmp/fm.md /tmp/body.md > "$DST_BASE/$name/SKILL.md"
done
```

- [ ] **Step 2: Verify with failing test for global MCP ban**

```bash
cat > tests/unit/tools-browser/no-mcp-refs.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
}

@test "no mcp__claude-in-chrome refs in any tools/ skill body" {
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "mcp__claude-in-chrome"; then
      echo "$name body still contains mcp__claude-in-chrome"
      return 1
    fi
  done
}

@test "no literal '\`playwright' or '\`npx playwright' CLI commands in tools/ skill bodies" {
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -qE '`(npx +)?playwright +[a-z][^`]*`'; then
      echo "$name body still contains playwright CLI"
      return 1
    fi
  done
}
EOF

cat > tests/unit/tools-browser/no-playwright-refs.bats <<'EOF'
#!/usr/bin/env bats

@test "no literal 'npx playwright' in tools/ skill bodies" {
  REPO="$BATS_TEST_DIRNAME/../../.."
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    echo "$body" | grep -q "npx playwright" && { echo "$name leak"; return 1; }
  done
  return 0
}
EOF
```

Run: `bats tests/unit/tools-browser/no-mcp-refs.bats tests/unit/tools-browser/no-playwright-refs.bats`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add tools/skills/ tests/unit/tools-browser/no-mcp-refs.bats tests/unit/tools-browser/no-playwright-refs.bats
git commit -m "feat(tools-browser): migrate 11 browser skills via import + rewriter pipeline"
```

---

## Task 6: Add driver-selection preamble to each browser skill

**Files:**
- Modify: `tools/skills/{browse,qa,qa-only,canary,benchmark,setup-browser-cookies,setup-gbrain,sync-gbrain,open-gstack-browser,aidesigner,aidesigner-frontend}/SKILL.md`

Spec §4 says: "skill's Preamble auto-source `drivers/browser/select-driver.sh`". Each browser skill body must start (after frontmatter) with a Preamble section telling the agent to source the selector before any verb call.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/tools-browser/preamble-sets-driver.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  BROWSER_SKILLS=(browse qa qa-only canary benchmark setup-browser-cookies
                  setup-gbrain sync-gbrain open-gstack-browser aidesigner
                  aidesigner-frontend)
}

@test "each browser skill body mentions select-driver.sh" {
  for name in "${BROWSER_SKILLS[@]}"; do
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$REPO/tools/skills/$name/SKILL.md")
    echo "$body" | grep -q "select-driver.sh" || { echo "$name preamble missing select-driver"; return 1; }
  done
}

@test "each browser skill body declares browser dependency in frontmatter" {
  for name in "${BROWSER_SKILLS[@]}"; do
    grep -q "^requires-driver: browser$" "$REPO/tools/skills/$name/SKILL.md" \
      || { echo "$name missing requires-driver: browser"; return 1; }
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Patch each skill**

```bash
for name in browse qa qa-only canary benchmark setup-browser-cookies \
            setup-gbrain sync-gbrain open-gstack-browser aidesigner \
            aidesigner-frontend; do
  file="tools/skills/$name/SKILL.md"
  # Inject requires-driver: browser before closing frontmatter
  awk '
    BEGIN{c=0}
    /^---$/{ c++; if(c==2){print "requires-driver: browser"} print; next }
    {print}
  ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"

  # Append preamble block right after frontmatter (after the second ---)
  awk -v p='
## Preamble (auto)

Before any browser verb call, source the driver selector:

```bash
source "$GPOWERS_HOME/tools/drivers/browser/select-driver.sh"
```

This exports `GPOWERS_BROWSER_DRIVER`. All browser interactions in this skill use `gpowers-browser <verb>` and never reference a specific MCP server or CLI tool by name.
' '
    BEGIN{c=0}
    /^---$/{ c++; print; if(c==2){print p} next }
    {print}
  ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
```

Note: the awk above embeds a literal preamble. Verify on one file before mass-running. If awk multi-line `-v` is finicky in your bash, write a `_inject-preamble.sh` helper script.

- [ ] **Step 3: Run preamble test**

Run: `bats tests/unit/tools-browser/preamble-sets-driver.bats`
Expected: PASS (2 tests × 11 skills).

- [ ] **Step 4: Commit**

```bash
git add tools/skills/
git commit -m "feat(tools-browser): inject driver-selection preamble in 11 browser skills"
```

---

## Task 7: Update tools/upstream-source.json

**Files:**
- Modify: `tools/upstream-source.json` (created by Plan #4)

Plan #4 left a placeholder `"browser_dependent": ["__pending_plan_5__"]`. Fill it now.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/tools-browser/upstream-source-recorded.bats <<'EOF'
#!/usr/bin/env bats

setup() { US="$BATS_TEST_DIRNAME/../../../tools/upstream-source.json"; }

@test "upstream-source lists 11 browser-dependent skills" {
  count=$(jq -r '.submodules.browser_dependent | length' < "$US")
  [ "$count" = "11" ]
}

@test "upstream-source has no pending sentinel" {
  ! jq -e '.submodules.browser_dependent | index("__pending_plan_5__")' < "$US" >/dev/null
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Update the file**

```bash
jq '.submodules.browser_dependent = [
  "browse","qa","qa-only","canary","benchmark","setup-browser-cookies",
  "setup-gbrain","sync-gbrain","open-gstack-browser","aidesigner","aidesigner-frontend"
]' tools/upstream-source.json > tools/upstream-source.json.tmp \
  && mv tools/upstream-source.json.tmp tools/upstream-source.json
```

- [ ] **Step 3: Run test + update manifest**

Run: `bats tests/unit/tools-browser/upstream-source-recorded.bats`
Expected: PASS.

```bash
source lib/manifest.sh
gpowers_manifest_set tools skill_count_browser 11
gpowers_manifest_set tools skill_count_total 28
```

- [ ] **Step 4: Commit**

```bash
git add tools/upstream-source.json manifest.json tests/unit/tools-browser/upstream-source-recorded.bats
git commit -m "feat(tools-browser): record 11 browser skills in upstream-source.json + manifest"
```

---

## Task 8: End-to-end smoke (mocked browser driver)

**Files:**
- Create: `tests/integration/tools-browser/browse-skill-smoke.bats`

Source select-driver in mock mode, invoke browse's recommended verb sequence, assert successful JSON pass-through.

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/tools-browser/browse-skill-smoke.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
  export GPOWERS_BROWSER_MOCK=1
  source "$GPOWERS_REPO/tools/drivers/browser/select-driver.sh"
}

@test "select-driver export GPOWERS_BROWSER_DRIVER" {
  [ -n "$GPOWERS_BROWSER_DRIVER" ]
  [ "$GPOWERS_BROWSER_DRIVER" != "missing" ]
}

@test "browse skill flow: open → read → close (mocked)" {
  open_out=$(echo '{"url":"https://example.com"}' | gpowers-browser open)
  tab=$(echo "$open_out" | jq -r .tab_id)
  [ -n "$tab" ]

  read_out=$(echo "{\"tab_id\":\"$tab\",\"mode\":\"text\"}" | gpowers-browser read)
  echo "$read_out" | jq -e '.content' >/dev/null

  close_out=$(echo "{\"tab_id\":\"$tab\"}" | gpowers-browser close)
  [ "$(echo "$close_out" | jq -r .ok)" = "true" ]
}

@test "qa skill verbs all available via dispatcher" {
  for verb in open click type wait read screenshot eval cookies close; do
    run bash -c "echo '{}' | gpowers-browser $verb 2>&1 || true"
    # We don't assert success (no real tab), only that the dispatcher accepted the verb
    if echo "$output" | grep -q "unknown verb"; then
      echo "verb rejected: $verb"; return 1
    fi
  done
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/integration/tools-browser/browse-skill-smoke.bats`
Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/tools-browser/browse-skill-smoke.bats
git commit -m "test(tools-browser): e2e smoke for browse skill through driver abstraction"
```

---

## Self-Review

### 1. Spec coverage (§4 browser-dependent subset + driver contract)

| Spec requirement | Task |
|---|---|
| Migrate browse, qa, qa-only, canary, benchmark | Task 5 |
| Migrate setup-browser-cookies, setup-gbrain, sync-gbrain | Task 5 |
| Migrate open-gstack-browser, aidesigner, aidesigner-frontend | Task 5 |
| skill body forbids `mcp__claude-in-chrome__*` | Tasks 3, 5 |
| skill body forbids literal `playwright` CLI | Tasks 3, 5 |
| skill Preamble sources `select-driver.sh` | Task 6 |
| `requires-driver: browser` frontmatter | Task 6 |
| upstream-source records browser skills | Task 7 |

### 2. Placeholder scan

- The Step 2 awk inside Task 6 is complex and known-finicky. Mitigation note in the task itself ("verify on one file before mass-running"). No silent TODOs.
- `open-gstack-browser` name retained as-is — explicitly documented; deprecation/rename discussion belongs in docs (Plan #12), not here.

### 3. Type / name consistency

- 9 verb names match Plan #3's interface.md.
- `GPOWERS_BROWSER_MOCK=1` env var matches Plan #3 Task 7.
- `tools/upstream-source.json` shape extends Plan #4 Task 10 without breaking it.

### 4. Decomposition

8 tasks. Rewriter is its own task with its own tests (Tasks 2, 3, 4). Migration is one task (Task 5). Preamble injection is separated (Task 6) so reviewers can audit it independently.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-tools-browser.md`. Depends on Plans #1, #3, #4. Choose subagent-driven or inline at execution time.
