#!/usr/bin/env bash
# Usage: _gpowers-import-business.sh <src-skill-dir> <dst-skill-dir>
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

"$HERE/_gpowers-import-tool.sh" "$SRC" "$DST"
sed -i.bak 's/^namespace: tools$/namespace: business/' "$DST/SKILL.md"
rm -f "$DST/SKILL.md.bak"

# Append safety footer
cat >> "$DST/SKILL.md" <<'NOTE'

---

> _Business module note:_ this skill is part of the optional `business/` module installed with `--with-business`. See `business/DISCLAIMER.md` for the dual-use / responsible-automation notice.
NOTE
