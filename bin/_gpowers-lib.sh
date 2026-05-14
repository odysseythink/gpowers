#!/usr/bin/env bash
# bin/_gpowers-lib.sh — Shared utilities for gpowers CLIs.

gpowers_die() {
  printf 'gpowers: %s\n' "$1" >&2
  exit "${2:-1}"
}

# Resolve the directory containing the calling script, following symlinks.
gpowers_script_dir() {
  local src="$1"
  while [ -L "$src" ]; do
    local dir
    dir="$(cd -P "$(dirname "$src")" && pwd)"
    src="$(readlink "$src")"
    case "$src" in
      /*) ;;
      *) src="$dir/$src" ;;
    esac
  done
  cd -P "$(dirname "$src")" && pwd
}

# Join path components with single slashes, stripping trailing slashes.
gpowers_path_join() {
  local out="$1"
  shift
  for part in "$@"; do
    out="${out%/}/${part#/}"
  done
  printf '%s\n' "$out"
}
