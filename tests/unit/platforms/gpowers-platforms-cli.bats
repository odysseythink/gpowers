#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "list prints all 7 platforms" {
  out=$(gpowers-platforms list)
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    echo "$out" | grep -q "$p" || { echo "missing $p"; return 1; }
  done
}

@test "verify all passes after a successful gen all" {
  gpowers-platforms gen all >/dev/null
  gpowers-platforms verify all
}

@test "unknown subcommand exits 2" {
  run gpowers-platforms doinator
  [ "$status" -eq 2 ]
}
