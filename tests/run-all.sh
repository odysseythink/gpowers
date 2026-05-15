#!/usr/bin/env bash
# tests/run-all.sh — Run all bats tests.
set -euo pipefail

SCRIPT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v bats >/dev/null; then
  printf 'error: bats-core not installed. Install with: brew install bats-core (or npm i -g bats)\n' >&2
  exit 1
fi
if ! command -v jq >/dev/null; then
  printf 'error: jq not installed. Install with: brew install jq\n' >&2
  exit 1
fi

bats "$SCRIPT_DIR/unit"
