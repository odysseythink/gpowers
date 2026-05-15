#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../.."
  source "$REPO/tests/helpers/platform-detect.sh"
  if ! platform_present "opencode"; then
    skip "opencode CLI not installed"
  fi
  HOME_TGT="$BATS_TEST_TMPDIR/gp-opencode"
  REPO="$REPO" "$REPO/tests/helpers/seed-gpowers-home.sh" "$HOME_TGT" >/dev/null
  export GPOWERS_HOME="$HOME_TGT"
  export PATH="$HOME_TGT/bin:$HOME_TGT/tools/bin:$PATH"
}

@test "opencode: gpowers-platforms verify opencode reports OK" {
  out=$(gpowers-platforms verify "opencode")
  echo "$out" | grep -q "OK: opencode"
}

@test "opencode: plugin/extension manifest is valid JSON" {
  manifest=$(jq --arg p "opencode" -r '.platforms[$p].manifest_filename' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  jq empty < "$GPOWERS_HOME/platforms/opencode/$manifest"
}

@test "opencode: at least one command file is present" {
  cmd_dir=$(jq --arg p "opencode" -r '.platforms[$p].command_dir' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  count=$(find "$GPOWERS_HOME/platforms/opencode/$cmd_dir" -mindepth 1 \( -name '*.md' -o -type d \) | wc -l)
  [ "$count" -gt 0 ]
}


