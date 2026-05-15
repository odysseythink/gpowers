#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/DRIVERS.md"; }

@test "DRIVERS.md exists" { [ -f "$F" ]; }

@test "names all 9 verbs" {
  for v in open click type read screenshot wait eval cookies close; do
    grep -q "browser\.$v" "$F" || { echo "missing verb: $v"; return 1; }
  done
}

@test "describes how to add a new driver" {
  grep -qi "add a new driver\|adding a driver\|new driver" "$F"
}

@test "documents JSON wire format" {
  grep -qi "json.*stdin\|stdin.*json" "$F"
}
