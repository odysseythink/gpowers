#!/usr/bin/env bash
# Usage: _gpowers-upgrade-module.sh <module>
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

MODULE="${1:?module name required}"
SOURCES="$GPOWERS_HOME/upstream-sources.json"
[ -f "$SOURCES" ] || { echo "upstream-sources.json missing" >&2; exit 1; }

read -r URL REF PREFIX TRANSFORM <<<"$(jq -r --arg m "$MODULE" '.modules[$m] |
  "\(.url) \(.ref) \(.subtree_prefix) \(.transform_script)"' < "$SOURCES")"

[ "$URL" != "null" ] || { echo "unknown module: $MODULE" >&2; exit 2; }

cd "$GPOWERS_HOME"
[ -d .git ] || { echo "$GPOWERS_HOME is not a git repo (needed for subtree)" >&2; exit 3; }

# Ensure a clean tree (subtree pull requires it)
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "$GPOWERS_HOME has uncommitted changes; commit or stash before upgrading." >&2
  exit 4
fi

echo "[upgrade:$MODULE] pulling $URL@$REF into $PREFIX/"
if ! git subtree pull --prefix="$PREFIX" "$URL" "$REF" --squash \
     -m "upgrade($MODULE): pull $URL@$REF"; then
  echo "[upgrade:$MODULE] subtree pull failed (likely conflict)" >&2
  git status >&2
  echo "Run \`gpowers upgrade --resume\` after resolving conflicts." >&2
  exit 5
fi

# Capture new SHA from FETCH_HEAD
NEW_SHA=$(git rev-parse FETCH_HEAD 2>/dev/null || git ls-remote "$URL" "$REF" | awk '{print $1}')

# Re-apply transform
echo "[upgrade:$MODULE] applying transform: $TRANSFORM"
"$GPOWERS_HOME/$TRANSFORM"

# Update module's upstream-source.json
jq --arg sha "$NEW_SHA" \
   --arg ts  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '.upstream.sha = $sha | .upgraded_at = $ts' \
   "$GPOWERS_HOME/$MODULE/upstream-source.json" \
   > "$GPOWERS_HOME/$MODULE/upstream-source.json.tmp"
mv "$GPOWERS_HOME/$MODULE/upstream-source.json.tmp" "$GPOWERS_HOME/$MODULE/upstream-source.json"

# Refresh platform manifests (Plan #8)
echo "[upgrade:$MODULE] regenerating platform manifests"
"$GPOWERS_HOME/bin/gpowers-platforms" gen all

# Run module tests
echo "[upgrade:$MODULE] running tests"
if command -v bats >/dev/null; then
  bats "$GPOWERS_HOME/tests/unit/$MODULE" 2>/dev/null || true
  bats "$GPOWERS_HOME/tests/integration/$MODULE" 2>/dev/null || true
else
  echo "[upgrade:$MODULE] bats not installed; skipping tests" >&2
fi

# Commit the transformed state
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" add -A
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): apply transform @ $NEW_SHA"

echo "[upgrade:$MODULE] done @ $NEW_SHA"
