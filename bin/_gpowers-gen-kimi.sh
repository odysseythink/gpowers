#!/usr/bin/env bash
# Generates platforms/kimi/adapters/ with flat-namespace adapter skills.
# Each adapter inlines (a) the using-gpowers preamble, (b) the source skill body.
set -euo pipefail

: "${GPOWERS_HOME:?GPOWERS_HOME required}"
ADAPTERS_DIR="$GPOWERS_HOME/platforms/kimi/adapters"
mkdir -p "$ADAPTERS_DIR"

# Preamble = using-gpowers body without its frontmatter
USING="$GPOWERS_HOME/core/skills/using-gpowers/SKILL.md"
[ -f "$USING" ] || { echo "using-gpowers missing: $USING" >&2; exit 1; }
PREAMBLE=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$USING")

# Router anchor — gpowers/ adapter just hosts the using-gpowers content
mkdir -p "$ADAPTERS_DIR/gpowers"
cat > "$ADAPTERS_DIR/gpowers/SKILL.md" <<HEADER
---
name: gpowers
description: gpowers entry — four-module model (core / roles / tools / business)
gpowers-source: core/skills/using-gpowers/SKILL.md
---

$PREAMBLE
HEADER

# Every other skill becomes gpowers-<original>
for module in core roles tools business; do
  dir="$GPOWERS_HOME/$module/skills"
  [ -d "$dir" ] || continue
  for s in "$dir"/*/; do
    [ -d "$s" ] || continue
    orig=$(basename "$s")
    [ "$orig" = "using-gpowers" ] && continue
    file="$s/SKILL.md"
    [ -f "$file" ] || continue

    adapter_name="gpowers-$orig"
    case "$orig" in gpowers-*) adapter_name="$orig" ;; esac
    adapter_dir="$ADAPTERS_DIR/$adapter_name"
    mkdir -p "$adapter_dir"

    src_frontmatter=$(awk 'BEGIN{fm=0} /^---$/{fm++; if(fm<=2)print; next} fm<2{print}' "$file")
    src_body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$file")
    orig_desc=$(awk -F': ' '/^description:/ {sub(/^description: /, ""); print; exit}' "$file")

    cat > "$adapter_dir/SKILL.md" <<ADAPTER
---
name: $adapter_name
description: $orig_desc (gpowers adapter for Kimi)
gpowers-source: $module/skills/$orig/SKILL.md
gpowers-module: $module
---

<!-- gpowers preamble (auto, four-module model) -->

$PREAMBLE

<!-- SOURCE: \$GPOWERS_HOME/$module/skills/$orig/SKILL.md -->

$src_body
ADAPTER
  done
done

# Flat manifest for Kimi discovery
find "$ADAPTERS_DIR" -mindepth 1 -maxdepth 1 -type d | sort | awk -F/ '{print $NF}' \
  | jq -R . | jq -s '{adapters: .}' \
  > "$GPOWERS_HOME/platforms/kimi/kimi-skills.json"

echo "Kimi adapters generated under $ADAPTERS_DIR"
