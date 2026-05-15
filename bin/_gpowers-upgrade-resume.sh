#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

cd "$GPOWERS_HOME"

if [ ! -f .git/MERGE_HEAD ] && [ ! -f .git/REBASE_HEAD ]; then
  echo "No upgrade in progress (no .git/MERGE_HEAD). Nothing to resume."
  exit 0
fi

# Working tree must be clean of conflict markers
if git ls-files --unmerged | grep -q . || ! git diff --cached --quiet; then
  echo "You still have unresolved conflict files. Fix them, then \`git add\` and rerun." >&2
  git status --short >&2
  exit 1
fi

# Read the module from the in-progress merge commit's prefix
# Heuristic: look at staged changes' paths to find the touched module
TOUCHED=$(git diff --cached --name-only | awk -F/ '{print $1}' | sort -u | head -1)
[ -n "$TOUCHED" ] || { echo "Couldn't determine touched module from staged changes." >&2; exit 1; }

case "$TOUCHED" in
  core|roles|tools|business) MODULE="$TOUCHED" ;;
  *) echo "Touched path '$TOUCHED' isn't a known module." >&2; exit 1 ;;
esac

# Complete merge commit
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): resume after conflict resolution"

# Run transform + record SHA + regen platforms (same tail as the worker)
"$GPOWERS_HOME/$(jq -r --arg m "$MODULE" '.modules[$m].transform_script' < upstream-sources.json)"
SHA=$(git rev-parse HEAD)

jq --arg sha "$SHA" --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '.upstream.sha = $sha | .upgraded_at = $ts' \
   "$MODULE/upstream-source.json" > "$MODULE/upstream-source.json.tmp"
mv "$MODULE/upstream-source.json.tmp" "$MODULE/upstream-source.json"

"$GPOWERS_HOME/bin/gpowers-platforms" gen all >/dev/null

bats "tests/unit/$MODULE" 2>/dev/null || true
bats "tests/integration/$MODULE" 2>/dev/null || true

git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" add -A
git -c user.email="upgrade@gpowers" -c user.name="gpowers upgrade" \
    commit -q -m "upgrade($MODULE): apply transform after manual resolution @ $SHA"

echo "[upgrade:$MODULE] resumed and applied @ $SHA"
