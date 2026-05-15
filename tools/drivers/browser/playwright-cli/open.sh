#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
URL=$(arg .url); [ -n "$URL" ] || die "url required" open
VW=$(arg .viewport.width 1280); VH=$(arg .viewport.height 800)

if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  tab_id=$(tab_alloc playwright-cli); tab_set "$tab_id" mock_url "$URL"
  jq -n --arg id "$tab_id" '{tab_id: $id}'; exit 0
fi

command -v playwright >/dev/null || command -v npx >/dev/null \
  || die "playwright not installed: npm i -g playwright" open

tab_id=$(tab_alloc playwright-cli)
"$DRIVER_DIR/_playwright-runner.sh" "$tab_id" "$URL" "$VW" "$VH"
jq -n --arg id "$tab_id" '{tab_id: $id}'
