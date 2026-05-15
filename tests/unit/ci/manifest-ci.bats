#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest records test runner path" {
  [ "$(jq -r '.tests.runner' < "$M")" = "tests/run.sh" ]
}
@test "manifest lists 3 test layers" {
  [ "$(jq -r '.tests.layers | length' < "$M")" = "3" ]
}
@test "manifest declares github-actions CI" {
  jq -e '.ci.platforms | index("github-actions")' < "$M" >/dev/null
}
@test "manifest lists both workflows" {
  jq -e '.ci.workflows | index("ci.yml") and index("release.yml")' < "$M" >/dev/null
}
