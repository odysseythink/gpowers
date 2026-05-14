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
