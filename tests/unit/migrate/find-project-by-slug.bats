#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TMP="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TMP/.gstack/projects/explicit-slug"
  mkdir -p "$TMP/.gstack/projects/findable-slug"
  mkdir -p "$TMP/.gstack/projects/missing-slug"

  # explicit: .repo-path file
  mkdir -p "$TMP/repos/explicit-repo"
  ( cd "$TMP/repos/explicit-repo" && git init -q )
  echo "$TMP/repos/explicit-repo" > "$TMP/.gstack/projects/explicit-slug/.repo-path"

  # findable: a directory matching slug name
  mkdir -p "$TMP/repos/findable-slug"
  ( cd "$TMP/repos/findable-slug" && git init -q )

  export HOME="$TMP"
  HELPER="$REPO/bin/_gpowers-find-project-by-slug.sh"
}

@test "explicit slug resolves via .repo-path" {
  out=$("$HELPER" explicit-slug)
  [ "$out" = "$HOME/repos/explicit-repo" ]
}

@test "findable slug resolves via find" {
  out=$("$HELPER" findable-slug)
  case "$out" in *"findable-slug"*) :;; *) echo "got: $out"; return 1;; esac
}

@test "missing slug returns empty (caller handles fallback)" {
  out=$("$HELPER" missing-slug)
  [ -z "$out" ]
}
