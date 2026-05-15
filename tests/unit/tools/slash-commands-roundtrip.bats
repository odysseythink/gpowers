#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
}

@test "each tools/ skill declares a unique slash command" {
  > "$BATS_TEST_TMPDIR/slashes.txt"
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    [ -n "$slash" ] || { echo "$name has no slash:"; return 1; }
    if grep -qFx "$slash" "$BATS_TEST_TMPDIR/slashes.txt"; then
      echo "duplicate slash: $slash ($name)"
      return 1
    fi
    echo "$slash" >> "$BATS_TEST_TMPDIR/slashes.txt"
  done
}

@test "tools/ slash commands do not collide with core/ skills" {
  for dir in "$REPO/tools/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    cname="${slash#/}"
    if [ -d "$REPO/core/skills/$cname" ]; then
      echo "tools/$(basename "$dir") collides with core/$cname"
      return 1
    fi
  done
}
