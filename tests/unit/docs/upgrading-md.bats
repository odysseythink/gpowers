#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/UPGRADING.md"; }

@test "UPGRADING.md exists" { [ -f "$F" ]; }
@test "documents --check" { grep -q "\\--check" "$F"; }
@test "documents --resume" { grep -q "\\--resume" "$F"; }
@test "documents conflict resolution" { grep -qi "conflict\|merge" "$F"; }
@test "names git subtree" { grep -q "git subtree" "$F"; }
