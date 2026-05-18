#!/usr/bin/env bats

setup() { US="$BATS_TEST_DIRNAME/../../../tools/upstream-source.json"; }

@test "upstream-source lists 11 browser-dependent skills" {
  count=$(jq -r '.submodules.browser_dependent | length' < "$US")
  [ "$count" = "9" ]
}

@test "upstream-source has no pending sentinel" {
  ! jq -e '.submodules.browser_dependent | index("__pending_plan_5__")' < "$US" >/dev/null
}
