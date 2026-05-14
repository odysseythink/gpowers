#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  BIN="$GPOWERS_REPO/tools/bin"
  command -v shellcheck >/dev/null || skip "shellcheck not installed"
}

@test "gpowers-ship-helper passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-ship-helper"
}

@test "gpowers-health passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-health"
}

@test "gpowers-update-check passes shellcheck" {
  shellcheck -S warning "$BIN/gpowers-update-check"
}
