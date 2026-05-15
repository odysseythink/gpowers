# gpowers Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the gpowers repository skeleton, install/uninstall scripts, runtime directory layout, and `gpowers-path` helper — the foundation that all later modules (core/roles/tools/business) depend on.

**Architecture:** Single git repo. Bash CLI scripts (POSIX-compliant) with Windows polyglot wrappers borrowed from superpowers. Single source of truth at `~/.gpowers/`; per-platform exposure via symlinks. Runtime data split between global `~/.gpowers/{config,state,cache,data,analytics,logs,tmp}` and per-project `<repo>/.gpowers/{plans,designs,evals,sessions,...}`. All path lookups must go through `gpowers-path` — no direct `~/.gpowers` string-concat in skills.

**Tech Stack:** Bash 4+ (POSIX where possible), `bats-core` for tests, `jq` for JSON, `realpath`/`readlink` for path normalization. No runtime dependencies on Bun/Node — that comes in later plans for browser-driver/tools.

**Spec reference:** `docs/superpowers/specs/2026-05-14-gpowers-merge-design.md` §1 (overall arch), §5 (install), §7 (runtime layout).

---

## File Structure

Files this plan creates:

```
gpowers/                                  ← new repo root (current working dir)
├── install                               main install script (Bash)
├── uninstall                             uninstall script
├── bin/
│   ├── gpowers                           top-level dispatcher (gpowers init|path|...)
│   ├── gpowers-path                      runtime path resolver (sourced by skills)
│   ├── gpowers-init                      `gpowers init` command implementation
│   ├── gpowers-detect-platforms          detect installed AI CLI platforms
│   └── _gpowers-lib.sh                   shared shell utilities
├── lib/
│   ├── runtime-dirs.sh                   defines GPOWERS_* env vars with defaults
│   ├── platform-paths.sh                 per-platform plugin-dir lookup table
│   └── manifest.sh                       read/write manifest.json
├── templates/
│   ├── project-gitignore                 template for <repo>/.gpowers/.gitignore
│   ├── project-readme.md                 template for <repo>/.gpowers/README.md
│   └── manifest.template.json            empty manifest
├── core/.placeholder                     empty placeholder (filled by plan-02)
├── roles/.placeholder                    (filled by plan-06)
├── tools/.placeholder                    (filled by plan-04/05)
├── business/.placeholder                 (filled by plan-07)
├── platforms/.placeholder                (filled by plan-08)
├── manifest.json                         shipped initial manifest
├── upstream-sources.json                 shipped initial upstream-sources
├── tests/
│   ├── unit/
│   │   ├── install.bats                  install script tests
│   │   ├── uninstall.bats
│   │   ├── gpowers-path.bats             path resolution tests
│   │   ├── gpowers-init.bats             gpowers init tests
│   │   └── platform-detect.bats
│   ├── helpers/
│   │   ├── setup.bash                    test setup (mock $HOME)
│   │   └── teardown.bash
│   └── fixtures/
│       ├── empty-repo/                   git repo with no .gpowers
│       └── initialized-repo/             git repo with .gpowers initialized
├── LICENSE                               MIT (matching gstack)
├── README.md                             one-paragraph + install instructions
└── .gitignore                            ignore tests/.tmp, etc.
```

Each script is a single-responsibility unit. `gpowers-path` is the most subtle — it must work both as a sourced library (for skills' Preamble) and as a CLI (`$(gpowers-path data plans)`).

---

## Task 1: Initialize Repository Skeleton

**Files:**
- Create: `LICENSE`
- Create: `README.md`
- Create: `.gitignore`
- Create: `core/.placeholder`, `roles/.placeholder`, `tools/.placeholder`, `business/.placeholder`, `platforms/.placeholder`
- Create: `manifest.json`, `upstream-sources.json`
- Create: `tests/unit/.gitkeep`, `tests/helpers/.gitkeep`, `tests/fixtures/.gitkeep`

- [ ] **Step 1.1: Write a test that asserts the skeleton exists**

Create `tests/unit/skeleton.bats`:

```bash
#!/usr/bin/env bats

@test "repo has all top-level module placeholders" {
  for dir in core roles tools business platforms; do
    [ -e "${BATS_TEST_DIRNAME}/../../${dir}/.placeholder" ]
  done
}

@test "repo has LICENSE, README, manifest, upstream-sources" {
  [ -f "${BATS_TEST_DIRNAME}/../../LICENSE" ]
  [ -f "${BATS_TEST_DIRNAME}/../../README.md" ]
  [ -f "${BATS_TEST_DIRNAME}/../../manifest.json" ]
  [ -f "${BATS_TEST_DIRNAME}/../../upstream-sources.json" ]
}

@test "manifest.json is valid JSON with required fields" {
  result=$(jq -r '.version, .installed_modules | type' "${BATS_TEST_DIRNAME}/../../manifest.json")
  [ "$(echo "$result" | head -1)" != "null" ]
  [ "$(echo "$result" | tail -1)" = "array" ]
}
```

- [ ] **Step 1.2: Run the test to verify it fails**

Run: `bats tests/unit/skeleton.bats`
Expected: FAIL — files don't exist yet.

- [ ] **Step 1.3: Create the skeleton files**

```bash
# Module placeholders
for d in core roles tools business platforms; do
  mkdir -p "$d"
  echo "Placeholder — populated by later plans." > "$d/.placeholder"
done

# Test directories
mkdir -p tests/unit tests/helpers tests/fixtures
touch tests/unit/.gitkeep tests/helpers/.gitkeep tests/fixtures/.gitkeep
```

Create `manifest.json`:

```json
{
  "version": "0.0.1",
  "schema_version": 1,
  "installed_modules": [],
  "installed_at": null,
  "install_location": null
}
```

Create `upstream-sources.json`:

```json
{
  "schema_version": 1,
  "modules": {
    "core": {
      "upstream": "https://github.com/obra/superpowers",
      "branch": "main",
      "subtree_prefix": "core",
      "last_sha": null
    },
    "roles": {
      "upstream": "https://github.com/garrytan/gstack",
      "branch": "main",
      "subtree_prefix": "roles",
      "last_sha": null,
      "skill_subset": ["office-hours", "plan-ceo-review", "plan-eng-review", "plan-design-review", "plan-devex-review", "design-consultation", "design-shotgun", "design-html", "design-review", "devex-review", "pr-review", "cso", "codex", "retro", "document-release", "investigate", "learn", "autoplan", "pair-agent", "plan-tune"]
    },
    "tools": {
      "upstream": "https://github.com/garrytan/gstack",
      "branch": "main",
      "subtree_prefix": "tools",
      "last_sha": null
    },
    "business": {
      "upstream": "https://github.com/garrytan/gstack",
      "branch": "main",
      "subtree_prefix": "business",
      "last_sha": null
    }
  }
}
```

Create `LICENSE` with standard MIT text (Copyright 2026 George Ran Wei).

Create `README.md`:

```markdown
# gpowers

Unified distribution combining [gstack](https://github.com/garrytan/gstack)'s
23 role-based commands with [superpowers](https://github.com/obra/superpowers)'s
14 methodology skills.

## Install

```bash
git clone https://github.com/<your>/gpowers ~/.gpowers/repo && \
  ~/.gpowers/repo/install
```

See `docs/superpowers/specs/2026-05-14-gpowers-merge-design.md` for design.
```

Create `.gitignore`:

```
tests/.tmp/
*.local.*
.DS_Store
```

- [ ] **Step 1.4: Run the test to verify it passes**

Run: `bats tests/unit/skeleton.bats`
Expected: 3 passing.

- [ ] **Step 1.5: Commit**

```bash
git add LICENSE README.md .gitignore manifest.json upstream-sources.json \
  core/.placeholder roles/.placeholder tools/.placeholder business/.placeholder platforms/.placeholder \
  tests/unit/.gitkeep tests/helpers/.gitkeep tests/fixtures/.gitkeep \
  tests/unit/skeleton.bats
git commit -m "foundation: initialize gpowers repository skeleton"
```

---

## Task 2: Define Runtime Directory Environment Variables

**Files:**
- Create: `lib/runtime-dirs.sh`
- Test: `tests/unit/runtime-dirs.bats`

This script is sourced by every CLI and by every skill's Preamble. It MUST be idempotent (sourcing twice is safe) and MUST not change current `cwd`.

- [ ] **Step 2.1: Write the failing test**

Create `tests/unit/runtime-dirs.bats`:

```bash
#!/usr/bin/env bats

load ../helpers/setup

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  unset GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
  unset GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP
  unset GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
}

teardown() {
  rm -rf "$HOME"
}

@test "defaults: GPOWERS_HOME = \$HOME/.gpowers" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_HOME" = "$HOME/.gpowers" ]
}

@test "defaults: GPOWERS_CONFIG = \$GPOWERS_HOME/config" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "$HOME/.gpowers/config" ]
}

@test "override: GPOWERS_HOME=/custom honored" {
  export GPOWERS_HOME=/custom
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_HOME" = "/custom" ]
  [ "$GPOWERS_CONFIG" = "/custom/config" ]
}

@test "override: GPOWERS_CONFIG independent of HOME" {
  export GPOWERS_CONFIG=/etc/gpowers
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "/etc/gpowers" ]
  [ "$GPOWERS_HOME" = "$HOME/.gpowers" ]
}

@test "all 7 dirs defined: HOME CONFIG STATE CACHE DATA ANALYTICS TMP" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  for var in GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP; do
    [ -n "${!var}" ]
  done
}

@test "sourcing twice is idempotent" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  first="$GPOWERS_CONFIG"
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "$first" ]
}

@test "sourcing does not change cwd" {
  cd "$BATS_TMPDIR"
  before="$(pwd)"
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$(pwd)" = "$before" ]
}
```

Create `tests/helpers/setup.bash`:

```bash
# Shared bats setup helpers.
# Currently empty; tests source this for symmetry with teardown.
:
```

- [ ] **Step 2.2: Run the test to verify it fails**

Run: `bats tests/unit/runtime-dirs.bats`
Expected: FAIL — `lib/runtime-dirs.sh` does not exist.

- [ ] **Step 2.3: Implement `lib/runtime-dirs.sh`**

```bash
#!/usr/bin/env bash
# lib/runtime-dirs.sh — Defines GPOWERS_* runtime directory env vars.
# Sourced by every CLI and skill. Idempotent. Does not change cwd.

: "${GPOWERS_HOME:=${HOME}/.gpowers}"
: "${GPOWERS_CONFIG:=${GPOWERS_HOME}/config}"
: "${GPOWERS_STATE:=${GPOWERS_HOME}/state}"
: "${GPOWERS_CACHE:=${GPOWERS_HOME}/cache}"
: "${GPOWERS_DATA:=${GPOWERS_HOME}/data}"
: "${GPOWERS_ANALYTICS:=${GPOWERS_HOME}/analytics}"
: "${GPOWERS_TMP:=${GPOWERS_HOME}/tmp}"

export GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
export GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP
```

- [ ] **Step 2.4: Run the test to verify it passes**

Run: `bats tests/unit/runtime-dirs.bats`
Expected: 7 passing.

- [ ] **Step 2.5: Commit**

```bash
git add lib/runtime-dirs.sh tests/unit/runtime-dirs.bats tests/helpers/setup.bash
git commit -m "foundation: define GPOWERS_* runtime directory env vars with override support"
```

---

## Task 3: Implement `gpowers-path` Helper — Global Mode

**Files:**
- Create: `bin/gpowers-path`
- Create: `bin/_gpowers-lib.sh`
- Test: `tests/unit/gpowers-path.bats`

`gpowers-path` resolves runtime paths. Used two ways:
- CLI: `gpowers-path config` prints `$GPOWERS_CONFIG`
- CLI: `gpowers-path cache models` prints `$GPOWERS_CACHE/models`
- CLI: `gpowers-path project plans ceo` prints `<repo>/.gpowers/plans/ceo` (project-mode, Task 5)

This task implements only global-mode (config, state, cache, data, analytics, tmp, home). Project-mode comes in Task 5.

- [ ] **Step 3.1: Write the failing test**

Create `tests/unit/gpowers-path.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  unset GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
  unset GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP
}

teardown() {
  rm -rf "$HOME"
}

PATH_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-path"

@test "gpowers-path home prints \$GPOWERS_HOME" {
  result="$("$PATH_BIN" home)"
  [ "$result" = "$HOME/.gpowers" ]
}

@test "gpowers-path config prints \$GPOWERS_CONFIG" {
  result="$("$PATH_BIN" config)"
  [ "$result" = "$HOME/.gpowers/config" ]
}

@test "gpowers-path config compact-rules joins subpaths" {
  result="$("$PATH_BIN" config compact-rules)"
  [ "$result" = "$HOME/.gpowers/config/compact-rules" ]
}

@test "gpowers-path cache models joins subpaths" {
  result="$("$PATH_BIN" cache models)"
  [ "$result" = "$HOME/.gpowers/cache/models" ]
}

@test "gpowers-path with multiple subpaths joins them" {
  result="$("$PATH_BIN" state security attempts)"
  [ "$result" = "$HOME/.gpowers/state/security/attempts" ]
}

@test "gpowers-path with unknown kind exits 2" {
  run "$PATH_BIN" unknown
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown kind"* ]]
}

@test "gpowers-path with no args exits 2 and prints usage" {
  run "$PATH_BIN"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage:"* ]]
}

@test "GPOWERS_HOME override is honored" {
  export GPOWERS_HOME=/custom/gp
  result="$("$PATH_BIN" config)"
  [ "$result" = "/custom/gp/config" ]
}
```

- [ ] **Step 3.2: Run the test to verify it fails**

Run: `bats tests/unit/gpowers-path.bats`
Expected: FAIL — `bin/gpowers-path` does not exist.

- [ ] **Step 3.3: Implement `bin/_gpowers-lib.sh`**

```bash
#!/usr/bin/env bash
# bin/_gpowers-lib.sh — Shared utilities for gpowers CLIs.

gpowers_die() {
  printf 'gpowers: %s\n' "$1" >&2
  exit "${2:-1}"
}

# Resolve the directory containing the calling script, following symlinks.
gpowers_script_dir() {
  local src="$1"
  while [ -L "$src" ]; do
    local dir
    dir="$(cd -P "$(dirname "$src")" && pwd)"
    src="$(readlink "$src")"
    case "$src" in
      /*) ;;
      *) src="$dir/$src" ;;
    esac
  done
  cd -P "$(dirname "$src")" && pwd
}

# Join path components with single slashes, stripping trailing slashes.
gpowers_path_join() {
  local out="$1"
  shift
  for part in "$@"; do
    out="${out%/}/${part#/}"
  done
  printf '%s\n' "$out"
}
```

- [ ] **Step 3.4: Implement `bin/gpowers-path`**

```bash
#!/usr/bin/env bash
# bin/gpowers-path — Resolve runtime data paths.
# Usage:
#   gpowers-path <kind> [subpath ...]
# Kinds (global mode): home config state cache data analytics tmp
# Kind (project mode): project [subpath ...]   (implemented in Task 5)

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_gpowers-lib.sh
. "$SCRIPT_DIR/_gpowers-lib.sh"
# shellcheck source=../lib/runtime-dirs.sh
. "$SCRIPT_DIR/../lib/runtime-dirs.sh"

usage() {
  cat <<'EOF' >&2
usage: gpowers-path <kind> [subpath ...]

Global kinds:
  home       -> $GPOWERS_HOME
  config     -> $GPOWERS_CONFIG
  state      -> $GPOWERS_STATE
  cache      -> $GPOWERS_CACHE
  data       -> $GPOWERS_DATA
  analytics  -> $GPOWERS_ANALYTICS
  tmp        -> $GPOWERS_TMP

Project kind (resolves to <repo>/.gpowers when in a project, else falls back to global data):
  project [subpath ...]
EOF
}

if [ $# -lt 1 ]; then
  usage
  exit 2
fi

kind="$1"
shift

case "$kind" in
  home)      base="$GPOWERS_HOME" ;;
  config)    base="$GPOWERS_CONFIG" ;;
  state)     base="$GPOWERS_STATE" ;;
  cache)     base="$GPOWERS_CACHE" ;;
  data)      base="$GPOWERS_DATA" ;;
  analytics) base="$GPOWERS_ANALYTICS" ;;
  tmp)       base="$GPOWERS_TMP" ;;
  project)
    gpowers_die "project mode not implemented yet (Task 5)" 3
    ;;
  *)
    gpowers_die "unknown kind: $kind" 2
    ;;
esac

if [ $# -eq 0 ]; then
  printf '%s\n' "$base"
else
  gpowers_path_join "$base" "$@"
fi
```

Make executable: `chmod +x bin/gpowers-path bin/_gpowers-lib.sh`.

- [ ] **Step 3.5: Run the test to verify it passes**

Run: `bats tests/unit/gpowers-path.bats`
Expected: 8 passing.

- [ ] **Step 3.6: Commit**

```bash
git add bin/gpowers-path bin/_gpowers-lib.sh tests/unit/gpowers-path.bats
chmod +x bin/gpowers-path
git update-index --chmod=+x bin/gpowers-path
git commit -m "foundation: implement gpowers-path helper for global runtime paths"
```

---

## Task 4: Project Root Detection

**Files:**
- Modify: `lib/runtime-dirs.sh` — add `GPOWERS_PROJECT_DIR` and `GPOWERS_PROJECT_DATA` detection
- Test: `tests/unit/project-detect.bats`

Per spec §7, project root detection priority:
1. `GPOWERS_PROJECT_DIR` env var (explicit)
2. Walk up from `cwd` looking for `.gpowers/` directory
3. Walk up from `cwd` looking for `.git` directory
4. Fallback: empty (signals "no project, use global")

- [ ] **Step 4.1: Write the failing test**

Create `tests/unit/project-detect.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  TESTREPO="${BATS_TMPDIR}/testrepo-$$"
  mkdir -p "$TESTREPO/sub/nested"
  unset GPOWERS_HOME GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

DETECT="${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"

@test "GPOWERS_PROJECT_DIR override is honored verbatim" {
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd /tmp
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
  [ "$GPOWERS_PROJECT_DATA" = "$TESTREPO/.gpowers" ]
}

@test "detects .gpowers/ directory by walking up" {
  mkdir -p "$TESTREPO/.gpowers"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
}

@test "detects .git/ directory by walking up if no .gpowers" {
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
}

@test ".gpowers/ takes priority over .git" {
  mkdir -p "$TESTREPO/.git"
  mkdir -p "$TESTREPO/sub/.gpowers"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO/sub" ]
}

@test "no project marker means empty PROJECT_DIR" {
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ -z "$GPOWERS_PROJECT_DIR" ]
  [ -z "$GPOWERS_PROJECT_DATA" ]
}

@test "PROJECT_DATA is PROJECT_DIR/.gpowers when project detected" {
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DATA" = "$TESTREPO/.gpowers" ]
}
```

- [ ] **Step 4.2: Run the test to verify it fails**

Run: `bats tests/unit/project-detect.bats`
Expected: FAIL — `GPOWERS_PROJECT_DIR` not set by `runtime-dirs.sh`.

- [ ] **Step 4.3: Extend `lib/runtime-dirs.sh`**

Append to `lib/runtime-dirs.sh`:

```bash
# Project root detection.
# Priority: $GPOWERS_PROJECT_DIR env, then walk up cwd for .gpowers, then .git, else empty.
gpowers_detect_project_dir() {
  if [ -n "${GPOWERS_PROJECT_DIR:-}" ]; then
    printf '%s\n' "$GPOWERS_PROJECT_DIR"
    return 0
  fi
  local dir="$PWD"
  # Phase 1: look for .gpowers/
  while [ "$dir" != "/" ] && [ -n "$dir" ]; do
    if [ -d "$dir/.gpowers" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  # Phase 2: look for .git/
  dir="$PWD"
  while [ "$dir" != "/" ] && [ -n "$dir" ]; do
    if [ -e "$dir/.git" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

if _detected="$(gpowers_detect_project_dir)"; then
  GPOWERS_PROJECT_DIR="$_detected"
  GPOWERS_PROJECT_DATA="${GPOWERS_PROJECT_DATA:-${GPOWERS_PROJECT_DIR}/.gpowers}"
  export GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
fi
unset _detected
```

- [ ] **Step 4.4: Run the test to verify it passes**

Run: `bats tests/unit/project-detect.bats`
Expected: 6 passing.

- [ ] **Step 4.5: Also re-run runtime-dirs tests to confirm no regression**

Run: `bats tests/unit/runtime-dirs.bats`
Expected: 7 passing (unchanged).

- [ ] **Step 4.6: Commit**

```bash
git add lib/runtime-dirs.sh tests/unit/project-detect.bats
git commit -m "foundation: detect project root via .gpowers, .git, or env override"
```

---

## Task 5: `gpowers-path project` Subcommand

**Files:**
- Modify: `bin/gpowers-path` — implement the `project` kind
- Test: extend `tests/unit/gpowers-path.bats`

Project mode resolves to `<repo>/.gpowers/<subpath>` when a project is detected, otherwise falls back to `$GPOWERS_DATA/<subpath>` (so callers always get a usable path).

- [ ] **Step 5.1: Add failing tests**

Append to `tests/unit/gpowers-path.bats`:

```bash
@test "gpowers-path project resolves to repo/.gpowers when in project" {
  TESTREPO="${BATS_TMPDIR}/proj-$$"
  mkdir -p "$TESTREPO/.git" "$TESTREPO/sub"
  cd "$TESTREPO/sub"
  result="$("$PATH_BIN" project)"
  [ "$result" = "$TESTREPO/.gpowers" ]
}

@test "gpowers-path project plans ceo joins subpath in project mode" {
  TESTREPO="${BATS_TMPDIR}/proj2-$$"
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO"
  result="$("$PATH_BIN" project plans ceo)"
  [ "$result" = "$TESTREPO/.gpowers/plans/ceo" ]
}

@test "gpowers-path project falls back to global data when no project" {
  cd "$BATS_TMPDIR"
  result="$("$PATH_BIN" project sessions)"
  [ "$result" = "$HOME/.gpowers/data/sessions" ]
}

@test "GPOWERS_PROJECT_DIR override is honored by project kind" {
  TESTREPO="${BATS_TMPDIR}/proj3-$$"
  mkdir -p "$TESTREPO"
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd "$BATS_TMPDIR"
  result="$("$PATH_BIN" project plans)"
  [ "$result" = "$TESTREPO/.gpowers/plans" ]
}
```

- [ ] **Step 5.2: Run to verify the new cases fail**

Run: `bats tests/unit/gpowers-path.bats`
Expected: 8 passing, 4 failing (the new project cases).

- [ ] **Step 5.3: Replace the project branch in `bin/gpowers-path`**

In `bin/gpowers-path`, replace this block:

```bash
  project)
    gpowers_die "project mode not implemented yet (Task 5)" 3
    ;;
```

with:

```bash
  project)
    if [ -n "${GPOWERS_PROJECT_DATA:-}" ]; then
      base="$GPOWERS_PROJECT_DATA"
    else
      base="$GPOWERS_DATA"
    fi
    ;;
```

- [ ] **Step 5.4: Run all gpowers-path tests**

Run: `bats tests/unit/gpowers-path.bats`
Expected: 12 passing.

- [ ] **Step 5.5: Commit**

```bash
git add bin/gpowers-path tests/unit/gpowers-path.bats
git commit -m "foundation: implement gpowers-path project mode with global fallback"
```

---

## Task 6: Platform Path Lookup Table

**Files:**
- Create: `lib/platform-paths.sh`
- Create: `bin/gpowers-detect-platforms`
- Test: `tests/unit/platform-detect.bats`

Defines where each of the 7 supported platforms looks for plugins/skills. `gpowers-detect-platforms` prints the platforms whose CLI binary or config directory is present.

- [ ] **Step 6.1: Write the failing test**

Create `tests/unit/platform-detect.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  export PATH="${BATS_TMPDIR}/fakepath-$$:$PATH"
  mkdir -p "${BATS_TMPDIR}/fakepath-$$"
}

teardown() {
  rm -rf "$HOME" "${BATS_TMPDIR}/fakepath-$$"
}

DETECT_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-detect-platforms"

@test "detects claude-code via ~/.claude directory" {
  mkdir -p "$HOME/.claude"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"claude-code"* ]]
}

@test "detects codex via ~/.codex directory" {
  mkdir -p "$HOME/.codex"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"codex"* ]]
}

@test "detects kimi via ~/.kimi directory" {
  mkdir -p "$HOME/.kimi"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"kimi"* ]]
}

@test "detects gemini via ~/.config/gemini directory" {
  mkdir -p "$HOME/.config/gemini"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"gemini"* ]]
}

@test "no platforms detected when no markers" {
  result="$("$DETECT_BIN")"
  [ -z "$result" ]
}

@test "lib/platform-paths.sh defines lookup table" {
  source "${BATS_TEST_DIRNAME}/../../lib/platform-paths.sh"
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_claude_code:-}" ]
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_codex:-}" ]
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_kimi:-}" ]
}
```

- [ ] **Step 6.2: Run the test to verify it fails**

Run: `bats tests/unit/platform-detect.bats`
Expected: FAIL — files do not exist.

- [ ] **Step 6.3: Implement `lib/platform-paths.sh`**

```bash
#!/usr/bin/env bash
# lib/platform-paths.sh — Per-platform plugin/skill directory lookup.
# Sourced by install/uninstall. Idempotent.

GPOWERS_PLATFORM_PLUGIN_DIR_claude_code="${HOME}/.claude/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_codex="${HOME}/.codex/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_gemini="${HOME}/.config/gemini/extensions/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_cursor="${HOME}/.cursor/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_opencode="${HOME}/.config/opencode/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_copilot="${HOME}/.config/copilot-cli/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_kimi="${HOME}/.kimi/skills"  # uses prefix mode, not symlink

# Marker directories used to detect "platform is installed on this machine".
GPOWERS_PLATFORM_MARKER_claude_code="${HOME}/.claude"
GPOWERS_PLATFORM_MARKER_codex="${HOME}/.codex"
GPOWERS_PLATFORM_MARKER_gemini="${HOME}/.config/gemini"
GPOWERS_PLATFORM_MARKER_cursor="${HOME}/.cursor"
GPOWERS_PLATFORM_MARKER_opencode="${HOME}/.config/opencode"
GPOWERS_PLATFORM_MARKER_copilot="${HOME}/.config/copilot-cli"
GPOWERS_PLATFORM_MARKER_kimi="${HOME}/.kimi"

GPOWERS_ALL_PLATFORMS="claude_code codex gemini cursor opencode copilot kimi"

export GPOWERS_ALL_PLATFORMS \
  GPOWERS_PLATFORM_PLUGIN_DIR_claude_code GPOWERS_PLATFORM_PLUGIN_DIR_codex \
  GPOWERS_PLATFORM_PLUGIN_DIR_gemini GPOWERS_PLATFORM_PLUGIN_DIR_cursor \
  GPOWERS_PLATFORM_PLUGIN_DIR_opencode GPOWERS_PLATFORM_PLUGIN_DIR_copilot \
  GPOWERS_PLATFORM_PLUGIN_DIR_kimi \
  GPOWERS_PLATFORM_MARKER_claude_code GPOWERS_PLATFORM_MARKER_codex \
  GPOWERS_PLATFORM_MARKER_gemini GPOWERS_PLATFORM_MARKER_cursor \
  GPOWERS_PLATFORM_MARKER_opencode GPOWERS_PLATFORM_MARKER_copilot \
  GPOWERS_PLATFORM_MARKER_kimi
```

- [ ] **Step 6.4: Implement `bin/gpowers-detect-platforms`**

```bash
#!/usr/bin/env bash
# bin/gpowers-detect-platforms — Print platforms with markers present.
# Output: one platform name per line (claude_code printed as claude-code).

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/platform-paths.sh
. "$SCRIPT_DIR/../lib/platform-paths.sh"

for platform in $GPOWERS_ALL_PLATFORMS; do
  marker_var="GPOWERS_PLATFORM_MARKER_${platform}"
  marker="${!marker_var:-}"
  if [ -n "$marker" ] && [ -d "$marker" ]; then
    # Print with dash form for user output
    printf '%s\n' "${platform//_/-}"
  fi
done
```

Make executable: `chmod +x bin/gpowers-detect-platforms`.

- [ ] **Step 6.5: Run the test to verify it passes**

Run: `bats tests/unit/platform-detect.bats`
Expected: 6 passing.

- [ ] **Step 6.6: Commit**

```bash
git add lib/platform-paths.sh bin/gpowers-detect-platforms tests/unit/platform-detect.bats
chmod +x bin/gpowers-detect-platforms
git update-index --chmod=+x bin/gpowers-detect-platforms
git commit -m "foundation: define platform plugin-dir lookup + detection"
```

---

## Task 7: Implement `install` — Argument Parsing & Defaults

**Files:**
- Create: `install`
- Test: `tests/unit/install.bats`

This task implements only argument parsing and a `--dry-run` mode that prints the plan without executing. Actual install action comes in Task 8.

- [ ] **Step 7.1: Write the failing test**

Create `tests/unit/install.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
}

teardown() {
  rm -rf "$HOME"
}

INSTALL="${BATS_TEST_DIRNAME}/../../install"

@test "install --help exits 0 with usage" {
  run "$INSTALL" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"usage:"* ]]
}

@test "install --dry-run --core-only prints intended actions" {
  run "$INSTALL" --dry-run --core-only
  [ "$status" -eq 0 ]
  [[ "$output" == *"core"* ]]
  [[ "$output" != *"business"* ]]
}

@test "install --dry-run --with-business includes business" {
  run "$INSTALL" --dry-run --with-business
  [ "$status" -eq 0 ]
  [[ "$output" == *"business"* ]]
}

@test "install --dry-run --no-tools skips tools" {
  run "$INSTALL" --dry-run --no-tools
  [ "$status" -eq 0 ]
  [[ "$output" != *"link tools"* ]]
}

@test "install --dry-run --location=/tmp/custom uses custom location" {
  run "$INSTALL" --dry-run --location=/tmp/custom
  [ "$status" -eq 0 ]
  [[ "$output" == *"/tmp/custom"* ]]
}

@test "install --dry-run --platforms=claude-code,kimi restricts platforms" {
  mkdir -p "$HOME/.claude" "$HOME/.kimi" "$HOME/.codex"
  run "$INSTALL" --dry-run --platforms=claude-code,kimi
  [ "$status" -eq 0 ]
  [[ "$output" == *"claude-code"* ]]
  [[ "$output" == *"kimi"* ]]
  [[ "$output" != *"codex"* ]]
}

@test "install --unknown-flag exits non-zero" {
  run "$INSTALL" --unknown-flag
  [ "$status" -ne 0 ]
}
```

- [ ] **Step 7.2: Run the test to verify it fails**

Run: `bats tests/unit/install.bats`
Expected: FAIL — `install` script does not exist.

- [ ] **Step 7.3: Implement `install`**

```bash
#!/usr/bin/env bash
# install — Install gpowers onto this machine.
# Resolves a target install location, plans which modules + platforms to wire up,
# then executes the plan (or prints it under --dry-run).

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/runtime-dirs.sh
. "$SCRIPT_DIR/lib/runtime-dirs.sh"
# shellcheck source=lib/platform-paths.sh
. "$SCRIPT_DIR/lib/platform-paths.sh"
# shellcheck source=bin/_gpowers-lib.sh
. "$SCRIPT_DIR/bin/_gpowers-lib.sh"

usage() {
  cat <<'EOF'
usage: gpowers install [options]

Options:
  --core-only          install only core/ (no roles, tools, business)
  --with-business      include business/ module (default: off)
  --no-tools           skip tools/ module
  --no-roles           skip roles/ module
  --platforms=LIST     comma-separated subset of platforms to register
                       valid: claude-code,codex,gemini,cursor,opencode,copilot,kimi
                       default: all detected
  --location=PATH      install location (default: $GPOWERS_HOME)
  --link               symlink source repo into location (dev mode)
  --dry-run            print planned actions, do not execute
  --help               show this message
EOF
}

# Defaults
opt_modules=(core roles tools)
opt_with_business=0
opt_no_tools=0
opt_no_roles=0
opt_platforms=""
opt_location="$GPOWERS_HOME"
opt_link=0
opt_dry_run=0

while [ $# -gt 0 ]; do
  case "$1" in
    --help) usage; exit 0 ;;
    --core-only) opt_modules=(core); opt_with_business=0; opt_no_tools=1; opt_no_roles=1 ;;
    --with-business) opt_with_business=1 ;;
    --no-tools) opt_no_tools=1 ;;
    --no-roles) opt_no_roles=1 ;;
    --platforms=*) opt_platforms="${1#--platforms=}" ;;
    --location=*) opt_location="${1#--location=}" ;;
    --link) opt_link=1 ;;
    --dry-run) opt_dry_run=1 ;;
    *) gpowers_die "unknown flag: $1" 2 ;;
  esac
  shift
done

# Compute final module set
final_modules=(core)
[ "$opt_no_roles" -eq 0 ] && final_modules+=(roles)
[ "$opt_no_tools" -eq 0 ] && final_modules+=(tools)
[ "$opt_with_business" -eq 1 ] && final_modules+=(business)
# If --core-only, override
if [ ${#opt_modules[@]} -eq 1 ] && [ "${opt_modules[0]}" = "core" ]; then
  final_modules=(core)
fi

# Compute final platform set
detected_platforms="$("$SCRIPT_DIR/bin/gpowers-detect-platforms" || true)"
if [ -n "$opt_platforms" ]; then
  IFS=',' read -r -a requested <<< "$opt_platforms"
  final_platforms=()
  for p in "${requested[@]}"; do
    final_platforms+=("$p")
  done
else
  final_platforms=()
  while IFS= read -r p; do
    [ -n "$p" ] && final_platforms+=("$p")
  done <<< "$detected_platforms"
fi

# Print plan
plan_line() {
  if [ "$opt_dry_run" -eq 1 ]; then
    printf '[plan] %s\n' "$1"
  else
    printf '[install] %s\n' "$1"
  fi
}

plan_line "install location: $opt_location"
plan_line "modules: ${final_modules[*]}"
plan_line "platforms: ${final_platforms[*]:-<none detected>}"
[ "$opt_link" -eq 1 ] && plan_line "mode: symlink (dev)"

if [ "$opt_no_tools" -eq 0 ]; then
  plan_line "link tools to platforms"
fi
for m in "${final_modules[@]}"; do
  plan_line "stage module: $m"
done
for p in "${final_platforms[@]}"; do
  plan_line "register platform: $p"
done

if [ "$opt_dry_run" -eq 1 ]; then
  exit 0
fi

gpowers_die "actual install action not implemented yet (Task 8)" 3
```

Make executable: `chmod +x install`.

- [ ] **Step 7.4: Run the test to verify it passes**

Run: `bats tests/unit/install.bats`
Expected: 7 passing.

- [ ] **Step 7.5: Commit**

```bash
git add install tests/unit/install.bats
chmod +x install
git update-index --chmod=+x install
git commit -m "foundation: install script argument parsing + dry-run plan"
```

---

## Task 8: Implement `install` — Execute the Plan

**Files:**
- Modify: `install`
- Test: extend `tests/unit/install.bats`

Replace the "not implemented yet" stub with the actual install:
1. Create `~/.gpowers/` (or `--location`) and copy/symlink source repo there
2. Create runtime dirs (`config state cache data analytics tmp`)
3. Update `manifest.json` with `installed_at`, `installed_modules`, `install_location`
4. For each platform, create plugin-dir symlink pointing to install location

- [ ] **Step 8.1: Add failing tests**

Append to `tests/unit/install.bats`:

```bash
@test "real install creates ~/.gpowers/ with module dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-real-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  [ -d "$HOME/.gpowers/core" ]
  [ -f "$HOME/.gpowers/manifest.json" ]
  rm -rf "$HOME"
}

@test "real install creates runtime dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-rt-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  for d in config state cache data analytics tmp; do
    [ -d "$HOME/.gpowers/$d" ]
  done
  rm -rf "$HOME"
}

@test "real install symlinks Claude Code plugin dir" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-cc-$$"
  mkdir -p "$HOME/.claude/plugins"
  export HOME
  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
  rm -rf "$HOME"
}

@test "real install updates manifest with installed_modules" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-mf-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  modules="$(jq -r '.installed_modules | join(",")' "$HOME/.gpowers/manifest.json")"
  [ "$modules" = "core" ]
  rm -rf "$HOME"
}

@test "real install with --with-business records business in manifest" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-biz-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --with-business
  [ "$status" -eq 0 ]
  modules="$(jq -r '.installed_modules | join(",")' "$HOME/.gpowers/manifest.json")"
  [[ "$modules" == *"business"* ]]
  rm -rf "$HOME"
}

@test "second install is idempotent (no error)" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-idem-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  "$REPO/install" --core-only
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  rm -rf "$HOME"
}
```

- [ ] **Step 8.2: Run to verify new cases fail**

Run: `bats tests/unit/install.bats`
Expected: 7 passing (parsing tests), 6 failing (real install).

- [ ] **Step 8.3: Implement `lib/manifest.sh`**

```bash
#!/usr/bin/env bash
# lib/manifest.sh — Read and write manifest.json.

gpowers_manifest_set_installed() {
  local manifest_path="$1"
  local location="$2"
  shift 2
  local modules=("$@")
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local modules_json
  modules_json="$(printf '%s\n' "${modules[@]}" | jq -R . | jq -s .)"
  local tmp
  tmp="$(mktemp)"
  jq --arg loc "$location" \
     --arg ts "$now" \
     --argjson mods "$modules_json" \
     '.installed_at = $ts | .install_location = $loc | .installed_modules = $mods' \
     "$manifest_path" > "$tmp"
  mv "$tmp" "$manifest_path"
}
```

- [ ] **Step 8.4: Replace the stub in `install`**

Replace the last block in `install`:

```bash
if [ "$opt_dry_run" -eq 1 ]; then
  exit 0
fi

gpowers_die "actual install action not implemented yet (Task 8)" 3
```

With:

```bash
if [ "$opt_dry_run" -eq 1 ]; then
  exit 0
fi

# shellcheck source=lib/manifest.sh
. "$SCRIPT_DIR/lib/manifest.sh"

# 1. Create install location
mkdir -p "$opt_location"

# 2. Stage source repo (copy or symlink)
if [ "$opt_link" -eq 1 ]; then
  # Symlink each top-level dir from source to install location
  for entry in core roles tools business platforms lib bin templates manifest.json upstream-sources.json install uninstall README.md LICENSE; do
    src="$SCRIPT_DIR/$entry"
    dst="$opt_location/$entry"
    if [ -e "$src" ]; then
      [ -e "$dst" ] || ln -s "$src" "$dst"
    fi
  done
else
  # Copy
  for entry in core roles tools business platforms lib bin templates manifest.json upstream-sources.json install uninstall README.md LICENSE; do
    src="$SCRIPT_DIR/$entry"
    if [ -e "$src" ]; then
      cp -R "$src" "$opt_location/" 2>/dev/null || true
    fi
  done
fi

# 3. Create runtime directories
for d in config state cache data analytics tmp logs; do
  mkdir -p "$opt_location/$d"
done

# 4. Update manifest.json
gpowers_manifest_set_installed \
  "$opt_location/manifest.json" \
  "$opt_location" \
  "${final_modules[@]}"

# 5. Register each detected platform
for p in "${final_platforms[@]}"; do
  platform_key="${p//-/_}"
  plugin_dir_var="GPOWERS_PLATFORM_PLUGIN_DIR_${platform_key}"
  plugin_dir="${!plugin_dir_var:-}"
  if [ -z "$plugin_dir" ]; then
    printf '[install] skip unknown platform: %s\n' "$p" >&2
    continue
  fi
  if [ "$platform_key" = "kimi" ]; then
    # Kimi uses prefix-mode (gpowers-* skill names); for foundation plan we
    # only ensure the parent directory exists. Per-skill adapter generation
    # is implemented in plan-08-platform-registration.
    mkdir -p "$plugin_dir"
    printf '[install] kimi parent dir ready: %s\n' "$plugin_dir"
  else
    parent="$(dirname "$plugin_dir")"
    mkdir -p "$parent"
    if [ -L "$plugin_dir" ]; then
      rm "$plugin_dir"
    elif [ -e "$plugin_dir" ]; then
      printf '[install] warn: %s exists and is not a symlink; skipping\n' "$plugin_dir" >&2
      continue
    fi
    ln -s "$opt_location" "$plugin_dir"
    printf '[install] linked %s -> %s\n' "$plugin_dir" "$opt_location"
  fi
done

printf '[install] done. location: %s\n' "$opt_location"
```

- [ ] **Step 8.5: Run all install tests**

Run: `bats tests/unit/install.bats`
Expected: 13 passing.

- [ ] **Step 8.6: Commit**

```bash
git add install lib/manifest.sh tests/unit/install.bats
git commit -m "foundation: install copies/links source, creates runtime dirs, registers platforms"
```

---

## Task 9: Implement `uninstall`

**Files:**
- Create: `uninstall`
- Test: `tests/unit/uninstall.bats`

Per spec §6/§7: by default keeps user data (`data/`, `config/`, `analytics/`, `logs/`, and per-project `<repo>/.gpowers/`); removes platform symlinks + module dirs + state/cache/tmp.

- [ ] **Step 9.1: Write the failing test**

Create `tests/unit/uninstall.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-un-$$"
  mkdir -p "$HOME/.claude" "$HOME/.kimi"
}

teardown() {
  rm -rf "$HOME"
}

@test "uninstall removes platform symlinks" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  [ -L "$HOME/.claude/plugins/gpowers" ]
  run "$REPO/uninstall" --dry-run
  [ "$status" -eq 0 ]
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -L "$HOME/.claude/plugins/gpowers" ]
}

@test "uninstall keeps user data by default" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "user content" > "$HOME/.gpowers/data/important.txt"
  echo "user config" > "$HOME/.gpowers/config/builder-profile"
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ -f "$HOME/.gpowers/data/important.txt" ]
  [ -f "$HOME/.gpowers/config/builder-profile" ]
}

@test "uninstall removes module dirs and state/cache/tmp" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "cache junk" > "$HOME/.gpowers/cache/junk.txt"
  echo "state junk" > "$HOME/.gpowers/state/junk.txt"
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -d "$HOME/.gpowers/core" ]
  [ ! -d "$HOME/.gpowers/cache" ]
  [ ! -d "$HOME/.gpowers/state" ]
  [ ! -d "$HOME/.gpowers/tmp" ]
}

@test "uninstall --remove-global-data also removes data dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "user content" > "$HOME/.gpowers/data/important.txt"
  run "$REPO/uninstall" --remove-global-data
  [ "$status" -eq 0 ]
  [ ! -d "$HOME/.gpowers/data" ]
  [ ! -d "$HOME/.gpowers/config" ]
}

@test "uninstall --dry-run prints actions and changes nothing" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  run "$REPO/uninstall" --dry-run
  [ "$status" -eq 0 ]
  [[ "$output" == *"would remove"* ]]
  [ -L "$HOME/.claude/plugins/gpowers" ]
}
```

- [ ] **Step 9.2: Run to verify it fails**

Run: `bats tests/unit/uninstall.bats`
Expected: FAIL — `uninstall` does not exist.

- [ ] **Step 9.3: Implement `uninstall`**

```bash
#!/usr/bin/env bash
# uninstall — Remove gpowers from this machine.
# Default keeps user data; --remove-global-data also removes data/config/analytics/logs.

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/runtime-dirs.sh
. "$SCRIPT_DIR/lib/runtime-dirs.sh"
# shellcheck source=lib/platform-paths.sh
. "$SCRIPT_DIR/lib/platform-paths.sh"
# shellcheck source=bin/_gpowers-lib.sh
. "$SCRIPT_DIR/bin/_gpowers-lib.sh"

usage() {
  cat <<'EOF'
usage: gpowers uninstall [options]

Options:
  --keep-data           keep ~/.gpowers/data/, config/, analytics/, logs/ (default)
  --remove-global-data  also remove data/config/analytics/logs
  --remove-all-data     also remove per-project <repo>/.gpowers/ (DANGEROUS: modifies git working trees)
  --platform=NAME       only unregister from one platform; do not touch ~/.gpowers/
  --dry-run             print actions, change nothing
  --help                show this message
EOF
}

opt_remove_global_data=0
opt_remove_all_data=0
opt_platform=""
opt_dry_run=0

while [ $# -gt 0 ]; do
  case "$1" in
    --help) usage; exit 0 ;;
    --keep-data) ;;  # default
    --remove-global-data) opt_remove_global_data=1 ;;
    --remove-all-data) opt_remove_all_data=1; opt_remove_global_data=1 ;;
    --platform=*) opt_platform="${1#--platform=}" ;;
    --dry-run) opt_dry_run=1 ;;
    *) gpowers_die "unknown flag: $1" 2 ;;
  esac
  shift
done

do_run() {
  if [ "$opt_dry_run" -eq 1 ]; then
    printf '[dry-run] would %s\n' "$1"
  else
    printf '[uninstall] %s\n' "$1"
    eval "$2"
  fi
}

# 1. Remove platform symlinks
platforms_to_clean="$GPOWERS_ALL_PLATFORMS"
if [ -n "$opt_platform" ]; then
  platforms_to_clean="${opt_platform//-/_}"
fi
for platform in $platforms_to_clean; do
  plugin_dir_var="GPOWERS_PLATFORM_PLUGIN_DIR_${platform}"
  plugin_dir="${!plugin_dir_var:-}"
  if [ -z "$plugin_dir" ]; then continue; fi
  if [ "$platform" = "kimi" ]; then
    # Remove gpowers-* prefixed entries in ~/.kimi/skills
    if [ -d "$plugin_dir" ]; then
      shopt -s nullglob
      for entry in "$plugin_dir"/gpowers-* "$plugin_dir"/gpowers; do
        do_run "remove $entry" "rm -rf '$entry'"
      done
      shopt -u nullglob
    fi
  else
    if [ -L "$plugin_dir" ] || [ -e "$plugin_dir" ]; then
      do_run "remove $plugin_dir" "rm -rf '$plugin_dir'"
    fi
  fi
done

# 2. If --platform was set, stop here
if [ -n "$opt_platform" ]; then
  exit 0
fi

# 3. Remove ~/.gpowers/ module dirs and ephemeral runtime dirs
for d in core roles tools business platforms bin lib templates state cache tmp; do
  path="$GPOWERS_HOME/$d"
  [ -e "$path" ] && do_run "remove $path" "rm -rf '$path'"
done

# Remove top-level shipped files
for f in install uninstall manifest.json upstream-sources.json README.md LICENSE; do
  path="$GPOWERS_HOME/$f"
  [ -e "$path" ] && do_run "remove $path" "rm -f '$path'"
done

# 4. Optionally remove data dirs
if [ "$opt_remove_global_data" -eq 1 ]; then
  for d in data config analytics logs; do
    path="$GPOWERS_HOME/$d"
    [ -e "$path" ] && do_run "remove $path" "rm -rf '$path'"
  done
  # If GPOWERS_HOME is now empty, remove it
  if [ -d "$GPOWERS_HOME" ] && [ -z "$(ls -A "$GPOWERS_HOME" 2>/dev/null)" ]; then
    do_run "remove empty $GPOWERS_HOME" "rmdir '$GPOWERS_HOME'"
  fi
fi

# 5. --remove-all-data: per-project <repo>/.gpowers/
if [ "$opt_remove_all_data" -eq 1 ]; then
  printf '[uninstall] note: --remove-all-data leaves per-project <repo>/.gpowers/ alone unless removed explicitly per-repo.\n' >&2
fi

if [ "$opt_dry_run" -eq 0 ]; then
  printf '[uninstall] done.\n'
fi
```

Make executable: `chmod +x uninstall`.

- [ ] **Step 9.4: Run tests**

Run: `bats tests/unit/uninstall.bats`
Expected: 5 passing.

- [ ] **Step 9.5: Commit**

```bash
git add uninstall tests/unit/uninstall.bats
chmod +x uninstall
git update-index --chmod=+x uninstall
git commit -m "foundation: uninstall script keeps user data by default, supports --remove-global-data"
```

---

## Task 10: Implement `gpowers init` — Per-Project Setup

**Files:**
- Create: `bin/gpowers-init`
- Create: `templates/project-gitignore`
- Create: `templates/project-readme.md`
- Test: `tests/unit/gpowers-init.bats`

`gpowers init` (run from inside a git repo) creates `<repo>/.gpowers/` with subdirectories, a `.gitignore`, and a brief README explaining what the directory is.

- [ ] **Step 10.1: Write the failing test**

Create `tests/unit/gpowers-init.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-init-$$"
  mkdir -p "$HOME"
  TESTREPO="${BATS_TMPDIR}/initrepo-$$"
  mkdir -p "$TESTREPO/.git"
  unset GPOWERS_PROJECT_DIR
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

INIT_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-init"

@test "gpowers-init creates <repo>/.gpowers/ tree" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers" ]
  [ -d "$TESTREPO/.gpowers/plans" ]
  [ -d "$TESTREPO/.gpowers/designs" ]
  [ -d "$TESTREPO/.gpowers/evals" ]
  [ -d "$TESTREPO/.gpowers/sessions" ]
  [ -d "$TESTREPO/.gpowers/retros" ]
  [ -d "$TESTREPO/.gpowers/learnings" ]
  [ -d "$TESTREPO/.gpowers/investigate" ]
  [ -d "$TESTREPO/.gpowers/canary" ]
  [ -d "$TESTREPO/.gpowers/health" ]
  [ -d "$TESTREPO/.gpowers/benchmark" ]
}

@test "gpowers-init writes .gitignore" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -f "$TESTREPO/.gpowers/.gitignore" ]
  grep -q "^logs/$" "$TESTREPO/.gpowers/.gitignore"
  grep -q "^tmp/$" "$TESTREPO/.gpowers/.gitignore"
}

@test "gpowers-init writes README explaining the directory" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -f "$TESTREPO/.gpowers/README.md" ]
  grep -qi "gpowers" "$TESTREPO/.gpowers/README.md"
}

@test "gpowers-init is idempotent" {
  cd "$TESTREPO"
  "$INIT_BIN"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
}

@test "gpowers-init refuses if not in a project (no .git, no GPOWERS_PROJECT_DIR)" {
  rm -rf "$TESTREPO/.git"
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -ne 0 ]
  [[ "$output" == *"no project"* ]]
}

@test "gpowers-init honors GPOWERS_PROJECT_DIR even outside a git repo" {
  rm -rf "$TESTREPO/.git"
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd /tmp
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers/plans" ]
}
```

- [ ] **Step 10.2: Run to verify failure**

Run: `bats tests/unit/gpowers-init.bats`
Expected: FAIL — files do not exist.

- [ ] **Step 10.3: Create `templates/project-gitignore`**

```
# gpowers project runtime data
# Default: commit decision artifacts (plans, designs, retros, etc.)
# Ignore noise and locks below:
logs/
tmp/
sessions/*.pid
sessions/*.lock
*.local.*
.cache/
ship-queue.lock
```

- [ ] **Step 10.4: Create `templates/project-readme.md`**

```markdown
# .gpowers/

This directory is created by [gpowers](https://github.com/<repo>/gpowers).
It holds project-scoped runtime data:

- `plans/` — CEO / eng / design / devex review plans
- `designs/` — design mockups and exploration
- `evals/` — evaluation results
- `sessions/` — agent session context snapshots
- `retros/` — project retrospectives
- `learnings/` — project-specific things learned
- `investigate/` — root-cause analysis records
- `canary/`, `health/`, `benchmark/` — historical metrics

Most contents **should be committed** (team-shared decision memory).
See `.gitignore` for what is excluded.
```

- [ ] **Step 10.5: Implement `bin/gpowers-init`**

```bash
#!/usr/bin/env bash
# bin/gpowers-init — Initialize <repo>/.gpowers/ for the current project.

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_gpowers-lib.sh
. "$SCRIPT_DIR/_gpowers-lib.sh"
# shellcheck source=../lib/runtime-dirs.sh
. "$SCRIPT_DIR/../lib/runtime-dirs.sh"

if [ -z "${GPOWERS_PROJECT_DIR:-}" ]; then
  gpowers_die "no project detected (no .gpowers/, no .git/, no GPOWERS_PROJECT_DIR)" 2
fi

target="$GPOWERS_PROJECT_DATA"
mkdir -p "$target"

for sub in plans plans/ceo plans/eng plans/design plans/devex plans/autoplan \
           designs evals sessions retros learnings investigate \
           canary health benchmark browser-skills logs tmp; do
  mkdir -p "$target/$sub"
done

# Templates
TEMPLATES="$SCRIPT_DIR/../templates"
if [ ! -f "$target/.gitignore" ]; then
  cp "$TEMPLATES/project-gitignore" "$target/.gitignore"
fi
if [ ! -f "$target/README.md" ]; then
  cp "$TEMPLATES/project-readme.md" "$target/README.md"
fi

printf '[gpowers-init] initialized %s\n' "$target"
```

Make executable: `chmod +x bin/gpowers-init`.

- [ ] **Step 10.6: Run tests**

Run: `bats tests/unit/gpowers-init.bats`
Expected: 6 passing.

- [ ] **Step 10.7: Commit**

```bash
git add bin/gpowers-init templates/project-gitignore templates/project-readme.md tests/unit/gpowers-init.bats
chmod +x bin/gpowers-init
git update-index --chmod=+x bin/gpowers-init
git commit -m "foundation: gpowers-init creates per-project .gpowers/ with templates"
```

---

## Task 11: Top-Level `gpowers` Dispatcher

**Files:**
- Create: `bin/gpowers`
- Test: `tests/unit/gpowers-cli.bats`

Single entry point: `gpowers <subcommand> [args]`. Routes to `bin/gpowers-<subcommand>`. Provides `gpowers --help` listing subcommands.

- [ ] **Step 11.1: Write failing test**

Create `tests/unit/gpowers-cli.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-cli-$$"
  mkdir -p "$HOME"
}

teardown() {
  rm -rf "$HOME"
}

GP="${BATS_TEST_DIRNAME}/../../bin/gpowers"

@test "gpowers --help lists subcommands" {
  run "$GP" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"init"* ]]
  [[ "$output" == *"path"* ]]
  [[ "$output" == *"detect-platforms"* ]]
}

@test "gpowers path config delegates to gpowers-path" {
  result="$("$GP" path config)"
  [ "$result" = "$HOME/.gpowers/config" ]
}

@test "gpowers detect-platforms delegates to bin script" {
  mkdir -p "$HOME/.claude"
  result="$("$GP" detect-platforms)"
  [[ "$result" == *"claude-code"* ]]
}

@test "gpowers unknown-subcommand exits 2" {
  run "$GP" unknown-subcommand
  [ "$status" -eq 2 ]
}

@test "gpowers with no args prints help and exits 0" {
  run "$GP"
  [ "$status" -eq 0 ]
  [[ "$output" == *"subcommands"* ]] || [[ "$output" == *"usage"* ]]
}
```

- [ ] **Step 11.2: Run to verify failure**

Run: `bats tests/unit/gpowers-cli.bats`
Expected: FAIL — `bin/gpowers` does not exist.

- [ ] **Step 11.3: Implement `bin/gpowers`**

```bash
#!/usr/bin/env bash
# bin/gpowers — Top-level dispatcher.

set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_gpowers-lib.sh
. "$SCRIPT_DIR/_gpowers-lib.sh"

usage() {
  cat <<'EOF'
usage: gpowers <subcommand> [args...]

Subcommands (foundation set):
  init                 initialize <repo>/.gpowers/ in current project
  path <kind> [sub..]  print a runtime path (kinds: home config state cache data analytics tmp project)
  detect-platforms     list AI CLI platforms with markers on this machine

Other subcommands are added by later plans (install, uninstall, upgrade, migrate).
Run with --help on any subcommand for details.
EOF
}

if [ $# -eq 0 ]; then
  usage
  exit 0
fi

case "$1" in
  --help|-h|help) usage; exit 0 ;;
esac

sub="$1"
shift
script="$SCRIPT_DIR/gpowers-$sub"
if [ ! -x "$script" ]; then
  gpowers_die "unknown subcommand: $sub" 2
fi
exec "$script" "$@"
```

Make executable: `chmod +x bin/gpowers`.

- [ ] **Step 11.4: Run tests**

Run: `bats tests/unit/gpowers-cli.bats`
Expected: 5 passing.

- [ ] **Step 11.5: Commit**

```bash
git add bin/gpowers tests/unit/gpowers-cli.bats
chmod +x bin/gpowers
git update-index --chmod=+x bin/gpowers
git commit -m "foundation: top-level gpowers CLI dispatcher"
```

---

## Task 12: Full-Suite Test Runner

**Files:**
- Create: `tests/run-all.sh`
- Modify: `README.md`

Single command to run the full foundation test suite.

- [ ] **Step 12.1: Create `tests/run-all.sh`**

```bash
#!/usr/bin/env bash
# tests/run-all.sh — Run all bats tests.
set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v bats >/dev/null; then
  printf 'error: bats-core not installed. Install with: brew install bats-core (or npm i -g bats)\n' >&2
  exit 1
fi
if ! command -v jq >/dev/null; then
  printf 'error: jq not installed. Install with: brew install jq\n' >&2
  exit 1
fi

bats "$SCRIPT_DIR/unit"
```

Make executable: `chmod +x tests/run-all.sh`.

- [ ] **Step 12.2: Run the full suite**

Run: `tests/run-all.sh`
Expected: All 7 test files pass (~50 individual tests total).

- [ ] **Step 12.3: Update README**

Append to `README.md`:

```markdown

## Development

```bash
# Run tests
./tests/run-all.sh

# Dry-run an install
./install --dry-run --with-business

# Detect platforms
./bin/gpowers detect-platforms
```

Requirements: bash 4+, jq, bats-core (for tests).
```

- [ ] **Step 12.4: Commit**

```bash
git add tests/run-all.sh README.md
chmod +x tests/run-all.sh
git update-index --chmod=+x tests/run-all.sh
git commit -m "foundation: full-suite test runner + dev usage in README"
```

---

## Task 13: Smoke Test — End-to-End Install + Init

**Files:**
- Create: `tests/unit/e2e-foundation.bats`

End-to-end: install gpowers, run `gpowers init` in a fake project, verify everything is wired up.

- [ ] **Step 13.1: Write the e2e test**

Create `tests/unit/e2e-foundation.bats`:

```bash
#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-e2e-$$"
  mkdir -p "$HOME/.claude/plugins"
  TESTREPO="${BATS_TMPDIR}/e2e-repo-$$"
  mkdir -p "$TESTREPO/.git"
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

REPO="${BATS_TEST_DIRNAME}/../.."

@test "e2e: install, then run gpowers init in a fake repo" {
  # 1. Install
  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
  [ -d "$HOME/.gpowers/config" ]

  # 2. Run gpowers init in repo
  cd "$TESTREPO"
  run "$HOME/.gpowers/bin/gpowers" init
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers/plans/ceo" ]

  # 3. gpowers path project plans resolves to the repo's .gpowers
  cd "$TESTREPO"
  result="$("$HOME/.gpowers/bin/gpowers" path project plans)"
  [ "$result" = "$TESTREPO/.gpowers/plans" ]
}

@test "e2e: uninstall, then re-install is clean" {
  "$REPO/install" --core-only --platforms=claude-code
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -L "$HOME/.claude/plugins/gpowers" ]
  [ ! -d "$HOME/.gpowers/core" ]

  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
}
```

- [ ] **Step 13.2: Run the e2e test**

Run: `bats tests/unit/e2e-foundation.bats`
Expected: 2 passing.

- [ ] **Step 13.3: Run the full suite to confirm no regression**

Run: `tests/run-all.sh`
Expected: All tests pass (8 test files, ~52 tests).

- [ ] **Step 13.4: Commit**

```bash
git add tests/unit/e2e-foundation.bats
git commit -m "foundation: end-to-end smoke test for install + init flow"
```

---

## Self-Review

### 1. Spec coverage check

| Spec section | Task |
|---|---|
| §1 Single repo, 4 modules, platforms/, install/upgrade scripts | Tasks 1, 7, 8 |
| §5 Install flags (`--with-business`, `--core-only`, `--no-tools`, `--platforms`, `--location`, `--link`, `--uninstall`, `--dry-run`) | Tasks 7, 8 (note: `--uninstall` is a separate `uninstall` script per spec, implemented in Task 9) |
| §5 Single-truth `~/.gpowers/` + per-platform symlink/prefix exposure | Task 8 |
| §6 Uninstall keeps user data by default | Task 9 |
| §7 Global runtime dirs (`config/state/cache/data/analytics/logs/tmp`) | Tasks 2, 8 |
| §7 Project root detection (env > .gpowers > .git > none) | Task 4 |
| §7 `gpowers-path` helper enforces no direct string-concat | Tasks 3, 5 |
| §7 `<repo>/.gpowers/` with plans/designs/evals/... | Task 10 |
| §7 GPOWERS_* env var override | Tasks 2, 4, 5 |
| §7 Project `.gitignore` template | Task 10 |
| §5 `manifest.json`, `upstream-sources.json` | Task 1 (init), Task 8 (update on install) |
| §5 Platform-specific plugin dirs (7 platforms) | Tasks 6, 8 |
| §5 Kimi prefix-mode is acknowledged but full adapter generation is deferred to plan-08 | Task 8 |

**Gaps consciously deferred to later plans:**
- `gpowers upgrade` → plan-09
- `gpowers migrate` → plan-10
- Kimi per-skill adapter generation → plan-08
- Real module content (core/roles/tools/business) → plans 02-07
- Real platforms/ command + skill registration files → plan-08
- Windows polyglot wrapper (`run-hook.cmd` style) → plan-02 (when hooks land) and plan-08 (per-platform)

### 2. Placeholder scan

Re-read the plan. No `TBD`, no "fill in", no "similar to Task N". One stub-implementation-then-replace exists (Task 7's "not implemented yet (Task 8)" stub) but that is intentional TDD: it lets the parsing tests pass before action logic exists, and Task 8 replaces it with full code. Acceptable.

### 3. Type / name consistency

- `GPOWERS_HOME`, `GPOWERS_CONFIG`, `GPOWERS_STATE`, `GPOWERS_CACHE`, `GPOWERS_DATA`, `GPOWERS_ANALYTICS`, `GPOWERS_TMP`, `GPOWERS_PROJECT_DIR`, `GPOWERS_PROJECT_DATA` — used consistently across Tasks 2, 3, 4, 5, 8, 9, 10.
- `gpowers-path` kinds: `home config state cache data analytics tmp project` — consistent.
- Platform names: `claude-code`/`claude_code` distinction. Underscore form is used in shell variable names (`GPOWERS_PLATFORM_PLUGIN_DIR_claude_code`); dash form in user-facing output (`gpowers-detect-platforms` prints `claude-code`). `install` converts dash to underscore via `${p//-/_}`. Consistent.
- `manifest.json` fields: `version`, `schema_version`, `installed_modules`, `installed_at`, `install_location` — defined in Task 1, updated in Task 8 via `gpowers_manifest_set_installed`. Consistent.
- `gpowers_die`, `gpowers_path_join`, `gpowers_script_dir`, `gpowers_manifest_set_installed` — all defined once, used by dependents.

### 4. Decomposition / file boundaries

- `lib/runtime-dirs.sh` — runtime env vars only (Tasks 2, 4)
- `lib/platform-paths.sh` — platform lookup table only (Task 6)
- `lib/manifest.sh` — manifest read/write only (Task 8)
- `bin/_gpowers-lib.sh` — shared shell utilities only (Task 3)
- `bin/gpowers-path`, `bin/gpowers-init`, `bin/gpowers-detect-platforms`, `bin/gpowers` — one CLI per file
- `install` and `uninstall` — entry-point scripts at root
- No file exceeds ~150 lines after this plan's tasks land.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-foundation.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
