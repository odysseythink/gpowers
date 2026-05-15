#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports gstack present" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.present == true' >/dev/null
}

@test "scan counts gstack projects" {
  out=$("$SCAN")
  [ "$(echo "$out" | jq -r '.gstack.projects | length')" -ge 1 ]
}

@test "scan finds builder-profile + developer-profile" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.profiles | index("builder-profile")' >/dev/null
  echo "$out" | jq -e '.gstack.profiles | index("developer-profile")' >/dev/null
}

@test "scan reports superpowers absent in gstack-only fixture" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.present == false' >/dev/null
}
