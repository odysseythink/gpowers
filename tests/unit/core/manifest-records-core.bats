#!/usr/bin/env bats

setup() {
  MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"
}

@test "manifest declares core module installed" {
  installed=$(jq -r '.modules.core.installed' < "$MANIFEST")
  [ "$installed" = "true" ]
}

@test "manifest records 14 core skills" {
  count=$(jq -r '.modules.core.skill_count' < "$MANIFEST")
  [ "$count" = "14" ]
}

@test "manifest references upstream tag" {
  jq -e '.modules.core.upstream | test("superpowers@v5\\.1\\.0")' < "$MANIFEST" >/dev/null
}
