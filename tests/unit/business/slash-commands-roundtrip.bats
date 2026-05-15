#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

@test "20 unique slashes in business/" {
  > "$BATS_TEST_TMPDIR/slashes.txt"
  count=0
  for dir in "$REPO/business/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    [ -n "$slash" ] || { echo "missing slash in $(basename "$dir")"; return 1; }
    if grep -qFx "$slash" "$BATS_TEST_TMPDIR/slashes.txt"; then
      echo "duplicate slash: $slash"
      return 1
    fi
    echo "$slash" >> "$BATS_TEST_TMPDIR/slashes.txt"
    count=$((count + 1))
  done
  [ "$count" -eq 20 ]
}

@test "business slashes do not collide with tools/" {
  > "$BATS_TEST_TMPDIR/tools-slashes.txt"
  for dir in "$REPO/tools/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    [ -n "$slash" ] && echo "$slash" >> "$BATS_TEST_TMPDIR/tools-slashes.txt"
  done
  for dir in "$REPO/business/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    if grep -qFx "$slash" "$BATS_TEST_TMPDIR/tools-slashes.txt"; then
      echo "collision $slash"
      return 1
    fi
  done
}

@test "business slashes do not collide with roles/" {
  > "$BATS_TEST_TMPDIR/roles-slashes.txt"
  for dir in "$REPO/roles/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    [ -n "$slash" ] && echo "$slash" >> "$BATS_TEST_TMPDIR/roles-slashes.txt"
  done
  for dir in "$REPO/business/skills"/*/; do
    slash=$(grep -m1 '^slash:' "$dir/SKILL.md" | awk '{print $2}')
    if grep -qFx "$slash" "$BATS_TEST_TMPDIR/roles-slashes.txt"; then
      echo "collision $slash"
      return 1
    fi
  done
}
