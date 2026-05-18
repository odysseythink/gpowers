#!/usr/bin/env bash
# Usage: _gpowers-docgen.sh <kind>
# kind: skills | commands | platforms
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

kind="${1:?kind required}"

case "$kind" in
  skills)
    printf '| Module | Skill | Description |\n'
    printf '|---|---|---|\n'
    for module in core roles tools; do
      dir="$GPOWERS_HOME/$module/skills"
      [ -d "$dir" ] || continue
      for s in "$dir"/*/; do
        [ -d "$s" ] || continue
        [ -f "$s/SKILL.md" ] || continue
        name=$(basename "$s")
        desc=$(awk -F': ' '/^description:/ {sub(/^description: /, ""); print; exit}' "$s/SKILL.md" \
               | tr '|' '/' | cut -c1-120)
        printf '| %s | %s | %s |\n' "$module" "$name" "$desc"
      done | sort
    done
    ;;
  commands)
    printf '| Slash | Module | Skill | Notes |\n'
    printf '|---|---|---|---|\n'
    "$GPOWERS_HOME/bin/_gpowers-list-slashes.sh" | while IFS=$'\t' read -r slash module skill driver; do
      note=""
      [ "$driver" = "browser" ] && note="requires browser driver"
      printf '| `%s` | %s | %s | %s |\n' "$slash" "$module" "$skill" "$note"
    done
    ;;
  platforms)
    printf '| Platform | Manifest | Hooks | Install link |\n'
    printf '|---|---|---|---|\n'
    for p in $(jq -r '.platforms | keys[]' < "$GPOWERS_HOME/platforms/_platform-shapes.json"); do
      mf=$(jq -r ".platforms.\"$p\".manifest_filename" < "$GPOWERS_HOME/platforms/_platform-shapes.json")
      hk=$(jq -r ".platforms.\"$p\".supports_hooks"   < "$GPOWERS_HOME/platforms/_platform-shapes.json")
      lt=$(jq -r ".platforms.\"$p\".install_link_target" < "$GPOWERS_HOME/platforms/_platform-shapes.json")
      printf '| %s | %s | %s | `%s` |\n' "$p" "$mf" "$hk" "$lt"
    done
    ;;
  *) echo "unknown kind: $kind" >&2; exit 2;;
esac
