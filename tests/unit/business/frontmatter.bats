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
