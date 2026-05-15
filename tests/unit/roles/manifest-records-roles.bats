#!/usr/bin/env bats

setup() { MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest declares roles module installed" {
  [ "$(jq -r '.modules.roles.installed' < "$MANIFEST")" = "true" ]
}

@test "manifest records 19 non-browser role skills" {
  [ "$(jq -r '.modules.roles.skill_count_non_browser' < "$MANIFEST")" = "19" ]
}

@test "manifest records 1 browser-dependent role skill" {
  [ "$(jq -r '.modules.roles.skill_count_browser' < "$MANIFEST")" = "1" ]
}

@test "manifest records 20 total role skills" {
  [ "$(jq -r '.modules.roles.skill_count_total' < "$MANIFEST")" = "20" ]
}

@test "manifest records roles upstream as gstack@main" {
  [ "$(jq -r '.modules.roles.upstream' < "$MANIFEST")" = "gstack@main" ]
}
