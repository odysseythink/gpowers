#!/usr/bin/env bash
# Usage: _gpowers-import-core.sh <source-skills-dir> <dest-skills-dir> <upstream-tag>
set -euo pipefail

_gpowers_transform_skill() {
  local src="$1" dst="$2" tag="$3"
  awk -v tag="$tag" '
    BEGIN { in_fm = 0; fm_done = 0; injected = 0 }
    NR == 1 && /^---$/ { in_fm = 1; print; next }
    in_fm && /^---$/ {
      if (!injected) { print "namespace: core"; print "upstream: " tag; injected = 1 }
      in_fm = 0; fm_done = 1; print; next
    }
    in_fm { print; next }
    fm_done { gsub(/superpowers:/, "gpowers:"); print; next }
    { print }
  ' "$src" > "$dst"
}

SRC="${1:?source dir required}"
DEST="${2:?destination dir required}"
TAG="${3:?upstream tag required}"

[ -d "$SRC" ] || { echo "source missing: $SRC" >&2; exit 1; }
mkdir -p "$DEST"

for src_dir in "$SRC"/*/; do
  name=$(basename "$src_dir")
  [ "$name" = "using-superpowers" ] && continue
  dst_dir="$DEST/$name"
  mkdir -p "$dst_dir"
  _gpowers_transform_skill "$src_dir/SKILL.md" "$dst_dir/SKILL.md" "$TAG"
done
