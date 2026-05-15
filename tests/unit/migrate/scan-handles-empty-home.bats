#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-empty"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports neither installed in empty home" {
  out=$("$SCAN")
  echo "$out" | jq -e '.gstack.present == false' >/dev/null
  echo "$out" | jq -e '.superpowers.present == false' >/dev/null
}

@test "scan exits 0 even on empty home" {
  run "$SCAN"
  [ "$status" -eq 0 ]
}
