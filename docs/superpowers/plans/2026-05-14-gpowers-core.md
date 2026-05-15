# gpowers core/ Methodology Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Populate `core/` with the 14 methodology skills from superpowers v5.1.0, rewrite `using-superpowers` → `using-gpowers` as the new entry skill, and wire the session-start hook so that supported platforms auto-load the entry skill on each session.

**Architecture:** `core/` is the auto-triggering methodology layer. Content is sourced from `github.com/obra/superpowers@v5.1.0` (Jesse Vincent), tracked via git subtree (mechanics in Plan #9). Each skill keeps its original SKILL.md content except for two narrow edits — a `namespace: core` frontmatter field and `superpowers:` → `gpowers:` reference rewrites. The entry skill `using-gpowers` replaces `using-superpowers` and teaches agents the four-module mental model (core/roles/tools/business). The session-start hook injects the entry skill content on each Claude Code / Codex / Gemini / Cursor / OpenCode / Copilot session; Kimi gets the entry text inlined into each adapted skill (Plan #8). Hook adapts superpowers' existing platform-detection branching and Windows polyglot wrapper.

**Tech Stack:** Markdown SKILL.md files, YAML frontmatter, Bash 4+ for session-start hook, JSON for `hooks.json` registration, bats-core for unit tests, `yq` for frontmatter manipulation in tests (`brew install yq` / `apt install yq`).

**Depends on:** Plan #1 (foundation) — needs `bin/gpowers-path`, `lib/runtime-dirs.sh`, `manifest.json`, `tests/helpers/`.

---

## File Structure

```
core/
├── skills/
│   ├── using-gpowers/SKILL.md                NEW — rewrite of using-superpowers
│   ├── brainstorming/SKILL.md                COPY from superpowers + frontmatter
│   ├── writing-plans/SKILL.md                COPY + frontmatter
│   ├── executing-plans/SKILL.md              COPY + frontmatter
│   ├── subagent-driven-development/SKILL.md
│   ├── test-driven-development/SKILL.md
│   ├── systematic-debugging/SKILL.md
│   ├── verification-before-completion/SKILL.md
│   ├── requesting-code-review/SKILL.md
│   ├── receiving-code-review/SKILL.md
│   ├── finishing-a-development-branch/SKILL.md
│   ├── dispatching-parallel-agents/SKILL.md
│   ├── using-git-worktrees/SKILL.md
│   └── writing-skills/SKILL.md
├── hooks/
│   ├── session-start                          Bash hook (Unix entry)
│   ├── session-start.ps1                      PowerShell entry (Windows)
│   ├── run-hook.cmd                           Polyglot wrapper (Windows shim)
│   └── hooks.json                             Claude Code registration
├── upstream-source.json                       sha + ref of superpowers checkout
└── .gitattributes                             merge=ours for hooks/run-hook.cmd
tests/unit/core/
├── core-skill-frontmatter.bats                14 skills all have namespace + upstream
├── core-skill-references.bats                 no `superpowers:` references in body
├── using-gpowers-content.bats                 entry skill teaches the 4 modules
├── session-start-hook.bats                    hook outputs expected envelope
└── hooks-json-shape.bats                      hooks.json valid + targets session-start
tests/fixtures/core/
├── fake-superpowers-checkout/                 14 minimal stub SKILL.md files
└── expected-hook-output.txt                   golden file for session-start
```

---

## Task 1: Stage superpowers v5.1.0 checkout fixture

**Files:**
- Create: `tests/fixtures/core/fake-superpowers-checkout/skills/<name>/SKILL.md` × 14

We need a deterministic source-of-truth to test "copy + transform" without hitting the network. The fixture is a tiny stub of superpowers v5.1.0 with the 14 skill directories present and one-line content.

- [ ] **Step 1: Create fixture directory and stub skills**

```bash
mkdir -p tests/fixtures/core/fake-superpowers-checkout/skills
for name in using-superpowers brainstorming writing-plans executing-plans \
            subagent-driven-development test-driven-development \
            systematic-debugging verification-before-completion \
            requesting-code-review receiving-code-review \
            finishing-a-development-branch dispatching-parallel-agents \
            using-git-worktrees writing-skills; do
  dir="tests/fixtures/core/fake-superpowers-checkout/skills/$name"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<EOF
---
name: $name
description: stub fixture for $name
---

# $name

Original superpowers $name content. References superpowers:writing-plans elsewhere.
EOF
done
```

- [ ] **Step 2: Add upstream-source.json fixture**

```bash
cat > tests/fixtures/core/fake-superpowers-checkout/.upstream <<EOF
repo: github.com/obra/superpowers
ref: v5.1.0
sha: 0000000000000000000000000000000000000000
EOF
```

- [ ] **Step 3: Commit fixture**

```bash
git add tests/fixtures/core/
git commit -m "test(core): superpowers v5.1.0 fixture for migration tests"
```

---

## Task 2: Write the failing test for skill frontmatter

**Files:**
- Create: `tests/unit/core/core-skill-frontmatter.bats`

Each migrated skill must carry `namespace: core` and `upstream: superpowers@v5.1.0` in its YAML frontmatter, in addition to the original `name:` and `description:` fields.

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/core/core-skill-frontmatter.bats <<'EOF'
#!/usr/bin/env bats

load ../../helpers/load.bash

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  CORE_SKILLS="$GPOWERS_REPO/core/skills"
}

@test "every core skill has SKILL.md" {
  for name in using-gpowers brainstorming writing-plans executing-plans \
              subagent-driven-development test-driven-development \
              systematic-debugging verification-before-completion \
              requesting-code-review receiving-code-review \
              finishing-a-development-branch dispatching-parallel-agents \
              using-git-worktrees writing-skills; do
    [ -f "$CORE_SKILLS/$name/SKILL.md" ] || {
      echo "missing: $CORE_SKILLS/$name/SKILL.md"
      return 1
    }
  done
}

@test "every core skill has namespace: core in frontmatter" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    grep -q "^namespace: core$" "$dir/SKILL.md" || {
      echo "$name: missing 'namespace: core'"
      return 1
    }
  done
}

@test "every core skill has upstream: superpowers@v5.1.0 except using-gpowers" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    [ "$name" = "using-gpowers" ] && continue
    grep -q "^upstream: superpowers@v5\.1\.0$" "$dir/SKILL.md" || {
      echo "$name: missing 'upstream: superpowers@v5.1.0'"
      return 1
    }
  done
}

@test "using-gpowers has upstream: gpowers-native" {
  grep -q "^upstream: gpowers-native$" "$CORE_SKILLS/using-gpowers/SKILL.md"
}
EOF
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bats tests/unit/core/core-skill-frontmatter.bats`
Expected: FAIL — directories do not exist yet.

- [ ] **Step 3: Commit failing test**

```bash
git add tests/unit/core/core-skill-frontmatter.bats
git commit -m "test(core): frontmatter requirements for 14 core skills"
```

---

## Task 3: Implement skill import script

**Files:**
- Create: `bin/_gpowers-import-core.sh` (private helper, not on PATH)

This script takes a superpowers checkout (the fixture or a real subtree later) and produces `core/skills/` with frontmatter rewritten. Plan #9 reuses it after `git subtree pull`.

- [ ] **Step 1: Write the importer**

```bash
cat > bin/_gpowers-import-core.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-import-core.sh <source-superpowers-skills-dir> <dest-core-skills-dir> <upstream-tag>
# Copies skills (except using-superpowers), adds namespace + upstream frontmatter,
# rewrites `superpowers:` references to `gpowers:` in body text.
set -euo pipefail

SRC="${1:?source dir required}"
DEST="${2:?destination dir required}"
TAG="${3:?upstream tag required, e.g. superpowers@v5.1.0}"

[ -d "$SRC" ] || { echo "source missing: $SRC" >&2; exit 1; }
mkdir -p "$DEST"

for src_dir in "$SRC"/*/; do
  name=$(basename "$src_dir")
  # using-superpowers is replaced by using-gpowers (separate task)
  [ "$name" = "using-superpowers" ] && continue
  dst_dir="$DEST/$name"
  mkdir -p "$dst_dir"
  _gpowers_transform_skill "$src_dir/SKILL.md" "$dst_dir/SKILL.md" "$TAG"
done

_gpowers_transform_skill() {
  local src="$1" dst="$2" tag="$3"
  awk -v tag="$tag" '
    BEGIN { in_fm = 0; fm_done = 0; injected = 0 }
    NR == 1 && /^---$/ { in_fm = 1; print; next }
    in_fm && /^---$/ {
      # Inject namespace + upstream just before frontmatter end
      if (!injected) { print "namespace: core"; print "upstream: " tag; injected = 1 }
      in_fm = 0; fm_done = 1; print; next
    }
    in_fm { print; next }
    fm_done { gsub(/superpowers:/, "gpowers:"); print }
  ' "$src" > "$dst"
}

# Functions must be defined before use in bash; reorder above is cosmetic.
# Rewrite: invoke explicitly after defining.
EOF
```

Note: bash hoists function definitions when they appear before invocation. The above has a bug — the `for` loop calls `_gpowers_transform_skill` before its definition. Fix in next step.

- [ ] **Step 2: Fix ordering (function defined before loop)**

```bash
cat > bin/_gpowers-import-core.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-import-core.sh <source-skills-dir> <dest-skills-dir> <upstream-tag>
set -euo pipefail

_gpowers_transform_skill() {
  local src="$1" dst="$2" tag="$3"
  awk -v tag="$tag" '
    BEGIN { in_fm = 0; fm_done = 0; injected = 0 }
    NR == 1 && /^---$/ { in_fm = 1; print; next }
    in_fm && /^---$/ {
      if (!injected) { print "namespace: core"; print "upstream: " tag; injected = 1 }
      in_fm = 0; fm_done = 1; print; next
    }
    in_fm { print; next }
    fm_done { gsub(/superpowers:/, "gpowers:"); print; next }
    { print }
  ' "$src" > "$dst"
}

SRC="${1:?source dir required}"
DEST="${2:?destination dir required}"
TAG="${3:?upstream tag required}"

[ -d "$SRC" ] || { echo "source missing: $SRC" >&2; exit 1; }
mkdir -p "$DEST"

for src_dir in "$SRC"/*/; do
  name=$(basename "$src_dir")
  [ "$name" = "using-superpowers" ] && continue
  dst_dir="$DEST/$name"
  mkdir -p "$dst_dir"
  _gpowers_transform_skill "$src_dir/SKILL.md" "$dst_dir/SKILL.md" "$TAG"
done
EOF
chmod +x bin/_gpowers-import-core.sh
```

- [ ] **Step 3: Run importer against fixture**

```bash
./bin/_gpowers-import-core.sh \
  tests/fixtures/core/fake-superpowers-checkout/skills \
  core/skills \
  superpowers@v5.1.0
```

Expected: 13 directories created under `core/skills/` (all 14 minus using-superpowers).

- [ ] **Step 4: Verify frontmatter on one sample**

```bash
head -6 core/skills/brainstorming/SKILL.md
```

Expected: shows `---` / `name: brainstorming` / `description: ...` / `namespace: core` / `upstream: superpowers@v5.1.0` / `---`.

- [ ] **Step 5: Commit importer**

```bash
git add bin/_gpowers-import-core.sh core/skills/
git commit -m "feat(core): import 13 superpowers skills with namespace+upstream frontmatter"
```

---

## Task 4: Write using-gpowers entry skill

**Files:**
- Create: `core/skills/using-gpowers/SKILL.md`

This is the *one* skill we rewrite rather than copy. It replaces `using-superpowers` and teaches agents the gpowers four-module model.

- [ ] **Step 1: Write the failing content test**

```bash
cat > tests/unit/core/using-gpowers-content.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  SKILL="$BATS_TEST_DIRNAME/../../../core/skills/using-gpowers/SKILL.md"
}

@test "using-gpowers exists" {
  [ -f "$SKILL" ]
}

@test "using-gpowers names all four modules" {
  for mod in core roles tools business; do
    grep -qw "$mod" "$SKILL" || { echo "module missing: $mod"; return 1; }
  done
}

@test "using-gpowers documents dual-track triggering" {
  grep -q "auto" "$SKILL"
  grep -q "explicit\|slash" "$SKILL"
}

@test "using-gpowers teaches namespace tags" {
  for tag in "(core)" "(roles)" "(tools)" "(business)"; do
    grep -qF "$tag" "$SKILL" || { echo "tag missing: $tag"; return 1; }
  done
}

@test "using-gpowers has correct frontmatter" {
  grep -q "^namespace: core$" "$SKILL"
  grep -q "^upstream: gpowers-native$" "$SKILL"
}
EOF
```

Run: `bats tests/unit/core/using-gpowers-content.bats` — expect FAIL.

- [ ] **Step 2: Write the skill**

```bash
mkdir -p core/skills/using-gpowers
cat > core/skills/using-gpowers/SKILL.md <<'EOF'
---
name: using-gpowers
description: Entry skill — establishes the four-module model (core/roles/tools/business) and dual-track triggering. Invoked automatically at session start on supported platforms.
namespace: core
upstream: gpowers-native
---

# Using gpowers

You have gpowers — a unified methodology + role + tools + business automation distribution. There are four modules, two trigger tracks, and one naming convention you must follow.

## The four modules

- **core/** — methodology skills (TDD, debugging, planning, brainstorming, code review, etc.). Apply these automatically when they fit the task. Tag `(core)` when you reference them in replies.
- **roles/** — role-based slash commands (`/pr-review`, `/cso`, `/plan-ceo-review`, `/investigate`, ...). Do NOT invoke these yourself. **Suggest** them to the user when their input matches a role's trigger. Tag `(roles)` when you reference them.
- **tools/** — capability skills (`/ship`, `/qa`, `/canary`, `/health`, ...). Call them on demand when the task requires that capability. Tag `(tools)`.
- **business/** — optional commercial automation (`/money`, `/money-content`, ...). Only present if installed with `--with-business`. Tag `(business)`.

## Dual-track triggering

- **Auto track** — `core/` only. The session-start hook injected this skill; from here, apply core methodology skills automatically when they apply. Example: bug report → invoke systematic-debugging (core). Implementation request → invoke writing-plans (core) before coding.
- **Explicit track** — `roles/`, `tools/`, `business/`. Wait for the user to type the slash command. You may *suggest* one when a trigger phrase appears: "preparing to ship" → suggest `/pr-review` + `/cso` + `/qa` before `/ship`.

## Namespace tags in replies

When you reference a gpowers skill in user-facing text, append the module tag in parentheses so the user knows where it lives:

- "I'll use brainstorming (core) to walk this through."
- "Consider `/cso` (roles) for a security review."
- "I'll run /qa (tools) against the staging URL."
- "money-content (business) covers that workflow."

## Skill priority

When multiple skills could apply, follow this order:
1. **Process skills first** (brainstorming, systematic-debugging, executing-plans)
2. **Implementation skills next** (writing-plans, TDD)
3. **Role / tool skills only when user-invoked** or suggested with explicit user confirmation

## Reading the rest

Use the `Skill` tool (Claude Code / Codex / OpenCode), `activate_skill` (Gemini), or skill-name reference (Kimi) to load any specific skill. Skill files live under `$GPOWERS_HOME/<module>/skills/<name>/SKILL.md` — never read them by absolute path; use the platform's skill mechanism so per-platform adaptations apply.

Path queries go through `gpowers-path` (`gpowers-path config`, `gpowers-path project plans`, ...) — never concatenate `~/.gpowers/` directly in skills.
EOF
```

- [ ] **Step 3: Run test to verify pass**

Run: `bats tests/unit/core/using-gpowers-content.bats`
Expected: PASS (5 tests).

- [ ] **Step 4: Run frontmatter test again — should now cover all 14**

Run: `bats tests/unit/core/core-skill-frontmatter.bats`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add core/skills/using-gpowers/ tests/unit/core/using-gpowers-content.bats
git commit -m "feat(core): using-gpowers entry skill replacing using-superpowers"
```

---

## Task 5: Verify no stray `superpowers:` references in body text

**Files:**
- Create: `tests/unit/core/core-skill-references.bats`

The importer rewrites `superpowers:` → `gpowers:` only in body (post-frontmatter). Test that no body text contains the old prefix.

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/core/core-skill-references.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  CORE_SKILLS="$BATS_TEST_DIRNAME/../../../core/skills"
}

@test "no 'superpowers:' refs in body of core skills" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    # strip frontmatter, then grep
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "superpowers:"; then
      echo "$name body still contains 'superpowers:'"
      return 1
    fi
  done
}

@test "frontmatter 'upstream: superpowers@...' is allowed and present" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    [ "$name" = "using-gpowers" ] && continue
    head -10 "$dir/SKILL.md" | grep -q "upstream: superpowers@" || {
      echo "$name missing upstream frontmatter"; return 1
    }
  done
}
EOF
```

- [ ] **Step 2: Run test**

Run: `bats tests/unit/core/core-skill-references.bats`
Expected: PASS (importer already did the rewrite).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/core/core-skill-references.bats
git commit -m "test(core): assert no body refs to 'superpowers:' prefix"
```

---

## Task 6: Write the session-start hook (Unix)

**Files:**
- Create: `core/hooks/session-start`

Adapts superpowers' SessionStart hook. Outputs a Claude Code "additional context" envelope containing the full text of `using-gpowers/SKILL.md`, plus the standard "<EXTREMELY_IMPORTANT>" preamble. Branches on `$CLAUDE_PLATFORM` (or detection) for Cursor / Codex / Gemini / OpenCode / Copilot variants.

- [ ] **Step 1: Write the failing hook test**

```bash
cat > tests/unit/core/session-start-hook.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  HOOK="$GPOWERS_REPO/core/hooks/session-start"
  export GPOWERS_HOME="$GPOWERS_REPO"
}

@test "session-start exists and is executable" {
  [ -x "$HOOK" ]
}

@test "session-start emits using-gpowers content" {
  out=$("$HOOK" claude-code)
  echo "$out" | grep -qF "Using gpowers"
  echo "$out" | grep -qF "core/"
  echo "$out" | grep -qF "roles/"
}

@test "session-start wraps output in EXTREMELY_IMPORTANT tag for claude-code" {
  out=$("$HOOK" claude-code)
  echo "$out" | grep -qF "<EXTREMELY_IMPORTANT>"
  echo "$out" | grep -qF "</EXTREMELY_IMPORTANT>"
}

@test "session-start emits raw content for cursor (no tag wrapper)" {
  out=$("$HOOK" cursor)
  ! echo "$out" | grep -qF "<EXTREMELY_IMPORTANT>"
  echo "$out" | grep -qF "Using gpowers"
}

@test "session-start exits 0 when GPOWERS_HOME missing using-gpowers" {
  GPOWERS_HOME=/nonexistent run "$HOOK" claude-code
  [ "$status" -ne 0 ]
}

@test "session-start unknown platform defaults to raw content" {
  out=$("$HOOK" unknown-platform)
  echo "$out" | grep -qF "Using gpowers"
}
EOF
```

Run: `bats tests/unit/core/session-start-hook.bats` — expect FAIL.

- [ ] **Step 2: Implement the hook**

```bash
mkdir -p core/hooks
cat > core/hooks/session-start <<'EOF'
#!/usr/bin/env bash
# gpowers session-start hook
# Usage: session-start <platform>
# Platforms: claude-code | codex | gemini | cursor | opencode | copilot
# Emits the using-gpowers skill content with platform-appropriate framing.
set -euo pipefail

PLATFORM="${1:-claude-code}"
: "${GPOWERS_HOME:?GPOWERS_HOME must be set}"

SKILL_FILE="$GPOWERS_HOME/core/skills/using-gpowers/SKILL.md"
[ -f "$SKILL_FILE" ] || {
  echo "gpowers: using-gpowers skill not found at $SKILL_FILE" >&2
  exit 1
}

# Strip YAML frontmatter for output
content=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$SKILL_FILE")

case "$PLATFORM" in
  claude-code|codex|opencode|copilot)
    printf '<EXTREMELY_IMPORTANT>\n%s\n</EXTREMELY_IMPORTANT>\n' "$content"
    ;;
  gemini)
    # GEMINI.md injection: bare markdown, no XML tags
    printf '%s\n' "$content"
    ;;
  cursor)
    # .cursorrules style: bare markdown
    printf '%s\n' "$content"
    ;;
  *)
    # Unknown platform: emit raw content; let caller decide framing
    printf '%s\n' "$content"
    ;;
esac
EOF
chmod +x core/hooks/session-start
```

- [ ] **Step 3: Run test to verify pass**

Run: `bats tests/unit/core/session-start-hook.bats`
Expected: PASS (6 tests).

- [ ] **Step 4: Commit**

```bash
git add core/hooks/session-start tests/unit/core/session-start-hook.bats
git commit -m "feat(core): session-start hook with per-platform framing"
```

---

## Task 7: Polyglot Windows wrapper (run-hook.cmd)

**Files:**
- Create: `core/hooks/run-hook.cmd`
- Create: `core/hooks/session-start.ps1` (PowerShell mirror, for native Windows)

Superpowers uses a CMD/PowerShell polyglot pattern. We mirror it so Windows Claude Code (which spawns `.cmd`) can run the hook.

- [ ] **Step 1: Write run-hook.cmd**

```bash
cat > core/hooks/run-hook.cmd <<'EOF'
@echo off
:: gpowers polyglot wrapper: invoked by Claude Code on Windows.
:: Delegates to PowerShell sibling .ps1 with the same basename as $1.
setlocal
set HOOK_NAME=%~1
if "%HOOK_NAME%"=="" (
  echo run-hook.cmd: missing hook name argument 1>&2
  exit /b 2
)
set PLATFORM=%~2
if "%PLATFORM%"=="" set PLATFORM=claude-code
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0%HOOK_NAME%.ps1" "%PLATFORM%"
exit /b %ERRORLEVEL%
EOF
```

- [ ] **Step 2: Write session-start.ps1**

```bash
cat > core/hooks/session-start.ps1 <<'EOF'
param([string]$Platform = "claude-code")
$ErrorActionPreference = "Stop"

if (-not $env:GPOWERS_HOME) {
  Write-Error "GPOWERS_HOME must be set"
  exit 1
}

$skillFile = Join-Path $env:GPOWERS_HOME "core/skills/using-gpowers/SKILL.md"
if (-not (Test-Path $skillFile)) {
  Write-Error "gpowers: using-gpowers skill not found at $skillFile"
  exit 1
}

# Strip YAML frontmatter
$lines = Get-Content $skillFile
$fmCount = 0
$body = @()
foreach ($line in $lines) {
  if ($line -eq "---") { $fmCount++; continue }
  if ($fmCount -ge 2) { $body += $line }
}
$content = $body -join "`n"

switch ($Platform) {
  { @("claude-code","codex","opencode","copilot") -contains $_ } {
    "<EXTREMELY_IMPORTANT>`n$content`n</EXTREMELY_IMPORTANT>"
  }
  default { $content }
}
EOF
```

- [ ] **Step 3: Smoke test on Unix (PowerShell Core if available, else skip)**

```bash
if command -v pwsh >/dev/null; then
  GPOWERS_HOME="$(pwd)" pwsh core/hooks/session-start.ps1 claude-code | head -5
fi
```

Expected: prints `<EXTREMELY_IMPORTANT>` line. (Test is skipped if pwsh absent; full Windows coverage lives in `tests/platform-smoke/windows.bat`, written in Plan #11.)

- [ ] **Step 4: Commit**

```bash
git add core/hooks/run-hook.cmd core/hooks/session-start.ps1
git commit -m "feat(core): Windows polyglot wrapper + PowerShell session-start mirror"
```

---

## Task 8: Write hooks.json (Claude Code registration)

**Files:**
- Create: `core/hooks/hooks.json`

Claude Code reads this file to register the hook with its `SessionStart` event.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/core/hooks-json-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  HOOKS_JSON="$BATS_TEST_DIRNAME/../../../core/hooks/hooks.json"
}

@test "hooks.json is valid JSON" {
  jq empty < "$HOOKS_JSON"
}

@test "hooks.json registers SessionStart" {
  jq -e '.hooks[] | select(.event == "SessionStart")' < "$HOOKS_JSON" >/dev/null
}

@test "hooks.json SessionStart command points to session-start" {
  cmd=$(jq -r '.hooks[] | select(.event == "SessionStart") | .command' "$HOOKS_JSON")
  case "$cmd" in *session-start*) :;; *) echo "got: $cmd"; return 1;; esac
}

@test "hooks.json declares Windows variant via run-hook.cmd" {
  jq -e '.hooks[] | select(.event == "SessionStart") | .windows' "$HOOKS_JSON" >/dev/null
}
EOF
```

Run: `bats tests/unit/core/hooks-json-shape.bats` — expect FAIL.

- [ ] **Step 2: Write hooks.json**

```bash
cat > core/hooks/hooks.json <<'EOF'
{
  "$schema": "https://docs.anthropic.com/claude-code/hooks.schema.json",
  "hooks": [
    {
      "event": "SessionStart",
      "description": "Inject using-gpowers entry skill on each session",
      "command": "${GPOWERS_HOME}/core/hooks/session-start claude-code",
      "windows": {
        "command": "%GPOWERS_HOME%\\core\\hooks\\run-hook.cmd session-start claude-code"
      },
      "timeout_seconds": 5
    }
  ]
}
EOF
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/core/hooks-json-shape.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add core/hooks/hooks.json tests/unit/core/hooks-json-shape.bats
git commit -m "feat(core): hooks.json registering SessionStart with Windows variant"
```

---

## Task 9: Write upstream-source.json for core/

**Files:**
- Create: `core/upstream-source.json`

This records which superpowers commit `core/` was last imported from. Plan #9's `gpowers upgrade core` reads this to compute the merge base.

- [ ] **Step 1: Write the file**

```bash
cat > core/upstream-source.json <<'EOF'
{
  "module": "core",
  "upstream": {
    "repo": "github.com/obra/superpowers",
    "ref": "v5.1.0",
    "sha": "0000000000000000000000000000000000000000"
  },
  "imported_at": "2026-05-14T00:00:00Z",
  "imported_skills": [
    "brainstorming",
    "writing-plans",
    "executing-plans",
    "subagent-driven-development",
    "test-driven-development",
    "systematic-debugging",
    "verification-before-completion",
    "requesting-code-review",
    "receiving-code-review",
    "finishing-a-development-branch",
    "dispatching-parallel-agents",
    "using-git-worktrees",
    "writing-skills"
  ],
  "skipped_skills": [
    "using-superpowers"
  ],
  "local_skills": [
    "using-gpowers"
  ]
}
EOF
```

Note: `sha` is the zero placeholder until Plan #9 wires a real subtree pull. Plan #9 updates this file as part of `gpowers upgrade core`.

- [ ] **Step 2: Validate JSON**

```bash
jq empty < core/upstream-source.json
```

Expected: silent success.

- [ ] **Step 3: Commit**

```bash
git add core/upstream-source.json
git commit -m "feat(core): record superpowers v5.1.0 as upstream source"
```

---

## Task 10: Update root manifest to record core/ installation

**Files:**
- Modify: `manifest.json`

After Plan #1, `manifest.json` has stubs for all four modules. Now mark `core` as installed and record skill count.

- [ ] **Step 1: Write the failing test**

```bash
cat > tests/unit/core/manifest-records-core.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"
}

@test "manifest declares core module installed" {
  installed=$(jq -r '.modules.core.installed' < "$MANIFEST")
  [ "$installed" = "true" ]
}

@test "manifest records 14 core skills" {
  count=$(jq -r '.modules.core.skill_count' < "$MANIFEST")
  [ "$count" = "14" ]
}

@test "manifest references upstream tag" {
  jq -e '.modules.core.upstream | test("superpowers@v5\\.1\\.0")' < "$MANIFEST" >/dev/null
}
EOF
```

Run: `bats tests/unit/core/manifest-records-core.bats` — expect FAIL.

- [ ] **Step 2: Update manifest (use Plan #1's `lib/manifest.sh` helper)**

```bash
source lib/manifest.sh
gpowers_manifest_set_installed core true
gpowers_manifest_set core skill_count 14
gpowers_manifest_set core upstream '"superpowers@v5.1.0"'
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/core/manifest-records-core.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add manifest.json
git commit -m "feat(core): manifest records core module installed with 14 skills"
```

---

## Task 11: End-to-end smoke — install + session-start round trip

**Files:**
- Create: `tests/integration/core/session-start-e2e.bats`

Real install dir, real GPOWERS_HOME pointed at repo root, run session-start hook and verify output. This catches issues where the hook script and the using-gpowers content drift apart.

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/core/session-start-e2e.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$PATH"
}

@test "session-start produces non-empty output" {
  run "$GPOWERS_HOME/core/hooks/session-start" claude-code
  [ "$status" -eq 0 ]
  [ -n "$output" ]
}

@test "session-start output names the 4 modules" {
  out=$("$GPOWERS_HOME/core/hooks/session-start" claude-code)
  for mod in core roles tools business; do
    echo "$out" | grep -qw "$mod" || { echo "missing module: $mod"; return 1; }
  done
}

@test "session-start output references gpowers-path helper" {
  out=$("$GPOWERS_HOME/core/hooks/session-start" claude-code)
  echo "$out" | grep -qF "gpowers-path"
}

@test "gpowers-path home returns GPOWERS_HOME" {
  result=$(gpowers-path home)
  [ "$result" = "$GPOWERS_HOME" ]
}
EOF
```

- [ ] **Step 2: Run test**

Run: `bats tests/integration/core/session-start-e2e.bats`
Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/core/session-start-e2e.bats
git commit -m "test(core): e2e smoke for session-start + gpowers-path integration"
```

---

## Self-Review

### 1. Spec coverage (§2 of design)

| Spec requirement | Task |
|---|---|
| `core/skills/` houses 14 skills incl. using-gpowers | Tasks 3, 4 |
| `core/hooks/session-start` adapted from superpowers | Task 6 |
| `run-hook.cmd` polyglot wrapper | Task 7 |
| `hooks.json` Claude Code registration | Task 8 |
| `upstream-source.json` | Task 9 |
| Rewrite `using-superpowers` → `using-gpowers` | Task 4 |
| Frontmatter `namespace: core` + `upstream:` | Tasks 2, 3, 4 |
| Body refs `superpowers:` → `gpowers:` | Tasks 3, 5 |
| Dual-track triggering documented in entry skill | Task 4 (using-gpowers content) |
| 14 skills original content preserved | Task 3 (importer copies verbatim except frontmatter + reference rewrite) |
| Cross-platform Markdown skills | Inherent — content is Markdown |
| Kimi adapter generation | Out of scope here → Plan #8 |

### 2. Placeholder scan

No TBDs / TODOs / "implement later" in any task. Every step has a complete bash command or file content. One known stub: `core/upstream-source.json` has `sha: 0000...` — explicitly called out as filled by Plan #9.

### 3. Type / name consistency

- Function name `_gpowers_transform_skill` consistent in Tasks 3.1 and 3.2.
- Hook arg `<platform>` consistent (claude-code | codex | gemini | cursor | opencode | copilot).
- `GPOWERS_HOME` env var used in Tasks 6, 7, 8, 11 — matches Plan #1's `lib/runtime-dirs.sh` definition.
- `gpowers-path` invocation in Task 11 matches Plan #1's Task 3 API.
- `manifest_set_installed` / `manifest_set` in Task 10 match Plan #1's `lib/manifest.sh` API.

### 4. Skill list integrity

13 imported + 1 native = 14. Matches spec §2.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-core.md`.

After Plan #1 lands, this plan can run independently of Plans 3-12. Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between
**2. Inline Execution** — batched via executing-plans skill

Choose at execution time.
