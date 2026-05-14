# gpowers migrate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `gpowers migrate` so existing gstack or superpowers users can move their runtime data into the new `~/.gpowers/` global layer + per-project `<repo>/.gpowers/` layout (spec §7). The migration is dry-run-first, reversible, and handles three starting states: gstack only, superpowers only, both installed. Also wire the `/review` → `/pr-review` slash-command alias with a 6-month deprecation banner.

**Architecture:** `bin/gpowers-migrate` is a Bash CLI with three phases — **scan** (detect what's installed and what data exists), **plan** (build a mapping table of source → destination), **apply** (move/copy with confirmation). Scan output is deterministic and shown to the user; plan output is a printable table; apply executes the plan with a per-file copy + verify + delete (move semantics, but safe). Project-scoped data uses a **slug → repo path** reverse lookup: gstack stored project data under `~/.gstack/projects/<slug>/`; we attempt to find that slug's repo by reading `~/.gstack/projects/<slug>/.repo-path` if it exists, falling back to `find ~ -type d -name .git` matching, falling back to `~/.gpowers/data/legacy-projects/<slug>/` so nothing is lost. The `/review` alias is implemented as a tiny dispatcher slash command in `platforms/<platform>/commands/review.md` that prints a deprecation banner and forwards to `/pr-review`; the banner is dated and self-removes after 6 months by reading `manifest.json`'s `install_date`.

**Tech Stack:** Bash 4+, `jq` for migration mapping JSON, `rsync` for atomic moves, `find` for project discovery, bats-core for tests. Optional: `mktemp` + `trap` for safe rollback.

**Depends on:** Plan #1 (foundation: `gpowers-path`, runtime layout, manifest.json), Plan #6 (the `/pr-review` skill must exist for the alias to forward to). Can be implemented after #1 and #6.

---

## File Structure

```
bin/
├── gpowers-migrate                   User-facing CLI: scan / plan / apply
├── _gpowers-migrate-scan.sh          Detect installed predecessors + data
├── _gpowers-migrate-plan.sh          Build mapping; emit JSON + table
├── _gpowers-migrate-apply.sh         Execute with verify + rollback
└── _gpowers-find-project-by-slug.sh  Reverse lookup helper
lib/
└── migration-rules.sh                The big src → dst rule table
templates/
└── review-alias-command.md           Template for /review deprecation alias
tests/fixtures/migrate/
├── fake-home-gstack/                 ~/.gstack/ tree to migrate
├── fake-home-superpowers/            ~/.config/superpowers/ tree
├── fake-home-both/                   both
└── fake-home-empty/                  neither (control)
tests/unit/migrate/
├── scan-detects-gstack.bats          scan finds ~/.gstack/, reports counts
├── scan-detects-superpowers.bats
├── scan-handles-empty-home.bats
├── plan-mapping-shape.bats           mapping JSON conforms to schema
├── plan-handles-project-slug.bats    reverse lookup or fallback to legacy-projects
├── plan-detects-conflicts.bats       target paths that exist → marked CONFLICT
├── apply-dry-run.bats                no filesystem changes when --dry-run
├── apply-respects-confirm.bats       interactive confirm gates execution
├── apply-rollback-on-failure.bats    partial failure undoes prior moves
└── review-alias-banner.bats          /review prints deprecation, forwards
tests/integration/migrate/
├── full-gstack-migration.bats        end-to-end from fake-home-gstack
└── both-installed-migration.bats     handles gstack + superpowers together
```

---

## Task 1: Stage fake-home fixtures

**Files:**
- Create: `tests/fixtures/migrate/fake-home-{gstack,superpowers,both,empty}/`

Each is a minimal tree mirroring the real-world layouts the migrator must handle. Tests `HOME=<fixture>` against the migrator to assert correct mapping.

- [ ] **Step 1: Build gstack fixture**

```bash
mkdir -p tests/fixtures/migrate
build_gstack() {
  local root="$1"
  mkdir -p "$root/.gstack"/{config,state,cache,analytics,data,sessions,retros,learnings}
  mkdir -p "$root/.gstack/projects/proj-alpha/ceo-plans"
  mkdir -p "$root/.gstack/projects/proj-alpha/designs"
  mkdir -p "$root/.gstack/security"
  mkdir -p "$root/.gstack/browse" "$root/.gstack/cache/chromium-profile"

  echo "installation-id-xyz" > "$root/.gstack/installation-id"
  echo "2026-04-01T00:00:00Z" > "$root/.gstack/last-update-check"
  echo "builder-profile content" > "$root/.gstack/builder-profile"
  echo "developer-profile content" > "$root/.gstack/developer-profile"
  echo "gbrain repo policy" > "$root/.gstack/gbrain-repo-policy"
  echo "salt" > "$root/.gstack/security/device-salt"
  echo "ceo plan body" > "$root/.gstack/projects/proj-alpha/ceo-plans/feature-x.md"
  echo "design body" > "$root/.gstack/projects/proj-alpha/designs/wireframe-1.html"

  # Also XDG compact
  mkdir -p "$root/.config/gstack"
  echo "compact toml" > "$root/.config/gstack/compact.toml"
}

mkdir -p tests/fixtures/migrate/fake-home-gstack
build_gstack tests/fixtures/migrate/fake-home-gstack
```

- [ ] **Step 2: Build superpowers fixture**

```bash
build_sp() {
  local root="$1"
  mkdir -p "$root/.config/superpowers/worktrees/myrepo/feature-branch"
  mkdir -p "$root/.config/superpowers/state"
  echo "wt-data" > "$root/.config/superpowers/worktrees/myrepo/feature-branch/.note"
  echo "sp-installation" > "$root/.config/superpowers/state/installation-id"
}

mkdir -p tests/fixtures/migrate/fake-home-superpowers
build_sp tests/fixtures/migrate/fake-home-superpowers
```

- [ ] **Step 3: Build "both" + empty fixtures**

```bash
mkdir -p tests/fixtures/migrate/fake-home-both
build_gstack tests/fixtures/migrate/fake-home-both
build_sp tests/fixtures/migrate/fake-home-both

mkdir -p tests/fixtures/migrate/fake-home-empty/.local/share
```

- [ ] **Step 4: Commit**

```bash
git add tests/fixtures/migrate/
git commit -m "test(migrate): fake-home fixtures (gstack / superpowers / both / empty)"
```

---

## Task 2: Implement scan

**Files:**
- Create: `bin/_gpowers-migrate-scan.sh`

Scan inspects `$HOME` for the predecessor installs and emits a JSON report: which install(s) are present, what to migrate, item counts.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/migrate/scan-detects-gstack.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports gstack present" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.present == true' >/dev/null
}

@test "scan counts gstack projects" {
  out=$("$SCAN")
  [ "$(echo "$out" | jq -r '.gstack.projects | length')" -ge 1 ]
}

@test "scan finds builder-profile + developer-profile" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.profiles | index("builder-profile")' >/dev/null
  echo "$out" | jq -e '.gstack.profiles | index("developer-profile")' >/dev/null
}

@test "scan reports superpowers absent in gstack-only fixture" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.present == false' >/dev/null
}
EOF

cat > tests/unit/migrate/scan-detects-superpowers.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-superpowers"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports superpowers present" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.present == true' >/dev/null
}

@test "scan enumerates superpowers worktrees" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.worktrees | index("myrepo/feature-branch")' >/dev/null
}
EOF

cat > tests/unit/migrate/scan-handles-empty-home.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-empty"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports neither installed in empty home" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.present == false' >/dev/null
  echo "$out" | jq -e '.superpowers.present == false' >/dev/null
}

@test "scan exits 0 even on empty home" {
  run "$SCAN"
  [ "$status" -eq 0 ]
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement scan**

```bash
cat > bin/_gpowers-migrate-scan.sh <<'EOF'
#!/usr/bin/env bash
# Emits a JSON scan report to stdout. Reads $HOME.
set -euo pipefail
: "${HOME:?HOME required}"

GSTACK_ROOT="$HOME/.gstack"
SP_ROOT="$HOME/.config/superpowers"

gstack_present=false
[ -d "$GSTACK_ROOT" ] && gstack_present=true

sp_present=false
[ -d "$SP_ROOT" ] && sp_present=true

# gstack details
g_profiles=()
[ -f "$GSTACK_ROOT/builder-profile" ] && g_profiles+=("builder-profile")
[ -f "$GSTACK_ROOT/developer-profile" ] && g_profiles+=("developer-profile")
[ -f "$GSTACK_ROOT/gbrain-repo-policy" ] && g_profiles+=("gbrain-repo-policy")

g_projects=()
if [ -d "$GSTACK_ROOT/projects" ]; then
  while IFS= read -r dir; do
    g_projects+=("$(basename "$dir")")
  done < <(find "$GSTACK_ROOT/projects" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
fi

g_has_compact=false
[ -f "$HOME/.config/gstack/compact.toml" ] && g_has_compact=true

# superpowers details
sp_worktrees=()
if [ -d "$SP_ROOT/worktrees" ]; then
  while IFS= read -r dir; do
    # Format: <project>/<branch>
    rel="${dir#"$SP_ROOT/worktrees/"}"
    sp_worktrees+=("$rel")
  done < <(find "$SP_ROOT/worktrees" -mindepth 2 -maxdepth 2 -type d 2>/dev/null)
fi

# Emit JSON
jq -n \
  --argjson gp "$gstack_present" \
  --argjson sp "$sp_present" \
  --argjson compact "$g_has_compact" \
  --arg gstack_root "$GSTACK_ROOT" \
  --arg sp_root "$SP_ROOT" \
  --argjson profiles "$(printf '%s\n' "${g_profiles[@]+"${g_profiles[@]}"}" | jq -R . | jq -s .)" \
  --argjson projects "$(printf '%s\n' "${g_projects[@]+"${g_projects[@]}"}" | jq -R . | jq -s .)" \
  --argjson worktrees "$(printf '%s\n' "${sp_worktrees[@]+"${sp_worktrees[@]}"}" | jq -R . | jq -s .)" \
  '{
     gstack: {
       present: $gp, root: $gstack_root,
       profiles: $profiles, projects: $projects, has_compact: $compact
     },
     superpowers: {
       present: $sp, root: $sp_root, worktrees: $worktrees
     }
   }'
EOF
chmod +x bin/_gpowers-migrate-scan.sh
```

- [ ] **Step 3: Run tests**

Run: `bats tests/unit/migrate/scan-detects-gstack.bats tests/unit/migrate/scan-detects-superpowers.bats tests/unit/migrate/scan-handles-empty-home.bats`
Expected: PASS (9 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-migrate-scan.sh tests/unit/migrate/scan-*.bats
git commit -m "feat(migrate): _gpowers-migrate-scan emits structured JSON report"
```

---

## Task 3: Implement migration rules library

**Files:**
- Create: `lib/migration-rules.sh`

A single source of truth for source → destination mappings. Plan reads this; apply consumes plan output.

- [ ] **Step 1: Write the rules**

```bash
cat > lib/migration-rules.sh <<'EOF'
# Source me. Defines the migration map.
# Each rule is: <type>|<src-pattern>|<dst-template>|<comment>
# type: file | dir | project-glob | worktree-glob
# Templates expand:
#   ${GPOWERS_HOME}           = gpowers home (default ~/.gpowers)
#   ${HOME}
#   ${slug}                   = matched project slug (project-glob only)
#   ${project_repo}           = resolved repo path or fallback
#   ${remainder}              = leftover path after pattern strip

GPOWERS_MIGRATION_RULES=(
  # ---------- gstack global config ----------
  "file|${HOME}/.config/gstack/compact.toml|${GPOWERS_HOME}/config/compact.toml|XDG compact config"
  "dir|${HOME}/.config/gstack/compact-rules|${GPOWERS_HOME}/config/compact-rules|XDG compact rules"
  "file|${HOME}/.gstack/builder-profile|${GPOWERS_HOME}/config/builder-profile|builder profile"
  "file|${HOME}/.gstack/developer-profile|${GPOWERS_HOME}/config/developer-profile|developer profile"
  "file|${HOME}/.gstack/gbrain-repo-policy|${GPOWERS_HOME}/config/gbrain-repo-policy|gbrain policy"
  "file|${HOME}/.gstack/plan-tune.toml|${GPOWERS_HOME}/config/plan-tune.toml|plan-tune config"

  # ---------- gstack state ----------
  "file|${HOME}/.gstack/installation-id|${GPOWERS_HOME}/state/installation-id|install id"
  "file|${HOME}/.gstack/last-update-check|${GPOWERS_HOME}/state/last-update-check|update check timestamp"
  "file|${HOME}/.gstack/update-snoozed|${GPOWERS_HOME}/state/update-snoozed|snooze marker"
  "file|${HOME}/.gstack/just-upgraded-from|${GPOWERS_HOME}/state/just-upgraded-from|prev version pointer"
  "dir|${HOME}/.gstack/security|${GPOWERS_HOME}/state/security|security state"

  # ---------- gstack cache ----------
  "dir|${HOME}/.gstack/browse|${GPOWERS_HOME}/cache/browse|browser cache"
  "dir|${HOME}/.gstack/cache/chromium-profile|${GPOWERS_HOME}/cache/browser/chromium-profile|chromium profile"
  "dir|${HOME}/.gstack/models|${GPOWERS_HOME}/cache/models|AI models"
  "dir|${HOME}/.gstack/repos|${GPOWERS_HOME}/cache/repos|clone mirrors"
  "dir|${HOME}/.gstack/cache|${GPOWERS_HOME}/cache|catch-all cache"

  # ---------- gstack analytics ----------
  "dir|${HOME}/.gstack/analytics|${GPOWERS_HOME}/analytics|telemetry"

  # ---------- gstack global data ----------
  "dir|${HOME}/.gstack/data/browser-skills|${GPOWERS_HOME}/data/browser-skills|user browser skills"
  "dir|${HOME}/.gstack/data/global-domain-skills|${GPOWERS_HOME}/data/global-domain-skills|domain skills"

  # ---------- gstack project-scoped data (slug-based) ----------
  "project-glob|${HOME}/.gstack/projects/<slug>/ceo-plans|${project_repo}/.gpowers/plans/ceo|per-project ceo plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/eng-plans|${project_repo}/.gpowers/plans/eng|per-project eng plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/design-plans|${project_repo}/.gpowers/plans/design|per-project design plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/devex-plans|${project_repo}/.gpowers/plans/devex|per-project devex plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/autoplans|${project_repo}/.gpowers/plans/autoplan|per-project autoplans"
  "project-glob|${HOME}/.gstack/projects/<slug>/designs|${project_repo}/.gpowers/designs|per-project designs"
  "project-glob|${HOME}/.gstack/projects/<slug>/evals|${project_repo}/.gpowers/evals|per-project evals"
  "project-glob|${HOME}/.gstack/projects/<slug>/canary|${project_repo}/.gpowers/canary|per-project canary"
  "project-glob|${HOME}/.gstack/projects/<slug>/health|${project_repo}/.gpowers/health|per-project health"
  "project-glob|${HOME}/.gstack/projects/<slug>/benchmark|${project_repo}/.gpowers/benchmark|per-project benchmark"
  "project-glob|${HOME}/.gstack/projects/<slug>/learnings|${project_repo}/.gpowers/learnings|per-project learnings"

  # ---------- superpowers worktrees ----------
  "worktree-glob|${HOME}/.config/superpowers/worktrees/<project>/<branch>|${GPOWERS_HOME}/state/worktrees/<project>/<branch>|worktree state"

  # ---------- legacy catch-all ----------
  "dir|${HOME}/.gstack/sessions|${GPOWERS_HOME}/data/sessions|global session catch-all"
  "dir|${HOME}/.gstack/retros|${GPOWERS_HOME}/data/retros/global|global retros"
  "dir|${HOME}/.gstack/learnings|${GPOWERS_HOME}/data/learnings/global|global learnings"
  "dir|${HOME}/.gstack/investigate-sessions|${GPOWERS_HOME}/data/investigate-sessions|global investigations"
)
EOF
```

- [ ] **Step 2: Smoke check the rules file**

```bash
bash -c 'source lib/migration-rules.sh && echo "rules: ${#GPOWERS_MIGRATION_RULES[@]}"'
```

Expected: prints "rules: N" where N matches the array.

- [ ] **Step 3: Commit**

```bash
git add lib/migration-rules.sh
git commit -m "feat(migrate): migration-rules.sh — single source of truth for src→dst map"
```

---

## Task 4: Project slug reverse-lookup helper

**Files:**
- Create: `bin/_gpowers-find-project-by-slug.sh`

Given a gstack project slug, try (in order):
1. `~/.gstack/projects/<slug>/.repo-path` (if gstack recorded it)
2. `find $HOME -type d -name .git` filter by slug match
3. Return empty (caller falls back to `~/.gpowers/data/legacy-projects/<slug>/`)

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/migrate/find-project-by-slug.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TMP/.gstack/projects/explicit-slug"
  mkdir -p "$TMP/.gstack/projects/findable-slug"
  mkdir -p "$TMP/.gstack/projects/missing-slug"

  # explicit: .repo-path file
  mkdir -p "$TMP/repos/explicit-repo"
  ( cd "$TMP/repos/explicit-repo" && git init -q )
  echo "$TMP/repos/explicit-repo" > "$TMP/.gstack/projects/explicit-slug/.repo-path"

  # findable: a directory matching slug name
  mkdir -p "$TMP/repos/findable-slug"
  ( cd "$TMP/repos/findable-slug" && git init -q )

  export HOME="$TMP"
  HELPER="$REPO/bin/_gpowers-find-project-by-slug.sh"
}

@test "explicit slug resolves via .repo-path" {
  out=$("$HELPER" explicit-slug)
  [ "$out" = "$HOME/repos/explicit-repo" ]
}

@test "findable slug resolves via find" {
  out=$("$HELPER" findable-slug)
  case "$out" in *"findable-slug"*) :;; *) echo "got: $out"; return 1;; esac
}

@test "missing slug returns empty (caller handles fallback)" {
  out=$("$HELPER" missing-slug)
  [ -z "$out" ]
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement**

```bash
cat > bin/_gpowers-find-project-by-slug.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-find-project-by-slug.sh <slug>
# Echoes resolved repo path or empty.
set -euo pipefail
SLUG="${1:?slug required}"
: "${HOME:?HOME required}"

# 1. explicit recording
RECORDED="$HOME/.gstack/projects/$SLUG/.repo-path"
if [ -f "$RECORDED" ]; then
  path=$(head -1 "$RECORDED" | tr -d '[:space:]')
  if [ -n "$path" ] && [ -d "$path" ]; then
    echo "$path"
    exit 0
  fi
fi

# 2. find by directory name match (cheap, bounded depth)
match=$(find "$HOME" -maxdepth 6 -type d -name "$SLUG" 2>/dev/null \
        | while read -r d; do
            if [ -d "$d/.git" ]; then echo "$d"; break; fi
          done | head -1)
if [ -n "$match" ]; then
  echo "$match"
  exit 0
fi

# 3. fallback empty
echo ""
EOF
chmod +x bin/_gpowers-find-project-by-slug.sh
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/migrate/find-project-by-slug.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-find-project-by-slug.sh tests/unit/migrate/find-project-by-slug.bats
git commit -m "feat(migrate): project-slug → repo-path resolver with 3-tier lookup"
```

---

## Task 5: Implement plan

**Files:**
- Create: `bin/_gpowers-migrate-plan.sh`

Reads `lib/migration-rules.sh`, resolves project slugs to repo paths, emits a JSON mapping array.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/migrate/plan-mapping-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  export GPOWERS_HOME="$HOME/.gpowers"
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
}

@test "plan emits valid JSON" {
  "$PLAN" | jq empty
}

@test "plan has 'mappings' array" {
  out=$("$PLAN")
  echo "$out" | jq -e '.mappings | type == "array"' >/dev/null
}

@test "each mapping has src/dst/comment fields" {
  out=$("$PLAN")
  echo "$out" | jq -e 'all(.mappings[]; has("src") and has("dst") and has("comment"))' >/dev/null
}

@test "plan maps builder-profile to gpowers/config/" {
  out=$("$PLAN")
  echo "$out" | jq -e '.mappings[] | select(.src | endswith("builder-profile")) | .dst | endswith("config/builder-profile")' >/dev/null
}

@test "plan-handles-project-slug — proj-alpha resolves to fallback" {
  out=$("$PLAN")
  # proj-alpha has no .repo-path, no matching dir — should land in legacy-projects
  echo "$out" | jq -e '.mappings[] | select(.src | contains("projects/proj-alpha")) | .dst | contains("legacy-projects/proj-alpha")' >/dev/null
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement plan**

```bash
cat > bin/_gpowers-migrate-plan.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${HOME:?HOME required}"
: "${GPOWERS_HOME:?GPOWERS_HOME required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

source "$HERE/../lib/migration-rules.sh"

mappings='[]'

resolve_project() {
  local slug="$1"
  local repo
  repo=$("$HERE/_gpowers-find-project-by-slug.sh" "$slug")
  if [ -z "$repo" ]; then
    echo "$GPOWERS_HOME/data/legacy-projects/$slug"
  else
    echo "$repo"
  fi
}

for rule in "${GPOWERS_MIGRATION_RULES[@]}"; do
  IFS='|' read -r type src_tmpl dst_tmpl comment <<<"$rule"

  case "$type" in
    file|dir)
      src=$(eval echo "$src_tmpl")
      dst=$(eval echo "$dst_tmpl")
      if [ -e "$src" ]; then
        mappings=$(echo "$mappings" | jq --arg s "$src" --arg d "$dst" --arg c "$comment" \
                   '. += [{type:"file_or_dir", src:$s, dst:$d, comment:$c}]')
      fi
      ;;
    project-glob)
      src_pattern=$(echo "$src_tmpl" | sed 's|<slug>|*|')
      while IFS= read -r match; do
        [ -e "$match" ] || continue
        slug=$(echo "$match" | sed "s|$HOME/.gstack/projects/||; s|/.*||")
        repo=$(resolve_project "$slug")
        dst_tail=$(echo "$dst_tmpl" | sed 's|^${project_repo}|REPO|')
        dst=${dst_tail/REPO/$repo}
        # Substitute ${slug} if used in dst_tmpl (legacy-projects fallback case)
        dst=${dst//\$\{slug\}/$slug}
        # If repo is the legacy fallback, prepend $GPOWERS_HOME path is already absolute
        mappings=$(echo "$mappings" | jq --arg s "$match" --arg d "$dst" --arg c "$comment" --arg slug "$slug" \
                   '. += [{type:"project", src:$s, dst:$d, comment:$c, slug:$slug}]')
      done < <(eval echo "$src_pattern" 2>/dev/null)
      ;;
    worktree-glob)
      src_pattern=$(echo "$src_tmpl" | sed 's|<project>|*|; s|<branch>|*|')
      while IFS= read -r match; do
        [ -e "$match" ] || continue
        rel=${match#"$HOME/.config/superpowers/worktrees/"}
        proj=$(echo "$rel" | cut -d/ -f1)
        branch=$(echo "$rel" | cut -d/ -f2)
        dst="$GPOWERS_HOME/state/worktrees/$proj/$branch"
        mappings=$(echo "$mappings" | jq --arg s "$match" --arg d "$dst" --arg c "$comment" \
                   '. += [{type:"worktree", src:$s, dst:$d, comment:$c}]')
      done < <(eval echo "$src_pattern" 2>/dev/null)
      ;;
  esac
done

# Compute conflicts: dst already exists
conflicts=$(echo "$mappings" | jq '[.[] | select(.dst as $d | (env.HOME // "") + "/" + $d | test("^/")) ] | length')

echo "$mappings" | jq --argjson cnt "$conflicts" \
  '{mappings: ., total: length, will_create_dirs: (map(.dst) | map(split("/")[:-1] | join("/")) | unique | length)}'
EOF
chmod +x bin/_gpowers-migrate-plan.sh
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/migrate/plan-mapping-shape.bats`
Expected: PASS (5 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-migrate-plan.sh tests/unit/migrate/plan-mapping-shape.bats
git commit -m "feat(migrate): _gpowers-migrate-plan — emits JSON mapping with slug resolution"
```

---

## Task 6: Conflict detection

**Files:**
- Create: `tests/unit/migrate/plan-detects-conflicts.bats`
- Modify: `bin/_gpowers-migrate-plan.sh`

Add a `conflicts:` array to the plan output: any destination path that already exists and is non-empty.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/migrate/plan-detects-conflicts.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  export GPOWERS_HOME="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$GPOWERS_HOME/config"
  echo "pre-existing" > "$GPOWERS_HOME/config/builder-profile"   # collision
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
}

@test "plan reports a conflict for pre-existing dst" {
  out=$("$PLAN")
  echo "$out" | jq -e '.conflicts | length > 0' >/dev/null
  echo "$out" | jq -e '.conflicts[] | select(.dst | endswith("builder-profile"))' >/dev/null
}

@test "non-conflicting dst is not in conflicts list" {
  out=$("$PLAN")
  echo "$out" | jq -e 'all(.conflicts[]; .dst | contains("builder-profile"))' >/dev/null
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Add conflict detection to plan**

Edit `bin/_gpowers-migrate-plan.sh` — at the end, before the final `jq`, add a pass that builds the conflicts array:

```bash
# Append after the rule loop, before the final jq emit:
conflicts='[]'
while read -r dst; do
  if [ -e "$dst" ]; then
    conflicts=$(echo "$conflicts" | jq --arg d "$dst" '. += [{dst:$d, reason:"target already exists"}]')
  fi
done < <(echo "$mappings" | jq -r '.[].dst')

echo "$mappings" | jq \
  --argjson conflicts "$conflicts" \
  '{mappings: ., conflicts: $conflicts, total: length, will_create_dirs: (map(.dst) | map(split("/")[:-1] | join("/")) | unique | length)}'
```

(Apply via Edit to the existing file, replacing the final `echo "$mappings" | jq ...` block.)

- [ ] **Step 3: Run test**

Run: `bats tests/unit/migrate/plan-detects-conflicts.bats`
Expected: PASS (2 tests).

- [ ] **Step 4: Commit**

```bash
git add bin/_gpowers-migrate-plan.sh tests/unit/migrate/plan-detects-conflicts.bats
git commit -m "feat(migrate): plan output includes pre-existing-dst conflicts list"
```

---

## Task 7: Implement apply

**Files:**
- Create: `bin/_gpowers-migrate-apply.sh`

Reads plan JSON on stdin, executes moves with `rsync -a --remove-source-files` (atomic per-file), records each move in a journal file under `$(gpowers-path state)/migrate-journal.jsonl`. On failure, replay journal in reverse for rollback.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/migrate/apply-dry-run.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  mkdir -p "$GPOWERS_HOME/state"
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
  APPLY="$REPO/bin/_gpowers-migrate-apply.sh"
}

@test "apply --dry-run does not modify source" {
  before=$(find "$HOME/.gstack" | sort)
  "$PLAN" | bash "$APPLY" --dry-run >/dev/null
  after=$(find "$HOME/.gstack" | sort)
  [ "$before" = "$after" ]
}

@test "apply --dry-run does not create destination" {
  "$PLAN" | bash "$APPLY" --dry-run >/dev/null
  ! [ -d "$GPOWERS_HOME/config" ]
}

@test "apply (no --dry-run, --yes) moves builder-profile" {
  "$PLAN" | bash "$APPLY" --yes >/dev/null
  [ -f "$GPOWERS_HOME/config/builder-profile" ]
  [ ! -e "$HOME/.gstack/builder-profile" ]
}

@test "apply writes a journal" {
  "$PLAN" | bash "$APPLY" --yes >/dev/null
  [ -s "$GPOWERS_HOME/state/migrate-journal.jsonl" ]
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Implement apply**

```bash
cat > bin/_gpowers-migrate-apply.sh <<'EOF'
#!/usr/bin/env bash
# Reads plan JSON on stdin. Executes moves with journal + rollback.
# Flags: --dry-run | --yes (skip interactive confirm)
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

DRY=false
YES=false
while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY=true; shift;;
    --yes)     YES=true; shift;;
    *) echo "unknown flag: $1" >&2; exit 2;;
  esac
done

PLAN=$(cat -)
TOTAL=$(echo "$PLAN" | jq -r '.total')
CONFLICTS=$(echo "$PLAN" | jq -r '.conflicts | length')

echo "Plan: $TOTAL moves, $CONFLICTS conflicts."

if [ "$CONFLICTS" -gt 0 ] && ! $YES; then
  echo "Refusing to proceed with conflicts. Resolve them or pass --yes to skip conflicting items." >&2
  exit 3
fi

if ! $YES && ! $DRY; then
  read -r -p "Proceed with migration? [y/N] " ans
  case "$ans" in y|Y|yes|YES) ;; *) echo "aborted."; exit 0;; esac
fi

JOURNAL="$GPOWERS_HOME/state/migrate-journal.jsonl"
$DRY || mkdir -p "$(dirname "$JOURNAL")"

rollback() {
  echo "Rolling back…" >&2
  if [ -s "$JOURNAL" ]; then
    # Replay in reverse: move dst back to src
    tac "$JOURNAL" | while read -r line; do
      src=$(echo "$line" | jq -r .src)
      dst=$(echo "$line" | jq -r .dst)
      if [ -e "$dst" ] && [ ! -e "$src" ]; then
        mkdir -p "$(dirname "$src")"
        mv "$dst" "$src"
      fi
    done
  fi
}
trap 'rollback' ERR

while read -r mapping; do
  src=$(echo "$mapping" | jq -r .src)
  dst=$(echo "$mapping" | jq -r .dst)
  [ -e "$src" ] || continue
  # Skip conflicting dsts
  if [ -e "$dst" ]; then
    echo "[skip] $src (dst exists)"
    continue
  fi
  if $DRY; then
    echo "[dry-run] $src → $dst"
    continue
  fi
  mkdir -p "$(dirname "$dst")"
  if [ -f "$src" ]; then
    mv "$src" "$dst"
  else
    rsync -a --remove-source-files "$src/" "$dst/" 2>/dev/null
    rmdir "$src" 2>/dev/null || true
  fi
  echo "$mapping" >> "$JOURNAL"
  echo "[ok]   $src → $dst"
done < <(echo "$PLAN" | jq -c '.mappings[]')

trap - ERR
echo "Migration complete."
EOF
chmod +x bin/_gpowers-migrate-apply.sh
```

- [ ] **Step 3: Run + commit**

```bash
bats tests/unit/migrate/apply-dry-run.bats
git add bin/_gpowers-migrate-apply.sh tests/unit/migrate/apply-dry-run.bats
git commit -m "feat(migrate): _gpowers-migrate-apply with journal + rollback"
```

---

## Task 8: Rollback test on injected failure

**Files:**
- Create: `tests/unit/migrate/apply-rollback-on-failure.bats`

Inject an unwriteable target (chmod 000 on a parent dir) mid-run, assert that successful prior moves are rolled back.

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/migrate/apply-rollback-on-failure.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  mkdir -p "$GPOWERS_HOME"
  # Lock down a sub-tree of $GPOWERS_HOME after a few moves succeed
  chmod 555 "$GPOWERS_HOME"
}

teardown() {
  chmod 755 "$GPOWERS_HOME" 2>/dev/null || true
}

@test "rollback restores source files when destination write fails" {
  set +e
  "$REPO/bin/_gpowers-migrate-plan.sh" \
    | "$REPO/bin/_gpowers-migrate-apply.sh" --yes >/dev/null 2>&1
  rc=$?
  set -e
  [ "$rc" -ne 0 ]
  # builder-profile should still exist in source after rollback
  [ -f "$HOME/.gstack/builder-profile" ]
}
EOF
```

Run: expect this test to PASS (rollback logic from Task 7 handles it). If not, fix Task 7's `trap … ERR` to also cover non-fatal failures (`set -e` failure modes).

- [ ] **Step 2: Commit**

```bash
git add tests/unit/migrate/apply-rollback-on-failure.bats
git commit -m "test(migrate): rollback restores source after destination write failure"
```

---

## Task 9: User-facing `gpowers-migrate` CLI

**Files:**
- Create: `bin/gpowers-migrate`

Composes scan / plan / apply with `--scan-only` / `--plan-only` / `--apply` modes and `--yes` / `--dry-run`.

- [ ] **Step 1: Write the CLI**

```bash
cat > bin/gpowers-migrate <<'EOF'
#!/usr/bin/env bash
# Usage:
#   gpowers migrate                  Scan + show plan, ask to apply
#   gpowers migrate --scan-only       Just scan
#   gpowers migrate --plan-only       Scan + emit JSON plan
#   gpowers migrate --apply --yes     Run apply non-interactively
#   gpowers migrate --dry-run         Same as --apply --dry-run
set -euo pipefail
: "${HOME:?HOME required}"
: "${GPOWERS_HOME:?GPOWERS_HOME required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

MODE=interactive
DRY=false
YES=false
while [ $# -gt 0 ]; do
  case "$1" in
    --scan-only) MODE=scan; shift;;
    --plan-only) MODE=plan; shift;;
    --apply)     MODE=apply; shift;;
    --dry-run)   MODE=apply; DRY=true; shift;;
    --yes)       YES=true; shift;;
    -h|--help)   sed -n '4,10p' "$0"; exit 0;;
    *) echo "Unknown flag: $1" >&2; exit 2;;
  esac
done

scan_json=$("$HERE/_gpowers-migrate-scan.sh")
if [ "$MODE" = "scan" ]; then echo "$scan_json"; exit 0; fi

# Only proceed if something to migrate
gp=$(echo "$scan_json" | jq -r .gstack.present)
sp=$(echo "$scan_json" | jq -r .superpowers.present)
if [ "$gp" = "false" ] && [ "$sp" = "false" ]; then
  echo "Nothing to migrate: no gstack or superpowers install detected." >&2
  exit 0
fi

plan_json=$("$HERE/_gpowers-migrate-plan.sh")
if [ "$MODE" = "plan" ]; then echo "$plan_json"; exit 0; fi

# Print human summary
total=$(echo "$plan_json" | jq -r .total)
conflicts=$(echo "$plan_json" | jq -r '.conflicts | length')
echo "$total moves planned, $conflicts conflicts."
if [ "$conflicts" -gt 0 ]; then
  echo "Conflicts (dst already exists):"
  echo "$plan_json" | jq -r '.conflicts[] | "  - \(.dst)"'
fi

flags=()
$DRY && flags+=(--dry-run)
$YES && flags+=(--yes)
echo "$plan_json" | "$HERE/_gpowers-migrate-apply.sh" "${flags[@]}"
EOF
chmod +x bin/gpowers-migrate
```

- [ ] **Step 2: Failing test**

```bash
cat > tests/unit/migrate/cli-modes.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "--scan-only emits valid JSON" {
  out=$(gpowers-migrate --scan-only)
  echo "$out" | jq empty
}

@test "--plan-only emits mappings" {
  out=$(gpowers-migrate --plan-only)
  echo "$out" | jq -e '.mappings | length > 0' >/dev/null
}

@test "--apply --dry-run does not move files" {
  before=$(find "$HOME/.gstack" 2>/dev/null | sort)
  gpowers-migrate --apply --dry-run --yes >/dev/null
  after=$(find "$HOME/.gstack" 2>/dev/null | sort)
  [ "$before" = "$after" ]
}

@test "empty home prints nothing-to-migrate message" {
  export HOME="$REPO/tests/fixtures/migrate/fake-home-empty"
  run gpowers-migrate
  [ "$status" -eq 0 ]
  echo "$output" | grep -qi "nothing to migrate"
}
EOF
bats tests/unit/migrate/cli-modes.bats
```

Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add bin/gpowers-migrate tests/unit/migrate/cli-modes.bats
git commit -m "feat(migrate): user-facing gpowers-migrate CLI (scan/plan/apply modes)"
```

---

## Task 10: `/review` deprecation alias

**Files:**
- Create: `templates/review-alias-command.md`
- Modify: install hook (Plan #1) to write the alias to each platform's `commands/`

The spec calls for 6 months of `/review` accepting input and forwarding to `/pr-review` with a deprecation banner. We implement it as a static command file (one per platform) that the agent reads when the user types `/review`.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/migrate/review-alias-banner.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TEMPLATE="$REPO/templates/review-alias-command.md"
}

@test "alias template exists" { [ -f "$TEMPLATE" ]; }

@test "alias template prints deprecation banner" {
  grep -qi "deprecated\|renamed\|/pr-review" "$TEMPLATE"
}

@test "alias template references the 6-month window" {
  grep -qi "6 month\|six month" "$TEMPLATE"
}

@test "every platform commands/ dir has a review.md after install" {
  for p in claude-code codex gemini cursor opencode copilot; do
    [ -f "$REPO/platforms/$p/commands/review.md" ] || { echo "missing review.md for $p"; return 1; }
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Write the template + populate platforms**

```bash
mkdir -p templates
cat > templates/review-alias-command.md <<'EOF'
---
slash: /review
module: roles
skill: pr-review
deprecated: true
deprecation_until: "2026-11-14"
forwards_to: /pr-review
---

<!-- gpowers deprecation alias -->

**Note**: `/review` was renamed to `/pr-review` to avoid conceptual collision with the methodology skill `requesting-code-review` (core). The renamed command is at `/pr-review`. This alias forwards to `/pr-review` for a 6-month window ending **2026-11-14**.

Please switch to `/pr-review` in your habits and any saved prompts.

This command invokes the gpowers skill **pr-review** (roles).
EOF
```

Add to install (insert at end of platform-gen section in install script — Plan #8 Task 8):

```bash
for p in claude-code codex gemini cursor opencode copilot; do
  cp "$GPOWERS_HOME/templates/review-alias-command.md" \
     "$GPOWERS_HOME/platforms/$p/commands/review.md"
done
```

For Kimi:

```bash
mkdir -p "$GPOWERS_HOME/platforms/kimi/adapters/gpowers-review"
cp "$GPOWERS_HOME/templates/review-alias-command.md" \
   "$GPOWERS_HOME/platforms/kimi/adapters/gpowers-review/SKILL.md"
```

Apply the install patch via Edit, then run the test.

- [ ] **Step 3: Run test (after running install / gen once to populate platforms)**

```bash
GPOWERS_HOME="$(pwd)" ./bin/gpowers-platforms gen all >/dev/null
for p in claude-code codex gemini cursor opencode copilot; do
  cp templates/review-alias-command.md "platforms/$p/commands/review.md"
done
bats tests/unit/migrate/review-alias-banner.bats
```

Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add templates/review-alias-command.md platforms/*/commands/review.md install tests/unit/migrate/review-alias-banner.bats
git commit -m "feat(migrate): /review deprecation alias forwarding to /pr-review (6-month window)"
```

---

## Task 11: End-to-end integration smoke

**Files:**
- Create: `tests/integration/migrate/full-gstack-migration.bats`
- Create: `tests/integration/migrate/both-installed-migration.bats`

- [ ] **Step 1: Write the gstack-only e2e**

```bash
cat > tests/integration/migrate/full-gstack-migration.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "e2e: scan → plan → apply (yes) moves all expected items" {
  gpowers-migrate --apply --yes >/dev/null
  [ -f "$GPOWERS_HOME/config/builder-profile" ]
  [ -f "$GPOWERS_HOME/config/developer-profile" ]
  [ -f "$GPOWERS_HOME/state/installation-id" ]
  [ -d "$GPOWERS_HOME/state/security" ]
  [ -d "$GPOWERS_HOME/data/legacy-projects/proj-alpha" ] || \
    [ -d "$GPOWERS_HOME/data/legacy-projects/proj-alpha/ceo-plans" ]
}

@test "e2e: source tree is empty (or only contains residual empties) after migrate" {
  gpowers-migrate --apply --yes >/dev/null
  # builder-profile / installation-id should be gone from source
  [ ! -f "$HOME/.gstack/builder-profile" ]
  [ ! -f "$HOME/.gstack/installation-id" ]
}
EOF
```

- [ ] **Step 2: Write the both-installed e2e**

```bash
cat > tests/integration/migrate/both-installed-migration.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-both/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "both: scan detects gstack + superpowers" {
  out=$(gpowers-migrate --scan-only)
  echo "$out" | jq -e '.gstack.present and .superpowers.present' >/dev/null
}

@test "both: apply migrates worktrees to ~/.gpowers/state/worktrees/" {
  gpowers-migrate --apply --yes >/dev/null
  [ -e "$GPOWERS_HOME/state/worktrees/myrepo/feature-branch" ]
}

@test "both: gstack and superpowers source dirs are emptied" {
  gpowers-migrate --apply --yes >/dev/null
  ! [ -d "$HOME/.config/superpowers/worktrees/myrepo/feature-branch" ]
}
EOF
```

- [ ] **Step 3: Run**

Run: `bats tests/integration/migrate/full-gstack-migration.bats tests/integration/migrate/both-installed-migration.bats`
Expected: PASS (5 tests total).

- [ ] **Step 4: Commit**

```bash
git add tests/integration/migrate/
git commit -m "test(migrate): e2e gstack-only + gstack+superpowers migrations"
```

---

## Task 12: Manifest record

**Files:**
- Modify: `manifest.json`

- [ ] **Step 1: Update + test**

```bash
source lib/manifest.sh
gpowers_manifest_set migrate implemented true
gpowers_manifest_set migrate slash_aliases '{"review":"pr-review"}'
gpowers_manifest_set migrate deprecation_until '"2026-11-14"'

cat > tests/unit/migrate/manifest-migrate.bats <<'EOF'
#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest declares migrate implemented" {
  [ "$(jq -r '.migrate.implemented' < "$M")" = "true" ]
}
@test "manifest records /review alias and deprecation date" {
  [ "$(jq -r '.migrate.slash_aliases.review' < "$M")" = "pr-review" ]
  [ "$(jq -r '.migrate.deprecation_until' < "$M")" = "2026-11-14" ]
}
EOF
bats tests/unit/migrate/manifest-migrate.bats
```

Expected: PASS.

- [ ] **Step 2: Commit**

```bash
git add manifest.json tests/unit/migrate/manifest-migrate.bats
git commit -m "feat(migrate): manifest records migrate subsystem + /review alias deprecation"
```

---

## Self-Review

### 1. Spec coverage (§6 migration + §7 cross-system data move)

| Spec entry | Task |
|---|---|
| Detect gstack and superpowers installs | Task 2 |
| Build mapping per §7 source→destination table | Tasks 3, 5 |
| Project slug → repo path reverse lookup with fallback | Task 4 |
| `legacy-projects/<slug>/` fallback location | Tasks 3, 4 |
| Dry-run-first, confirm before apply | Tasks 7, 9 |
| Journal + rollback | Tasks 7, 8 |
| `/review` → `/pr-review` alias with deprecation banner | Task 10 |
| 6-month window | Task 10 |
| Handles gstack-only / superpowers-only / both / neither | Tasks 2, 11 |

### 2. Placeholder scan

No TBDs. Task 8's rollback test relies on `chmod 555` to inject a write failure; if a test environment runs as root that won't fail — we note that the test asserts behavior, not coverage of every kernel-level error mode.

### 3. Type / name consistency

- `_gpowers-migrate-{scan,plan,apply}.sh` matches Task 9 CLI's dispatch.
- `migration-rules.sh` rule format (type|src|dst|comment) consistent in Tasks 3, 5.
- Slash alias `review` → `pr-review` matches Plan #6 Task 5's rename.
- `deprecation_until: 2026-11-14` consistent in template (Task 10), test (Task 10), manifest (Task 12).

### 4. Decomposition

12 tasks. Scan, plan, and apply are isolated tasks each. The slug resolver is its own task (Task 4) because it's the trickiest piece of logic. The `/review` alias is its own task — orthogonal to data migration.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-migrate.md`. Depends on Plans #1 and #6 (and #8 for the alias to land in platforms/). Choose subagent-driven or inline at execution time.
