#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/CONTRIBUTING.md"; }

@test "CONTRIBUTING.md exists" { [ -f "$F" ]; }
@test "describes how to add a skill" {
  grep -qi "add.*skill\|new skill" "$F"
}
@test "describes how to add a driver" {
  grep -qi "add.*driver\|new driver" "$F"
}
@test "links to TDD discipline" { grep -qi "TDD\|test.driven" "$F"; }
@test "references PR conventions" { grep -qi "pull request\|PR" "$F"; }
