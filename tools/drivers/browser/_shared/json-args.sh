# Source me. Reads JSON object from stdin into associative-like access.
# Usage: source json-args.sh; ARGS_JSON="$(read_args)"; arg .url; arg .tab_id

read_args() {
  cat -
}

arg() {
  # arg <jq-path> [<default>] — extracts a scalar from $ARGS_JSON
  local path="$1" default="${2-}"
  local val
  val=$(printf '%s' "$ARGS_JSON" | jq -r "$path // empty" 2>/dev/null || echo "")
  if [ -z "$val" ] && [ -n "$default" ]; then echo "$default"; else echo "$val"; fi
}

emit() {
  # emit <jq-json-template> — prints a JSON object to stdout
  printf '%s\n' "$1"
}

die() {
  # die <message> [<verb>] [<tab_id>]
  local msg="$1" verb="${2:-unknown}" tab="${3:-}"
  jq -n --arg e "$msg" --arg v "$verb" --arg t "$tab" \
     '{error: $e, verb: $v, tab_id: $t}' >&2
  exit 1
}
