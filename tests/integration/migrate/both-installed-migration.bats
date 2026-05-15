#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-both/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  export PATH="$REPO/bin:$PATH"
}

@test "both: scan detects gstack + superpowers" {
  out=$(gpowers-migrate --scan-only)
  echo "$out" | jq -e '.gstack.present and .superpowers.present' >/dev/null
}

@test "both: apply migrates worktrees to ~/.gpowers/state/worktrees/" {
  gpowers-migrate --apply --yes >/dev/null
  [ -e "$GPOWERS_HOME/state/worktrees/myrepo/feature-branch" ]
}

@test "both: gstack and superpowers source dirs are emptied" {
  gpowers-migrate --apply --yes >/dev/null
  ! [ -d "$HOME/.config/superpowers/worktrees/myrepo/feature-branch" ]
}
