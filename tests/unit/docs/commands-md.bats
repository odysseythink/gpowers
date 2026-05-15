#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  F="$REPO/docs/COMMANDS.md"
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "COMMANDS.md exists" { [ -f "$F" ]; }

@test "COMMANDS.md has generated block" {
  grep -q "gpowers:generated:begin" "$F"
}

@test "COMMANDS.md mentions /pr-review explicitly" {
  grep -q "/pr-review" "$F"
}

@test "COMMANDS.md notes /review is deprecated" {
  grep -qi "/review.*deprecat\|deprecat.*/review\|rename.*review" "$F"
}
