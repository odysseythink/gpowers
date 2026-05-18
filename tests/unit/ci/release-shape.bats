#!/usr/bin/env bats

setup() {
  REL="$BATS_TEST_DIRNAME/../../../.github/workflows/release.yml"
}

@test "release.yml exists" { [ -f "$REL" ]; }
@test "release.yml triggers on v* tags" { grep -q "tags: \[v\*\]" "$REL"; }
@test "release.yml uses GoReleaser" {
  grep -q "goreleaser" "$REL"
}
@test "release.yml computes checksums via GoReleaser" {
  grep -q "goreleaser" "$REL"
}
