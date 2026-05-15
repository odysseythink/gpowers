#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
TAB_ID=$(arg .tab_id)
[ -n "$TAB_ID" ] || die "tab_id required" wait

if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  jq -n '{ok:true}'
  exit 0
fi

MCP_TAB=$(tab_get "$TAB_ID" mcp_tab_id) || die "tab not initialized" wait "$TAB_ID"
echo "GPOWERS_MCP_INSTRUCTION: invoke mcp__claude-in-chrome__find for $MCP_TAB with args from $ARGS_JSON" >&2
jq -n '{ok:true}'
