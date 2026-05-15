#!/usr/bin/env bash
# Usage: seed-repo.sh <tmp-base> <kind>
#   kind = superpowers | gstack-roles | gstack-tools | gstack-business
# Creates a bare repo under <tmp-base>/<kind>.git with one initial commit
# of stub skills mirroring the production layout. Echoes the bare repo path.
set -euo pipefail
BASE="${1:?base dir required}"
KIND="${2:?kind required}"

BARE="$BASE/$KIND.git"
WORK="$BASE/$KIND.work"
mkdir -p "$BARE" "$WORK"
git init --bare -q "$BARE"

cd "$WORK"
git init -q
git -c user.email=t@t -c user.name=t commit --allow-empty -q -m initial

case "$KIND" in
  superpowers)
    for n in brainstorming writing-plans test-driven-development; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: $n upstream
---

# $n

Upstream content. Body references superpowers:writing-plans.
F
    done
    ;;
  gstack-tools)
    for n in ship health make-pdf; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack tool $n
slash: /$n
---

# $n

Body references ~/.gstack/state and gstack-$n CLI.
F
    done
    ;;
  gstack-roles)
    for n in pr-review cso; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack role $n
slash: /$n
---

# $n

Role body. Reads ~/.gstack/config.
F
    done
    ;;
  gstack-business)
    for n in money money-content; do
      mkdir -p "skills/$n"
      cat > "skills/$n/SKILL.md" <<F
---
name: $n
description: gstack business $n
slash: /$n
---

# $n

Business strategy stub. Data: ~/.gstack/data/$n/.
F
    done
    ;;
esac

git add -A
git -c user.email=t@t -c user.name=t commit -q -m "seed $KIND"
git push -q "$BARE" master:main 2>/dev/null || git push -q "$BARE" HEAD:main
echo "$BARE"
