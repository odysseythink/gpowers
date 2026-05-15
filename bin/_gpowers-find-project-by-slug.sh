#!/usr/bin/env bash
# Usage: _gpowers-find-project-by-slug.sh <slug>
# Echoes resolved repo path or empty.
set -euo pipefail
SLUG="${1:?slug required}"
: "${HOME:?HOME required}"

# 1. explicit recording
RECORDED="$HOME/.gstack/projects/$SLUG/.repo-path"
if [ -f "$RECORDED" ]; then
  path=$(head -1 "$RECORDED" | tr -d '[:space:]')
  if [ -n "$path" ] && [ -d "$path" ]; then
    echo "$path"
    exit 0
  fi
fi

# 2. find by directory name match (cheap, bounded depth)
match=$(find "$HOME" -maxdepth 6 -type d -name "$SLUG" 2>/dev/null \
        | while read -r d; do
            if [ -d "$d/.git" ]; then echo "$d"; break; fi
          done | head -1)
if [ -n "$match" ]; then
  echo "$match"
  exit 0
fi

# 3. fallback empty
echo ""
