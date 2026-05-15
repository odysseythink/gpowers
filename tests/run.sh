#!/usr/bin/env bash
# Usage:
#   tests/run.sh                     # run unit + integration (smoke skipped by default)
#   tests/run.sh unit
#   tests/run.sh integration
#   tests/run.sh smoke               # run all platform-smoke (each may skip)
#   tests/run.sh smoke claude-code   # only one platform
#   tests/run.sh all                 # everything including smoke
#   tests/run.sh unit roles          # filter: only tests/unit/roles/
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
SCOPE="${1:-default}"
FILTER="${2:-}"

run_dir() {
  local d="$1"
  [ -d "$d" ] || return 0
  local found
  if [ -n "$FILTER" ]; then
    found=$(find "$d" -path "*$FILTER*" -name '*.bats' 2>/dev/null)
  else
    found=$(find "$d" -name '*.bats' 2>/dev/null)
  fi
  if [ -z "$found" ]; then
    echo "[run] no tests under $d"
    return 0
  fi
  echo "[run] $d"
  echo "$found" | xargs bats
}

case "$SCOPE" in
  unit)         run_dir "$REPO/tests/unit" ;;
  integration)  run_dir "$REPO/tests/integration" ;;
  smoke)        run_dir "$REPO/tests/platform-smoke" ;;
  all)
    run_dir "$REPO/tests/unit"
    run_dir "$REPO/tests/integration"
    run_dir "$REPO/tests/platform-smoke"
    ;;
  default|"")
    run_dir "$REPO/tests/unit"
    run_dir "$REPO/tests/integration"
    ;;
  *) echo "unknown scope: $SCOPE" >&2; exit 2 ;;
esac
