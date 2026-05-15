#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/srepo"
  "$REPO/tests/fixtures/build-sample-repo.sh" "$TMP"
  cd "$TMP"
}

@test "sample-repo has 4 commits on main" {
  count=$(git log --oneline | wc -l)
  [ "$count" -ge 4 ]
}

@test "sample-repo has feature/sample branch" {
  git rev-parse feature/sample >/dev/null
}

@test "sample-repo has package.json + README + src/" {
  [ -f package.json ]
  [ -f README.md ]
  [ -d src ]
}
