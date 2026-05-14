#!/usr/bin/env bash
# Usage: _gpowers-import-role.sh <src-skill-dir> <dst-skill-dir>
# Wraps _gpowers-import-tool.sh and changes namespace: tools → namespace: roles.
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

"$HERE/_gpowers-import-tool.sh" "$SRC" "$DST"

# Swap namespace label
sed -i.bak 's/^namespace: tools$/namespace: roles/' "$DST/SKILL.md"
rm -f "$DST/SKILL.md.bak"
