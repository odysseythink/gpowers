#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; T="$REPO/platforms/gemini"; }

@test "gemini uses extension.json (not plugin.json)" {
  [ -f "$T/extension.json" ]
  [ ! -f "$T/plugin.json" ]
}

@test "gemini has no hooks.json (hooks are via-injection)" {
  [ ! -f "$T/hooks.json" ]
}
