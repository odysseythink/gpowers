#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-init-$$"
  mkdir -p "$HOME"
  TESTREPO="${BATS_TMPDIR}/initrepo-$$"
  mkdir -p "$TESTREPO/.git"
  unset GPOWERS_PROJECT_DIR
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

INIT_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-init"

@test "gpowers-init creates <repo>/.gpowers/ tree" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers" ]
  [ -d "$TESTREPO/.gpowers/plans" ]
  [ -d "$TESTREPO/.gpowers/designs" ]
  [ -d "$TESTREPO/.gpowers/evals" ]
  [ -d "$TESTREPO/.gpowers/sessions" ]
  [ -d "$TESTREPO/.gpowers/retros" ]
  [ -d "$TESTREPO/.gpowers/learnings" ]
  [ -d "$TESTREPO/.gpowers/investigate" ]
  [ -d "$TESTREPO/.gpowers/canary" ]
  [ -d "$TESTREPO/.gpowers/health" ]
  [ -d "$TESTREPO/.gpowers/benchmark" ]
}

@test "gpowers-init writes .gitignore" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -f "$TESTREPO/.gpowers/.gitignore" ]
  grep -q "^logs/$" "$TESTREPO/.gpowers/.gitignore"
  grep -q "^tmp/$" "$TESTREPO/.gpowers/.gitignore"
}

@test "gpowers-init writes README explaining the directory" {
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -f "$TESTREPO/.gpowers/README.md" ]
  grep -qi "gpowers" "$TESTREPO/.gpowers/README.md"
}

@test "gpowers-init is idempotent" {
  cd "$TESTREPO"
  "$INIT_BIN"
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
}

@test "gpowers-init refuses if not in a project (no .git, no GPOWERS_PROJECT_DIR)" {
  rm -rf "$TESTREPO/.git"
  cd "$TESTREPO"
  run "$INIT_BIN"
  [ "$status" -ne 0 ]
  [[ "$output" == *"no project"* ]]
}

@test "gpowers-init honors GPOWERS_PROJECT_DIR even outside a git repo" {
  rm -rf "$TESTREPO/.git"
  export GPOWERS_PROJECT_DIR="$TESTREPO"
  cd /tmp
  run "$INIT_BIN"
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers/plans" ]
}
