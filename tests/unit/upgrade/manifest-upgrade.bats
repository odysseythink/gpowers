#!/usr/bin/env bats
setup() { M="$BATS_TEST_DIRNAME/../../../manifest.json"; }
@test "manifest declares upgrade implemented" {
  [ "$(jq -r '.upgrade.implemented' < "$M")" = "true" ]
}
@test "manifest lists --resume subcommand" {
  jq -e '.upgrade.subcommands | index("--resume")' < "$M" >/dev/null
}
