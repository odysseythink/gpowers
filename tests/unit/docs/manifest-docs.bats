#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest lists 10 docs entries" {
  [ "$(jq -r '.docs.entries | length' < "$M")" = "10" ]
}

@test "manifest declares root README" {
  [ "$(jq -r '.docs.root_readme' < "$M")" = "true" ]
}
