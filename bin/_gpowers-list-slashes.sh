#!/usr/bin/env bash
# Emits TSV: slash<TAB>module<TAB>skill_dir<TAB>requires_driver<TAB>description
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

for module in core roles tools; do
  dir="$GPOWERS_HOME/$module/skills"
  [ -d "$dir" ] || continue
  for skill in "$dir"/*/; do
    [ -d "$skill" ] || continue
    file="$skill/SKILL.md"
    [ -f "$file" ] || continue
    skill_dir=$(basename "$skill")
    [ "$skill_dir" != "using-gpowers" ] || continue
    slash=$(awk -F': ' '/^slash:/ {print $2; exit}' "$file")
    [ -n "$slash" ] || slash="/$skill_dir"
    driver=$(awk -F': ' '/^requires-driver:/ {print $2; exit}' "$file")
    [ -n "$driver" ] || driver="none"
    # description: take first line of description field (ignores multi-line YAML)
    desc=$(awk -F': ' '/^description:/ {sub(/^description: /, ""); print; exit}' "$file")
    printf '%s\t%s\t%s\t%s\t%s\n' "$slash" "$module" "$skill_dir" "$driver" "$desc"
  done
done | sort
