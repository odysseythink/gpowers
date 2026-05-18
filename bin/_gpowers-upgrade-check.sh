#!/usr/bin/env bash
# Usage: _gpowers-upgrade-check.sh [<module>]
# Compares <module>/upstream-source.json sha against remote ls-remote.
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"
SOURCES="$GPOWERS_HOME/upstream-sources.json"

check_one() {
  local module="$1"
  local url ref local_sha remote_sha
  url=$(jq -r --arg m "$module" '.modules[$m].url' < "$SOURCES")
  ref=$(jq -r --arg m "$module" '.modules[$m].ref' < "$SOURCES")
  local_sha=$(jq -r '.upstream.sha' < "$GPOWERS_HOME/$module/upstream-source.json")
  remote_sha=$(git ls-remote "$url" "$ref" 2>/dev/null | awk '{print $1}' | head -1) || true

  if [ -z "$remote_sha" ]; then
    printf '%-10s %-42s %s\n' "$module" "(unreachable)" "$url"
    return
  fi

  local status="up-to-date"
  if [ "$local_sha" != "$remote_sha" ]; then status="new version available"; fi
  printf '%-10s %s %s\n' "$module" "$remote_sha" "$status"
}

if [ $# -ge 1 ]; then
  check_one "$1"
else
  for m in core roles tools; do check_one "$m"; done
fi
