#!/usr/bin/env bash
# Emits TSV: slash<TAB>module<TAB>skill_dir<TAB>requires_driver
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

for module in core roles tools business; do
  dir="$GPOWERS_HOME/$module/skills"
  [ -d "$dir" ] || continue
  for skill in "$dir"/*/; do
    [ -d "$skill" ] || continue
    file="$skill/SKILL.md"
    [ -f "$file" ] || continue
    slash=$(awk -F': ' '/^slash:/ {print $2; exit}' "$file")
    [ -n "$slash" ] || continue
    driver=$(awk -F': ' '/^requires-driver:/ {print $2; exit}' "$file")
    [ -n "$driver" ] || driver="none"
    skill_dir=$(basename "$skill")
    printf '%s\t%s\t%s\t%s\n' "$slash" "$module" "$skill_dir" "$driver"
  done
done | sort
