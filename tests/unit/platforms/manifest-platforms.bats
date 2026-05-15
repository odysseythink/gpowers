#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest lists 7 supported platforms" {
  [ "$(jq -r '.platforms.supported | length' < "$M")" = "7" ]
}

@test "manifest names kimi explicitly" {
  jq -e '.platforms.supported | index("kimi")' < "$M" >/dev/null
}
