#!/usr/bin/env bash
# Spawns a Node child running playwright in headless mode, exchanging JSON
# requests/responses over a pair of FIFOs in $tab_dir.
set -euo pipefail

DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

TAB_ID="${1:?tab_id required}"
URL="${2:?initial URL required}"
VW="${3:-1280}"
VH="${4:-800}"

TAB_DIR="$(gpowers-path state)/browser/tabs/$TAB_ID"
mkfifo "$TAB_DIR/req" "$TAB_DIR/res"

NODE_SCRIPT="$DRIVER_DIR/runner.mjs"

setsid node "$NODE_SCRIPT" "$TAB_DIR/req" "$TAB_DIR/res" "$URL" "$VW" "$VH" \
  </dev/null >"$TAB_DIR/stdout" 2>"$TAB_DIR/stderr" &
echo $! > "$TAB_DIR/pid"
