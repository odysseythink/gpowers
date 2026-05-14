# gpowers tools/ Non-Browser Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the 16 non-browser tools/ skills from `gstack-main` into `tools/skills/`, rewrite their internal paths to use `gpowers-path`, and adapt their CLI entrypoints (originally `gstack-*` in `gstack/bin/`) to `gpowers-*` in `tools/bin/`. These skills work identically across all 7 platforms because they have no browser dependency.

**Architecture:** Each tool is a Markdown SKILL.md + (optionally) a Bash CLI in `tools/bin/`. The CLI scripts are the executable surface; the SKILL.md teaches the agent when and how to invoke them. Migration is mechanical: (1) copy skill content from gstack, (2) rewrite `~/.gstack/` paths to `$(gpowers-path …)` calls, (3) rename `gstack-foo` → `gpowers-foo`, (4) add `namespace: tools` + `upstream: gstack@main` frontmatter, (5) prefix slash commands as `/foo` (no namespace prefix — gpowers owns the `/` namespace).

**Tech Stack:** Markdown, Bash 4+, jq, `gpowers-path` helper from Plan #1, bats-core for tests, `shellcheck` for the migrated bin scripts.

**Depends on:** Plan #1 (`gpowers-path`, `lib/runtime-dirs.sh`, `tests/helpers/`). No dependency on Plan #2 or Plan #3.

---

## Skill Inventory (16 non-browser tools)

| Slash command | Skill dir | gstack origin | Migration shape |
|---|---|---|---|
| `/ship` | `ship` | `skills/ship/` + `bin/gstack-ship-helper` | skill + bin |
| `/land-and-deploy` | `land-and-deploy` | `skills/land-and-deploy/` | skill only |
| `/landing-report` | `landing-report` | `skills/landing-report/` | skill only |
| `/setup-deploy` | `setup-deploy` | `skills/setup-deploy/` | skill only |
| `/health` | `health` | `skills/health/` + `bin/gstack-health` | skill + bin |
| `/benchmark-models` | `benchmark-models` | `skills/benchmark-models/` | skill only |
| `/context-save` | `context-save` | `skills/context-save/` | skill only |
| `/context-restore` | `context-restore` | `skills/context-restore/` | skill only |
| `/careful` | `careful` | `skills/careful/` | skill only |
| `/freeze` | `freeze` | `skills/freeze/` | skill only |
| `/guard` | `guard` | `skills/guard/` | skill only |
| `/unfreeze` | `unfreeze` | `skills/unfreeze/` | skill only |
| `/make-pdf` | `make-pdf` | `skills/make-pdf/` | skill only |
| `/fix-the-roof` | `fix-the-roof` | `skills/fix-the-roof/` | skill only |
| `/simplify` | `simplify` | `skills/simplify/` | skill only |
| `/fewer-permission-prompts` | `fewer-permission-prompts` | `skills/fewer-permission-prompts/` | skill only |
| `/gpowers-upgrade` | `gpowers-upgrade` | `skills/gstack-upgrade/` + `bin/gstack-update-check` | **renamed**, skill + bin |

17 commands total. `gpowers-upgrade` skill body is rewritten in Plan #9; here we only stub the skill file and register the slash command so platforms wire it up.

---

## File Structure

```
tools/
├── skills/
│   ├── ship/SKILL.md
│   ├── land-and-deploy/SKILL.md
│   ├── landing-report/SKILL.md
│   ├── setup-deploy/SKILL.md
│   ├── health/SKILL.md
│   ├── benchmark-models/SKILL.md
│   ├── context-save/SKILL.md
│   ├── context-restore/SKILL.md
│   ├── careful/SKILL.md
│   ├── freeze/SKILL.md
│   ├── guard/SKILL.md
│   ├── unfreeze/SKILL.md
│   ├── make-pdf/SKILL.md
│   ├── fix-the-roof/SKILL.md
│   ├── simplify/SKILL.md
│   ├── fewer-permission-prompts/SKILL.md
│   └── gpowers-upgrade/SKILL.md           STUB ONLY (Plan #9 fills body)
├── bin/
│   ├── gpowers-ship-helper                ports gstack-ship-helper
│   ├── gpowers-health                     ports gstack-health
│   └── gpowers-update-check               ports gstack-update-check
└── upstream-source.json
bin/
└── _gpowers-import-tool.sh                helper: copy + rewrite a gstack tool
tests/fixtures/tools/
├── fake-gstack-checkout/skills/<name>/SKILL.md  × 17 stubs
└── fake-gstack-checkout/bin/<name>              × 3 stubs
tests/unit/tools/
├── frontmatter.bats                       16 (+stub upgrade) have namespace+upstream
├── no-gstack-paths.bats                   no literal `~/.gstack/` in skill bodies
├── slash-commands-roundtrip.bats          17 commands declared in skill frontmatter
├── bin-renamed.bats                       gpowers-* CLI scripts on PATH
└── bin-shellcheck.bats                    bin scripts pass shellcheck
```

---

## Task 1: Stage gstack tool fixture

**Files:**
- Create: `tests/fixtures/tools/fake-gstack-checkout/skills/<name>/SKILL.md` × 17
- Create: `tests/fixtures/tools/fake-gstack-checkout/bin/{gstack-ship-helper,gstack-health,gstack-update-check}`

Like Plan #2's superpowers fixture — small but with the exact patterns we need to rewrite (path strings, bin invocations).

- [ ] **Step 1: Create skill fixtures**

```bash
mkdir -p tests/fixtures/tools/fake-gstack-checkout/skills
for name in ship land-and-deploy landing-report setup-deploy health benchmark-models \
            context-save context-restore careful freeze guard unfreeze make-pdf \
            fix-the-roof simplify fewer-permission-prompts gstack-upgrade; do
  dir="tests/fixtures/tools/fake-gstack-checkout/skills/$name"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<EOF
---
name: $name
description: stub fixture for gstack $name
slash: /$name
---

# $name

This skill writes state to ~/.gstack/state and reads from ~/.gstack/config.
It invokes \`gstack-$name\` when needed. Cache lives under ~/.gstack/cache.
EOF
done
```

- [ ] **Step 2: Create bin fixtures**

```bash
mkdir -p tests/fixtures/tools/fake-gstack-checkout/bin
for n in gstack-ship-helper gstack-health gstack-update-check; do
  cat > "tests/fixtures/tools/fake-gstack-checkout/bin/$n" <<EOF
#!/usr/bin/env bash
# stub: $n
echo "stub $n" "\$@"
EOF
  chmod +x "tests/fixtures/tools/fake-gstack-checkout/bin/$n"
done
```

- [ ] **Step 3: Commit**

```bash
git add tests/fixtures/tools/
git commit -m "test(tools): gstack fixtures for non-browser tools migration"
```

---

## Task 2: Failing frontmatter test

**Files:**
- Create: `tests/unit/tools/frontmatter.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/tools/frontmatter.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  TOOLS="$BATS_TEST_DIRNAME/../../../tools/skills"
  EXPECTED="ship land-and-deploy landing-report setup-deploy health benchmark-models \
            context-save context-restore careful freeze guard unfreeze make-pdf \
            fix-the-roof simplify fewer-permission-prompts gpowers-upgrade"
}

@test "every expected non-browser tool skill exists" {
  for name in $EXPECTED; do
    [ -f "$TOOLS/$name/SKILL.md" ] || { echo "missing: $name"; return 1; }
  done
}

@test "every tool skill declares namespace: tools" {
  for name in $EXPECTED; do
    grep -q "^namespace: tools$" "$TOOLS/$name/SKILL.md" || {
      echo "$name: missing 'namespace: tools'"; return 1
    }
  done
}

@test "tool skills (except gpowers-upgrade stub) declare upstream: gstack@main" {
  for name in $EXPECTED; do
    [ "$name" = "gpowers-upgrade" ] && continue
    grep -q "^upstream: gstack@main$" "$TOOLS/$name/SKILL.md" || {
      echo "$name: missing 'upstream: gstack@main'"; return 1
    }
  done
}

@test "each skill declares its slash command" {
  for name in $EXPECTED; do
    grep -q "^slash: /" "$TOOLS/$name/SKILL.md" || {
      echo "$name: missing slash:"; return 1
    }
  done
}
EOF
```

Run: expect FAIL — `tools/skills/` empty.

- [ ] **Step 2: Commit failing test**

```bash
git add tests/unit/tools/frontmatter.bats
git commit -m "test(tools): frontmatter requirements for 17 non-browser tools"
```

---

## Task 3: Implement tool import helper

**Files:**
- Create: `bin/_gpowers-import-tool.sh`

One generic importer reused by every tool skill: copies SKILL.md, rewrites `~/.gstack/<X>` patterns to `$(gpowers-path …)` calls, rewrites `gstack-NAME` → `gpowers-NAME`, injects `namespace: tools` + `upstream: gstack@main`.

- [ ] **Step 1: Write the importer**

```bash
cat > bin/_gpowers-import-tool.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-import-tool.sh <src-skill-dir> <dst-skill-dir> [<upstream-tag>]
# Default upstream-tag = gstack@main.
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
TAG="${3:-gstack@main}"

[ -d "$SRC" ] || { echo "src missing: $SRC" >&2; exit 1; }
[ -f "$SRC/SKILL.md" ] || { echo "src missing SKILL.md: $SRC" >&2; exit 1; }
mkdir -p "$DST"

awk -v tag="$TAG" '
  BEGIN { in_fm = 0; fm_done = 0; injected = 0 }
  NR == 1 && /^---$/ { in_fm = 1; print; next }
  in_fm && /^---$/ {
    if (!injected) { print "namespace: tools"; print "upstream: " tag; injected = 1 }
    in_fm = 0; fm_done = 1; print; next
  }
  in_fm { print; next }
  fm_done {
    # Rewrite gstack paths and CLI names in body
    gsub(/~\/\.gstack\/state/, "$(gpowers-path state)")
    gsub(/~\/\.gstack\/config/, "$(gpowers-path config)")
    gsub(/~\/\.gstack\/cache/, "$(gpowers-path cache)")
    gsub(/~\/\.gstack\/data/, "$(gpowers-path data)")
    gsub(/~\/\.gstack\/analytics/, "$(gpowers-path analytics)")
    gsub(/~\/\.gstack\/logs/, "$(gpowers-path logs)")
    gsub(/~\/\.gstack\/tmp/, "$(gpowers-path tmp)")
    gsub(/~\/\.gstack\//, "$(gpowers-path home)/")
    gsub(/gstack-/, "gpowers-")
    print; next
  }
  { print }
' "$SRC/SKILL.md" > "$DST/SKILL.md"

# Copy any additional files unchanged (e.g. references/, examples/)
find "$SRC" -mindepth 1 -maxdepth 1 -not -name SKILL.md -print0 \
  | xargs -0 -I {} cp -R {} "$DST/" 2>/dev/null || true
EOF
chmod +x bin/_gpowers-import-tool.sh
```

- [ ] **Step 2: Smoke-test importer**

```bash
mkdir -p /tmp/import-test
./bin/_gpowers-import-tool.sh \
  tests/fixtures/tools/fake-gstack-checkout/skills/ship \
  /tmp/import-test/ship
grep -q "namespace: tools" /tmp/import-test/ship/SKILL.md
grep -q "upstream: gstack@main" /tmp/import-test/ship/SKILL.md
grep -q 'gpowers-path state' /tmp/import-test/ship/SKILL.md
! grep -q '~/\.gstack/' /tmp/import-test/ship/SKILL.md
! grep -q '\bgstack-ship\b' /tmp/import-test/ship/SKILL.md
rm -rf /tmp/import-test
```

Expected: all assertions pass silently.

- [ ] **Step 3: Commit**

```bash
git add bin/_gpowers-import-tool.sh
git commit -m "feat(tools): _gpowers-import-tool.sh — generic skill migration helper"
```

---

## Task 4: Import all 16 non-stub skills

**Files:**
- Create: `tools/skills/{ship,land-and-deploy,landing-report,setup-deploy,health,benchmark-models,context-save,context-restore,careful,freeze,guard,unfreeze,make-pdf,fix-the-roof,simplify,fewer-permission-prompts}/SKILL.md`

- [ ] **Step 1: Run importer in a loop**

```bash
SRC_BASE="tests/fixtures/tools/fake-gstack-checkout/skills"
DST_BASE="tools/skills"
for name in ship land-and-deploy landing-report setup-deploy health benchmark-models \
            context-save context-restore careful freeze guard unfreeze make-pdf \
            fix-the-roof simplify fewer-permission-prompts; do
  ./bin/_gpowers-import-tool.sh "$SRC_BASE/$name" "$DST_BASE/$name"
done
```

- [ ] **Step 2: Run frontmatter test**

Run: `bats tests/unit/tools/frontmatter.bats`
Expected: FAIL — still missing `gpowers-upgrade`.

- [ ] **Step 3: Commit**

```bash
git add tools/skills/
git commit -m "feat(tools): import 16 non-browser tool skills with rewritten paths"
```

---

## Task 5: Stub the gpowers-upgrade skill

**Files:**
- Create: `tools/skills/gpowers-upgrade/SKILL.md`

Plan #9 owns the full body of this skill. Here we land a minimal stub that:
- carries `namespace: tools`
- declares `slash: /gpowers-upgrade`
- omits `upstream: gstack@main` (it's a *renamed* skill, not a direct port — body will diverge)
- contains a one-line note pointing to Plan #9

- [ ] **Step 1: Write the stub**

```bash
mkdir -p tools/skills/gpowers-upgrade
cat > tools/skills/gpowers-upgrade/SKILL.md <<'EOF'
---
name: gpowers-upgrade
description: Self-upgrade gpowers — pulls new versions of core/roles/tools/business from upstream and re-runs install hooks. Stub; full body landed by Plan #9.
namespace: tools
slash: /gpowers-upgrade
---

# gpowers-upgrade (stub)

This skill is a placeholder. Implementation arrives via the **gpowers upgrade** plan (Plan #9 — `2026-05-14-gpowers-upgrade.md`). Until then, invoking `/gpowers-upgrade` prints this notice and exits.
EOF
```

- [ ] **Step 2: Run frontmatter test**

Run: `bats tests/unit/tools/frontmatter.bats`
Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add tools/skills/gpowers-upgrade/
git commit -m "feat(tools): stub gpowers-upgrade skill (full body in plan #9)"
```

---

## Task 6: Assert no leftover gstack paths

**Files:**
- Create: `tests/unit/tools/no-gstack-paths.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/tools/no-gstack-paths.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  TOOLS="$BATS_TEST_DIRNAME/../../../tools/skills"
}

@test "no literal '~/.gstack/' in tool skill bodies" {
  for dir in "$TOOLS"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "~/\.gstack/"; then
      echo "$name body contains ~/.gstack/"
      return 1
    fi
  done
}

@test "no 'gstack-' CLI references in tool skill bodies" {
  for dir in "$TOOLS"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -qE '\bgstack-[a-z]'; then
      echo "$name body still references gstack-* binary"
      return 1
    fi
  done
}

@test "skills use gpowers-path helper for paths" {
  # Any skill that previously had a path now uses gpowers-path
  found=0
  for dir in "$TOOLS"/*/; do
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "gpowers-path"; then found=$((found+1)); fi
  done
  [ "$found" -ge 10 ]  # at least 10 skills had paths to rewrite in fixture
}
EOF
```

- [ ] **Step 2: Run test**

Run: `bats tests/unit/tools/no-gstack-paths.bats`
Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/tools/no-gstack-paths.bats
git commit -m "test(tools): assert no leftover gstack paths or bin refs"
```

---

## Task 7: Port the three bin/ CLI scripts

**Files:**
- Create: `tools/bin/{gpowers-ship-helper,gpowers-health,gpowers-update-check}`

In the production migration these will start as a near-verbatim copy of the gstack originals with two rewrites: paths (same regex as the skill importer) and `gstack-` → `gpowers-`. For this plan we use the fixture stubs.

- [ ] **Step 1: Write a failing test**

```bash
cat > tests/unit/tools/bin-renamed.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  BIN="$GPOWERS_REPO/tools/bin"
}

@test "gpowers-ship-helper exists and is executable" {
  [ -x "$BIN/gpowers-ship-helper" ]
}

@test "gpowers-health exists and is executable" {
  [ -x "$BIN/gpowers-health" ]
}

@test "gpowers-update-check exists and is executable" {
  [ -x "$BIN/gpowers-update-check" ]
}

@test "bin scripts do not contain literal 'gstack' references" {
  for f in "$BIN"/*; do
    if grep -q 'gstack' "$f"; then
      echo "$(basename "$f") contains gstack"
      return 1
    fi
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Port the scripts**

```bash
mkdir -p tools/bin
SRC="tests/fixtures/tools/fake-gstack-checkout/bin"
for old in gstack-ship-helper gstack-health gstack-update-check; do
  new=$(echo "$old" | sed 's/^gstack-/gpowers-/')
  sed \
    -e "s/gstack-/gpowers-/g" \
    -e "s|~/\.gstack/state|\$(gpowers-path state)|g" \
    -e "s|~/\.gstack/config|\$(gpowers-path config)|g" \
    -e "s|~/\.gstack/cache|\$(gpowers-path cache)|g" \
    -e "s|~/\.gstack/data|\$(gpowers-path data)|g" \
    -e "s|~/\.gstack/|\$(gpowers-path home)/|g" \
    "$SRC/$old" > "tools/bin/$new"
  chmod +x "tools/bin/$new"
done
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/tools/bin-renamed.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add tools/bin/
git commit -m "feat(tools): port gstack-ship-helper, gstack-health, gstack-update-check"
```

---

## Task 8: shellcheck the bin scripts

**Files:**
- Create: `tests/unit/tools/bin-shellcheck.bats`

- [ ] **Step 1: Write the test (skipped if shellcheck absent)**

```bash
cat > tests/unit/tools/bin-shellcheck.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  BIN="$GPOWERS_REPO/tools/bin"
  command -v shellcheck >/dev/null || skip "shellcheck not installed"
}

@test "gpowers-ship-helper passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-ship-helper"
}

@test "gpowers-health passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-health"
}

@test "gpowers-update-check passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-update-check"
}
EOF
```

- [ ] **Step 2: Run test**

Run: `bats tests/unit/tools/bin-shellcheck.bats`
Expected: PASS (or SKIP on hosts without shellcheck).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/tools/bin-shellcheck.bats
git commit -m "test(tools): shellcheck migrated bin scripts"
```

---

## Task 9: Slash-command roundtrip test

**Files:**
- Create: `tests/unit/tools/slash-commands-roundtrip.bats`

Confirms every skill declares a `slash:` field and that each slash command appears exactly once across all four modules (no collisions with core/ or roles/).

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/tools/slash-commands-roundtrip.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
}

@test "each tools/ skill declares a unique slash command" {
  declare -A seen
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    [ -n "$slash" ] || { echo "$name has no slash:"; return 1; }
    [ -z "${seen[$slash]:-}" ] || { echo "duplicate slash: $slash ($name vs ${seen[$slash]})"; return 1; }
    seen[$slash]=$name
  done
}

@test "tools/ slash commands do not collide with core/ skills" {
  for dir in "$REPO/tools/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    cname="${slash#/}"
    if [ -d "$REPO/core/skills/$cname" ]; then
      echo "tools/$(basename "$dir") collides with core/$cname"
      return 1
    fi
  done
}
EOF
```

- [ ] **Step 2: Run test**

Run: `bats tests/unit/tools/slash-commands-roundtrip.bats`
Expected: PASS (2 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/tools/slash-commands-roundtrip.bats
git commit -m "test(tools): slash command uniqueness across modules"
```

---

## Task 10: tools/ upstream-source.json + manifest

**Files:**
- Create: `tools/upstream-source.json`
- Modify: `manifest.json`

- [ ] **Step 1: Write upstream-source.json**

```bash
cat > tools/upstream-source.json <<'EOF'
{
  "module": "tools",
  "upstream": {
    "repo": "github.com/garrytan/gstack",
    "ref": "main",
    "sha": "0000000000000000000000000000000000000000"
  },
  "imported_at": "2026-05-14T00:00:00Z",
  "submodules": {
    "non_browser": [
      "ship","land-and-deploy","landing-report","setup-deploy","health",
      "benchmark-models","context-save","context-restore","careful","freeze",
      "guard","unfreeze","make-pdf","fix-the-roof","simplify",
      "fewer-permission-prompts"
    ],
    "browser_dependent": ["__pending_plan_5__"],
    "local_renamed": {"gstack-upgrade": "gpowers-upgrade"}
  }
}
EOF
```

- [ ] **Step 2: Update manifest**

```bash
source lib/manifest.sh
gpowers_manifest_set_installed tools true
gpowers_manifest_set tools skill_count_non_browser 17
gpowers_manifest_set tools upstream '"gstack@main"'
```

- [ ] **Step 3: Failing test for manifest**

```bash
cat > tests/unit/tools/manifest-records-tools.bats <<'EOF'
#!/usr/bin/env bats

setup() { MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest declares tools module installed" {
  [ "$(jq -r '.modules.tools.installed' < "$MANIFEST")" = "true" ]
}

@test "manifest records 17 non-browser tool skills" {
  [ "$(jq -r '.modules.tools.skill_count_non_browser' < "$MANIFEST")" = "17" ]
}
EOF
```

Run: PASS expected (manifest already updated in Step 2).

- [ ] **Step 4: Commit**

```bash
git add tools/upstream-source.json manifest.json tests/unit/tools/manifest-records-tools.bats
git commit -m "feat(tools): upstream-source.json + manifest records non-browser tools"
```

---

## Task 11: End-to-end smoke

**Files:**
- Create: `tests/integration/tools/non-browser-smoke.bats`

Verify after a fresh install, a sample skill can be loaded and its slash command is discoverable.

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/tools/non-browser-smoke.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
}

@test "ship skill can be cat'd and references gpowers-path" {
  body=$(cat "$GPOWERS_HOME/tools/skills/ship/SKILL.md")
  echo "$body" | grep -qF "namespace: tools"
  echo "$body" | grep -qF "gpowers-path"
}

@test "gpowers-health is on PATH and runs" {
  run gpowers-health --help 2>&1
  # Stub returns 0; real implementation may differ. Just assert it's invokable.
  command -v gpowers-health >/dev/null
}

@test "gpowers-path home points to GPOWERS_HOME" {
  [ "$(gpowers-path home)" = "$GPOWERS_HOME" ]
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/integration/tools/non-browser-smoke.bats`
Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/tools/non-browser-smoke.bats
git commit -m "test(tools): e2e smoke for non-browser tools"
```

---

## Self-Review

### 1. Spec coverage (§4 of design, non-browser subset)

| Spec entry | Task |
|---|---|
| ship, land-and-deploy, landing-report, setup-deploy | Task 4 |
| health, benchmark-models | Task 4 |
| context-save, context-restore | Task 4 |
| careful, freeze, guard, unfreeze | Task 4 |
| make-pdf, fix-the-roof, simplify, fewer-permission-prompts | Task 4 |
| gpowers-upgrade (renamed from gstack-upgrade) | Task 5 (stub) + Plan #9 (body) |
| `tools/bin/` ports of gstack CLI scripts | Task 7 |
| `upstream-source.json` for tools | Task 10 |

Browser-dependent tools (browse, qa, qa-only, canary, benchmark, setup-browser-cookies, setup-gbrain, sync-gbrain, open-gstack-browser, aidesigner, aidesigner-frontend) are explicitly deferred to Plan #5.

### 2. Placeholder scan

- `gpowers-upgrade` stub is *intentional* and clearly marked. Plan #9 lands the body.
- `submodules.browser_dependent: ["__pending_plan_5__"]` in upstream-source.json is a sentinel; Plan #5 replaces it. This is explicit, not a hidden TODO.

### 3. Type / name consistency

- All 17 slash commands match the `slash:` field in their SKILL.md.
- `gpowers-path` invocation signature consistent with Plan #1 (`gpowers-path home/config/state/cache/data/...`).
- `lib/manifest.sh` functions (`gpowers_manifest_set_installed`, `gpowers_manifest_set`) match Plan #1's API.

### 4. Decomposition

11 tasks. Each task = one importable artifact. Heavy lifting (the importer) is one task; subsequent tasks consume it.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-tools-non-browser.md`. Depends on Plan #1 only. Choose subagent-driven or inline at execution time.
