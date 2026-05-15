#!/usr/bin/env bash
# Reads plan JSON on stdin. Executes moves with journal + rollback.
# Flags: --dry-run | --yes (skip interactive confirm)
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

DRY=false
YES=false
while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY=true; shift;;
    --yes)     YES=true; shift;;
    *) echo "unknown flag: $1" >&2; exit 2;;
  esac
done

PLAN=$(cat -)
TOTAL=$(echo "$PLAN" | jq -r '.total')
CONFLICTS=$(echo "$PLAN" | jq -r '.conflicts | length')

echo "Plan: $TOTAL moves, $CONFLICTS conflicts."

if [ "$CONFLICTS" -gt 0 ] && ! $YES; then
  echo "Refusing to proceed with conflicts. Resolve them or pass --yes to skip conflicting items." >&2
  exit 3
fi

if ! $YES && ! $DRY; then
  read -r -p "Proceed with migration? [y/N] " ans
  case "$ans" in y|Y|yes|YES) ;; *) echo "aborted."; exit 0;; esac
fi

JOURNAL="$GPOWERS_HOME/state/migrate-journal.jsonl"
$DRY || mkdir -p "$(dirname "$JOURNAL")"

rollback() {
  echo "Rolling back…" >&2
  if [ -s "$JOURNAL" ]; then
    # Replay in reverse: move dst back to src
    tac "$JOURNAL" | while read -r line; do
      src=$(echo "$line" | jq -r .src)
      dst=$(echo "$line" | jq -r .dst)
      if [ -e "$dst" ] && [ ! -e "$src" ]; then
        mkdir -p "$(dirname "$src")"
        mv "$dst" "$src"
      fi
    done
  fi
}
trap 'rollback' ERR

while read -r mapping; do
  src=$(echo "$mapping" | jq -r .src)
  dst=$(echo "$mapping" | jq -r .dst)
  [ -e "$src" ] || continue
  # Skip conflicting dsts
  if [ -e "$dst" ]; then
    echo "[skip] $src (dst exists)"
    continue
  fi
  if $DRY; then
    echo "[dry-run] $src → $dst"
    continue
  fi
  mkdir -p "$(dirname "$dst")"
  if [ -f "$src" ]; then
    mv "$src" "$dst"
  else
    rsync -a --remove-source-files "$src/" "$dst/" 2>/dev/null
    rmdir "$src" 2>/dev/null || true
  fi
  echo "$mapping" >> "$JOURNAL"
  echo "[ok]   $src → $dst"
done < <(echo "$PLAN" | jq -c '.mappings[]')

trap - ERR
echo "Migration complete."
