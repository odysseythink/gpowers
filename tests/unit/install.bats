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

@test "real install creates ~/.gpowers/ with module dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-real-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  [ -d "$HOME/.gpowers/core" ]
  [ -f "$HOME/.gpowers/manifest.json" ]
  rm -rf "$HOME"
}

@test "real install creates runtime dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-rt-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  for d in config state cache data analytics tmp; do
    [ -d "$HOME/.gpowers/$d" ]
  done
  rm -rf "$HOME"
}

@test "real install symlinks Claude Code plugin dir" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-cc-$$"
  mkdir -p "$HOME/.claude/plugins"
  export HOME
  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
  rm -rf "$HOME"
}

@test "real install updates manifest with installed_modules" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-mf-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  modules="$(jq -r '.installed_modules | join(",")' "$HOME/.gpowers/manifest.json")"
  [ "$modules" = "core" ]
  rm -rf "$HOME"
}

@test "real install with --with-business records business in manifest" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-biz-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  run "$REPO/install" --with-business
  [ "$status" -eq 0 ]
  modules="$(jq -r '.installed_modules | join(",")' "$HOME/.gpowers/manifest.json")"
  [[ "$modules" == *"business"* ]]
  rm -rf "$HOME"
}

@test "second install is idempotent (no error)" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  HOME="${BATS_TMPDIR}/fakehome-idem-$$"
  mkdir -p "$HOME/.claude"
  export HOME
  "$REPO/install" --core-only
  run "$REPO/install" --core-only
  [ "$status" -eq 0 ]
  rm -rf "$HOME"
}
