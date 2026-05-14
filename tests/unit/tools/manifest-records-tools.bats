#!/usr/bin/env bats

setup() { MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest declares tools module installed" {
  [ "$(jq -r '.modules.tools.installed' < "$MANIFEST")" = "true" ]
}

@test "manifest records 17 non-browser tool skills" {
  [ "$(jq -r '.modules.tools.skill_count_non_browser' < "$MANIFEST")" = "17" ]
}
