# gpowers roles/ Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the 20 role skills from `gstack-main` into `roles/skills/`, applying the same import + frontmatter machinery as `tools/`. Critical rename: gstack's `/review` becomes gpowers' `/pr-review` to avoid conceptual collision with superpowers' `requesting-code-review` (core). One role (`design-review`) uses the browser driver from Plan #3 — its preamble sources `select-driver.sh`.

**Architecture:** roles/ is the *explicit-trigger* module — users invoke roles by slash command; the agent never auto-invokes them. Each role is a SKILL.md with structured prompts that walk the agent through a role-specific review. Migration reuses `_gpowers-import-tool.sh` from Plan #4 with `namespace: roles` instead of `namespace: tools` and `upstream: gstack@main`. A new test catches the `/review` → `/pr-review` rename in three places: the directory name, the `slash:` frontmatter field, and any user-facing prose that mentioned the old command.

**Tech Stack:** Markdown, Bash 4+, the import helper from Plan #4, the browser-skill preamble injector pattern from Plan #5 (one skill only), bats-core, jq.

**Depends on:** Plan #1 (foundation), Plan #4 (`_gpowers-import-tool.sh`). Plan #3 + #5 only needed for the `design-review` browser preamble (1 of 20). Can be implemented in parallel with Plans #2 and #7.

---

## Skill Inventory (20 roles, 5 domain groups)

**Product / Strategy**
| Slash | Skill dir | gstack source |
|---|---|---|
| `/office-hours` | `office-hours` | `skills/office-hours/` |
| `/plan-ceo-review` | `plan-ceo-review` | `skills/plan-ceo-review/` |
| `/autoplan` | `autoplan` | `skills/autoplan/` |

**Engineering**
| Slash | Skill dir | gstack source |
|---|---|---|
| `/plan-eng-review` | `plan-eng-review` | `skills/plan-eng-review/` |
| `/plan-devex-review` | `plan-devex-review` | `skills/plan-devex-review/` |
| `/devex-review` | `devex-review` | `skills/devex-review/` |
| `/investigate` | `investigate` | `skills/investigate/` |
| `/codex` | `codex` | `skills/codex/` |
| `/pr-review` ⚠️ renamed | `pr-review` | `skills/review/` |

**Design**
| Slash | Skill dir | gstack source |
|---|---|---|
| `/plan-design-review` | `plan-design-review` | `skills/plan-design-review/` |
| `/design-consultation` | `design-consultation` | `skills/design-consultation/` |
| `/design-shotgun` | `design-shotgun` | `skills/design-shotgun/` |
| `/design-html` | `design-html` | `skills/design-html/` |
| `/design-review` ⚠️ browser | `design-review` | `skills/design-review/` |

**Security**
| Slash | Skill dir | gstack source |
|---|---|---|
| `/cso` | `cso` | `skills/cso/` |

**Retro / Memory / Collaboration**
| Slash | Skill dir | gstack source |
|---|---|---|
| `/retro` | `retro` | `skills/retro/` |
| `/document-release` | `document-release` | `skills/document-release/` |
| `/learn` | `learn` | `skills/learn/` |
| `/pair-agent` | `pair-agent` | `skills/pair-agent/` |
| `/plan-tune` | `plan-tune` | `skills/plan-tune/` |

Total: **20 roles**. Renames: 1 (`review` → `pr-review`). Browser deps: 1 (`design-review`); `pair-agent` may use browser features but the skill itself is platform-agnostic and degrades gracefully (the spec §5 already calls out that pair-agent is "degraded" on non-Claude-Code platforms — that's documented in the skill prose, not enforced by `requires-driver`).

---

## File Structure

```
roles/
├── skills/
│   ├── office-hours/SKILL.md
│   ├── plan-ceo-review/SKILL.md
│   ├── autoplan/SKILL.md
│   ├── plan-eng-review/SKILL.md
│   ├── plan-devex-review/SKILL.md
│   ├── devex-review/SKILL.md
│   ├── investigate/SKILL.md
│   ├── codex/SKILL.md
│   ├── pr-review/SKILL.md                  RENAMED from review
│   ├── plan-design-review/SKILL.md
│   ├── design-consultation/SKILL.md
│   ├── design-shotgun/SKILL.md
│   ├── design-html/SKILL.md
│   ├── design-review/SKILL.md              browser-dependent
│   ├── cso/SKILL.md
│   ├── retro/SKILL.md
│   ├── document-release/SKILL.md
│   ├── learn/SKILL.md
│   ├── pair-agent/SKILL.md
│   └── plan-tune/SKILL.md
└── upstream-source.json
tests/fixtures/roles/
└── fake-gstack-checkout/skills/<name>/SKILL.md   × 20 stubs (note: source name is `review/` for pr-review)
tests/unit/roles/
├── frontmatter.bats                        20 skills, namespace: roles, upstream
├── pr-review-rename.bats                   no /review refs anywhere; /pr-review present
├── slash-commands-roundtrip.bats           20 unique slashes, no collision with tools/core
├── design-review-browser-preamble.bats     design-review sources select-driver.sh
└── no-gstack-paths.bats                    no ~/.gstack/ literals
bin/_gpowers-import-role.sh                 wraps Plan #4 importer with namespace: roles
```

---

## Task 1: Stage role fixtures

**Files:**
- Create: `tests/fixtures/roles/fake-gstack-checkout/skills/<name>/SKILL.md` × 20

Important: the *source* directory for `pr-review` must be named `review/` (gstack original name) so the rename test has something to assert.

- [ ] **Step 1: Create the 19 same-named stubs**

```bash
mkdir -p tests/fixtures/roles/fake-gstack-checkout/skills
for name in office-hours plan-ceo-review autoplan plan-eng-review plan-devex-review \
            devex-review investigate codex plan-design-review design-consultation \
            design-shotgun design-html design-review cso retro document-release \
            learn pair-agent plan-tune; do
  dir="tests/fixtures/roles/fake-gstack-checkout/skills/$name"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<EOF
---
name: $name
description: stub fixture for gstack role $name
slash: /$name
---

# $name

Role-based stub. Reads from ~/.gstack/state and writes audit notes to ~/.gstack/data/$name/.
EOF
done
```

- [ ] **Step 2: Create the rename fixture (`review/`, not `pr-review/`)**

```bash
dir="tests/fixtures/roles/fake-gstack-checkout/skills/review"
mkdir -p "$dir"
cat > "$dir/SKILL.md" <<'EOF'
---
name: review
description: pre-merge PR review (gstack original)
slash: /review
---

# review

Run `/review` against the current branch. Read ~/.gstack/config for reviewer profile.
EOF
```

- [ ] **Step 3: Make design-review fixture include MCP references (so rewriter applies)**

```bash
cat > tests/fixtures/roles/fake-gstack-checkout/skills/design-review/SKILL.md <<'EOF'
---
name: design-review
description: post-implementation visual review (gstack)
slash: /design-review
---

# design-review

Visual walkthrough:
1. Use mcp__claude-in-chrome__navigate to the staging URL.
2. mcp__claude-in-chrome__computer for screenshots.
3. mcp__claude-in-chrome__read_page for DOM check.
On non-CC: `npx playwright test design.spec.ts`.

State: ~/.gstack/data/design-review/<slug>.md
EOF
```

- [ ] **Step 4: Commit**

```bash
git add tests/fixtures/roles/
git commit -m "test(roles): gstack fixtures including review/ (to rename) and design-review (browser)"
```

---

## Task 2: Wrap Plan #4 importer with namespace: roles

**Files:**
- Create: `bin/_gpowers-import-role.sh`

A thin wrapper that calls `_gpowers-import-tool.sh` but post-processes the frontmatter to set `namespace: roles` instead of `tools`. Easier than parameterizing the upstream helper.

- [ ] **Step 1: Write the wrapper**

```bash
cat > bin/_gpowers-import-role.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-import-role.sh <src-skill-dir> <dst-skill-dir>
# Wraps _gpowers-import-tool.sh and changes namespace: tools → namespace: roles.
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

"$HERE/_gpowers-import-tool.sh" "$SRC" "$DST"

# Swap namespace label
sed -i.bak 's/^namespace: tools$/namespace: roles/' "$DST/SKILL.md"
rm -f "$DST/SKILL.md.bak"
EOF
chmod +x bin/_gpowers-import-role.sh
```

- [ ] **Step 2: Smoke test**

```bash
mkdir -p /tmp/role-test
./bin/_gpowers-import-role.sh \
  tests/fixtures/roles/fake-gstack-checkout/skills/cso \
  /tmp/role-test/cso
grep -q "^namespace: roles$" /tmp/role-test/cso/SKILL.md
grep -q "^upstream: gstack@main$" /tmp/role-test/cso/SKILL.md
! grep -q "namespace: tools" /tmp/role-test/cso/SKILL.md
rm -rf /tmp/role-test
```

- [ ] **Step 3: Commit**

```bash
git add bin/_gpowers-import-role.sh
git commit -m "feat(roles): _gpowers-import-role.sh — wraps importer with namespace: roles"
```

---

## Task 3: Failing frontmatter test

**Files:**
- Create: `tests/unit/roles/frontmatter.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/roles/frontmatter.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  ROLES="$BATS_TEST_DIRNAME/../../../roles/skills"
  EXPECTED="office-hours plan-ceo-review autoplan plan-eng-review plan-devex-review
            devex-review investigate codex pr-review plan-design-review
            design-consultation design-shotgun design-html design-review cso
            retro document-release learn pair-agent plan-tune"
}

@test "every expected role skill exists" {
  for name in $EXPECTED; do
    [ -f "$ROLES/$name/SKILL.md" ] || { echo "missing: $name"; return 1; }
  done
}

@test "exactly 20 role skills present" {
  count=$(find "$ROLES" -mindepth 1 -maxdepth 1 -type d | wc -l)
  [ "$count" -eq 20 ]
}

@test "every role declares namespace: roles" {
  for name in $EXPECTED; do
    grep -q "^namespace: roles$" "$ROLES/$name/SKILL.md" || { echo "$name: no namespace"; return 1; }
  done
}

@test "every role declares upstream: gstack@main" {
  for name in $EXPECTED; do
    grep -q "^upstream: gstack@main$" "$ROLES/$name/SKILL.md" || { echo "$name: no upstream"; return 1; }
  done
}

@test "every role declares its slash command" {
  for name in $EXPECTED; do
    grep -q "^slash: /" "$ROLES/$name/SKILL.md" || { echo "$name: no slash"; return 1; }
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Commit failing test**

```bash
git add tests/unit/roles/frontmatter.bats
git commit -m "test(roles): frontmatter requirements for 20 role skills"
```

---

## Task 4: Migrate 19 same-named roles

**Files:**
- Create: `roles/skills/{office-hours,plan-ceo-review,autoplan,plan-eng-review,plan-devex-review,devex-review,investigate,codex,plan-design-review,design-consultation,design-shotgun,design-html,design-review,cso,retro,document-release,learn,pair-agent,plan-tune}/SKILL.md`

- [ ] **Step 1: Mass import**

```bash
SRC="tests/fixtures/roles/fake-gstack-checkout/skills"
DST="roles/skills"
mkdir -p "$DST"

for name in office-hours plan-ceo-review autoplan plan-eng-review plan-devex-review \
            devex-review investigate codex plan-design-review design-consultation \
            design-shotgun design-html design-review cso retro document-release \
            learn pair-agent plan-tune; do
  ./bin/_gpowers-import-role.sh "$SRC/$name" "$DST/$name"
done
```

- [ ] **Step 2: Run frontmatter test (will still FAIL — pr-review missing)**

Run: `bats tests/unit/roles/frontmatter.bats`
Expected: FAIL (pr-review not yet migrated; count = 19, expected 20).

- [ ] **Step 3: Commit**

```bash
git add roles/skills/
git commit -m "feat(roles): import 19 same-named role skills with namespace+upstream"
```

---

## Task 5: Migrate `review/` → `pr-review/` with rename machinery

**Files:**
- Create: `roles/skills/pr-review/SKILL.md`

Run the importer with destination `pr-review`, then do additional in-body rewrites: `name: review` → `name: pr-review`, `slash: /review` → `slash: /pr-review`, and any `/review` user-facing prose → `/pr-review` (mentioning that the old name is an alias during the deprecation window, per spec §6).

- [ ] **Step 1: Write failing rename test**

```bash
cat > tests/unit/roles/pr-review-rename.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  PR="$BATS_TEST_DIRNAME/../../../roles/skills/pr-review/SKILL.md"
}

@test "pr-review skill exists" {
  [ -f "$PR" ]
}

@test "frontmatter name is pr-review (not review)" {
  grep -q "^name: pr-review$" "$PR"
  ! grep -q "^name: review$" "$PR"
}

@test "slash command is /pr-review" {
  grep -q "^slash: /pr-review$" "$PR"
}

@test "body mentions both the new name and the alias note" {
  grep -q "/pr-review" "$PR"
  grep -qi "alias\|formerly\|previously /review\|legacy" "$PR"
}

@test "no role skill anywhere declares slash: /review" {
  REPO="$BATS_TEST_DIRNAME/../../../roles/skills"
  for dir in "$REPO"/*/; do
    if grep -q "^slash: /review$" "$dir/SKILL.md"; then
      echo "$(basename "$dir") still declares /review"; return 1
    fi
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Import + transform**

```bash
SRC="tests/fixtures/roles/fake-gstack-checkout/skills/review"
DST="roles/skills/pr-review"
./bin/_gpowers-import-role.sh "$SRC" "$DST"

# Rewrite name, slash, and inject alias note in body
sed -i.bak \
    -e 's/^name: review$/name: pr-review/' \
    -e 's|^slash: /review$|slash: /pr-review|' \
    -e 's|`/review`|`/pr-review`|g' \
    -e 's|/review |/pr-review |g' \
    "$DST/SKILL.md"
rm -f "$DST/SKILL.md.bak"

# Append alias note before final newline
cat >> "$DST/SKILL.md" <<'EOF'

## Renamed from /review

This skill was previously invoked as `/review` in gstack. The name changed to `/pr-review` to avoid confusion with `requesting-code-review` (core, methodology). The legacy `/review` slash is aliased to `/pr-review` during a 6-month deprecation window (see `gpowers migrate`, Plan #10).
EOF
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/roles/pr-review-rename.bats`
Expected: PASS (5 tests).

Then re-run `bats tests/unit/roles/frontmatter.bats` — now PASS (5 tests × 20 skills).

- [ ] **Step 4: Commit**

```bash
git add roles/skills/pr-review/ tests/unit/roles/pr-review-rename.bats
git commit -m "feat(roles): rename gstack /review → /pr-review with alias note"
```

---

## Task 6: Apply browser preamble to design-review

**Files:**
- Modify: `roles/skills/design-review/SKILL.md`

`design-review` is the only role that requires a browser driver. Apply the same preamble pattern as Plan #5 (rewriter pass + `requires-driver: browser` frontmatter + driver-select preamble).

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/roles/design-review-browser-preamble.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  DR="$BATS_TEST_DIRNAME/../../../roles/skills/design-review/SKILL.md"
}

@test "design-review declares requires-driver: browser" {
  grep -q "^requires-driver: browser$" "$DR"
}

@test "design-review body sources select-driver.sh" {
  body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$DR")
  echo "$body" | grep -q "select-driver.sh"
}

@test "design-review body uses gpowers-browser verbs (not MCP)" {
  body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$DR")
  echo "$body" | grep -q "gpowers-browser"
  ! echo "$body" | grep -q "mcp__claude-in-chrome"
}

@test "design-review body has no literal playwright CLI" {
  body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$DR")
  ! echo "$body" | grep -qE '`(npx +)?playwright +[a-z][^`]*`'
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Apply rewriter + preamble**

```bash
file="roles/skills/design-review/SKILL.md"

# Body rewrite via Plan #5 rewriter
fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$file")
head -n "$fm_end" "$file" > /tmp/fm.md
tail -n +$((fm_end+1)) "$file" | ./bin/_gpowers-rewrite-browser.py > /tmp/body.md

# Inject requires-driver: browser into frontmatter
awk '/^---$/{c++; if(c==2){print "requires-driver: browser"} print; next} {print}' \
    /tmp/fm.md > /tmp/fm2.md

# Splice frontmatter + preamble + rewritten body
cat /tmp/fm2.md > "$file"
cat >> "$file" <<'PREAMBLE'

## Preamble (auto)

Before any browser verb call, source the driver selector:

```bash
source "$GPOWERS_HOME/tools/drivers/browser/select-driver.sh"
```

This exports `GPOWERS_BROWSER_DRIVER`. All browser interactions use `gpowers-browser <verb>`.

PREAMBLE
cat /tmp/body.md >> "$file"
```

- [ ] **Step 3: Run tests**

Run: `bats tests/unit/roles/design-review-browser-preamble.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add roles/skills/design-review/SKILL.md tests/unit/roles/design-review-browser-preamble.bats
git commit -m "feat(roles): design-review browser preamble + verb rewrite"
```

---

## Task 7: Assert no leftover gstack paths

**Files:**
- Create: `tests/unit/roles/no-gstack-paths.bats`

- [ ] **Step 1: Write + run test**

```bash
cat > tests/unit/roles/no-gstack-paths.bats <<'EOF'
#!/usr/bin/env bats

setup() { ROLES="$BATS_TEST_DIRNAME/../../../roles/skills"; }

@test "no '~/.gstack/' literals in role bodies" {
  for dir in "$ROLES"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "~/\.gstack/"; then
      echo "$name leaks ~/.gstack/"; return 1
    fi
  done
}

@test "no 'gstack-<name>' CLI references in role bodies" {
  for dir in "$ROLES"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -qE '\bgstack-[a-z]'; then
      echo "$name references gstack-* CLI"; return 1
    fi
  done
}
EOF

bats tests/unit/roles/no-gstack-paths.bats
```

Expected: PASS (importer already rewrote during Tasks 4–6).

- [ ] **Step 2: Commit**

```bash
git add tests/unit/roles/no-gstack-paths.bats
git commit -m "test(roles): assert no leftover gstack paths"
```

---

## Task 8: Slash command uniqueness across modules

**Files:**
- Create: `tests/unit/roles/slash-commands-roundtrip.bats`

Confirms 20 unique slashes in `roles/`, plus no collision with `tools/` or `core/`.

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/roles/slash-commands-roundtrip.bats <<'EOF'
#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

slashes_from() {
  for dir in "$1"/*/; do
    grep -m1 '^slash:' "$dir/SKILL.md" 2>/dev/null | awk '{print $2}'
  done
}

@test "20 unique slashes in roles/" {
  mapfile -t all < <(slashes_from "$REPO/roles/skills")
  count=${#all[@]}
  [ "$count" -eq 20 ]
  uniq=$(printf '%s\n' "${all[@]}" | sort -u | wc -l)
  [ "$uniq" -eq 20 ]
}

@test "roles/ slashes do not collide with tools/" {
  mapfile -t roles  < <(slashes_from "$REPO/roles/skills")
  mapfile -t tools  < <(slashes_from "$REPO/tools/skills" 2>/dev/null || true)
  for r in "${roles[@]}"; do
    for t in "${tools[@]}"; do
      [ "$r" = "$t" ] && { echo "collision: $r"; return 1; }
    done
  done
}

@test "roles/ slashes do not collide with core/ skill names" {
  for dir in "$REPO/roles/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    cname="${slash#/}"
    if [ -d "$REPO/core/skills/$cname" ]; then
      echo "$(basename "$dir") collides with core/$cname"; return 1
    fi
  done
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/unit/roles/slash-commands-roundtrip.bats`
Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/unit/roles/slash-commands-roundtrip.bats
git commit -m "test(roles): slash command uniqueness across modules"
```

---

## Task 9: roles/upstream-source.json + manifest

**Files:**
- Create: `roles/upstream-source.json`
- Modify: `manifest.json`

- [ ] **Step 1: Write upstream-source.json**

```bash
cat > roles/upstream-source.json <<'EOF'
{
  "module": "roles",
  "upstream": {
    "repo": "github.com/garrytan/gstack",
    "ref": "main",
    "sha": "0000000000000000000000000000000000000000"
  },
  "imported_at": "2026-05-14T00:00:00Z",
  "renames": { "review": "pr-review" },
  "browser_dependent": ["design-review"],
  "imported_skills": [
    "office-hours","plan-ceo-review","autoplan",
    "plan-eng-review","plan-devex-review","devex-review","investigate","codex","pr-review",
    "plan-design-review","design-consultation","design-shotgun","design-html","design-review",
    "cso",
    "retro","document-release","learn","pair-agent","plan-tune"
  ]
}
EOF
```

- [ ] **Step 2: Update manifest**

```bash
source lib/manifest.sh
gpowers_manifest_set_installed roles true
gpowers_manifest_set roles skill_count 20
gpowers_manifest_set roles upstream '"gstack@main"'
gpowers_manifest_set roles renames '{"review":"pr-review"}'
```

- [ ] **Step 3: Failing test**

```bash
cat > tests/unit/roles/manifest-records-roles.bats <<'EOF'
#!/usr/bin/env bats

setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest installed: roles" { [ "$(jq -r '.modules.roles.installed' < "$M")" = "true" ]; }
@test "manifest skill_count 20" { [ "$(jq -r '.modules.roles.skill_count' < "$M")" = "20" ]; }
@test "manifest renames review→pr-review" {
  [ "$(jq -r '.modules.roles.renames.review' < "$M")" = "pr-review" ]
}
EOF
```

Run: PASS.

- [ ] **Step 4: Commit**

```bash
git add roles/upstream-source.json manifest.json tests/unit/roles/manifest-records-roles.bats
git commit -m "feat(roles): upstream-source + manifest record 20 roles with rename"
```

---

## Task 10: End-to-end smoke

**Files:**
- Create: `tests/integration/roles/roles-smoke.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/roles/roles-smoke.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "pr-review skill exists and slash is /pr-review" {
  [ -f "$REPO/roles/skills/pr-review/SKILL.md" ]
  grep -q '^slash: /pr-review$' "$REPO/roles/skills/pr-review/SKILL.md"
}

@test "cso skill exists with namespace roles" {
  grep -q '^namespace: roles$' "$REPO/roles/skills/cso/SKILL.md"
}

@test "design-review is the only role with requires-driver" {
  count=$(grep -l '^requires-driver: browser$' "$REPO/roles/skills"/*/SKILL.md | wc -l)
  [ "$count" -eq 1 ]
  grep -q '^requires-driver: browser$' "$REPO/roles/skills/design-review/SKILL.md"
}

@test "all 20 roles round-trip through gpowers-path home" {
  # Sanity that the install layout is correct relative to GPOWERS_HOME
  count=$(find "$(gpowers-path home)/roles/skills" -name SKILL.md | wc -l)
  [ "$count" -eq 20 ]
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/integration/roles/roles-smoke.bats`
Expected: PASS (4 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/roles/roles-smoke.bats
git commit -m "test(roles): e2e smoke for 20 roles + design-review browser"
```

---

## Self-Review

### 1. Spec coverage (§3 of design)

| Spec entry | Task |
|---|---|
| 20 roles imported | Tasks 4, 5 |
| /review → /pr-review rename | Task 5 |
| design-review uses browser driver | Task 6 |
| namespace: roles + upstream: gstack@main | Tasks 2, 3 |
| no slash-command collisions | Task 8 |
| no gstack path leakage | Task 7 |
| upstream-source + manifest record | Task 9 |

### 2. Placeholder scan

No TBDs. The pair-agent "degraded on non-CC" caveat is documented in the skill body itself (preserved from gstack source) — no design action needed in this plan.

### 3. Type / name consistency

- 20 expected slashes match the inventory table exactly.
- `requires-driver: browser` token matches Plan #5 frontmatter.
- `_gpowers-import-role.sh` matches `_gpowers-import-tool.sh` from Plan #4 in argument shape (src, dst).

### 4. Decomposition

10 tasks. The 19-skill bulk import is one task; the unique `review → pr-review` rename is isolated as its own task (Task 5) so reviewers can audit the rename machinery separately. design-review browser preamble is its own task.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-roles.md`. Depends on Plans #1 and #4. (Task 6 also needs Plans #3 and #5 — that single task may run after Plan #5 completes.) Choose subagent-driven or inline at execution time.
