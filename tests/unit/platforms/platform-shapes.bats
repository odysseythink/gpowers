#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../platforms/_platform-shapes.json"; }

@test "platform shapes JSON is valid" { jq empty < "$F"; }

@test "all 7 platforms declared" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    jq -e ".platforms.\"$p\"" < "$F" >/dev/null || { echo "missing: $p"; return 1; }
  done
}

@test "each platform has manifest_filename + command_dir" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    jq -e ".platforms.\"$p\".manifest_filename" < "$F" >/dev/null
    jq -e ".platforms.\"$p\".command_dir" < "$F" >/dev/null
  done
}

@test "kimi uses flat-prefix namespace_mode" {
  [ "$(jq -r '.platforms.kimi.namespace_mode' < "$F")" = "flat-prefix" ]
}
