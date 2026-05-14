#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  unset GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
  unset GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP
}

teardown() {
  rm -rf "$HOME"
}

PATH_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-path"

@test "gpowers-path home prints \$GPOWERS_HOME" {
  result="$("$PATH_BIN" home)"
  [ "$result" = "$HOME/.gpowers" ]
}

@test "gpowers-path config prints \$GPOWERS_CONFIG" {
  result="$("$PATH_BIN" config)"
  [ "$result" = "$HOME/.gpowers/config" ]
}

@test "gpowers-path config compact-rules joins subpaths" {
  result="$("$PATH_BIN" config compact-rules)"
  [ "$result" = "$HOME/.gpowers/config/compact-rules" ]
}

@test "gpowers-path cache models joins subpaths" {
  result="$("$PATH_BIN" cache models)"
  [ "$result" = "$HOME/.gpowers/cache/models" ]
}

@test "gpowers-path with multiple subpaths joins them" {
  result="$("$PATH_BIN" state security attempts)"
  [ "$result" = "$HOME/.gpowers/state/security/attempts" ]
}

@test "gpowers-path with unknown kind exits 2" {
  run "$PATH_BIN" unknown
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown kind"* ]]
}

@test "gpowers-path with no args exits 2 and prints usage" {
  run "$PATH_BIN"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage:"* ]]
}

@test "GPOWERS_HOME override is honored" {
  export GPOWERS_HOME=/custom/gp
  result="$("$PATH_BIN" config)"
  [ "$result" = "/custom/gp/config" ]
}
