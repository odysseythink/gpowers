#!/usr/bin/env bats

@test "no literal 'npx playwright' in tools/ skill bodies" {
  REPO="$BATS_TEST_DIRNAME/../../.."
  for dir in "$REPO/tools/skills"/*/; do
    [ -f "$dir/SKILL.md" ] || continue
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    echo "$body" | grep -q "npx playwright" && { echo "$name leak"; return 1; }
  done
  return 0
}
