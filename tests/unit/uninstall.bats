#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-un-$$"
  mkdir -p "$HOME/.claude" "$HOME/.kimi"
}

teardown() {
  rm -rf "$HOME"
}

@test "uninstall removes platform symlinks" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  [ -L "$HOME/.claude/plugins/gpowers" ]
  run "$REPO/uninstall" --dry-run
  [ "$status" -eq 0 ]
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -L "$HOME/.claude/plugins/gpowers" ]
}

@test "uninstall keeps user data by default" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "user content" > "$HOME/.gpowers/data/important.txt"
  echo "user config" > "$HOME/.gpowers/config/builder-profile"
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ -f "$HOME/.gpowers/data/important.txt" ]
  [ -f "$HOME/.gpowers/config/builder-profile" ]
}

@test "uninstall removes module dirs and state/cache/tmp" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "cache junk" > "$HOME/.gpowers/cache/junk.txt"
  echo "state junk" > "$HOME/.gpowers/state/junk.txt"
  run "$REPO/uninstall"
  [ "$status" -eq 0 ]
  [ ! -d "$HOME/.gpowers/core" ]
  [ ! -d "$HOME/.gpowers/cache" ]
  [ ! -d "$HOME/.gpowers/state" ]
  [ ! -d "$HOME/.gpowers/tmp" ]
}

@test "uninstall --remove-global-data also removes data dirs" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  echo "user content" > "$HOME/.gpowers/data/important.txt"
  run "$REPO/uninstall" --remove-global-data
  [ "$status" -eq 0 ]
  [ ! -d "$HOME/.gpowers/data" ]
  [ ! -d "$HOME/.gpowers/config" ]
}

@test "uninstall --dry-run prints actions and changes nothing" {
  cd "${BATS_TEST_DIRNAME}/../.."
  REPO="$(pwd)"
  "$REPO/install" --core-only --platforms=claude-code
  run "$REPO/uninstall" --dry-run
  [ "$status" -eq 0 ]
  [[ "$output" == *"would remove"* ]]
  [ -L "$HOME/.claude/plugins/gpowers" ]
}
