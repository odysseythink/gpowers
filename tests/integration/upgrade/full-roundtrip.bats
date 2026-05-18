#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/up"
  mkdir -p "$TMP/gp"
  tar -C "$REPO" -cf - . 2>/dev/null | tar -C "$TMP/gp" -xf -
  cd "$TMP/gp"
  rm -rf .git
  git init -q
  git config user.email "t@t"
  git config user.name "t"
  git add -A
  git commit -q -m initial
  export GPOWERS_HOME="$TMP/gp"
  export PATH="$GPOWERS_HOME/bin:$PATH"

  BARE=$("$REPO/tests/fixtures/upgrade/seed-repo.sh" "$TMP" gstack-tools)
  jq --arg u "file://$BARE" '.modules.tools.url = $u | .modules.tools.ref = "main"' \
     upstream-sources.json > upstream-sources.json.tmp && mv upstream-sources.json.tmp upstream-sources.json
  git -c user.email=t@t -c user.name=t add -A
  git -c user.email=t@t -c user.name=t commit -q -m "point tools at fixture"

  # Set up subtree tracking
  mv tools tools.bak
  rm -rf tools
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m "remove tools for subtree" || true
  git subtree add --prefix=tools "file://$BARE" main --squash -q -m "add tools subtree"
  cp tools.bak/_upgrade-transform.sh tools/
  cp tools.bak/upstream-source.json tools/
  rm -rf tools.bak
  git add -A
  git -c user.email=t@t -c user.name=t commit -q -m "add local tools files" || true
}

@test "check reports new SHA for tools" {
  out=$(gpowers-upgrade --check tools)
  echo "$out" | grep -qE 'tools[[:space:]]+[0-9a-f]{40}'
}

@test "full upgrade round-trip succeeds" {
  before=$(jq -r '.upstream.sha' < tools/upstream-source.json)
  run gpowers-upgrade tools
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  after=$(jq -r '.upstream.sha' < tools/upstream-source.json)
  [ "$before" != "$after" ]
}

@test "after upgrade, manifests regenerated and skills still pass frontmatter test" {
  gpowers-upgrade tools >/dev/null
  # Skills should still satisfy frontmatter requirements
  for d in tools/skills/*/; do
    grep -q "^namespace: tools$" "$d/SKILL.md"
  done
  # Platform manifests still valid
  gpowers-platforms verify all >/dev/null
}
