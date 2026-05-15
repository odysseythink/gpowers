#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../.."
  source "$REPO/tests/helpers/platform-detect.sh"
  if ! platform_present "claude-code"; then
    skip "claude-code CLI not installed"
  fi
  HOME_TGT="$BATS_TEST_TMPDIR/gp-claude-code"
  REPO="$REPO" "$REPO/tests/helpers/seed-gpowers-home.sh" "$HOME_TGT" >/dev/null
  export GPOWERS_HOME="$HOME_TGT"
  export PATH="$HOME_TGT/bin:$HOME_TGT/tools/bin:$PATH"
}

@test "claude-code: gpowers-platforms verify claude-code reports OK" {
  out=$(gpowers-platforms verify "claude-code")
  echo "$out" | grep -q "OK: claude-code"
}

@test "claude-code: plugin/extension manifest is valid JSON" {
  manifest=$(jq --arg p "claude-code" -r '.platforms[$p].manifest_filename' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  jq empty < "$GPOWERS_HOME/platforms/claude-code/$manifest"
}

@test "claude-code: at least one command file is present" {
  cmd_dir=$(jq --arg p "claude-code" -r '.platforms[$p].command_dir' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  count=$(find "$GPOWERS_HOME/platforms/claude-code/$cmd_dir" -mindepth 1 \( -name '*.md' -o -type d \) | wc -l)
  [ "$count" -gt 0 ]
}


