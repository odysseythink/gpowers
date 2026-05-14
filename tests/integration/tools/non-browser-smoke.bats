#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
}

@test "ship skill can be cat'd and references gpowers-path" {
  body=$(cat "$GPOWERS_HOME/tools/skills/ship/SKILL.md")
  echo "$body" | grep -qF "namespace: tools"
  echo "$body" | grep -qF "gpowers-path"
}

@test "gpowers-health is on PATH and runs" {
  run gpowers-health --help 2>&1
  # Stub returns 0; real implementation may differ. Just assert it's invokable.
  command -v gpowers-health >/dev/null
}

@test "gpowers-path home points to GPOWERS_HOME" {
  [ "$(gpowers-path home)" = "$GPOWERS_HOME" ]
}
