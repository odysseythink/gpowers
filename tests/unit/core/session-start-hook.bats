#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  HOOK="$GPOWERS_REPO/core/hooks/session-start"
  export GPOWERS_HOME="$GPOWERS_REPO"
}

@test "session-start exists and is executable" {
  [ -x "$HOOK" ]
}

@test "session-start emits using-gpowers content" {
  out=$("$HOOK" claude-code)
  echo "$out" | grep -qF "Using gpowers"
  echo "$out" | grep -qF "core/"
  echo "$out" | grep -qF "roles/"
}

@test "session-start wraps output in EXTREMELY_IMPORTANT tag for claude-code" {
  out=$("$HOOK" claude-code)
  echo "$out" | grep -qF "<EXTREMELY_IMPORTANT>"
  echo "$out" | grep -qF "</EXTREMELY_IMPORTANT>"
}

@test "session-start emits raw content for cursor (no tag wrapper)" {
  out=$("$HOOK" cursor)
  ! echo "$out" | grep -qF "<EXTREMELY_IMPORTANT>"
  echo "$out" | grep -qF "Using gpowers"
}

@test "session-start exits non-zero when GPOWERS_HOME missing using-gpowers" {
  GPOWERS_HOME=/nonexistent run "$HOOK" claude-code
  [ "$status" -ne 0 ]
}

@test "session-start unknown platform defaults to raw content" {
  out=$("$HOOK" unknown-platform)
  echo "$out" | grep -qF "Using gpowers"
}
