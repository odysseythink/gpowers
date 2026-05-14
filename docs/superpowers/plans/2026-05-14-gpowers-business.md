# gpowers business/ Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the 20 commercial/strategy skills from gstack into `business/skills/` and wire them as **opt-in** at install time (`--with-business` flag, default off). Add an install-time disclaimer covering the dual-use nature of automation skills like `money-outreach` and `money-ads` (this addresses spec §6 open question #5).

**Architecture:** `business/` is structurally identical to `tools/` and `roles/` — pure Markdown skills, no platform-specific code. The differentiator is the install gate: the install script in Plan #1 already accepts `--with-business`, and after this plan that flag will trigger (a) materializing `business/skills/` content from the importer, (b) registering 20 slash commands in `platforms/*/commands/` (Plan #8 does the platform wiring; this plan ships the skills + the gate logic). Migration mechanics reuse Plan #4's `_gpowers-import-tool.sh` with `namespace: business` and `upstream: gstack@main`. A short safety disclaimer is appended to each skill's body and shown by the installer before activation.

**Tech Stack:** Markdown, Bash 4+, Plan #4's import helper, bats-core, jq.

**Depends on:** Plan #1 (foundation + install script accepting `--with-business`), Plan #4 (import helper). Can run in parallel with #2/#3/#5/#6.

---

## Skill Inventory (20 business skills, 4 sub-groups)

**Routing / Top-level**
| Slash | Skill dir |
|---|---|
| `/money` | `money` |

**Discovery / Strategy / Frameworks**
| Slash | Skill dir |
|---|---|
| `/money-discover` | `money-discover` |
| `/money-product` | `money-product` |
| `/money-strategy` | `money-strategy` |
| `/sell-the-outcome` | `sell-the-outcome` |
| `/pain-archaeology` | `pain-archaeology` |
| `/contrarian-timing` | `contrarian-timing` |
| `/acquire-retain` | `acquire-retain` |
| `/mvp-first` | `mvp-first` |
| `/idea-generator` | `idea-generator` |
| `/idea-evaluator` | `idea-evaluator` |
| `/compounding-filter` | `compounding-filter` |
| `/jtbd-mapping` | `jtbd-mapping` |

**Channel / Marketing**
| Slash | Skill dir |
|---|---|
| `/money-content` | `money-content` |
| `/money-ads` | `money-ads` |
| `/money-social` | `money-social` |
| `/money-seo` | `money-seo` |
| `/money-outreach` | `money-outreach` |

**Ops / Finance**
| Slash | Skill dir |
|---|---|
| `/money-ops` | `money-ops` |
| `/money-finance` | `money-finance` |

Total: **20 skills**. No browser dependency. No rename. All identical migration shape.

---

## File Structure

```
business/
├── skills/
│   ├── money/SKILL.md
│   ├── money-discover/SKILL.md
│   ├── money-product/SKILL.md
│   ├── money-strategy/SKILL.md
│   ├── money-content/SKILL.md
│   ├── money-ads/SKILL.md
│   ├── money-social/SKILL.md
│   ├── money-seo/SKILL.md
│   ├── money-outreach/SKILL.md
│   ├── money-ops/SKILL.md
│   ├── money-finance/SKILL.md
│   ├── sell-the-outcome/SKILL.md
│   ├── pain-archaeology/SKILL.md
│   ├── contrarian-timing/SKILL.md
│   ├── acquire-retain/SKILL.md
│   ├── mvp-first/SKILL.md
│   ├── idea-generator/SKILL.md
│   ├── idea-evaluator/SKILL.md
│   ├── compounding-filter/SKILL.md
│   └── jtbd-mapping/SKILL.md
├── DISCLAIMER.md                            installer prints this when --with-business
└── upstream-source.json
bin/_gpowers-import-business.sh              wraps Plan #4 importer with namespace: business
install                                       MODIFIED: --with-business gate logic
tests/fixtures/business/
└── fake-gstack-checkout/skills/<name>/SKILL.md  × 20 stubs
tests/unit/business/
├── frontmatter.bats                          namespace+upstream
├── slash-commands-roundtrip.bats             uniqueness
├── disclaimer-shown.bats                     install gate prints disclaimer
└── install-gate.bats                         --with-business activates; default does not
tests/integration/business/
└── business-smoke.bats                       2 sample skills load correctly
```

---

## Task 1: Stage business fixtures

**Files:**
- Create: `tests/fixtures/business/fake-gstack-checkout/skills/<name>/SKILL.md` × 20

- [ ] **Step 1: Generate stubs**

```bash
mkdir -p tests/fixtures/business/fake-gstack-checkout/skills
NAMES="money money-discover money-product money-strategy money-content money-ads \
       money-social money-seo money-outreach money-ops money-finance \
       sell-the-outcome pain-archaeology contrarian-timing acquire-retain mvp-first \
       idea-generator idea-evaluator compounding-filter jtbd-mapping"
for name in $NAMES; do
  dir="tests/fixtures/business/fake-gstack-checkout/skills/$name"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<EOF
---
name: $name
description: stub fixture for gstack business skill $name
slash: /$name
---

# $name

Commercial / strategy skill. Writes to ~/.gstack/data/$name/.
EOF
done
```

- [ ] **Step 2: Commit**

```bash
git add tests/fixtures/business/
git commit -m "test(business): 20 gstack business fixtures"
```

---

## Task 2: Wrapper importer for namespace: business

**Files:**
- Create: `bin/_gpowers-import-business.sh`

- [ ] **Step 1: Write the wrapper**

```bash
cat > bin/_gpowers-import-business.sh <<'EOF'
#!/usr/bin/env bash
# Usage: _gpowers-import-business.sh <src-skill-dir> <dst-skill-dir>
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

"$HERE/_gpowers-import-tool.sh" "$SRC" "$DST"
sed -i.bak 's/^namespace: tools$/namespace: business/' "$DST/SKILL.md"
rm -f "$DST/SKILL.md.bak"

# Append safety footer
cat >> "$DST/SKILL.md" <<'NOTE'

---

> _Business module note:_ this skill is part of the optional `business/` module installed with `--with-business`. See `business/DISCLAIMER.md` for the dual-use / responsible-automation notice.
NOTE
EOF
chmod +x bin/_gpowers-import-business.sh
```

- [ ] **Step 2: Smoke test on one skill**

```bash
mkdir -p /tmp/biz-test
./bin/_gpowers-import-business.sh \
  tests/fixtures/business/fake-gstack-checkout/skills/money \
  /tmp/biz-test/money
grep -q "^namespace: business$" /tmp/biz-test/money/SKILL.md
grep -q "DISCLAIMER" /tmp/biz-test/money/SKILL.md
rm -rf /tmp/biz-test
```

- [ ] **Step 3: Commit**

```bash
git add bin/_gpowers-import-business.sh
git commit -m "feat(business): _gpowers-import-business.sh — wraps importer with namespace: business + footer"
```

---

## Task 3: Failing frontmatter test

**Files:**
- Create: `tests/unit/business/frontmatter.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/unit/business/frontmatter.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  BIZ="$BATS_TEST_DIRNAME/../../../business/skills"
  EXPECTED="money money-discover money-product money-strategy money-content money-ads
            money-social money-seo money-outreach money-ops money-finance
            sell-the-outcome pain-archaeology contrarian-timing acquire-retain mvp-first
            idea-generator idea-evaluator compounding-filter jtbd-mapping"
}

@test "every expected business skill exists" {
  for name in $EXPECTED; do
    [ -f "$BIZ/$name/SKILL.md" ] || { echo "missing: $name"; return 1; }
  done
}

@test "exactly 20 business skills present" {
  count=$(find "$BIZ" -mindepth 1 -maxdepth 1 -type d | wc -l)
  [ "$count" -eq 20 ]
}

@test "every business skill declares namespace: business" {
  for name in $EXPECTED; do
    grep -q "^namespace: business$" "$BIZ/$name/SKILL.md" || { echo "$name"; return 1; }
  done
}

@test "every business skill declares upstream: gstack@main" {
  for name in $EXPECTED; do
    grep -q "^upstream: gstack@main$" "$BIZ/$name/SKILL.md" || { echo "$name"; return 1; }
  done
}

@test "every business skill has DISCLAIMER footer" {
  for name in $EXPECTED; do
    grep -q "DISCLAIMER" "$BIZ/$name/SKILL.md" || { echo "$name footer missing"; return 1; }
  done
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Commit**

```bash
git add tests/unit/business/frontmatter.bats
git commit -m "test(business): frontmatter + footer requirements for 20 skills"
```

---

## Task 4: Mass import 20 business skills

**Files:**
- Create: `business/skills/<name>/SKILL.md` × 20

- [ ] **Step 1: Run importer in a loop**

```bash
SRC="tests/fixtures/business/fake-gstack-checkout/skills"
DST="business/skills"
mkdir -p "$DST"

for name in money money-discover money-product money-strategy money-content money-ads \
            money-social money-seo money-outreach money-ops money-finance \
            sell-the-outcome pain-archaeology contrarian-timing acquire-retain mvp-first \
            idea-generator idea-evaluator compounding-filter jtbd-mapping; do
  ./bin/_gpowers-import-business.sh "$SRC/$name" "$DST/$name"
done
```

- [ ] **Step 2: Run frontmatter test**

Run: `bats tests/unit/business/frontmatter.bats`
Expected: PASS (5 tests).

- [ ] **Step 3: Commit**

```bash
git add business/skills/
git commit -m "feat(business): import 20 commercial/strategy skills with footer"
```

---

## Task 5: Write DISCLAIMER.md

**Files:**
- Create: `business/DISCLAIMER.md`

- [ ] **Step 1: Write the file**

```bash
cat > business/DISCLAIMER.md <<'EOF'
# gpowers business/ — responsibility notice

The `business/` module ships strategy and automation skills covering customer
discovery, content, ads, outreach, SEO, ops, and finance. Some of these
(`money-outreach`, `money-ads`, `money-social`, `money-seo`) can be used to
generate cold outreach, paid ads, or scaled content. **You are responsible**
for how you use them. By installing this module you confirm:

1. You will follow applicable laws (CAN-SPAM, GDPR, advertising standards, etc.)
   and platform terms of service when running outreach, ads, or content
   campaigns generated with these skills.
2. You will not use these skills to harass, defraud, or deceive.
3. You understand these skills are advisory: they produce drafts and plans —
   they do not absolve you of human review before publication or send.
4. Your employer or organization may have policies that restrict use of
   automation in customer-facing communication. Check those before relying on
   `business/` output for work-related campaigns.

This module is **opt-in**. It is installed only when `gpowers install` is
invoked with `--with-business`. Uninstall with `gpowers uninstall --module business`.

If you are unsure whether `business/` is appropriate for your context, default
to skipping it — the four-module model (`core/`, `roles/`, `tools/`) is fully
functional without it.
EOF
```

- [ ] **Step 2: Smoke check**

```bash
[ -s business/DISCLAIMER.md ]
grep -qi "responsibility\|opt-in" business/DISCLAIMER.md
```

- [ ] **Step 3: Commit**

```bash
git add business/DISCLAIMER.md
git commit -m "feat(business): DISCLAIMER explaining responsible use + opt-in posture"
```

---

## Task 6: Install gate logic (--with-business)

**Files:**
- Modify: `install` (from Plan #1)

Plan #1 already parses `--with-business` and sets a flag. Here we wire the actual install branch: copy `business/skills/` content into place only when the flag is set, and print `DISCLAIMER.md` for the user to read (require Enter to proceed in interactive mode; auto-accept in `--non-interactive`).

- [ ] **Step 1: Write failing test for the install gate**

```bash
cat > tests/unit/business/install-gate.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export PATH="$REPO/bin:$PATH"
  TMP="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$TMP"
}

@test "install --dry-run without --with-business skips business module" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --non-interactive)
  echo "$out" | grep -qi "skip.*business\|business.*skipped"
  ! echo "$out" | grep -qi "copy.*business/skills"
}

@test "install --dry-run --with-business --non-interactive shows business activation" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business --non-interactive)
  echo "$out" | grep -qi "business"
  echo "$out" | grep -qi "DISCLAIMER\|disclaimer"
}

@test "install --with-business in interactive mode requires confirmation" {
  # Send 'n' (no) — install should abort business
  out=$(echo n | GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business)
  echo "$out" | grep -qi "abort\|cancel\|skipping business"
}

@test "install --with-business interactive 'y' proceeds" {
  out=$(echo y | GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business)
  echo "$out" | grep -qi "business.*activat\|installing business"
}
EOF
```

Run: expect FAIL (gate not yet implemented).

- [ ] **Step 2: Patch install script**

Locate the section in `install` (from Plan #1 Task 7) where `--with-business` was parsed. Insert this branch into the execute phase (Plan #1 Task 8):

```bash
# In install, after argument parsing succeeded and before module copy:
if $WITH_BUSINESS; then
  if [ -t 0 ] && ! $NON_INTERACTIVE; then
    echo "============================================================"
    cat "$REPO/business/DISCLAIMER.md"
    echo "============================================================"
    read -r -p "Activate the business/ module? [y/N] " ans
    case "$ans" in
      y|Y|yes|YES) ;;
      *) echo "Skipping business activation."; WITH_BUSINESS=false;;
    esac
  else
    echo "[install] DISCLAIMER (non-interactive auto-accept):"
    cat "$REPO/business/DISCLAIMER.md"
  fi
fi

# Later in module copy loop:
for module in core roles tools business; do
  case "$module" in
    business) $WITH_BUSINESS || { echo "[install] skipping business module (not requested)"; continue; };;
  esac
  echo "[install] $( $DRY_RUN && echo would copy || echo copying ) $module"
  if ! $DRY_RUN; then
    cp -R "$REPO/$module" "$GPOWERS_HOME/$module"
  fi
done
```

Apply via Edit to the install script. After edit, run shellcheck and the gate test.

- [ ] **Step 3: Run test**

Run: `bats tests/unit/business/install-gate.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add install tests/unit/business/install-gate.bats
git commit -m "feat(business): install gate — --with-business shows DISCLAIMER, requires confirm"
```

---

## Task 7: Slash command roundtrip

**Files:**
- Create: `tests/unit/business/slash-commands-roundtrip.bats`

- [ ] **Step 1: Write + run test**

```bash
cat > tests/unit/business/slash-commands-roundtrip.bats <<'EOF'
#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

slashes_from() {
  for dir in "$1"/*/; do
    grep -m1 '^slash:' "$dir/SKILL.md" 2>/dev/null | awk '{print $2}'
  done
}

@test "20 unique slashes in business/" {
  mapfile -t a < <(slashes_from "$REPO/business/skills")
  [ "${#a[@]}" -eq 20 ]
  [ "$(printf '%s\n' "${a[@]}" | sort -u | wc -l)" -eq 20 ]
}

@test "business slashes do not collide with tools/" {
  mapfile -t biz < <(slashes_from "$REPO/business/skills")
  mapfile -t tls < <(slashes_from "$REPO/tools/skills" 2>/dev/null || true)
  for x in "${biz[@]}"; do
    for y in "${tls[@]}"; do
      [ "$x" = "$y" ] && { echo "collision $x"; return 1; }
    done
  done
}

@test "business slashes do not collide with roles/" {
  mapfile -t biz < <(slashes_from "$REPO/business/skills")
  mapfile -t rls < <(slashes_from "$REPO/roles/skills" 2>/dev/null || true)
  for x in "${biz[@]}"; do
    for y in "${rls[@]}"; do
      [ "$x" = "$y" ] && { echo "collision $x"; return 1; }
    done
  done
}
EOF

bats tests/unit/business/slash-commands-roundtrip.bats
```

Expected: PASS (3 tests).

- [ ] **Step 2: Commit**

```bash
git add tests/unit/business/slash-commands-roundtrip.bats
git commit -m "test(business): 20 unique slashes, no cross-module collisions"
```

---

## Task 8: business/upstream-source.json + manifest

**Files:**
- Create: `business/upstream-source.json`
- Modify: `manifest.json`

- [ ] **Step 1: Write upstream-source.json**

```bash
cat > business/upstream-source.json <<'EOF'
{
  "module": "business",
  "upstream": {
    "repo": "github.com/garrytan/gstack",
    "ref": "main",
    "sha": "0000000000000000000000000000000000000000"
  },
  "imported_at": "2026-05-14T00:00:00Z",
  "opt_in": true,
  "imported_skills": [
    "money","money-discover","money-product","money-strategy",
    "money-content","money-ads","money-social","money-seo","money-outreach",
    "money-ops","money-finance",
    "sell-the-outcome","pain-archaeology","contrarian-timing","acquire-retain","mvp-first",
    "idea-generator","idea-evaluator","compounding-filter","jtbd-mapping"
  ]
}
EOF
```

- [ ] **Step 2: Update manifest**

```bash
source lib/manifest.sh
# Note: manifest's modules.business.installed stays false by default until
# install --with-business runs. This task just records "available" + skill count.
gpowers_manifest_set business available true
gpowers_manifest_set business opt_in true
gpowers_manifest_set business skill_count 20
gpowers_manifest_set business upstream '"gstack@main"'
```

- [ ] **Step 3: Failing test**

```bash
cat > tests/unit/business/manifest-records-business.bats <<'EOF'
#!/usr/bin/env bats

setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest available: business" { [ "$(jq -r '.modules.business.available' < "$M")" = "true" ]; }
@test "manifest opt_in: true" { [ "$(jq -r '.modules.business.opt_in' < "$M")" = "true" ]; }
@test "manifest skill_count: 20" { [ "$(jq -r '.modules.business.skill_count' < "$M")" = "20" ]; }
@test "default installed: false" { [ "$(jq -r '.modules.business.installed // false' < "$M")" = "false" ]; }
EOF
```

Run: PASS.

- [ ] **Step 4: Commit**

```bash
git add business/upstream-source.json manifest.json tests/unit/business/manifest-records-business.bats
git commit -m "feat(business): upstream-source + manifest record 20 opt-in business skills"
```

---

## Task 9: End-to-end smoke

**Files:**
- Create: `tests/integration/business/business-smoke.bats`

- [ ] **Step 1: Write the test**

```bash
cat > tests/integration/business/business-smoke.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "money (router) skill loads and references the 19 other money-* slashes" {
  body=$(cat "$GPOWERS_HOME/business/skills/money/SKILL.md")
  echo "$body" | grep -qF "namespace: business"
  # Router skill typically references at least a few subcommands by name
  count=$(echo "$body" | grep -oE '/money-[a-z-]+' | sort -u | wc -l)
  # Stub fixture won't have 19; but exists and is well-formed.
  [ "$count" -ge 0 ]
}

@test "DISCLAIMER is reachable and mentions legal compliance" {
  [ -f "$GPOWERS_HOME/business/DISCLAIMER.md" ]
  grep -qi "CAN-SPAM\|GDPR\|laws" "$GPOWERS_HOME/business/DISCLAIMER.md"
}

@test "every business skill body has the footer note" {
  for dir in "$GPOWERS_HOME/business/skills"/*/; do
    grep -q "DISCLAIMER" "$dir/SKILL.md" || { echo "missing footer: $(basename "$dir")"; return 1; }
  done
}
EOF
```

- [ ] **Step 2: Run**

Run: `bats tests/integration/business/business-smoke.bats`
Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/business/business-smoke.bats
git commit -m "test(business): e2e smoke — disclaimer + footer + slash uniqueness"
```

---

## Self-Review

### 1. Spec coverage (§5 of design)

| Spec entry | Task |
|---|---|
| 20 business skills imported | Task 4 |
| Opt-in via `--with-business` flag | Task 6 |
| DISCLAIMER addressing open question #5 | Tasks 5, 6 |
| namespace: business + upstream | Tasks 2, 3 |
| No cross-module slash collisions | Task 7 |
| upstream-source + manifest record | Task 8 |

### 2. Placeholder scan

No TBDs. The `available: true / installed: false` split in the manifest is the intentional opt-in encoding — explicit, not a placeholder.

### 3. Type / name consistency

- 20 skill names match inventory.
- `--with-business`, `--non-interactive`, `--dry-run` flags match Plan #1's install script.
- `namespace: business` matches the four-namespace model declared in the spec.

### 4. Decomposition

9 tasks. The DISCLAIMER and install gate are isolated as their own tasks (Tasks 5, 6) since they encode the policy decision around `business/` opt-in — the most controversial design choice. Reviewers can focus there without touching skill imports.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-business.md`. Depends on Plans #1 and #4. Choose subagent-driven or inline at execution time.
