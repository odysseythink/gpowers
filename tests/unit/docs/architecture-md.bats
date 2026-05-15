#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/ARCHITECTURE.md"; }

@test "ARCHITECTURE.md exists" { [ -f "$F" ]; }

@test "names all four modules" {
  for m in core roles tools business; do
    grep -qw "$m" "$F" || { echo "missing module: $m"; return 1; }
  done
}

@test "describes dual-track triggering" {
  grep -qi "dual.track\|auto.*trigger\|explicit.*trigger" "$F"
}

@test "references browser driver abstraction" {
  grep -qi "browser.*driver\|9.verb\|drivers/browser" "$F"
}

@test "references runtime layout (global + project)" {
  grep -qi "runtime layout\|RUNTIME_LAYOUT\|project.*directory" "$F"
}
