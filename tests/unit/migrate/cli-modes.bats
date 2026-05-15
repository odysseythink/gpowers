#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "--scan-only emits valid JSON" {
  out=$(gpowers-migrate --scan-only)
  echo "$out" | jq empty
}

@test "--plan-only emits mappings" {
  out=$(gpowers-migrate --plan-only)
  echo "$out" | jq -e '.mappings | length > 0' >/dev/null
}

@test "--apply --dry-run does not move files" {
  before=$(find "$HOME/.gstack" 2>/dev/null | sort)
  gpowers-migrate --apply --dry-run --yes >/dev/null
  after=$(find "$HOME/.gstack" 2>/dev/null | sort)
  [ "$before" = "$after" ]
}

@test "empty home prints nothing-to-migrate message" {
  export HOME="$REPO/tests/fixtures/migrate/fake-home-empty"
  run gpowers-migrate
  [ "$status" -eq 0 ]
  echo "$output" | grep -qi "nothing to migrate"
}
