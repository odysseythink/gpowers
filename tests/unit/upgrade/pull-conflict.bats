#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  mkdir -p "$TMP/gp"
  tar -C "$REPO" -cf - . 2>/dev/null | tar -C "$TMP/gp" -xf -
  cd "$TMP/gp"
  rm -rf .git
  git init -q
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m initial
  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"
}

@test "resume exits 0 with no in-progress merge (no-op)" {
  out=$(bash _gpowers-upgrade-resume.sh 2>&1)
  echo "$out" | grep -qi "no upgrade in progress\|nothing to resume"
}

@test "resume requires a clean working tree (after manual fix)" {
  # Simulate: pretend MERGE_HEAD exists but tree dirty
  echo conflict > /tmp/gp_conflict
  touch "$GPOWERS_HOME/.git/MERGE_HEAD" 2>/dev/null || true
  if [ -f "$GPOWERS_HOME/.git/MERGE_HEAD" ]; then
    cp /tmp/gp_conflict "$GPOWERS_HOME/dirty.txt"
    git -C "$GPOWERS_HOME" add dirty.txt
    run bash _gpowers-upgrade-resume.sh
    [ "$status" -ne 0 ]
    echo "$output" | grep -qi "still have unresolved\|working tree not clean\|conflicts remain"
  else
    skip "platform doesn't allow direct MERGE_HEAD touch"
  fi
}
