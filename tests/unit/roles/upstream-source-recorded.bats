#!/usr/bin/env bats

setup() { US="$BATS_TEST_DIRNAME/../../../roles/upstream-source.json"; }

@test "upstream-source lists 1 browser-dependent skill" {
  count=$(jq -r '.submodules.browser_dependent | length' < "$US")
  [ "$count" = "1" ]
}

@test "upstream-source has no pending sentinel" {
  ! jq -e '.submodules.browser_dependent | index("__pending_plan_5__")' < "$US" >/dev/null
}
