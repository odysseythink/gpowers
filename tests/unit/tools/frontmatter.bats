#!/usr/bin/env bats

setup() {
  TOOLS="$BATS_TEST_DIRNAME/../../../tools/skills"
  EXPECTED="ship land-and-deploy landing-report setup-deploy health benchmark-models \
            context-save context-restore careful freeze guard unfreeze make-pdf \
            fix-the-roof simplify fewer-permission-prompts gpowers-upgrade"
}

@test "every expected non-browser tool skill exists" {
  for name in $EXPECTED; do
    [ -f "$TOOLS/$name/SKILL.md" ] || { echo "missing: $name"; return 1; }
  done
}

@test "every tool skill declares namespace: tools" {
  for name in $EXPECTED; do
    grep -q "^namespace: tools$" "$TOOLS/$name/SKILL.md" || {
      echo "$name: missing 'namespace: tools'"; return 1
    }
  done
}

@test "tool skills (except gpowers-upgrade stub) declare upstream: gstack@main" {
  for name in $EXPECTED; do
    [ "$name" = "gpowers-upgrade" ] && continue
    grep -q "^upstream: gstack@main$" "$TOOLS/$name/SKILL.md" || {
      echo "$name: missing 'upstream: gstack@main'"; return 1
    }
  done
}

@test "each skill declares its slash command" {
  for name in $EXPECTED; do
    slash=$(grep -m1 '^slash:' "$TOOLS/$name/SKILL.md" | awk '{print $2}')
    if [ -z "$slash" ]; then
      slash="/$name"
    fi
    [[ "$slash" == /* ]] || { echo "$name: slash must start with /"; return 1; }
  done
}
