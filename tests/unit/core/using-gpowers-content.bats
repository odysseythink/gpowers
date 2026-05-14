#!/usr/bin/env bats

setup() {
  SKILL="$BATS_TEST_DIRNAME/../../../core/skills/using-gpowers/SKILL.md"
}

@test "using-gpowers exists" {
  [ -f "$SKILL" ]
}

@test "using-gpowers names all four modules" {
  for mod in core roles tools business; do
    grep -qw "$mod" "$SKILL" || { echo "module missing: $mod"; return 1; }
  done
}

@test "using-gpowers documents dual-track triggering" {
  grep -q "auto" "$SKILL"
  grep -q "explicit\|slash" "$SKILL"
}

@test "using-gpowers teaches namespace tags" {
  for tag in "(core)" "(roles)" "(tools)" "(business)"; do
    grep -qF "$tag" "$SKILL" || { echo "tag missing: $tag"; return 1; }
  done
}

@test "using-gpowers has correct frontmatter" {
  grep -q "^namespace: core$" "$SKILL"
  grep -q "^upstream: gpowers-native$" "$SKILL"
}
