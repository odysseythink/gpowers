#!/usr/bin/env bash
# Usage: seed-gpowers-home.sh <target-dir>
# Copies repo contents into target-dir and runs `gpowers-platforms gen all`.
set -euo pipefail
TARGET="${1:?target required}"
REPO="${REPO:-$(cd "$(dirname "$0")/../.." && pwd)}"

rm -rf "$TARGET"
mkdir -p "$TARGET"
# Copy modules + bin + lib + platforms shape + manifest + upstream-sources
cp -R "$REPO/core" "$REPO/roles" "$REPO/tools" "$TARGET/"
[ -d "$REPO/business" ] && cp -R "$REPO/business" "$TARGET/"
cp -R "$REPO/bin" "$REPO/lib" "$TARGET/"
mkdir -p "$TARGET/platforms"
cp "$REPO/platforms/_platform-shapes.json" "$TARGET/platforms/"
cp "$REPO/manifest.json" "$TARGET/manifest.json"
cp "$REPO/upstream-sources.json" "$TARGET/upstream-sources.json"

export GPOWERS_HOME="$TARGET"
export PATH="$TARGET/bin:$TARGET/tools/bin:$PATH"

# Regenerate per-platform assets so platform-smoke tests have real files
"$TARGET/bin/gpowers-platforms" gen all >/dev/null

echo "$TARGET"
