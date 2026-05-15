#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-cli-$$"
  mkdir -p "$HOME"
}

teardown() {
  rm -rf "$HOME"
}

GP="${BATS_TEST_DIRNAME}/../../bin/gpowers"

@test "gpowers --help lists subcommands" {
  run "$GP" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"init"* ]]
  [[ "$output" == *"path"* ]]
  [[ "$output" == *"detect-platforms"* ]]
}

@test "gpowers path config delegates to gpowers-path" {
  result="$("$GP" path config)"
  [ "$result" = "$HOME/.gpowers/config" ]
}

@test "gpowers detect-platforms delegates to bin script" {
  mkdir -p "$HOME/.claude"
  result="$("$GP" detect-platforms)"
  [[ "$result" == *"claude-code"* ]]
}

@test "gpowers unknown-subcommand exits 2" {
  run "$GP" unknown-subcommand
  [ "$status" -eq 2 ]
}

@test "gpowers with no args prints help and exits 0" {
  run "$GP"
  [ "$status" -eq 0 ]
  [[ "$output" == *"subcommands"* ]] || [[ "$output" == *"usage"* ]]
}
