#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  export GPOWERS_HOME="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$GPOWERS_HOME/config"
  echo "pre-existing" > "$GPOWERS_HOME/config/builder-profile"   # collision
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
}

@test "plan reports a conflict for pre-existing dst" {
  out=$("$PLAN")
  echo "$out" | jq -e '.conflicts | length > 0' >/dev/null
  echo "$out" | jq -e '.conflicts[] | select(.dst | endswith("builder-profile"))' >/dev/null
}

@test "non-conflicting dst is not in conflicts list" {
  out=$("$PLAN")
  echo "$out" | jq -e 'all(.conflicts[]; .dst | contains("builder-profile"))' >/dev/null
}
