#!/usr/bin/env bats

load ../../helpers/setup.bash

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  CORE_SKILLS="$GPOWERS_REPO/core/skills"
}

@test "every core skill has SKILL.md" {
  for name in using-gpowers brainstorming writing-plans executing-plans \
              subagent-driven-development test-driven-development \
              systematic-debugging verification-before-completion \
              requesting-code-review receiving-code-review \
              finishing-a-development-branch dispatching-parallel-agents \
              using-git-worktrees writing-skills; do
    [ -f "$CORE_SKILLS/$name/SKILL.md" ] || {
      echo "missing: $CORE_SKILLS/$name/SKILL.md"
      return 1
    }
  done
}

@test "every core skill has namespace: core in frontmatter" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    grep -q "^namespace: core$" "$dir/SKILL.md" || {
      echo "$name: missing 'namespace: core'"
      return 1
    }
  done
}

@test "every core skill has upstream: superpowers@v5.1.0 except using-gpowers" {
  for dir in "$CORE_SKILLS"/*/; do
    name=$(basename "$dir")
    [ "$name" = "using-gpowers" ] && continue
    grep -q "^upstream: superpowers@v5\.1\.0$" "$dir/SKILL.md" || {
      echo "$name: missing 'upstream: superpowers@v5.1.0'"
      return 1
    }
  done
}

@test "using-gpowers has upstream: gpowers-native" {
  grep -q "^upstream: gpowers-native$" "$CORE_SKILLS/using-gpowers/SKILL.md"
}
