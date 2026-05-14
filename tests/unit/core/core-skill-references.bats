#!/usr/bin/env bats

setup() {
  CORE_SKILLS="$BATS_TEST_DIRNAME/../../../core/skills"
}

@test "no 'superpowers:' refs in body of core skills" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    # strip frontmatter, then grep
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$dir/SKILL.md")
    if echo "$body" | grep -q "superpowers:"; then
      echo "$name body still contains 'superpowers:'"
      return 1
    fi
  done
}

@test "frontmatter 'upstream: superpowers@...' is allowed and present" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    [ "$name" = "using-gpowers" ] && continue
    head -10 "$dir/SKILL.md" | grep -q "upstream: superpowers@" || {
      echo "$name missing upstream frontmatter"; return 1
    }
  done
}
