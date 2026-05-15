#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest declares migrate implemented" {
  [ "$(jq -r '.migrate.implemented' < "$M")" = "true" ]
}
@test "manifest records /review alias and deprecation date" {
  [ "$(jq -r '.migrate.slash_aliases.review' < "$M")" = "pr-review" ]
  [ "$(jq -r '.migrate.deprecation_until' < "$M")" = "2026-11-14" ]
}
