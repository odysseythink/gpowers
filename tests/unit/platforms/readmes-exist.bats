#!/usr/bin/env bats
setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }
@test "each platform has README.md naming itself" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    F="$REPO/platforms/$p/README.md"
    [ -f "$F" ] || { echo "missing: $F"; return 1; }
    grep -q "$p" "$F" || { echo "$F doesn't mention platform name"; return 1; }
  done
}
