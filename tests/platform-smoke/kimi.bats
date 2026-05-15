#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../.."
  source "$REPO/tests/helpers/platform-detect.sh"
  if ! platform_present "kimi"; then
    skip "kimi CLI not installed"
  fi
  HOME_TGT="$BATS_TEST_TMPDIR/gp-kimi"
  REPO="$REPO" "$REPO/tests/helpers/seed-gpowers-home.sh" "$HOME_TGT" >/dev/null
  export GPOWERS_HOME="$HOME_TGT"
  export PATH="$HOME_TGT/bin:$HOME_TGT/tools/bin:$PATH"
}

@test "kimi: gpowers-platforms verify kimi reports OK" {
  out=$(gpowers-platforms verify "kimi")
  echo "$out" | grep -q "OK: kimi"
}

@test "kimi: plugin/extension manifest is valid JSON" {
  manifest=$(jq --arg p "kimi" -r '.platforms[$p].manifest_filename' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  jq empty < "$GPOWERS_HOME/platforms/kimi/$manifest"
}

@test "kimi: at least one command file is present" {
  cmd_dir=$(jq --arg p "kimi" -r '.platforms[$p].command_dir' < "$GPOWERS_HOME/platforms/_platform-shapes.json")
  count=$(find "$GPOWERS_HOME/platforms/kimi/$cmd_dir" -mindepth 1 \( -name '*.md' -o -type d \) | wc -l)
  [ "$count" -gt 0 ]
}


@test "kimi: adapters dir has gpowers-* entries" {
  count=$(find "$GPOWERS_HOME/platforms/kimi/adapters" -mindepth 1 -maxdepth 1 -type d -name "gpowers*" | wc -l)
  [ "$count" -ge 1 ]
}

