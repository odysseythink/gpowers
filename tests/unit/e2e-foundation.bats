#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-e2e-$$"
  mkdir -p "$HOME/.claude/plugins"
  TESTREPO="${BATS_TMPDIR}/e2e-repo-$$"
  mkdir -p "$TESTREPO/.git"
}

teardown() {
  rm -rf "$HOME" "$TESTREPO"
}

REPO="${BATS_TEST_DIRNAME}/../.."

@test "e2e: install, then run gpowers init in a fake repo" {
  # 1. Install
  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
  [ -d "$HOME/.gpowers/config" ]

  # 2. Run gpowers init in repo
  cd "$TESTREPO"
  run "$HOME/.gpowers/bin/gpowers" init
  [ "$status" -eq 0 ]
  [ -d "$TESTREPO/.gpowers/plans/ceo" ]

  # 3. gpowers path project plans resolves to the repo's .gpowers
  cd "$TESTREPO"
  result="$("$HOME/.gpowers/bin/gpowers" path project plans)"
  [ "$result" = "$TESTREPO/.gpowers/plans" ]
}

@test "e2e: uninstall, then re-install is clean" {
  "$REPO/install" --core-only --platforms=claude-code
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -L "$HOME/.claude/plugins/gpowers" ]
  [ ! -d "$HOME/.gpowers/core" ]

  run "$REPO/install" --core-only --platforms=claude-code
  [ "$status" -eq 0 ]
  [ -L "$HOME/.claude/plugins/gpowers" ]
}
