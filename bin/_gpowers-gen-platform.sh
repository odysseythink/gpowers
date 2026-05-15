#!/usr/bin/env bash
# Usage: _gpowers-gen-platform.sh <platform>
# Reads $GPOWERS_HOME/platforms/_platform-shapes.json + the slash catalog,
# writes $GPOWERS_HOME/platforms/<platform>/{<manifest>,skills.json,commands/...,hooks.json}.
set -euo pipefail

: "${GPOWERS_HOME:?GPOWERS_HOME required}"
PLATFORM="${1:?platform name required}"
SHAPES="$GPOWERS_HOME/platforms/_platform-shapes.json"
[ -f "$SHAPES" ] || { echo "shapes file missing: $SHAPES" >&2; exit 1; }

# Pull this platform's shape
shape() { jq -r --arg p "$PLATFORM" ".platforms[\$p].$1" < "$SHAPES"; }
MANIFEST=$(shape manifest_filename)
COMMAND_DIR=$(shape command_dir)
SUPPORTS_HOOKS=$(shape supports_hooks)
NS_MODE=$(shape namespace_mode)
[ "$MANIFEST" != "null" ] || { echo "unknown platform: $PLATFORM" >&2; exit 1; }

OUT="$GPOWERS_HOME/platforms/$PLATFORM"
mkdir -p "$OUT/$COMMAND_DIR"

# 1) plugin/extension manifest
jq -n --arg name gpowers \
      --arg version "1.0.0" \
      --arg ns "$NS_MODE" \
      '{
        "$schema": "https://gpowers.dev/schemas/plugin.json",
        name: $name,
        version: $version,
        namespace_mode: $ns,
        description: "gpowers — unified methodology + roles + tools + business automation",
        modules: ["core","roles","tools","business"]
      }' > "$OUT/$MANIFEST"

# 2) skills.json: complete index across all four modules
SKILLS_JSON='{ "skills": [] }'
for module in core roles tools business; do
  dir="$GPOWERS_HOME/$module/skills"
  [ -d "$dir" ] || continue
  for s in "$dir"/*/; do
    [ -d "$s" ] || continue
    file="$s/SKILL.md"
    [ -f "$file" ] || continue
    name=$(basename "$s")
    desc=$(awk -F': ' '/^description:/ {sub(/^description: /, ""); print; exit}' "$file")
    SKILLS_JSON=$(echo "$SKILLS_JSON" | jq \
      --arg n "$name" \
      --arg m "$module" \
      --arg d "$desc" \
      --arg p "$module/skills/$name/SKILL.md" \
      '.skills += [{name: $n, module: $m, description: $d, path: $p}]')
  done
done
echo "$SKILLS_JSON" > "$OUT/skills.json"

# 3) Command files (one .md per slash)
"$GPOWERS_HOME/bin/_gpowers-list-slashes.sh" | while IFS=$'\t' read -r slash module skill_dir driver; do
  cmd_name="${slash#/}"
  cmd_file="$OUT/$COMMAND_DIR/$cmd_name.md"
  cat > "$cmd_file" <<MD
---
slash: $slash
module: $module
skill: $skill_dir
requires_driver: $driver
---

<!-- SOURCE: \$GPOWERS_HOME/$module/skills/$skill_dir/SKILL.md -->

This command invokes the gpowers skill **$skill_dir** ($module).

Refer to the source SKILL.md (above) for the full workflow. The platform's skill mechanism will load it on demand.
MD
done

# 4) hooks.json (where supported)
if [ "$SUPPORTS_HOOKS" = "true" ]; then
  cp "$GPOWERS_HOME/core/hooks/hooks.json" "$OUT/hooks.json"
fi

echo "platforms/$PLATFORM/ generated."
