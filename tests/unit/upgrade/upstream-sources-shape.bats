#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../upstream-sources.json"; }

@test "upstream-sources.json is valid JSON" { jq empty < "$F"; }

@test "every module has a remote entry" {
  for m in core roles tools; do
    jq -e ".modules.\"$m\".repo" < "$F" >/dev/null
    jq -e ".modules.\"$m\".ref"  < "$F" >/dev/null
  done
}

@test "core upstream is superpowers" {
  [ "$(jq -r '.modules.core.repo' < "$F")" = "github.com/obra/superpowers" ]
}

@test "roles tools business upstream is gstack" {
  for m in roles tools; do
    [ "$(jq -r ".modules.\"$m\".repo" < "$F")" = "github.com/garrytan/gstack" ]
  done
}
