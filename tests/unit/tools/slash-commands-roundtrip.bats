#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
}

# Helper: read explicit slash frontmatter, or default to /<skill-dir>
get_slash() {
  local dir="$1"
  local name=$(basename "$dir")
  local slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
  if [ -z "$slash" ]; then
    slash="/$name"
  fi
  printf '%s' "$slash"
}

@test "each tools/ skill declares a unique slash command" {
  > "$BATS_TEST_TMPDIR/slashes.txt"
  for dir in "$REPO/tools/skills"/*/; do
    name=$(basename "$dir")
    slash=$(get_slash "$dir")
    if grep -qFx "$slash" "$BATS_TEST_TMPDIR/slashes.txt"; then
      echo "duplicate slash: $slash ($name)"
      return 1
    fi
    echo "$slash" >> "$BATS_TEST_TMPDIR/slashes.txt"
  done
}

@test "tools/ slash commands do not collide with core/ skills" {
  for dir in "$REPO/tools/skills"/*/; do
    slash=$(get_slash "$dir")
    cname="${slash#/}"
    if [ -d "$REPO/core/skills/$cname" ]; then
      echo "tools/$(basename "$dir") collides with core/$cname"
      return 1
    fi
  done
}
