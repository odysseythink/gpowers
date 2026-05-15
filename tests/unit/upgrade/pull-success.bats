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

  # Build a fake upstream
  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  # Re-point upstream-sources.json at the bare repo
  jq --arg u "file://$BARE" '.modules.tools.url = $u | .modules.tools.ref = "main"' \
     "$GPOWERS_HOME/upstream-sources.json" > "$GPOWERS_HOME/upstream-sources.json.tmp"
  mv "$GPOWERS_HOME/upstream-sources.json.tmp" "$GPOWERS_HOME/upstream-sources.json"

  # Preserve local-only files, then subtree add upstream content
  mv tools tools.bak
  rm -rf tools
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m "remove tools for subtree" || true

  # Simulate initial install via subtree add
  git subtree add --prefix=tools "file://$BARE" main --squash -q \
    -m "install(tools): add subtree from fixture"

  # Layer local files back on top of subtree
  cp tools.bak/_upgrade-transform.sh tools/
  cp tools.bak/upstream-source.json tools/
  rm -rf tools.bak
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m "add local tools files" || true
}

@test "upgrade-module tools succeeds against fake remote" {
  run bash _gpowers-upgrade-module.sh tools
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
}

@test "after pull, tools/upstream-source.json has new SHA" {
  bash _gpowers-upgrade-module.sh tools >/dev/null
  sha=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/tools/upstream-source.json")
  [ "$sha" != "0000000000000000000000000000000000000000" ]
  [ ${#sha} -eq 40 ]
}

@test "after pull, tools skills have namespace: tools applied" {
  bash _gpowers-upgrade-module.sh tools >/dev/null
  for d in "$GPOWERS_HOME/tools/skills"/*/; do
    grep -q "^namespace: tools$" "$d/SKILL.md"
  done
}
