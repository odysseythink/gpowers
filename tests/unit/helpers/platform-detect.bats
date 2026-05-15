#!/usr/bin/env bats

setup() { source "$BATS_TEST_DIRNAME/../../helpers/platform-detect.sh"; }

@test "platform_present returns 0 or 1 cleanly" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    run platform_present "$p"
    case "$status" in 0|1) ;; *) echo "unexpected status $status for $p"; return 1;; esac
  done
}

@test "platform_present errors for unknown name" {
  run platform_present "bogus"
  [ "$status" -ne 0 ]
}

@test "platforms_present emits a subset of known platforms" {
  out=$(platforms_present)
  for p in $out; do
    case "$p" in claude-code|codex|gemini|cursor|opencode|copilot|kimi) ;;
                 *) echo "unknown: $p"; return 1;; esac
  done
}
