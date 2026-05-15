#!/usr/bin/env bats

setup() { S="$BATS_TEST_DIRNAME/../../../tools/skills/gpowers-upgrade/SKILL.md"; }

@test "skill no longer marked as stub" {
  ! grep -qi "^\* stub\|^stub\|placeholder" "$S"
  ! grep -qi "Plan #9 \(landed below\|fills body\)" "$S"
}

@test "skill names the four modules" {
  for m in core roles tools business; do
    grep -qw "$m" "$S"
  done
}

@test "skill documents --check, --resume, --dry-run" {
  grep -q -- "--check" "$S"
  grep -q -- "--resume" "$S"
  grep -q -- "--dry-run" "$S"
}

@test "skill explains conflict resolution path" {
  grep -qi "conflict\|merge" "$S"
}
