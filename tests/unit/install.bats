#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
}

teardown() {
  rm -rf "$HOME"
}

INSTALL="${BATS_TEST_DIRNAME}/../../install"

@test "install --help exits 0 with usage" {
  run "$INSTALL" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"usage:"* ]]
}

@test "install --dry-run --core-only prints intended actions" {
  run "$INSTALL" --dry-run --core-only
  [ "$status" -eq 0 ]
  [[ "$output" == *"core"* ]]
  [[ "$output" != *"business"* ]]
}

@test "install --dry-run --with-business includes business" {
  run "$INSTALL" --dry-run --with-business
  [ "$status" -eq 0 ]
  [[ "$output" == *"business"* ]]
}

@test "install --dry-run --no-tools skips tools" {
  run "$INSTALL" --dry-run --no-tools
  [ "$status" -eq 0 ]
  [[ "$output" != *"link tools"* ]]
}

@test "install --dry-run --location=/tmp/custom uses custom location" {
  run "$INSTALL" --dry-run --location=/tmp/custom
  [ "$status" -eq 0 ]
  [[ "$output" == *"/tmp/custom"* ]]
}

@test "install --dry-run --platforms=claude-code,kimi restricts platforms" {
  mkdir -p "$HOME/.claude" "$HOME/.kimi" "$HOME/.codex"
  run "$INSTALL" --dry-run --platforms=claude-code,kimi
  [ "$status" -eq 0 ]
  [[ "$output" == *"claude-code"* ]]
  [[ "$output" == *"kimi"* ]]
  [[ "$output" != *"codex"* ]]
}

@test "install --unknown-flag exits non-zero" {
  run "$INSTALL" --unknown-flag
  [ "$status" -ne 0 ]
}
