#!/usr/bin/env bats

setup() {
  export HOME="${BATS_TMPDIR}/fakehome-$$"
  mkdir -p "$HOME"
  export PATH="${BATS_TMPDIR}/fakepath-$$:$PATH"
  mkdir -p "${BATS_TMPDIR}/fakepath-$$"
}

teardown() {
  rm -rf "$HOME" "${BATS_TMPDIR}/fakepath-$$"
}

DETECT_BIN="${BATS_TEST_DIRNAME}/../../bin/gpowers-detect-platforms"

@test "detects claude-code via ~/.claude directory" {
  mkdir -p "$HOME/.claude"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"claude-code"* ]]
}

@test "detects codex via ~/.codex directory" {
  mkdir -p "$HOME/.codex"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"codex"* ]]
}

@test "detects kimi via ~/.kimi directory" {
  mkdir -p "$HOME/.kimi"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"kimi"* ]]
}

@test "detects gemini via ~/.config/gemini directory" {
  mkdir -p "$HOME/.config/gemini"
  result="$("$DETECT_BIN")"
  [[ "$result" == *"gemini"* ]]
}

@test "no platforms detected when no markers" {
  result="$("$DETECT_BIN")"
  [ -z "$result" ]
}

@test "lib/platform-paths.sh defines lookup table" {
  source "${BATS_TEST_DIRNAME}/../../lib/platform-paths.sh"
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_claude_code:-}" ]
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_codex:-}" ]
  [ -n "${GPOWERS_PLATFORM_PLUGIN_DIR_kimi:-}" ]
}
