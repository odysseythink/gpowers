#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  mkdir -p "$GPOWERS_HOME"
  # Lock down a sub-tree of $GPOWERS_HOME after a few moves succeed
  chmod 555 "$GPOWERS_HOME"
}

teardown() {
  chmod 755 "$GPOWERS_HOME" 2>/dev/null || true
}

@test "rollback restores source files when destination write fails" {
  set +e
  "$REPO/bin/_gpowers-migrate-plan.sh" \
    | "$REPO/bin/_gpowers-migrate-apply.sh" --yes >/dev/null 2>&1
  rc=$?
  set -e
  [ "$rc" -ne 0 ]
  # builder-profile should still exist in source after rollback
  [ -f "$HOME/.gstack/builder-profile" ]
}
