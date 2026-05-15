#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

# Tools split: non-browser uses Plan #4 importer; browser ones additionally
# get Plan #5's _gpowers-rewrite-browser.py applied to the body.
SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/tools/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/tools/skills"

BROWSER_LIST=$(jq -r '.submodules.browser_dependent[]' < "$GPOWERS_HOME/tools/upstream-source.json")

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  "$GPOWERS_HOME/bin/_gpowers-import-tool.sh" "$src" "$GPOWERS_HOME/tools/skills/$name"

  # If this is a browser-dependent skill, run the rewriter on its body
  if echo "$BROWSER_LIST" | grep -qx "$name"; then
    file="$GPOWERS_HOME/tools/skills/$name/SKILL.md"
    fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$file")
    head -n "$fm_end" "$file" > "$SCRATCH/fm.md"
    tail -n +$((fm_end+1)) "$file" \
      | "$GPOWERS_HOME/bin/_gpowers-rewrite-browser.py" > "$SCRATCH/body.md"
    cat "$SCRATCH/fm.md" "$SCRATCH/body.md" > "$file"
  fi
done

rm -rf "$SCRATCH"
