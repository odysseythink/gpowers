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

@test "gpowers-path home prints $GPOWERS_HOME" {
  result="$("$PATH_BIN" home)"
  [ "$result" = "$HOME/.gpowers" ]
}

@test "gpowers-path config prints $GPOWERS_CONFIG" {
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

@test "gpowers-path project resolves to repo/.gpowers when in project" {
  TESTREPO="${BATS_TMPDIR}/proj-$$"
  mkdir -p "$TESTREPO/.git" "$TESTREPO/sub"
  cd "$TESTREPO/sub"
  result="$("$PATH_BIN" project)"
  [ "$result" = "$TESTREPO/.gpowers" ]
}

@test "gpowers-path project plans ceo joins subpath in project mode" {
  TESTREPO="${BATS_TMPDIR}/proj2-$$"
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO"
  result="$("$PATH_BIN" project plans ceo)"
  [ "$result" = "$TESTREPO/.gpowers/plans/ceo" ]
}

@test "gpowers-path project falls back to global data when no project" {
  cd "$BATS_TMPDIR"
  result="$("$PATH_BIN" project sessions)"
  [ "$result" = "$HOME/.gpowers/data/sessions" ]
}

@test "GPOWERS_PROJECT_DIR override is honored by project kind" {
  TESTREPO="${BATS_TMPDIR}/proj3-$$"
  mkdir -p "$TESTREPO"
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd "$BATS_TMPDIR"
  result="$("$PATH_BIN" project plans)"
  [ "$result" = "$TESTREPO/.gpowers/plans" ]
}
