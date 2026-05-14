#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
URL=$(arg .url)
VW=$(arg .viewport.width 1280)
VH=$(arg .viewport.height 800)

[ -n "$URL" ] || die "url required" open

# In agent context, the actual MCP call is performed by the LLM after
# reading this driver's README.md. For automated CI / unit test we shim
# via $GPOWERS_BROWSER_MOCK if set.
if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  tab_id=$(tab_alloc claude-in-chrome)
  tab_set "$tab_id" mock_url "$URL"
  jq -n --arg id "$tab_id" '{tab_id: $id}'
  exit 0
fi

# Production path: emit MCP invocation instruction.
tab_id=$(tab_alloc claude-in-chrome)
tab_set "$tab_id" pending_open "$URL"
cat >&2 <<MCP
GPOWERS_MCP_INSTRUCTION: invoke mcp__claude-in-chrome__tabs_create_mcp with {"url":"$URL"}, then mcp__claude-in-chrome__resize_window {"width":$VW,"height":$VH}. Bind the returned tab id to gpowers tab_id "$tab_id" via: tab_set "$tab_id" mcp_tab_id <returned>.
MCP
jq -n --arg id "$tab_id" '{tab_id: $id}'
