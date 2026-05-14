#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$PATH"
}

@test "session-start produces non-empty output" {
  run "$GPOWERS_HOME/core/hooks/session-start" claude-code
  [ "$status" -eq 0 ]
  [ -n "$output" ]
}

@test "session-start output names the 4 modules" {
  out=$("$GPOWERS_HOME/core/hooks/session-start" claude-code)
  for mod in core roles tools business; do
    echo "$out" | grep -qw "$mod" || { echo "missing module: $mod"; return 1; }
  done
}

@test "session-start output references gpowers-path helper" {
  out=$("$GPOWERS_HOME/core/hooks/session-start" claude-code)
  echo "$out" | grep -qF "gpowers-path"
}

@test "gpowers-path home returns GPOWERS_HOME" {
  result=$(gpowers-path home)
  [ "$result" = "$GPOWERS_HOME" ]
}
