#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$TMP"
}

@test "install --dry-run mentions gpowers-platforms gen" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --non-interactive)
  echo "$out" | grep -qi "gpowers-platforms gen\|platforms gen\|generate platform"
}

@test "install --dry-run --platforms=claude-code,kimi only mentions those two" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --non-interactive --platforms=claude-code,kimi)
  echo "$out" | grep -q "claude-code"
  echo "$out" | grep -q "kimi"
  ! echo "$out" | grep -q "gemini\|cursor\|opencode\|copilot\|codex"
}
