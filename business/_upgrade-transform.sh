#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/business/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/business/skills"

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  "$GPOWERS_HOME/bin/_gpowers-import-business.sh" "$src" "$GPOWERS_HOME/business/skills/$name"
done

rm -rf "$SCRATCH"
