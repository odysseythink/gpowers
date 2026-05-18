#!/usr/bin/env bats

setup() {
  ROLES="$BATS_TEST_DIRNAME/../../../roles/skills"
  EXPECTED="office-hours plan-ceo-review autoplan plan-eng-review plan-devex-review
            devex-review investigate codex pr-review plan-design-review
            design-consultation design-shotgun design-html design-review cso
            retro document-release learn pair-agent plan-tune"
}

@test "every expected role skill exists" {
  for name in $EXPECTED; do
    [ -f "$ROLES/$name/SKILL.md" ] || { echo "missing: $name"; return 1; }
  done
}

@test "exactly 20 role skills present" {
  count=$(find "$ROLES" -mindepth 1 -maxdepth 1 -type d | wc -l)
  [ "$count" -eq 20 ]
}

@test "every role declares namespace: roles" {
  for name in $EXPECTED; do
    grep -q "^namespace: roles$" "$ROLES/$name/SKILL.md" || { echo "$name: no namespace"; return 1; }
  done
}

@test "every role declares upstream: gstack@main" {
  for name in $EXPECTED; do
    grep -q "^upstream: gstack@main$" "$ROLES/$name/SKILL.md" || { echo "$name: no upstream"; return 1; }
  done
}

@test "every role declares its slash command" {
  for name in $EXPECTED; do
    slash=$(grep -m1 '^slash:' "$ROLES/$name/SKILL.md" | awk '{print $2}')
    if [ -z "$slash" ]; then
      slash="/$name"
    fi
    [[ "$slash" == /* ]] || { echo "$name: slash must start with /"; return 1; }
  done
}
