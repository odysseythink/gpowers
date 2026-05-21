#!/usr/bin/env bash
# Ultrawork Stop Hook for kimi-cli
# Reads assistant end-of-turn from wire.jsonl; blocks unverified <promise>DONE</promise>.
# Exit 0 = allow turn to complete.
# Exit 2 + stderr = block and inject reason into next turn.
set -euo pipefail

# Kimi passes session metadata as JSON on stdin. We only need session_id.
read -r stdin_json
session_id=$(echo "$stdin_json" | python3 -c "import sys,json; print(json.load(sys.stdin).get('session_id',''))" 2>/dev/null || echo "")

if [ -z "$session_id" ]; then
  # Cannot identify session — fail-open per Kimi design
  exit 0
fi

# Derive session directory from Kimi's share dir convention
SESSION_DIR="${KIMI_SESSION_DIR:-$HOME/.kimi/sessions/$session_id}"
FLOW_FLAG="$SESSION_DIR/.ultrawork-flow-active"
ITER_FILE="$SESSION_DIR/.ultrawork-iter"
HOOK_LOG="$SESSION_DIR/.ultrawork-hook.log"
WIRE_LOG="$SESSION_DIR/wire.jsonl"

heartbeat() {
  echo "[$(date -Iseconds)] $1" >> "$HOOK_LOG" 2>/dev/null || true
}

heartbeat "stop-hook invoked"

# Dormancy check — hook only acts during active flow runs
if [ ! -f "$FLOW_FLAG" ]; then
  heartbeat "flow inactive → exit 0"
  exit 0
fi

# Guard: wire log must exist
if [ ! -f "$WIRE_LOG" ]; then
  heartbeat "wire.jsonl missing → exit 0"
  exit 0
fi

last_block=$(tac "$WIRE_LOG" 2>/dev/null | awk '
  BEGIN { capture=0 }
  /"event":"TurnBegin"/ { if(capture) exit; capture=1; next }
  /"event":"TurnEnd"/   { next }
  capture { print }
' | tac)

if [ -z "$last_block" ]; then
  heartbeat "no block extracted → exit 0"
  exit 0
fi

PROMISE_RE='(?i)<promise>\s*([^<]+?)\s*</promise>'

promises=$(echo "$last_block" | grep -oP "$PROMISE_RE" 2>/dev/null || true)

if [ -z "$promises" ]; then
  heartbeat "no promise tag → exit 0"
  exit 0
fi

has_verified=$(echo "$last_block" | grep -cP '(?i)Agent:\s*Oracle.*<promise>\s*VERIFIED\s*</promise>' 2>/dev/null || echo "0")

not_verified=$(echo "$last_block" | grep -oP '(?i)Agent:\s*Oracle.*<promise>\s*NOT-VERIFIED:\s*([^<]+?)\s*</promise>' 2>/dev/null || true)

iter=0
if [ -f "$ITER_FILE" ]; then
  iter=$(cat "$ITER_FILE" 2>/dev/null | tr -d '[:space:]' || echo "0")
  case "$iter" in
    ''|*[!0-9]*) iter=0; heartbeat "iter file corrupted, resetting to 0" ;;
  esac
fi

if [ -n "$not_verified" ]; then
  iter=$((iter + 1))
  echo "$iter" > "$ITER_FILE" 2>/dev/null || true
  reason=$(echo "$not_verified" | grep -oP '(?i)<promise>\s*NOT-VERIFIED:\s*\K([^<]+?)(?=\s*</promise>)' || echo "unknown reason")
  heartbeat "NOT-VERIFIED at iter=$iter: $reason → exit 2"
  echo "Ultrawork: Oracle rejected — $reason" >&2
  exit 2
fi

done_present=$(echo "$promises" | grep -ic 'DONE' || echo "0")

if [ "$done_present" -gt 0 ] && [ "$has_verified" -eq 0 ]; then
  iter=$((iter + 1))
  echo "$iter" > "$ITER_FILE" 2>/dev/null || true
  if [ "$iter" -ge 100 ]; then
    summary_file="$SESSION_DIR/.ultrawork-iter-summary"
    echo "Ultrawork: iteration cap reached ($iter). Aborting loop. See $summary_file" > "$summary_file"
    heartbeat "CAP REACHED at iter=$iter → exit 2"
    echo "Ultrawork: iteration cap reached ($iter). Aborting loop. See $summary_file" >&2
    exit 2
  fi
  heartbeat "DONE without VERIFIED at iter=$iter → exit 2"
  echo "Ultrawork: emit Oracle verdict before stopping. Dispatch Agent(subagent_type='oracle', prompt=<task + recent diff>)." >&2
  exit 2
fi

if [ "$has_verified" -gt 0 ]; then
  heartbeat "VERIFIED found → exit 0 (loop ends)"
  rm -f "$FLOW_FLAG" 2>/dev/null || true
  exit 0
fi

heartbeat "fallback → exit 0"
exit 0
