#!/usr/bin/env bash
set -euo pipefail
: "${GPOWERS_HOME:?GPOWERS_HOME required}"

SCRATCH=$(mktemp -d)
mv "$GPOWERS_HOME/roles/skills" "$SCRATCH/upstream-skills"
mkdir -p "$GPOWERS_HOME/roles/skills"

for src in "$SCRATCH/upstream-skills"/*/; do
  name=$(basename "$src")
  # Apply the same /review → /pr-review rename as Plan #6 Task 5
  if [ "$name" = "review" ]; then
    "$GPOWERS_HOME/bin/_gpowers-import-role.sh" "$src" "$GPOWERS_HOME/roles/skills/pr-review"
    sed -i.bak \
        -e 's/^name: review$/name: pr-review/' \
        -e 's|^slash: /review$|slash: /pr-review|' \
        "$GPOWERS_HOME/roles/skills/pr-review/SKILL.md"
    rm -f "$GPOWERS_HOME/roles/skills/pr-review/SKILL.md.bak"
  else
    "$GPOWERS_HOME/bin/_gpowers-import-role.sh" "$src" "$GPOWERS_HOME/roles/skills/$name"
  fi
done

# Apply browser preamble to design-review if it exists
DR="$GPOWERS_HOME/roles/skills/design-review/SKILL.md"
if [ -f "$DR" ] && ! grep -q "^requires-driver: browser$" "$DR"; then
  fm_end=$(awk '/^---$/{c++; if(c==2){print NR; exit}}' "$DR")
  head -n "$fm_end" "$DR" > "$SCRATCH/fm.md"
  tail -n +$((fm_end+1)) "$DR" | "$GPOWERS_HOME/bin/_gpowers-rewrite-browser.py" > "$SCRATCH/body.md"
  awk '/^---$/{c++; if(c==2)print "requires-driver: browser"; print; next} {print}' \
       "$SCRATCH/fm.md" > "$DR"
  cat "$SCRATCH/body.md" >> "$DR"
fi

rm -rf "$SCRATCH"
