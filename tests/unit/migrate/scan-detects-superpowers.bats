#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-superpowers"
  SCAN="$REPO/bin/_gpowers-migrate-scan.sh"
}

@test "scan reports superpowers present" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.present == true' >/dev/null
}

@test "scan enumerates superpowers worktrees" {
  out=$("$SCAN")
  echo "$out" | jq -e '.superpowers.worktrees | index("myrepo/feature-branch")' >/dev/null
}
