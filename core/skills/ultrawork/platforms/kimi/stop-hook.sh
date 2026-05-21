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

# Reverse-read wire.jsonl to isolate the most recent assistant block.
# After tac, the last TurnEnd is first; we capture everything after it
# until the next TurnBegin (which is the start of the most recent turn).
last_block=$(tac "$WIRE_LOG" 2>/dev/null | awk '
  BEGIN { after_end=0 }
  /"event":"TurnEnd"/   { after_end=1; next }
  /"event":"TurnBegin"/ { exit }
  after_end { print }
' | tac)

if [ -z "$last_block" ]; then
  heartbeat "no block extracted → exit 0"
  exit 0
fi

# Extract plain text content from the JSONL block (concatenate all MessageDelta contents)
last_block_text=$(echo "$last_block" | python3 -c "
import sys, json
contents = []
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        obj = json.loads(line)
        if obj.get('event') == 'MessageDelta' and isinstance(obj.get('content'), str):
            contents.append(obj['content'])
    except (json.JSONDecodeError, UnicodeDecodeError):
        continue
print(''.join(contents))
" 2>/dev/null || echo "")

if [ -z "$last_block_text" ]; then
  heartbeat "no text extracted from block → exit 0"
  exit 0
fi

# Use python3 for all regex matching (BSD grep lacks -P; python3 is already a kimi-cli dependency)
_parse_result=$(echo "$last_block_text" | python3 -c "
import sys, re

text = sys.stdin.read()
# Case-insensitive matching
text_lower = text.lower()

# Find all promise tags
promise_re = re.compile(r'<promise>\s*([^<]+?)\s*</promise>', re.IGNORECASE)
promises = promise_re.findall(text)

# Check for Oracle VERIFIED (Agent: Oracle anywhere before VERIFIED tag)
has_verified = 0
if re.search(r'Agent:\s*Oracle.*<promise>\s*VERIFIED\s*</promise>', text, re.IGNORECASE | re.DOTALL):
    has_verified = 1

# Check for NOT-VERIFIED
not_verified_match = re.search(r'Agent:\s*Oracle.*<promise>\s*NOT-VERIFIED:\s*([^<]+?)\s*</promise>', text, re.IGNORECASE | re.DOTALL)
not_verified_tag = ''
reason = ''
if not_verified_match:
    not_verified_tag = not_verified_match.group(0)
    reason = not_verified_match.group(1).strip()

# Check for exact <promise>DONE</promise> (not substring in NOT-VERIFIED reasons)
done_present = 0
for p in promises:
    if p.strip().upper() == 'DONE':
        done_present = 1
        break

# Output as tab-separated fields
print('\\t'.join([
    str(len(promises)),
    str(has_verified),
    '1' if not_verified_tag else '0',
    reason,
    str(done_present)
]))
" 2>/dev/null || echo "0	0	0		0")

# Parse result: promises_count has_verified not_verified_flag reason done_present
promise_count=$(echo "$_parse_result" | cut -f1)
has_verified=$(echo "$_parse_result" | cut -f2)
not_verified_flag=$(echo "$_parse_result" | cut -f3)
reason=$(echo "$_parse_result" | cut -f4)
done_present=$(echo "$_parse_result" | cut -f5)

if [ "$promise_count" -eq 0 ]; then
  heartbeat "no promise tag → exit 0"
  exit 0
fi

iter=0
if [ -f "$ITER_FILE" ]; then
  iter=$(cat "$ITER_FILE" 2>/dev/null | tr -d '[:space:]' || echo "0")
  case "$iter" in
    ''|*[!0-9]*) iter=0; heartbeat "iter file corrupted, resetting to 0" ;;
  esac
fi

if [ "$not_verified_flag" -eq 1 ]; then
  iter=$((iter + 1))
  echo "$iter" > "$ITER_FILE" 2>/dev/null || true
  heartbeat "NOT-VERIFIED at iter=$iter: $reason → exit 2"
  echo "Ultrawork: Oracle rejected — $reason" >&2
  if [ "$iter" -ge 100 ]; then
    summary_file="$SESSION_DIR/.ultrawork-iter-summary"
    echo "Ultrawork: iteration cap reached ($iter). Aborting loop. See $summary_file" > "$summary_file"
    heartbeat "CAP REACHED at iter=$iter → exit 2"
    echo "Ultrawork: iteration cap reached ($iter). Aborting loop. See $summary_file" >&2
    exit 2
  fi
  exit 2
fi

if [ "$done_present" -eq 1 ] && [ "$has_verified" -eq 0 ]; then
  # Blocking unverified DONE is a gate, not an iteration. Do NOT increment iter here.
  heartbeat "DONE without VERIFIED at iter=$iter → exit 2"
  echo "Ultrawork: emit Oracle verdict before stopping. Dispatch Agent(subagent_type='oracle', prompt=<task + recent diff>)." >&2
  exit 2
fi

if [ "$has_verified" -eq 1 ]; then
  heartbeat "VERIFIED found → exit 0 (loop ends)"
  rm -f "$FLOW_FLAG" 2>/dev/null || true
  exit 0
fi

heartbeat "fallback → exit 0"
exit 0
