#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "e2e: scan → plan → apply (yes) moves all expected items" {
  gpowers-migrate --apply --yes >/dev/null
  [ -f "$GPOWERS_HOME/config/builder-profile" ]
  [ -f "$GPOWERS_HOME/config/developer-profile" ]
  [ -f "$GPOWERS_HOME/state/installation-id" ]
  [ -d "$GPOWERS_HOME/state/security" ]
  [ -d "$GPOWERS_HOME/data/legacy-projects/proj-alpha" ] || \
    [ -d "$GPOWERS_HOME/data/legacy-projects/proj-alpha/ceo-plans" ]
}

@test "e2e: source tree is empty (or only contains residual empties) after migrate" {
  gpowers-migrate --apply --yes >/dev/null
  # builder-profile / installation-id should be gone from source
  [ ! -f "$HOME/.gstack/builder-profile" ]
  [ ! -f "$HOME/.gstack/installation-id" ]
}
