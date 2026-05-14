#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
TAB_ID=$(arg .tab_id); [ -n "$TAB_ID" ] || die "tab_id required" wait

if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  jq -n '{ok:true}'
  exit 0
fi

TAB_DIR="$(gpowers-path state)/browser/tabs/$TAB_ID"
[ -p "$TAB_DIR/req" ] || die "tab not opened" wait "$TAB_ID"

REQ=$(echo "$ARGS_JSON" | jq --arg v wait '.verb = $v')
printf '%s\n' "$REQ" > "$TAB_DIR/req" &
RES=$(head -n1 "$TAB_DIR/res")
printf '%s\n' "$RES"
