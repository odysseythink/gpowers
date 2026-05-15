#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  cp -R "$REPO/tests/fixtures/migrate/fake-home-gstack/." "$TMP/"
  export HOME="$TMP"
  export GPOWERS_HOME="$TMP/.gpowers"
  mkdir -p "$GPOWERS_HOME/state"
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
  APPLY="$REPO/bin/_gpowers-migrate-apply.sh"
}

@test "apply --dry-run does not modify source" {
  before=$(find "$HOME/.gstack" | sort)
  "$PLAN" | bash "$APPLY" --dry-run >/dev/null
  after=$(find "$HOME/.gstack" | sort)
  [ "$before" = "$after" ]
}

@test "apply --dry-run does not create destination" {
  "$PLAN" | bash "$APPLY" --dry-run >/dev/null
  ! [ -d "$GPOWERS_HOME/config" ]
}

@test "apply (no --dry-run, --yes) moves builder-profile" {
  "$PLAN" | bash "$APPLY" --yes >/dev/null
  [ -f "$GPOWERS_HOME/config/builder-profile" ]
  [ ! -e "$HOME/.gstack/builder-profile" ]
}

@test "apply writes a journal" {
  "$PLAN" | bash "$APPLY" --yes >/dev/null
  [ -s "$GPOWERS_HOME/state/migrate-journal.jsonl" ]
}
