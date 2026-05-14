#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
}

@test "no mcp__claude-in-chrome refs in any tools/ skill body" {
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "mcp__claude-in-chrome"; then
      echo "$name body still contains mcp__claude-in-chrome"
      return 1
    fi
  done
}

@test "no literal '`playwright' or '`npx playwright' CLI commands in tools/ skill bodies" {
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -qE '`(npx +)?playwright +[a-z][^`]*`'; then
      echo "$name body still contains playwright CLI"
      return 1
    fi
  done
}
