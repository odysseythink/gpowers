#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/RUNTIME_LAYOUT.md"; }

@test "RUNTIME_LAYOUT.md exists" { [ -f "$F" ]; }

@test "documents 2-layer split (global + project)" {
  grep -qi "global" "$F"
  grep -qi "project" "$F"
}

@test "names all GPOWERS_* env vars" {
  for v in GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP GPOWERS_PROJECT_DIR; do
    grep -q "$v" "$F" || { echo "missing $v"; return 1; }
  done
}

@test "has migration section" {
  grep -qi "migration\|migrate" "$F"
}
