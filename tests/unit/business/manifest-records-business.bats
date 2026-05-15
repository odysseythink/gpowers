#!/usr/bin/env bats

setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }

@test "manifest available: business" { [ "$(jq -r '.modules.business.available' < "$M")" = "true" ]; }
@test "manifest opt_in: true" { [ "$(jq -r '.modules.business.opt_in' < "$M")" = "true" ]; }
@test "manifest skill_count: 20" { [ "$(jq -r '.modules.business.skill_count' < "$M")" = "20" ]; }
@test "default installed: false" { [ "$(jq -r '.modules.business.installed // false' < "$M")" = "false" ]; }
