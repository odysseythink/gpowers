#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
TAB_ID=$(arg .tab_id)
[ -n "$TAB_ID" ] || die "tab_id required" close

if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  tab_release "$TAB_ID"
  jq -n '{ok:true}'
  exit 0
fi

MCP_TAB=$(tab_get "$TAB_ID" mcp_tab_id) || die "tab not initialized" close "$TAB_ID"
echo "GPOWERS_MCP_INSTRUCTION: invoke mcp__claude-in-chrome__tabs_close_mcp for $MCP_TAB" >&2
tab_release "$TAB_ID"
jq -n '{ok:true}'
