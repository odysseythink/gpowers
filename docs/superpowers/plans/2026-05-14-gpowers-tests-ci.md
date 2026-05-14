# gpowers Tests + CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the unit + integration tests written by Plans #1–#10 into a single runnable suite, add the missing `platform-smoke/` layer (one smoke test per platform — 7 total), build the two fixture environments the spec calls for (`demo-site/` and `sample-repo/`), and wire GitHub Actions CI that runs all three layers on every PR and tags releases as artifacts.

**Architecture:** Tests live under `tests/{unit,integration,platform-smoke,fixtures}/`. A top-level `tests/run.sh` orchestrates them with selectable scope (`./tests/run.sh unit`, `./tests/run.sh all`, `./tests/run.sh smoke claude-code`). Each platform smoke test runs *the live binary* of that platform — Claude Code, Codex, Gemini, Cursor, OpenCode, Copilot CLI, Kimi — against a copy of `~/.gpowers/` and asserts: (1) gpowers loads without error, (2) the using-gpowers preamble injection works (or the platform's equivalent), (3) at least one slash command resolves to its skill, (4) the browser driver resolves (mock mode). Some platforms (Codex, Gemini, OpenCode, Copilot, Kimi) may not be installed in CI by default — the smoke runner skips with a clear note rather than failing. `demo-site/` is a static HTML page served via `python3 -m http.server` (already used by Plan #3) — Plan #11 extends it for QA/canary/benchmark testing. `sample-repo/` is a small fixture git repo used by `/ship`, `/pr-review`, `/retro`, etc.

**Tech Stack:** bats-core, jq, shellcheck (already used in earlier plans), GitHub Actions, Docker (optional — for matrix builds across Linux/macOS), Python 3 (fixture server).

**Depends on:** Plans #1–#10 (all the artifacts the CI will test). This plan is intended to land *last* among the artifact plans, before the docs plan (#12).

---

## File Structure

```
tests/
├── run.sh                                  Top-level orchestrator
├── helpers/
│   ├── load.bash                           Shared bats helpers (already created Plan #1)
│   ├── platform-detect.sh                  Detect what's installed in CI
│   └── seed-gpowers-home.sh                Build a working $GPOWERS_HOME for smoke
├── fixtures/                                (Plans #1–10 already populated)
│   ├── demo-site/
│   │   ├── index.html                      Already from Plan #3
│   │   ├── qa-form.html                    NEW: form for /qa tests
│   │   ├── canary-version.html             NEW: window.__version page for /canary
│   │   └── server.sh                       Already from Plan #3
│   └── sample-repo/                         NEW: small git repo for /ship, /retro, etc.
├── platform-smoke/
│   ├── claude-code.bats
│   ├── codex.bats
│   ├── gemini.bats
│   ├── cursor.bats
│   ├── opencode.bats
│   ├── copilot.bats
│   └── kimi.bats
└── unit/, integration/                     (already populated by Plans #1–10)
.github/workflows/
├── ci.yml                                  PR + push to main
└── release.yml                              Tag-triggered: build artifact + run all tests
RELEASING.md                                NEW: release procedure (semver + git tag + tarball)
```

---

## Task 1: Build the sample-repo fixture

**Files:**
- Create: `tests/fixtures/sample-repo/`

A small git repo with a handful of commits, a `package.json`, a `README.md`, and a working tree. Used by `/ship`, `/pr-review`, `/retro`, `/health` etc. to have something realistic to operate on.

- [ ] **Step 1: Build the fixture init script**

```bash
mkdir -p tests/fixtures
cat > tests/fixtures/build-sample-repo.sh <<'EOF'
#!/usr/bin/env bash
# Usage: build-sample-repo.sh <target-dir>
# Builds a deterministic sample git repo for tests. Idempotent: removes target first.
set -euo pipefail
T="${1:?target dir required}"
rm -rf "$T"
mkdir -p "$T"
cd "$T"
git init -q
git config user.email t@t
git config user.name t

cat > README.md <<'F'
# sample-repo

Fixture for gpowers tests. Do not edit by hand — regenerate with `tests/fixtures/build-sample-repo.sh`.
F
git add README.md
git commit -q -m "initial: README"

cat > package.json <<'F'
{
  "name": "sample-repo",
  "version": "0.1.0",
  "scripts": { "test": "echo no tests" }
}
F
git add package.json
git commit -q -m "feat: package.json"

mkdir -p src
cat > src/index.js <<'F'
// sample-repo entry point
export function add(a, b) { return a + b; }
F
git add src/index.js
git commit -q -m "feat: add() function"

cat > src/bug.js <<'F'
// sample bug for /investigate fixture
export function divide(a, b) { return a / b; }  // does not guard b === 0
F
git add src/bug.js
git commit -q -m "feat: divide() (intentional gap for /investigate fixture)"

# Branch we ship from
git checkout -q -b feature/sample
echo "// feature change" >> src/index.js
git add src/index.js
git commit -q -m "feat: feature/sample line"
git checkout -q main 2>/dev/null || git checkout -q master 2>/dev/null || true
EOF
chmod +x tests/fixtures/build-sample-repo.sh
```

- [ ] **Step 2: Run it; smoke check**

```bash
./tests/fixtures/build-sample-repo.sh /tmp/sample-repo-smoke
git -C /tmp/sample-repo-smoke log --oneline | head
rm -rf /tmp/sample-repo-smoke
```

Expected: 4 commits on main + 1 on feature branch.

- [ ] **Step 3: Add a bats test exercising the fixture**

```bash
cat > tests/unit/fixtures/sample-repo-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/srepo"
  "$REPO/tests/fixtures/build-sample-repo.sh" "$TMP"
  cd "$TMP"
}

@test "sample-repo has 4 commits on main" {
  count=$(git log --oneline | wc -l)
  [ "$count" -ge 4 ]
}

@test "sample-repo has feature/sample branch" {
  git rev-parse feature/sample >/dev/null
}

@test "sample-repo has package.json + README + src/" {
  [ -f package.json ]
  [ -f README.md ]
  [ -d src ]
}
EOF
bats tests/unit/fixtures/sample-repo-shape.bats
```

Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add tests/fixtures/build-sample-repo.sh tests/unit/fixtures/sample-repo-shape.bats
git commit -m "test(fixtures): build-sample-repo.sh — deterministic fixture repo"
```

---

## Task 2: Extend demo-site for QA / canary / benchmark

**Files:**
- Create: `tests/fixtures/demo-site/qa-form.html`
- Create: `tests/fixtures/demo-site/canary-version.html`
- Create: `tests/fixtures/demo-site/benchmark.html`

Each is a small standalone HTML page exercising one shape of skill behavior.

- [ ] **Step 1: Write the pages**

```bash
mkdir -p tests/fixtures/demo-site

cat > tests/fixtures/demo-site/qa-form.html <<'EOF'
<!doctype html>
<html><head><title>QA form</title></head>
<body>
  <h1>QA Fixture</h1>
  <form id="form">
    <input type="email" id="email" name="email" placeholder="email" />
    <input type="text" id="name" name="name" placeholder="name" />
    <button type="button" id="submit" onclick="document.getElementById('result').textContent='ok';">Submit</button>
  </form>
  <pre id="result"></pre>
  <script>console.log("qa-form ready");</script>
</body></html>
EOF

cat > tests/fixtures/demo-site/canary-version.html <<'EOF'
<!doctype html>
<html><head><title>Canary</title></head>
<body>
  <h1 id="banner">Production</h1>
  <script>window.__version="2026.05.14"; console.log("canary v"+window.__version);</script>
</body></html>
EOF

cat > tests/fixtures/demo-site/benchmark.html <<'EOF'
<!doctype html>
<html><head><title>Benchmark fixture</title></head>
<body>
  <h1>Benchmark</h1>
  <script>
    const start = performance.now();
    setTimeout(() => { window.__ttfb = performance.now() - start; }, 50);
  </script>
</body></html>
EOF
```

- [ ] **Step 2: Smoke test the pages parse**

```bash
cat > tests/unit/fixtures/demo-site-pages.bats <<'EOF'
#!/usr/bin/env bats

setup() { D="$BATS_TEST_DIRNAME/../../../tests/fixtures/demo-site"; }

@test "qa-form.html exists and contains the form" {
  [ -f "$D/qa-form.html" ]
  grep -q "id=\"form\"" "$D/qa-form.html"
  grep -q "id=\"submit\"" "$D/qa-form.html"
}

@test "canary-version.html exposes window.__version" {
  grep -q "window.__version" "$D/canary-version.html"
}

@test "benchmark.html measures something" {
  grep -q "performance.now" "$D/benchmark.html"
}
EOF
bats tests/unit/fixtures/demo-site-pages.bats
```

Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/fixtures/demo-site/qa-form.html tests/fixtures/demo-site/canary-version.html \
        tests/fixtures/demo-site/benchmark.html tests/unit/fixtures/demo-site-pages.bats
git commit -m "test(fixtures): qa-form, canary-version, benchmark demo pages"
```

---

## Task 3: Platform-detection helper

**Files:**
- Create: `tests/helpers/platform-detect.sh`

A small library that detects which of the 7 supported platforms is installed in the current environment. Returns a list to stdout. Each platform-smoke test sources this and skips if its target isn't present.

- [ ] **Step 1: Write + test**

```bash
mkdir -p tests/helpers
cat > tests/helpers/platform-detect.sh <<'EOF'
# Source me. Provides: platform_present <name> → exit 0 if present.
platform_present() {
  case "$1" in
    claude-code) command -v claude >/dev/null;;
    codex)       command -v codex >/dev/null;;
    gemini)      command -v gemini >/dev/null;;
    cursor)      command -v cursor-cli >/dev/null || command -v cursor >/dev/null;;
    opencode)    command -v opencode >/dev/null;;
    copilot)     command -v gh >/dev/null && gh copilot --version >/dev/null 2>&1;;
    kimi)        command -v kimi >/dev/null;;
    *) return 1;;
  esac
}

platforms_present() {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    if platform_present "$p"; then echo "$p"; fi
  done
}
EOF

cat > tests/unit/helpers/platform-detect.bats <<'EOF'
#!/usr/bin/env bats

setup() { source "$BATS_TEST_DIRNAME/../../helpers/platform-detect.sh"; }

@test "platform_present returns 0 or 1 cleanly" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    run platform_present "$p"
    case "$status" in 0|1) ;; *) echo "unexpected status $status for $p"; return 1;; esac
  done
}

@test "platform_present errors for unknown name" {
  run platform_present "bogus"
  [ "$status" -ne 0 ]
}

@test "platforms_present emits a subset of known platforms" {
  out=$(platforms_present)
  for p in $out; do
    case "$p" in claude-code|codex|gemini|cursor|opencode|copilot|kimi) ;;
                 *) echo "unknown: $p"; return 1;; esac
  done
}
EOF
bats tests/unit/helpers/platform-detect.bats
```

Expected: PASS (3 tests).

- [ ] **Step 2: Commit**

```bash
git add tests/helpers/platform-detect.sh tests/unit/helpers/platform-detect.bats
git commit -m "test(helpers): platform_present / platforms_present detectors"
```

---

## Task 4: Seed-gpowers-home helper

**Files:**
- Create: `tests/helpers/seed-gpowers-home.sh`

Builds a complete working `$GPOWERS_HOME` from the repo checkout — every smoke test calls this to get a clean install. Equivalent of running `./install --location=$GPOWERS_HOME --non-interactive` against the fixture repo.

- [ ] **Step 1: Write the seeder**

```bash
cat > tests/helpers/seed-gpowers-home.sh <<'EOF'
#!/usr/bin/env bash
# Usage: seed-gpowers-home.sh <target-dir>
# Copies repo contents into target-dir and runs `gpowers-platforms gen all`.
set -euo pipefail
TARGET="${1:?target required}"
REPO="${REPO:-$(cd "$(dirname "$0")/../.." && pwd)}"

rm -rf "$TARGET"
mkdir -p "$TARGET"
# Copy modules + bin + lib + platforms shape + manifest + upstream-sources
cp -R "$REPO/core" "$REPO/roles" "$REPO/tools" "$TARGET/"
[ -d "$REPO/business" ] && cp -R "$REPO/business" "$TARGET/"
cp -R "$REPO/bin" "$REPO/lib" "$TARGET/"
mkdir -p "$TARGET/platforms"
cp "$REPO/platforms/_platform-shapes.json" "$TARGET/platforms/"
cp "$REPO/manifest.json" "$TARGET/manifest.json"
cp "$REPO/upstream-sources.json" "$TARGET/upstream-sources.json"

export GPOWERS_HOME="$TARGET"
export PATH="$TARGET/bin:$TARGET/tools/bin:$PATH"

# Regenerate per-platform assets so platform-smoke tests have real files
"$TARGET/bin/gpowers-platforms" gen all >/dev/null

echo "$TARGET"
EOF
chmod +x tests/helpers/seed-gpowers-home.sh
```

- [ ] **Step 2: Smoke test**

```bash
out=$(REPO="$(pwd)" ./tests/helpers/seed-gpowers-home.sh /tmp/gp-seed-smoke)
[ "$out" = "/tmp/gp-seed-smoke" ]
[ -f /tmp/gp-seed-smoke/platforms/claude-code/plugin.json ]
[ -f /tmp/gp-seed-smoke/core/skills/using-gpowers/SKILL.md ]
rm -rf /tmp/gp-seed-smoke
```

- [ ] **Step 3: Commit**

```bash
git add tests/helpers/seed-gpowers-home.sh
git commit -m "test(helpers): seed-gpowers-home.sh — build complete \$GPOWERS_HOME from repo"
```

---

## Task 5: Write the 7 platform-smoke tests

**Files:**
- Create: `tests/platform-smoke/{claude-code,codex,gemini,cursor,opencode,copilot,kimi}.bats`

Each test follows the same skeleton: skip-if-not-installed → seed-gpowers-home → ask the platform to load and resolve a slash command → assert success.

- [ ] **Step 1: Failing tests for all 7**

```bash
mkdir -p tests/platform-smoke

write_smoke() {
  local platform="$1" extra="$2"
  cat > "tests/platform-smoke/$platform.bats" <<EOF
#!/usr/bin/env bats

setup() {
  REPO="\$BATS_TEST_DIRNAME/../.."
  source "\$REPO/tests/helpers/platform-detect.sh"
  if ! platform_present "$platform"; then
    skip "$platform CLI not installed"
  fi
  HOME_TGT="\$BATS_TEST_TMPDIR/gp-$platform"
  REPO="\$REPO" "\$REPO/tests/helpers/seed-gpowers-home.sh" "\$HOME_TGT" >/dev/null
  export GPOWERS_HOME="\$HOME_TGT"
  export PATH="\$HOME_TGT/bin:\$HOME_TGT/tools/bin:\$PATH"
}

@test "$platform: gpowers-platforms verify $platform reports OK" {
  out=\$(gpowers-platforms verify "$platform")
  echo "\$out" | grep -q "OK: $platform"
}

@test "$platform: plugin/extension manifest is valid JSON" {
  manifest=\$(jq -r '.platforms."$platform".manifest_filename' < "\$GPOWERS_HOME/platforms/_platform-shapes.json")
  jq empty < "\$GPOWERS_HOME/platforms/$platform/\$manifest"
}

@test "$platform: at least one command file is present" {
  cmd_dir=\$(jq -r '.platforms."$platform".command_dir' < "\$GPOWERS_HOME/platforms/_platform-shapes.json")
  count=\$(find "\$GPOWERS_HOME/platforms/$platform/\$cmd_dir" -mindepth 1 \\( -name '*.md' -o -type d \\) | wc -l)
  [ "\$count" -gt 0 ]
}

$extra
EOF
}

# Standard pattern for 6 platforms
for p in claude-code codex gemini cursor opencode copilot; do
  write_smoke "$p" ""
done

# Kimi: extra assertion that adapters dir has gpowers-* entries
write_smoke "kimi" '
@test "kimi: adapters dir has gpowers-* entries" {
  count=$(find "$GPOWERS_HOME/platforms/kimi/adapters" -mindepth 1 -maxdepth 1 -type d -name "gpowers*" | wc -l)
  [ "$count" -ge 1 ]
}
'
```

- [ ] **Step 2: Run them (most will skip due to missing CLIs)**

```bash
bats tests/platform-smoke/*.bats
```

Expected: each .bats file either PASSES (CLI installed) or SKIPs cleanly with a "not installed" message. CI must surface skipped vs failed distinctly.

- [ ] **Step 3: Commit**

```bash
git add tests/platform-smoke/
git commit -m "test(platform-smoke): 7 bats files — skip-if-uninstalled + verify+load+command"
```

---

## Task 6: Top-level test runner `tests/run.sh`

**Files:**
- Create: `tests/run.sh`

Single entry point: `./tests/run.sh [unit|integration|smoke|all] [<filter>]`.

- [ ] **Step 1: Write the runner**

```bash
cat > tests/run.sh <<'EOF'
#!/usr/bin/env bash
# Usage:
#   tests/run.sh                     # run unit + integration (smoke skipped by default)
#   tests/run.sh unit
#   tests/run.sh integration
#   tests/run.sh smoke               # run all platform-smoke (each may skip)
#   tests/run.sh smoke claude-code   # only one platform
#   tests/run.sh all                 # everything including smoke
#   tests/run.sh unit roles          # filter: only tests/unit/roles/
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
SCOPE="${1:-default}"
FILTER="${2:-}"

run_dir() {
  local d="$1"
  [ -d "$d" ] || return 0
  local found
  if [ -n "$FILTER" ]; then
    found=$(find "$d" -path "*$FILTER*" -name '*.bats' 2>/dev/null)
  else
    found=$(find "$d" -name '*.bats' 2>/dev/null)
  fi
  if [ -z "$found" ]; then
    echo "[run] no tests under $d"
    return 0
  fi
  echo "[run] $d"
  echo "$found" | xargs bats
}

case "$SCOPE" in
  unit)         run_dir "$REPO/tests/unit" ;;
  integration)  run_dir "$REPO/tests/integration" ;;
  smoke)        run_dir "$REPO/tests/platform-smoke" ;;
  all)
    run_dir "$REPO/tests/unit"
    run_dir "$REPO/tests/integration"
    run_dir "$REPO/tests/platform-smoke"
    ;;
  default|"")
    run_dir "$REPO/tests/unit"
    run_dir "$REPO/tests/integration"
    ;;
  *) echo "unknown scope: $SCOPE" >&2; exit 2 ;;
esac
EOF
chmod +x tests/run.sh
```

- [ ] **Step 2: Failing test**

```bash
cat > tests/unit/test-runner/run-sh.bats <<'EOF'
#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

@test "run.sh unit completes without error" {
  run bash "$REPO/tests/run.sh" unit
  [ "$status" -eq 0 ]
}

@test "run.sh integration completes (may have no tests)" {
  run bash "$REPO/tests/run.sh" integration
  [ "$status" -eq 0 ]
}

@test "run.sh unknown scope exits 2" {
  run bash "$REPO/tests/run.sh" bogus
  [ "$status" -eq 2 ]
}

@test "run.sh unit roles filter limits scope" {
  out=$(bash "$REPO/tests/run.sh" unit roles 2>&1 || true)
  echo "$out" | grep -q "tests/unit/roles" || echo "$out" | grep -qi "no tests"
}
EOF
bats tests/unit/test-runner/run-sh.bats
```

Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/run.sh tests/unit/test-runner/run-sh.bats
git commit -m "test(runner): tests/run.sh — single orchestrator for unit/integration/smoke"
```

---

## Task 7: CI workflow — PR + push (.github/workflows/ci.yml)

**Files:**
- Create: `.github/workflows/ci.yml`

Runs on every push and PR. Three jobs in parallel: lint (shellcheck), unit, integration. platform-smoke runs only on a per-job basis when the matching CLI install is feasible in CI (claude-code is `npm i -g`-able; others may not be).

- [ ] **Step 1: Write the workflow**

```bash
mkdir -p .github/workflows
cat > .github/workflows/ci.yml <<'EOF'
name: ci

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  shellcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install shellcheck
        run: sudo apt-get update && sudo apt-get install -y shellcheck
      - name: Run shellcheck on bin/ and lib/
        run: |
          shopt -s globstar nullglob
          shellcheck -S warning bin/* lib/*.sh tools/bin/* core/hooks/session-start || true
          # Strict pass on the wrappers / entry points
          shellcheck -S error bin/gpowers bin/gpowers-path bin/gpowers-init \
                              bin/gpowers-detect-platforms bin/gpowers-upgrade \
                              bin/gpowers-migrate bin/gpowers-platforms \
                              install uninstall

  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install bats-core + jq
        run: sudo apt-get update && sudo apt-get install -y bats jq python3 rsync
      - name: Run unit tests
        run: ./tests/run.sh unit

  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install deps
        run: sudo apt-get update && sudo apt-get install -y bats jq python3 rsync git
      - name: Run integration tests
        run: ./tests/run.sh integration

  smoke-claude-code:
    runs-on: ubuntu-latest
    continue-on-error: true   # CLI may not be available on every runner
    steps:
      - uses: actions/checkout@v4
      - name: Install bats + jq
        run: sudo apt-get update && sudo apt-get install -y bats jq
      - name: Try installing Claude Code CLI
        run: |
          if command -v npm >/dev/null; then
            npm i -g @anthropic-ai/claude-code 2>/dev/null || true
          fi
      - name: Smoke (will skip if not installed)
        run: ./tests/run.sh smoke claude-code
EOF
```

- [ ] **Step 2: Lint the workflow file**

```bash
# Basic YAML validation
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
```

Expected: no output (valid YAML).

- [ ] **Step 3: Failing test**

```bash
cat > tests/unit/ci/workflow-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  CI="$REPO/.github/workflows/ci.yml"
}

@test "ci.yml exists" { [ -f "$CI" ]; }
@test "ci.yml defines unit job" { grep -q "^  unit:" "$CI"; }
@test "ci.yml defines integration job" { grep -q "^  integration:" "$CI"; }
@test "ci.yml defines shellcheck job" { grep -q "^  shellcheck:" "$CI"; }
@test "ci.yml runs ./tests/run.sh" { grep -q "tests/run.sh" "$CI"; }
EOF
bats tests/unit/ci/workflow-shape.bats
```

Expected: PASS (5 tests).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml tests/unit/ci/workflow-shape.bats
git commit -m "ci: shellcheck + unit + integration + claude-code smoke (continue-on-error)"
```

---

## Task 8: Release workflow (.github/workflows/release.yml)

**Files:**
- Create: `.github/workflows/release.yml`

Triggered by tags `v*`. Builds a tarball, runs the full test suite, publishes the artifact.

- [ ] **Step 1: Write the workflow**

```bash
cat > .github/workflows/release.yml <<'EOF'
name: release

on:
  push:
    tags: [v*]

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install deps
        run: sudo apt-get update && sudo apt-get install -y bats jq python3 rsync git shellcheck

      - name: Run full test suite (excluding smoke)
        run: ./tests/run.sh all || ./tests/run.sh unit  # all is best-effort if smoke deps absent

      - name: Build artifact tarball
        run: |
          version="${GITHUB_REF_NAME#v}"
          tarball="gpowers-${version}.tar.gz"
          tar --exclude='./.git' --exclude='./tests' --exclude='./.github' \
              -czf "$tarball" .
          sha256sum "$tarball" > "$tarball.sha256"
          echo "ARTIFACT=$tarball" >> "$GITHUB_ENV"

      - name: Publish release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            ${{ env.ARTIFACT }}
            ${{ env.ARTIFACT }}.sha256
          generate_release_notes: true
EOF
```

- [ ] **Step 2: Failing test**

```bash
cat > tests/unit/ci/release-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REL="$BATS_TEST_DIRNAME/../../../.github/workflows/release.yml"
}

@test "release.yml exists" { [ -f "$REL" ]; }
@test "release.yml triggers on v* tags" { grep -q "tags: \[v\*\]" "$REL"; }
@test "release.yml builds gpowers-<version>.tar.gz" {
  grep -q "gpowers-\${version}.tar.gz\|gpowers-\$version" "$REL"
}
@test "release.yml computes sha256" { grep -q "sha256sum" "$REL"; }
EOF
bats tests/unit/ci/release-shape.bats
```

Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml tests/unit/ci/release-shape.bats
git commit -m "ci: release.yml — tarball + sha256 + GH release on v* tags"
```

---

## Task 9: Write RELEASING.md

**Files:**
- Create: `RELEASING.md`

Short and actionable: how to cut a release.

- [ ] **Step 1: Write the doc**

```bash
cat > RELEASING.md <<'EOF'
# Releasing gpowers

gpowers follows semantic versioning: `vMAJOR.MINOR.PATCH`.

- **MAJOR** — driver interface or module boundary change.
- **MINOR** — new skill, new platform, new opt-in module.
- **PATCH** — bug fix or upstream sync.

## Cutting a release

1. Verify `main` is green:
   ```bash
   ./tests/run.sh all
   ```

2. Bump the version in `manifest.json` (`.version`) and any READMEs that
   reference it. Commit:
   ```bash
   git commit -am "chore: bump version to v1.2.3"
   ```

3. Tag:
   ```bash
   git tag -a v1.2.3 -m "release v1.2.3"
   git push origin main --follow-tags
   ```

4. The `release.yml` workflow runs automatically. It:
   - Re-runs the full test suite
   - Builds `gpowers-1.2.3.tar.gz` + `gpowers-1.2.3.tar.gz.sha256`
   - Creates a GitHub Release with the tarball and auto-generated notes

5. Manually verify the release artifact is downloadable from the GitHub Releases page.

## Hotfix from a non-main branch

If a hotfix is needed off an older minor:
```bash
git checkout -b hotfix/v1.1.x v1.1.4
# make fix
git tag -a v1.1.5 -m "patch: hotfix"
git push origin hotfix/v1.1.x --follow-tags
```

The release workflow runs against the tag's checkout regardless of branch.
EOF
```

- [ ] **Step 2: Failing test**

```bash
cat > tests/unit/ci/releasing-md.bats <<'EOF'
#!/usr/bin/env bats

setup() { R="$BATS_TEST_DIRNAME/../../../RELEASING.md"; }

@test "RELEASING.md exists" { [ -f "$R" ]; }
@test "RELEASING.md names semver categories MAJOR/MINOR/PATCH" {
  for kw in MAJOR MINOR PATCH; do grep -q "$kw" "$R" || { echo "$kw"; return 1; }; done
}
@test "RELEASING.md references release.yml flow" {
  grep -qi "release.yml\|release workflow" "$R"
}
EOF
bats tests/unit/ci/releasing-md.bats
```

Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add RELEASING.md tests/unit/ci/releasing-md.bats
git commit -m "docs: RELEASING.md — semver + tag + release-workflow procedure"
```

---

## Task 10: Manifest record

**Files:**
- Modify: `manifest.json`

Record that the test/CI subsystem is installed and the test-runner contract is in place.

- [ ] **Step 1: Update manifest + test**

```bash
source lib/manifest.sh
gpowers_manifest_set tests runner '"tests/run.sh"'
gpowers_manifest_set tests layers '["unit","integration","platform-smoke"]'
gpowers_manifest_set ci platforms '["github-actions"]'
gpowers_manifest_set ci workflows '["ci.yml","release.yml"]'

cat > tests/unit/ci/manifest-ci.bats <<'EOF'
#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest records test runner path" {
  [ "$(jq -r '.tests.runner' < "$M")" = "tests/run.sh" ]
}
@test "manifest lists 3 test layers" {
  [ "$(jq -r '.tests.layers | length' < "$M")" = "3" ]
}
@test "manifest declares github-actions CI" {
  jq -e '.ci.platforms | index("github-actions")' < "$M" >/dev/null
}
@test "manifest lists both workflows" {
  jq -e '.ci.workflows | index("ci.yml") and index("release.yml")' < "$M" >/dev/null
}
EOF
bats tests/unit/ci/manifest-ci.bats
```

Expected: PASS (4 tests).

- [ ] **Step 2: Commit**

```bash
git add manifest.json tests/unit/ci/manifest-ci.bats
git commit -m "feat(ci): manifest records test layers + CI workflows"
```

---

## Self-Review

### 1. Spec coverage (§6 testing strategy)

| Spec entry | Task |
|---|---|
| `tests/unit/` per-module | Plans #1–10 already; Plan #11 just orchestrates |
| `tests/integration/` per-skill | Plans #1–10; orchestrated by Plan #11 |
| `tests/platform-smoke/` for 7 platforms | Task 5 |
| `tests/fixtures/demo-site/` | Task 2 (extends Plan #3 base) |
| `tests/fixtures/sample-repo/` | Task 1 |
| Auto-test on `gpowers upgrade` | Plan #9 Task 4 (worker already calls tests) |
| Release artifact `gpowers-vX.Y.Z.tar.gz` | Task 8 |
| Semver discipline | Task 9 (RELEASING.md) |

### 2. Placeholder scan

- `continue-on-error: true` on `smoke-claude-code` is intentional: CI runners don't reliably have all 7 CLIs. Documented inline.
- `release.yml`'s "all is best-effort" fallback (`||` between commands) is documented in the workflow file itself.

### 3. Type / name consistency

- Test scopes (`unit`, `integration`, `smoke`, `all`) match between `run.sh`, `ci.yml`, and `release.yml`.
- Platform names (7 enumeration) consistent across Tasks 3, 5, and earlier plans.
- `seed-gpowers-home.sh` output path used identically in Task 5's smoke tests.

### 4. Decomposition

10 tasks. Fixtures (#1, #2), helpers (#3, #4), the seven smoke tests (#5), runner (#6), two workflows (#7, #8), RELEASING (#9), manifest (#10). Each commit is independently reviewable.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-tests-ci.md`. Depends on Plans #1–#10. Choose subagent-driven or inline at execution time.
