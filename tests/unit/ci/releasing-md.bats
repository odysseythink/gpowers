#!/usr/bin/env bats

setup() { R="$BATS_TEST_DIRNAME/../../../RELEASING.md"; }

@test "RELEASING.md exists" { [ -f "$R" ]; }
@test "RELEASING.md names semver categories MAJOR/MINOR/PATCH" {
  for kw in MAJOR MINOR PATCH; do grep -q "$kw" "$R" || { echo "$kw"; return 1; }; done
}
@test "RELEASING.md references release.yml flow" {
  grep -qi "release.yml\|release workflow" "$R"
}
