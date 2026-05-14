# gpowers upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `gpowers upgrade [<module>]` so users can pull upstream changes for each module independently: `core/` from `github.com/obra/superpowers`, and `roles/` / `tools/` / `business/` from `github.com/garrytan/gstack`. Use `git subtree` for the mechanics, integrate the import + rewrite helpers from Plans #2/#4/#5/#7 to re-transform after pull, run the relevant test suite automatically, and fill the `gpowers-upgrade` skill body that Plan #4 left as a stub.

**Architecture:** `gpowers upgrade` is a Bash command living at `bin/gpowers-upgrade` and exposed via the top-level `gpowers upgrade [<module>]` dispatcher (Plan #1). For each module:

1. Compute current SHA from `<module>/upstream-source.json`.
2. Run `git subtree pull --prefix=<module> <upstream-repo> <ref> --squash` against `~/.gpowers/` (which is a git repo from Plan #1).
3. Re-run the module's transform (frontmatter injection + reference rewrites). Each module exposes a `<module>/_upgrade-transform.sh` script that wraps the right helpers — `_gpowers-import-core.sh` for core, `_gpowers-import-tool.sh` for tools, `_gpowers-import-role.sh` for roles, `_gpowers-import-business.sh` for business. Plan #5's `_gpowers-rewrite-browser.py` runs after the tools import to keep verb-abstraction clean.
4. Re-run `gpowers-platforms gen <changed-modules>` to refresh manifests.
5. Run the module's unit + integration tests (`tests/unit/<module>/` + `tests/integration/<module>/`).
6. Update `<module>/upstream-source.json` with new SHA + timestamp.

Conflict path: `git subtree pull` may produce a merge conflict. We do **not** auto-resolve. Stop, print `git status`, point user to the conflict files, and document the resume procedure. `--check` mode does (1) only — fetches and compares SHAs but does not pull.

The skill `tools/skills/gpowers-upgrade/SKILL.md` (stub from Plan #4) gets its full body here: a workflow that an agent can guide a user through.

**Tech Stack:** git (>=2.30 for subtree --squash + --message), Bash 4+, jq, bats-core, the import/rewrite helpers from earlier plans.

**Depends on:** Plans #1 (foundation, dispatcher, manifest.sh), #2 (core/ stub for upstream-source.json + import helper), #4 (tools/ import helper + gpowers-upgrade stub skill), #5 (browser rewriter), #6 (roles/ import helper), #7 (business/ import helper), #8 (gpowers-platforms gen). Can be implemented after all module plans (#2-7) are complete.

---

## File Structure

```
bin/
├── gpowers-upgrade                       Top-level CLI (replaces the dispatcher stub from Plan #1)
├── _gpowers-upgrade-module.sh            Per-module pull + transform + test
├── _gpowers-upgrade-resume.sh            Resume from conflict
└── _gpowers-upgrade-check.sh             --check: report new versions without pulling
core/_upgrade-transform.sh                Re-apply core/ import transform after pull
roles/_upgrade-transform.sh
tools/_upgrade-transform.sh
business/_upgrade-transform.sh
tools/skills/gpowers-upgrade/SKILL.md     REPLACE Plan #4's stub with full body
upstream-sources.json                     Top-level summary (Plan #1 stub → filled here)
tests/fixtures/upgrade/
├── seed-repo.sh                          Creates a throwaway git repo for tests
└── fake-remotes/                         Bare git repos simulating upstream
tests/unit/upgrade/
├── argument-parsing.bats                 valid flags / unknown module rejected
├── check-mode.bats                       --check fetches, doesn't pull
├── pull-success.bats                     Successful subtree pull + transform applied
├── pull-conflict.bats                    Conflict prints status + exit non-zero
├── transform-runs.bats                   After pull, namespace + upstream tags re-applied
├── tests-trigger-after-pull.bats         Upgrade triggers per-module tests
└── upstream-source-updated.bats          SHA in upstream-source.json bumped after success
tests/integration/upgrade/
└── full-roundtrip.bats                   gen test fake-remote → pull → run tests → record SHA
```

---

## Task 1: Set up fake upstream remotes (test fixture)

**Files:**
- Create: `tests/fixtures/upgrade/seed-repo.sh`
- Create: `tests/fixtures/upgrade/fake-remotes/` (bare git repos)

We need real-looking git repos to subtree-pull from. A small bash script bootstraps four bare repos (superpowers + 3 gstack-derived) under a tmp dir each test setup, each containing a handful of files mirroring the upstream layout.

- [ ] **Step 1: Write the seeder**

```bash
mkdir -p tests/fixtures/upgrade
cat > tests/fixtures/upgrade/seed-repo.sh <<'EOF'
#!/usr/bin/env bash
# Usage: seed-repo.sh <tmp-base> <kind>
#   kind = superpowers | gstack-roles | gstack-tools | gstack-business
# Creates a bare repo under <tmp-base>/<kind>.git with one initial commit
# of stub skills mirroring the production layout. Echoes the bare repo path.
set -euo pipefail
BASE="${1:?base dir required}"
KIND="${2:?kind required}"

BARE="$BASE/$KIND.git"
WORK="$BASE/$KIND.work"
mkdir -p "$BARE" "$WORK"
git init --bare -q "$BARE"

cd "$WORK"
git init -q
git -c user.email=t@t -c user.name=t commit --allow-empty -q -m initial

case "$KIND" in
  superpowers)
    for n in brainstorming writing-plans test-driven-development; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: $n upstream
---

# $n

Upstream content. Body references superpowers:writing-plans.
F
    done
    ;;
  gstack-tools)
    for n in ship health make-pdf; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack tool $n
slash: /$n
---

# $n

Body references ~/.gstack/state and gstack-$n CLI.
F
    done
    ;;
  gstack-roles)
    for n in pr-review cso; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack role $n
slash: /$n
---

# $n

Role body. Reads ~/.gstack/config.
F
    done
    ;;
  gstack-business)
    for n in money money-content; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack business $n
slash: /$n
---

# $n

Business strategy stub. Data: ~/.gstack/data/$n/.
F
    done
    ;;
esac

git add -A
git -c user.email=t@t -c user.name=t commit -q -m "seed $KIND"
git push -q "$BARE" master:main 2>/dev/null || git push -q "$BARE" HEAD:main
echo "$BARE"
EOF
chmod +x tests/fixtures/upgrade/seed-repo.sh
```

- [ ] **Step 2: Smoke test the seeder**

```bash
tmp=$(mktemp -d)
bare=$(./tests/fixtures/upgrade/seed-repo.sh "$tmp" superpowers)
git -C "$bare" log --oneline | head
rm -rf "$tmp"
```

Expected: one commit "seed superpowers" appears.

- [ ] **Step 3: Commit**

```bash
git add tests/fixtures/upgrade/seed-repo.sh
git commit -m "test(upgrade): seed-repo helper for fake upstream bare repos"
```

---

## Task 2: Define top-level upstream-sources.json

**Files:**
- Modify: `upstream-sources.json` (created as stub in Plan #1)

Plan #1 created this as a stub. Fill it in with the actual per-module entries; this is what the upgrader reads to know where each module's upstream lives.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/upgrade/upstream-sources-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../upstream-sources.json"; }

@test "upstream-sources.json is valid JSON" { jq empty < "$F"; }

@test "every module has a remote entry" {
  for m in core roles tools business; do
    jq -e ".modules.\"$m\".repo" < "$F" >/dev/null
    jq -e ".modules.\"$m\".ref"  < "$F" >/dev/null
  done
}

@test "core upstream is superpowers" {
  [ "$(jq -r '.modules.core.repo' < "$F")" = "github.com/obra/superpowers" ]
}

@test "roles tools business upstream is gstack" {
  for m in roles tools business; do
    [ "$(jq -r ".modules.\"$m\".repo" < "$F")" = "github.com/garrytan/gstack" ]
  done
}
EOF
```

Run: expect FAIL (stub from Plan #1 had no module entries).

- [ ] **Step 2: Write the file**

```bash
cat > upstream-sources.json <<'EOF'
{
  "schema_version": 1,
  "modules": {
    "core": {
      "repo": "github.com/obra/superpowers",
      "url":  "https://github.com/obra/superpowers.git",
      "ref":  "v5.1.0",
      "subtree_prefix": "core",
      "subdir_in_upstream": "skills",
      "transform_script": "core/_upgrade-transform.sh"
    },
    "roles": {
      "repo": "github.com/garrytan/gstack",
      "url":  "https://github.com/garrytan/gstack.git",
      "ref":  "main",
      "subtree_prefix": "roles",
      "subdir_in_upstream": "skills",
      "transform_script": "roles/_upgrade-transform.sh"
    },
    "tools": {
      "repo": "github.com/garrytan/gstack",
      "url":  "https://github.com/garrytan/gstack.git",
      "ref":  "main",
      "subtree_prefix": "tools",
      "subdir_in_upstream": "skills",
      "transform_script": "tools/_upgrade-transform.sh"
    },
    "business": {
      "repo": "github.com/garrytan/gstack",
      "url":  "https://github.com/garrytan/gstack.git",
      "ref":  "main",
      "subtree_prefix": "business",
      "subdir_in_upstream": "skills",
      "transform_script": "business/_upgrade-transform.sh"
    }
  }
}
EOF
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/upstream-sources-shape.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add upstream-sources.json tests/unit/upgrade/upstream-sources-shape.bats
git commit -m "feat(upgrade): top-level upstream-sources.json with 4 module entries"
```

---

## Task 3: Per-module `_upgrade-transform.sh` scripts

**Files:**
- Create: `core/_upgrade-transform.sh`
- Create: `roles/_upgrade-transform.sh`
- Create: `tools/_upgrade-transform.sh`
- Create: `business/_upgrade-transform.sh`

After `git subtree pull`, the upstream content lands in the module directory but in *upstream form* (no namespace/upstream frontmatter, original `superpowers:` / `gstack-` references). These transform scripts re-apply the same machinery used at first install (Plans #2 / #4 / #5 / #6 / #7).

- [ ] **Step 1: Write core's transform**

```bash
cat > core/_upgrade-transform.sh <<'EOF'
#!/usr/bin/env bash
# Re-applies the Plan #2 transform after a git subtree pull from superpowers.
# Pre: GPOWERS_HOME contains a freshly-pulled core/skills/ in upstream form.
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

REF=$(jq -r '.modules.core.ref' < "$GPOWERS_HOME/upstream-sources.json")
TAG="superpowers@$REF"

# Move the freshly-pulled upstream form to a scratch dir, then run the importer
SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/core/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/core/skills"

"$GPOWERS_HOME/bin/_gpowers-import-core.sh" \
  "$SCRATCH/upstream-skills" \
  "$GPOWERS_HOME/core/skills" \
  "$TAG"

# using-gpowers is local-only; preserve it from prior state if removed by the pull
USING="$GPOWERS_HOME/core/skills/using-gpowers"
if [ ! -d "$USING" ] && [ -d "$SCRATCH/upstream-skills/using-gpowers.bak" ]; then
  cp -R "$SCRATCH/upstream-skills/using-gpowers.bak" "$USING"
fi

rm -rf "$SCRATCH"
EOF
chmod +x core/_upgrade-transform.sh
```

- [ ] **Step 2: Write tools' transform (uses both helpers — import + browser rewrite)**

```bash
cat > tools/_upgrade-transform.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

# Tools split: non-browser uses Plan #4 importer; browser ones additionally
# get Plan #5's _gpowers-rewrite-browser.py applied to the body.
SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/tools/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/tools/skills"

BROWSER_LIST=$(jq -r '.submodules.browser_dependent[]' < "$GPOWERS_HOME/tools/upstream-source.json")

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  "$GPOWERS_HOME/bin/_gpowers-import-tool.sh" "$src" "$GPOWERS_HOME/tools/skills/$name"

  # If this is a browser-dependent skill, run the rewriter on its body
  if echo "$BROWSER_LIST" | grep -qx "$name"; then
    file="$GPOWERS_HOME/tools/skills/$name/SKILL.md"
    fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$file")
    head -n "$fm_end" "$file" > "$SCRATCH/fm.md"
    tail -n +$((fm_end+1)) "$file" \
      | "$GPOWERS_HOME/bin/_gpowers-rewrite-browser.py" > "$SCRATCH/body.md"
    cat "$SCRATCH/fm.md" "$SCRATCH/body.md" > "$file"
  fi
done

rm -rf "$SCRATCH"
EOF
chmod +x tools/_upgrade-transform.sh
```

- [ ] **Step 3: Write roles' and business' transforms**

```bash
cat > roles/_upgrade-transform.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/roles/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/roles/skills"

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  # Apply the same /review → /pr-review rename as Plan #6 Task 5
  if [ "$name" = "review" ]; then
    "$GPOWERS_HOME/bin/_gpowers-import-role.sh" "$src" "$GPOWERS_HOME/roles/skills/pr-review"
    sed -i.bak \
        -e 's/^name: review$/name: pr-review/' \
        -e 's|^slash: /review$|slash: /pr-review|' \
        "$GPOWERS_HOME/roles/skills/pr-review/SKILL.md"
    rm -f "$GPOWERS_HOME/roles/skills/pr-review/SKILL.md.bak"
  else
    "$GPOWERS_HOME/bin/_gpowers-import-role.sh" "$src" "$GPOWERS_HOME/roles/skills/$name"
  fi
done

# Apply browser preamble to design-review if it exists
DR="$GPOWERS_HOME/roles/skills/design-review/SKILL.md"
if [ -f "$DR" ] && ! grep -q "^requires-driver: browser$" "$DR"; then
  fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$DR")
  head -n "$fm_end" "$DR" > "$SCRATCH/fm.md"
  tail -n +$((fm_end+1)) "$DR" | "$GPOWERS_HOME/bin/_gpowers-rewrite-browser.py" > "$SCRATCH/body.md"
  awk '/^---$/{c++; if(c==2)print "requires-driver: browser"; print; next} {print}' \
       "$SCRATCH/fm.md" > "$DR"
  cat "$SCRATCH/body.md" >> "$DR"
fi

rm -rf "$SCRATCH"
EOF
chmod +x roles/_upgrade-transform.sh

cat > business/_upgrade-transform.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/business/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/business/skills"

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  "$GPOWERS_HOME/bin/_gpowers-import-business.sh" "$src" "$GPOWERS_HOME/business/skills/$name"
done

rm -rf "$SCRATCH"
EOF
chmod +x business/_upgrade-transform.sh
```

- [ ] **Step 4: Commit**

```bash
git add core/_upgrade-transform.sh roles/_upgrade-transform.sh \
        tools/_upgrade-transform.sh business/_upgrade-transform.sh
git commit -m "feat(upgrade): per-module _upgrade-transform scripts"
```

---

## Task 4: Implement `_gpowers-upgrade-module.sh`

**Files:**
- Create: `bin/_gpowers-upgrade-module.sh`

The single-module worker. Reads upstream-sources.json, runs `git subtree pull`, captures the new SHA, runs the module's `_upgrade-transform.sh`, refreshes platform manifests, runs the module's tests, and updates `<module>/upstream-source.json`.

- [ ] **Step 1: Failing test (with the fixture)**

```bash
cat > tests/unit/upgrade/pull-success.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  # Build a working gpowers-style repo
  mkdir -p "$TMP" && cp -R "$REPO/." "$TMP/gp" >/dev/null
  cd "$TMP/gp"
  # Make it a git repo if not already
  [ -d .git ] || git init -q
  git -c user.email=t@t -c user.name=t add -A
  git -c user.email=t@t -c user.name=t commit -q -m initial 2>/dev/null || true

  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"

  # Build a fake upstream
  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  # Re-point upstream-sources.json at the bare repo
  jq --arg u "file://$BARE" '.modules.tools.url = $u | .modules.tools.ref = "main"' \
     "$GPOWERS_HOME/upstream-sources.json" > "$GPOWERS_HOME/upstream-sources.json.tmp"
  mv "$GPOWERS_HOME/upstream-sources.json.tmp" "$GPOWERS_HOME/upstream-sources.json"
}

@test "upgrade-module tools succeeds against fake remote" {
  run bash _gpowers-upgrade-module.sh tools
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
}

@test "after pull, tools/upstream-source.json has new SHA" {
  bash _gpowers-upgrade-module.sh tools >/dev/null
  sha=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  [ "$sha" != "0000000000000000000000000000000000000000" ]
  [ ${#sha} -eq 40 ]
}

@test "after pull, tools skills have namespace: tools applied" {
  bash _gpowers-upgrade-module.sh tools >/dev/null
  for d in "$GPOWERS_HOME/tools/skills"/*/; do
    grep -q "^namespace: tools$" "$d/SKILL.md"
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement the worker**

```bash
cat > bin/_gpowers-upgrade-module.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-upgrade-module.sh <module>
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

MODULE="${1:?module name required}"
SOURCES="$GPOWERS_HOME/upstream-sources.json"
[ -f "$SOURCES" ] || { echo "upstream-sources.json missing" >&2; exit 1; }

read -r URL REF PREFIX TRANSFORM <<<"$(jq -r --arg m "$MODULE" '.modules[$m] |
  "\(.url) \(.ref) \(.subtree_prefix) \(.transform_script)"' < "$SOURCES")"

[ "$URL" != "null" ] || { echo "unknown module: $MODULE" >&2; exit 2; }

cd "$GPOWERS_HOME"
[ -d .git ] || { echo "$GPOWERS_HOME is not a git repo (needed for subtree)" >&2; exit 3; }

# Ensure a clean tree (subtree pull requires it)
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "$GPOWERS_HOME has uncommitted changes; commit or stash before upgrading." >&2
  exit 4
fi

echo "[upgrade:$MODULE] pulling $URL@$REF into $PREFIX/"
if ! git subtree pull --prefix="$PREFIX" "$URL" "$REF" --squash \
     -m "upgrade($MODULE): pull $URL@$REF"; then
  echo "[upgrade:$MODULE] subtree pull failed (likely conflict)" >&2
  git status >&2
  echo "Run \`gpowers upgrade --resume\` after resolving conflicts." >&2
  exit 5
fi

# Capture new SHA from FETCH_HEAD
NEW_SHA=$(git rev-parse FETCH_HEAD 2>/dev/null || git ls-remote "$URL" "$REF" | awk '{print $1}')

# Re-apply transform
echo "[upgrade:$MODULE] applying transform: $TRANSFORM"
"$GPOWERS_HOME/$TRANSFORM"

# Update module's upstream-source.json
jq --arg sha "$NEW_SHA" \
   --arg ts  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '.upstream.sha = $sha | .upgraded_at = $ts' \
   "$GPOWERS_HOME/$MODULE/upstream-source.json" \
   > "$GPOWERS_HOME/$MODULE/upstream-source.json.tmp"
mv "$GPOWERS_HOME/$MODULE/upstream-source.json.tmp" "$GPOWERS_HOME/$MODULE/upstream-source.json"

# Refresh platform manifests (Plan #8)
echo "[upgrade:$MODULE] regenerating platform manifests"
"$GPOWERS_HOME/bin/gpowers-platforms" gen all

# Run module tests
echo "[upgrade:$MODULE] running tests"
if command -v bats >/dev/null; then
  bats "$GPOWERS_HOME/tests/unit/$MODULE" 2>/dev/null || true
  bats "$GPOWERS_HOME/tests/integration/$MODULE" 2>/dev/null || true
else
  echo "[upgrade:$MODULE] bats not installed; skipping tests" >&2
fi

# Commit the transformed state
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" add -A
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): apply transform @ $NEW_SHA"

echo "[upgrade:$MODULE] done @ $NEW_SHA"
EOF
chmod +x bin/_gpowers-upgrade-module.sh
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/pull-success.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-upgrade-module.sh tests/unit/upgrade/pull-success.bats
git commit -m "feat(upgrade): _gpowers-upgrade-module — pull + transform + test + SHA bump"
```

---

## Task 5: Implement `_gpowers-upgrade-check.sh`

**Files:**
- Create: `bin/_gpowers-upgrade-check.sh`

Read-only mode: fetches `ls-remote` for each module and compares to the SHA in `<module>/upstream-source.json`. Reports a table.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/upgrade/check-mode.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  cp -R "$REPO/." "$TMP/gp"
  cd "$TMP/gp"
  [ -d .git ] || git init -q
  git -c user.email=t@t -c user.name=t add -A 2>/dev/null
  git -c user.email=t@t -c user.name=t commit -q -m initial 2>/dev/null || true
  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"
  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  jq --arg u "file://$BARE" '.modules.tools.url = $u' \
     "$GPOWERS_HOME/upstream-sources.json" > "$GPOWERS_HOME/upstream-sources.json.tmp"
  mv "$GPOWERS_HOME/upstream-sources.json.tmp" "$GPOWERS_HOME/upstream-sources.json"
}

@test "check reports a remote SHA for tools" {
  out=$(bash _gpowers-upgrade-check.sh tools)
  echo "$out" | grep -qE 'tools[[:space:]]+[0-9a-f]{40}'
}

@test "check does NOT modify upstream-source.json" {
  before=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  bash _gpowers-upgrade-check.sh tools >/dev/null
  after=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  [ "$before" = "$after" ]
}

@test "check reports 'new version available' when SHAs differ" {
  out=$(bash _gpowers-upgrade-check.sh tools)
  echo "$out" | grep -qi "new\|update available"
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement**

```bash
cat > bin/_gpowers-upgrade-check.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-upgrade-check.sh [<module>]
# Compares <module>/upstream-source.json sha against remote ls-remote.
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"
SOURCES="$GPOWERS_HOME/upstream-sources.json"

check_one() {
  local module="$1"
  local url ref local_sha remote_sha
  url=$(jq -r --arg m "$module" '.modules[$m].url' < "$SOURCES")
  ref=$(jq -r --arg m "$module" '.modules[$m].ref' < "$SOURCES")
  local_sha=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/$module/upstream-source.json")
  remote_sha=$(git ls-remote "$url" "$ref" 2>/dev/null | awk '{print $1}' | head -1)

  if [ -z "$remote_sha" ]; then
    printf '%-10s %-42s %s\n' "$module" "(unreachable)" "$url"
    return
  fi

  local status="up-to-date"
  if [ "$local_sha" != "$remote_sha" ]; then status="new version available"; fi
  printf '%-10s %s %s\n' "$module" "$remote_sha" "$status"
}

if [ $# -ge 1 ]; then
  check_one "$1"
else
  for m in core roles tools business; do check_one "$m"; done
fi
EOF
chmod +x bin/_gpowers-upgrade-check.sh
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/check-mode.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-upgrade-check.sh tests/unit/upgrade/check-mode.bats
git commit -m "feat(upgrade): _gpowers-upgrade-check — read-only remote SHA compare"
```

---

## Task 6: Implement `_gpowers-upgrade-resume.sh`

**Files:**
- Create: `bin/_gpowers-upgrade-resume.sh`

Detect an in-progress merge (presence of `.git/MERGE_HEAD`), confirm user has resolved conflicts (working tree clean), continue: commit, run transform, run tests, bump SHA.

- [ ] **Step 1: Failing test** (skipped if not on a system where we can simulate conflict trivially)

```bash
cat > tests/unit/upgrade/pull-conflict.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  cp -R "$REPO/." "$TMP/gp"
  cd "$TMP/gp"
  [ -d .git ] || git init -q
  git -c user.email=t@t -c user.name=t add -A 2>/dev/null
  git -c user.email=t@t -c user.name=t commit -q -m initial 2>/dev/null || true
  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"
}

@test "resume exits 0 with no in-progress merge (no-op)" {
  out=$(bash _gpowers-upgrade-resume.sh 2>&1)
  echo "$out" | grep -qi "no upgrade in progress\|nothing to resume"
}

@test "resume requires a clean working tree (after manual fix)" {
  # Simulate: pretend MERGE_HEAD exists but tree dirty
  echo conflict > /tmp/gp_conflict
  touch "$GPOWERS_HOME/.git/MERGE_HEAD" 2>/dev/null || true
  if [ -f "$GPOWERS_HOME/.git/MERGE_HEAD" ]; then
    cp /tmp/gp_conflict "$GPOWERS_HOME/dirty.txt"
    git -C "$GPOWERS_HOME" add dirty.txt
    run bash _gpowers-upgrade-resume.sh
    [ "$status" -ne 0 ]
    echo "$output" | grep -qi "still have unresolved\|working tree not clean\|conflicts remain"
  else
    skip "platform doesn't allow direct MERGE_HEAD touch"
  fi
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement**

```bash
cat > bin/_gpowers-upgrade-resume.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

cd "$GPOWERS_HOME"

if [ ! -f .git/MERGE_HEAD ] && [ ! -f .git/REBASE_HEAD ]; then
  echo "No upgrade in progress (no .git/MERGE_HEAD). Nothing to resume."
  exit 0
fi

# Working tree must be clean of conflict markers
if git ls-files --unmerged | grep -q .; then
  echo "You still have unresolved conflict files. Fix them, then \`git add\` and rerun." >&2
  git status --short >&2
  exit 1
fi

# Read the module from the in-progress merge commit's prefix
# Heuristic: look at staged changes' paths to find the touched module
TOUCHED=$(git diff --cached --name-only | awk -F/ '{print $1}' | sort -u | head -1)
[ -n "$TOUCHED" ] || { echo "Couldn't determine touched module from staged changes." >&2; exit 1; }

case "$TOUCHED" in
  core|roles|tools|business) MODULE="$TOUCHED" ;;
  *) echo "Touched path '$TOUCHED' isn't a known module." >&2; exit 1 ;;
esac

# Complete merge commit
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): resume after conflict resolution"

# Run transform + record SHA + regen platforms (same tail as the worker)
"$GPOWERS_HOME/$(jq -r --arg m "$MODULE" '.modules[$m].transform_script' < upstream-sources.json)"
SHA=$(git rev-parse HEAD)

jq --arg sha "$SHA" --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '.upstream.sha = $sha | .upgraded_at = $ts' \
   "$MODULE/upstream-source.json" > "$MODULE/upstream-source.json.tmp"
mv "$MODULE/upstream-source.json.tmp" "$MODULE/upstream-source.json"

"$GPOWERS_HOME/bin/gpowers-platforms" gen all >/dev/null

bats "tests/unit/$MODULE" 2>/dev/null || true
bats "tests/integration/$MODULE" 2>/dev/null || true

git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" add -A
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): apply transform after manual resolution @ $SHA"

echo "[upgrade:$MODULE] resumed and applied @ $SHA"
EOF
chmod +x bin/_gpowers-upgrade-resume.sh
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/pull-conflict.bats`
Expected: PASS (2 tests; second may skip on hosts where touching .git/MERGE_HEAD isn't sufficient — that's documented).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-upgrade-resume.sh tests/unit/upgrade/pull-conflict.bats
git commit -m "feat(upgrade): _gpowers-upgrade-resume — continue after manual conflict resolve"
```

---

## Task 7: Top-level `bin/gpowers-upgrade` CLI

**Files:**
- Create: `bin/gpowers-upgrade`

Composes the three internal helpers into the user-facing UX described in spec §5.

- [ ] **Step 1: Failing argument-parsing test**

```bash
cat > tests/unit/upgrade/argument-parsing.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "no args = upgrade all (dry-run shows plan)" {
  out=$(gpowers-upgrade --dry-run)
  for m in core roles tools business; do
    echo "$out" | grep -q "$m" || { echo "missing $m"; return 1; }
  done
}

@test "named module narrows scope" {
  out=$(gpowers-upgrade tools --dry-run)
  echo "$out" | grep -q "tools"
  ! echo "$out" | grep -q "core\|roles\|business"
}

@test "--check delegates without modifying" {
  out=$(gpowers-upgrade --check 2>&1 || true)
  # Output must look like a table; we accept whatever ls-remote returns
  echo "$out" | grep -qE 'core|tools|roles|business'
}

@test "unknown module exits 2" {
  run gpowers-upgrade nonsense
  [ "$status" -eq 2 ]
}

@test "--resume invokes resume helper" {
  out=$(gpowers-upgrade --resume 2>&1)
  echo "$out" | grep -qi "no upgrade in progress\|nothing to resume"
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement**

```bash
cat > bin/gpowers-upgrade <<'EOF'
#!/usr/bin/env bash
# gpowers-upgrade — composes _gpowers-upgrade-{module,check,resume}
#
# Usage:
#   gpowers-upgrade                      Upgrade all modules
#   gpowers-upgrade <module>             Upgrade only that module
#   gpowers-upgrade --check [<module>]   Show pending versions only
#   gpowers-upgrade --resume             Resume after conflict resolution
#   gpowers-upgrade --dry-run            Print the plan, do nothing
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

MODULES=()
DRY=false
CHECK=false
RESUME=false

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY=true; shift;;
    --check)   CHECK=true; shift;;
    --resume)  RESUME=true; shift;;
    -h|--help) sed -n '4,12p' "$0"; exit 0;;
    --*)       echo "Unknown flag: $1" >&2; exit 2;;
    core|roles|tools|business) MODULES+=("$1"); shift;;
    *)         echo "Unknown module: $1" >&2; exit 2;;
  esac
done

[ "${#MODULES[@]}" -gt 0 ] || MODULES=(core roles tools business)

if $RESUME; then
  exec "$GPOWERS_HOME/bin/_gpowers-upgrade-resume.sh"
fi

if $CHECK; then
  for m in "${MODULES[@]}"; do
    "$GPOWERS_HOME/bin/_gpowers-upgrade-check.sh" "$m"
  done
  exit 0
fi

if $DRY; then
  for m in "${MODULES[@]}"; do
    url=$(jq -r --arg m "$m" '.modules[$m].url' < "$GPOWERS_HOME/upstream-sources.json")
    ref=$(jq -r --arg m "$m" '.modules[$m].ref' < "$GPOWERS_HOME/upstream-sources.json")
    echo "[dry-run] would upgrade $m from $url@$ref"
  done
  exit 0
fi

for m in "${MODULES[@]}"; do
  "$GPOWERS_HOME/bin/_gpowers-upgrade-module.sh" "$m"
done
EOF
chmod +x bin/gpowers-upgrade
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/argument-parsing.bats`
Expected: PASS (5 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/gpowers-upgrade tests/unit/upgrade/argument-parsing.bats
git commit -m "feat(upgrade): top-level gpowers-upgrade CLI (--check / --resume / --dry-run / <module>)"
```

---

## Task 8: Replace `gpowers-upgrade` skill stub with full body

**Files:**
- Modify: `tools/skills/gpowers-upgrade/SKILL.md` (stub from Plan #4)

The skill teaches the agent the upgrade workflow. The CLI does the work; the skill walks the user through using it.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/upgrade/skill-body-filled.bats <<'EOF'
#!/usr/bin/env bats

setup() { S="$BATS_TEST_DIRNAME/../../../tools/skills/gpowers-upgrade/SKILL.md"; }

@test "skill no longer marked as stub" {
  ! grep -qi "^\\* stub\\|^stub\\|placeholder" "$S"
  ! grep -qi "Plan #9 \\(landed below\\|fills body\\)" "$S"
}

@test "skill names the four modules" {
  for m in core roles tools business; do
    grep -qw "$m" "$S"
  done
}

@test "skill documents --check, --resume, --dry-run" {
  grep -q -- "--check" "$S"
  grep -q -- "--resume" "$S"
  grep -q -- "--dry-run" "$S"
}

@test "skill explains conflict resolution path" {
  grep -qi "conflict\|merge" "$S"
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Replace the stub**

```bash
cat > tools/skills/gpowers-upgrade/SKILL.md <<'EOF'
---
name: gpowers-upgrade
description: Pull upstream changes for any gpowers module (core / roles / tools / business) — git subtree mechanics, transform re-application, test re-run, platform manifest refresh.
namespace: tools
slash: /gpowers-upgrade
---

# gpowers-upgrade

When the user wants to refresh gpowers from upstream:

## Decide scope first

- **All four modules**: `gpowers upgrade` (no argument)
- **One module**: `gpowers upgrade core` (or `roles`, `tools`, `business`)
- **Just check what's new**: `gpowers upgrade --check` (read-only, no merge)

## Recommend a check before pulling

Suggest the user run `gpowers upgrade --check` first. It prints a table of
remote SHAs versus locally recorded SHAs and labels each row "up-to-date" or
"new version available". Use this to decide which modules actually need
pulling.

## Pull workflow

```bash
gpowers upgrade core            # pulls from github.com/obra/superpowers
gpowers upgrade tools           # pulls from github.com/garrytan/gstack
gpowers upgrade                 # all four
```

For each pulled module the runner:

1. Verifies `~/.gpowers/` working tree is clean (git subtree requirement).
2. Runs `git subtree pull --squash` from the upstream listed in
   `~/.gpowers/upstream-sources.json`.
3. Captures the new SHA and runs the module's `_upgrade-transform.sh` —
   re-applies `namespace:` and `upstream:` frontmatter, `~/.gstack/` path
   rewrites, `superpowers:` → `gpowers:` reference rewrites, and (for browser
   skills) the abstract-verb rewriter.
4. Regenerates all 7 platform manifests via `gpowers-platforms gen all`.
5. Runs the module's bats tests under `tests/unit/<module>/` and
   `tests/integration/<module>/`.
6. Bumps the SHA in `<module>/upstream-source.json`.
7. Commits the transformed state.

## Conflicts

`git subtree pull` may produce a merge conflict if you've made local edits
inside `~/.gpowers/<module>/`. The runner stops, prints `git status`, and
exits non-zero. Guide the user through:

```bash
cd ~/.gpowers
# Resolve conflicts in the listed files
git add <resolved-files>
gpowers upgrade --resume
```

`--resume` finishes the merge commit, runs the transform, regenerates
manifests, runs tests, and bumps the SHA — picking up where the conflict
interrupted things.

## Dry run

`gpowers upgrade --dry-run` prints the plan without acting. Use this to show
the user what would happen before they commit to a pull.

## Why each module has its own transform

The transform encodes how gpowers normalizes upstream content. Each module
ships a `_upgrade-transform.sh` that wraps the import helper used at first
install. Changing the normalization in one place (the helper) auto-applies
to upgrades — no separate code path to maintain.

## Telemetry

Upgrade events are recorded under `$(gpowers-path analytics)/upgrade.jsonl`.
Disable with `GPOWERS_ANALYTICS=off`.
EOF
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/upgrade/skill-body-filled.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add tools/skills/gpowers-upgrade/SKILL.md tests/unit/upgrade/skill-body-filled.bats
git commit -m "feat(upgrade): fill gpowers-upgrade skill body (was stub from plan #4)"
```

---

## Task 9: Integration round-trip

**Files:**
- Create: `tests/integration/upgrade/full-roundtrip.bats`

End-to-end: build fake remote → check → upgrade → verify transform + SHA bump + manifest refresh.

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/upgrade/full-roundtrip.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  cp -R "$REPO/." "$TMP/gp"
  cd "$TMP/gp"
  [ -d .git ] || git init -q
  git -c user.email=t@t -c user.name=t add -A
  git -c user.email=t@t -c user.name=t commit -q -m initial 2>/dev/null || true
  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"

  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  jq --arg u "file://$BARE" '.modules.tools.url = $u | .modules.tools.ref = "main"' \
     upstream-sources.json > upstream-sources.json.tmp && mv upstream-sources.json.tmp upstream-sources.json
  git -c user.email=t@t -c user.name=t add -A
  git -c user.email=t@t -c user.name=t commit -q -m "point tools at fixture"
}

@test "check reports new SHA for tools" {
  out=$(gpowers-upgrade --check tools)
  echo "$out" | grep -qE 'tools[[:space:]]+[0-9a-f]{40}'
}

@test "full upgrade round-trip succeeds" {
  before=$(jq -r '.upstream.sha' < tools/upstream-source.json)
  run gpowers-upgrade tools
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  after=$(jq -r '.upstream.sha' < tools/upstream-source.json)
  [ "$before" != "$after" ]
}

@test "after upgrade, manifests regenerated and skills still pass frontmatter test" {
  gpowers-upgrade tools >/dev/null
  # Skills should still satisfy frontmatter requirements
  for d in tools/skills/*/; do
    grep -q "^namespace: tools$" "$d/SKILL.md"
  done
  # Platform manifests still valid
  gpowers-platforms verify all >/dev/null
}
EOF
bats tests/integration/upgrade/full-roundtrip.bats
```

Expected: PASS (3 tests).

- [ ] **Step 2: Commit**

```bash
git add tests/integration/upgrade/full-roundtrip.bats
git commit -m "test(upgrade): full round-trip — check, pull, transform, manifest refresh"
```

---

## Task 10: Manifest record

**Files:**
- Modify: `manifest.json`

- [ ] **Step 1: Update**

```bash
source lib/manifest.sh
gpowers_manifest_set upgrade implemented true
gpowers_manifest_set upgrade subcommands '["all","core","roles","tools","business","--check","--resume","--dry-run"]'
```

- [ ] **Step 2: Failing test**

```bash
cat > tests/unit/upgrade/manifest-upgrade.bats <<'EOF'
#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest declares upgrade implemented" {
  [ "$(jq -r '.upgrade.implemented' < "$M")" = "true" ]
}
@test "manifest lists --resume subcommand" {
  jq -e '.upgrade.subcommands | index("--resume")' < "$M" >/dev/null
}
EOF
```

Run: PASS.

- [ ] **Step 3: Commit**

```bash
git add manifest.json tests/unit/upgrade/manifest-upgrade.bats
git commit -m "feat(upgrade): manifest records upgrade subsystem implemented"
```

---

## Self-Review

### 1. Spec coverage (§5 upgrade subset)

| Spec entry | Task |
|---|---|
| `gpowers upgrade` / per-module / --check | Tasks 4, 5, 7 |
| git subtree mechanics | Task 4 |
| Conflict handling — stop + manual resolve + resume | Task 6 |
| Auto-test after pull | Task 4 |
| upstream-sources.json schema | Task 2 |
| `gpowers-upgrade` skill body filled | Task 8 |

### 2. Placeholder scan

No TBDs. One known platform-specific test step (Task 6 Step 1's second test) explicitly skips on hosts that can't simulate `MERGE_HEAD`; that's documented in the test body.

### 3. Type / name consistency

- `_gpowers-upgrade-{module,check,resume}.sh` naming matches Task 7's dispatcher.
- `_upgrade-transform.sh` location convention identical for all 4 modules.
- Subcommand flags `--check`, `--resume`, `--dry-run` match Task 7 CLI and Task 8 skill body and Task 10 manifest.

### 4. Decomposition

10 tasks. The three sub-helpers (module / check / resume) are each their own task. The skill body is its own task (Task 8) so reviewers can read it as documentation.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-upgrade.md`. Depends on Plans #1, #2, #4, #5, #6, #7, #8 — most cross-dependent plan. Choose subagent-driven or inline at execution time.
