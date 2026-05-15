#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  TESTREPO="${BATS_TMPDIR}/testrepo-$$"
  mkdir -p "$TESTREPO/sub/nested"
  unset GPOWERS_HOME GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

DETECT="${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"

@test "GPOWERS_PROJECT_DIR override is honored verbatim" {
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd /tmp
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
  [ "$GPOWERS_PROJECT_DATA" = "$TESTREPO/.gpowers" ]
}

@test "detects .gpowers/ directory by walking up" {
  mkdir -p "$TESTREPO/.gpowers"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
}

@test "detects .git/ directory by walking up if no .gpowers" {
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO" ]
}

@test ".gpowers/ takes priority over .git" {
  mkdir -p "$TESTREPO/.git"
  mkdir -p "$TESTREPO/sub/.gpowers"
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DIR" = "$TESTREPO/sub" ]
}

@test "no project marker means empty PROJECT_DIR" {
  cd "$TESTREPO/sub/nested"
  source "$DETECT"
  [ -z "$GPOWERS_PROJECT_DIR" ]
  [ -z "$GPOWERS_PROJECT_DATA" ]
}

@test "PROJECT_DATA is PROJECT_DIR/.gpowers when project detected" {
  mkdir -p "$TESTREPO/.git"
  cd "$TESTREPO"
  source "$DETECT"
  [ "$GPOWERS_PROJECT_DATA" = "$TESTREPO/.gpowers" ]
}
