#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

@test "gemini fragment has begin/end markers" {
  F="$REPO/platforms/gemini/GEMINI.md.fragment"
  [ -f "$F" ]
  grep -q "gpowers:begin" "$F"
  grep -q "gpowers:end" "$F"
}

@test "cursor fragment has begin/end markers" {
  F="$REPO/platforms/cursor/cursor-rules.md.fragment"
  [ -f "$F" ]
  grep -q "gpowers:begin" "$F"
  grep -q "gpowers:end" "$F"
}
