#!/usr/bin/env bash
# Re-applies the Plan #2 transform after a git subtree pull from superpowers.
# Pre: GPOWERS_HOME contains a freshly-pulled core/skills/ in upstream form.
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

REF=$(jq -r '.modules.core.ref' < "$GPOWERS_HOME/upstream-sources.json")
TAG="superpowers@$REF"

# Move the freshly-pulled upstream form to a scratch dir, then run the importer
SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/core/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/core/skills"

"$GPOWERS_HOME/bin/_gpowers-import-core.sh" \
  "$SCRATCH/upstream-skills" \
  "$GPOWERS_HOME/core/skills" \
  "$TAG"

# using-gpowers is local-only; preserve it from prior state if removed by the pull
USING="$GPOWERS_HOME/core/skills/using-gpowers"
if [ ! -d "$USING" ] && [ -d "$SCRATCH/upstream-skills/using-gpowers.bak" ]; then
  cp -R "$SCRATCH/upstream-skills/using-gpowers.bak" "$USING"
fi

rm -rf "$SCRATCH"
