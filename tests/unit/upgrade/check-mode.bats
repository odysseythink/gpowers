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
  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  jq --arg u "file://$BARE" '.modules.tools.url = $u' \
     "$GPOWERS_HOME/upstream-sources.json" > "$GPOWERS_HOME/upstream-sources.json.tmp"
  mv "$GPOWERS_HOME/upstream-sources.json.tmp" "$GPOWERS_HOME/upstream-sources.json"
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m "point tools at fixture" || true
}

@test "check reports a remote SHA for tools" {
  out=$(bash _gpowers-upgrade-check.sh tools)
  echo "$out" | grep -qE 'tools[[:space:]]+[0-9a-f]{40}'
}

@test "check does NOT modify upstream-source.json" {
  before=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  bash _gpowers-upgrade-check.sh tools >/dev/null
  after=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  [ "$before" = "$after" ]
}

@test "check reports 'new version available' when SHAs differ" {
  out=$(bash _gpowers-upgrade-check.sh tools)
  echo "$out" | grep -qi "new\|update available"
}
