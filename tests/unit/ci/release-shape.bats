#!/usr/bin/env bats

setup() {
  REL="$BATS_TEST_DIRNAME/../../../.github/workflows/release.yml"
}

@test "release.yml exists" { [ -f "$REL" ]; }
@test "release.yml triggers on v* tags" { grep -q "tags: \[v\*\]" "$REL"; }
@test "release.yml builds gpowers-<version>.tar.gz" {
  grep -q "gpowers-\${version}.tar.gz\|gpowers-\$version" "$REL"
}
@test "release.yml computes sha256" { grep -q "sha256sum" "$REL"; }
