#!/usr/bin/env bats

load ../helpers/setup

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  unset GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
  unset GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP
  unset GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
}

teardown() {
  rm -rf "$HOME"
}

@test "defaults: GPOWERS_HOME = \$HOME/.gpowers" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_HOME" = "$HOME/.gpowers" ]
}

@test "defaults: GPOWERS_CONFIG = \$GPOWERS_HOME/config" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "$HOME/.gpowers/config" ]
}

@test "override: GPOWERS_HOME=/custom honored" {
  export GPOWERS_HOME=/custom
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_HOME" = "/custom" ]
  [ "$GPOWERS_CONFIG" = "/custom/config" ]
}

@test "override: GPOWERS_CONFIG independent of HOME" {
  export GPOWERS_CONFIG=/etc/gpowers
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "/etc/gpowers" ]
  [ "$GPOWERS_HOME" = "$HOME/.gpowers" ]
}

@test "all 7 dirs defined: HOME CONFIG STATE CACHE DATA ANALYTICS TMP" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  for var in GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP; do
    [ -n "${!var}" ]
  done
}

@test "sourcing twice is idempotent" {
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  first="$GPOWERS_CONFIG"
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$GPOWERS_CONFIG" = "$first" ]
}

@test "sourcing does not change cwd" {
  cd "$BATS_TMPDIR"
  before="$(pwd)"
  source "${BATS_TEST_DIRNAME}/../../lib/runtime-dirs.sh"
  [ "$(pwd)" = "$before" ]
}
