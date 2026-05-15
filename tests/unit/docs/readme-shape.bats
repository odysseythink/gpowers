#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../README.md"; }

@test "README.md exists" { [ -f "$F" ]; }
@test "README has Quickstart section" { grep -qi "quickstart\|getting started" "$F"; }
@test "README links to ARCHITECTURE.md" { grep -q "ARCHITECTURE.md" "$F"; }
@test "README links to INSTALL.md" { grep -q "INSTALL.md" "$F"; }
@test "README mentions the four modules" {
  for m in core roles tools business; do
    grep -qw "$m" "$F" || { echo "missing $m"; return 1; }
  done
}
