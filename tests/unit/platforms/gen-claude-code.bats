#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
  TARGET="$REPO/platforms/claude-code"
}

@test "generator runs for claude-code without error" {
  run bash _gpowers-gen-platform.sh claude-code
  [ "$status" -eq 0 ]
}

@test "platforms/claude-code/.claude-plugin/plugin.json is valid JSON" {
  jq empty < "$TARGET/.claude-plugin/plugin.json"
}

@test "plugin.json declares name=gpowers and version" {
  [ "$(jq -r .name < "$TARGET/.claude-plugin/plugin.json")" = "gpowers" ]
  jq -e .version < "$TARGET/.claude-plugin/plugin.json" >/dev/null
}

@test "skills.json lists every skill" {
  REPO="$BATS_TEST_DIRNAME/../../.."
  total=$(find "$REPO/core/skills" "$REPO/roles/skills" "$REPO/tools/skills" "$REPO/business/skills" \
              -name SKILL.md 2>/dev/null | wc -l)
  jq_count=$(jq -r '.skills | length' < "$TARGET/skills.json")
  [ "$jq_count" -eq "$total" ]
}

@test "commands/<slash>.md exists for every slash in catalog" {
  while IFS=$'\t' read -r slash _ skill _; do
    name="${slash#/}"
    [ -f "$TARGET/commands/$name.md" ] || { echo "missing command file: $name"; return 1; }
  done < <(GPOWERS_HOME="$REPO" "$REPO/bin/_gpowers-list-slashes.sh")
}

@test "commands/<slash>.md body references the skill SOURCE" {
  for f in "$TARGET/commands"/*.md; do
    [ "$(basename "$f")" = "review.md" ] && continue
    grep -q "SOURCE:" "$f" || { echo "no SOURCE ref in $(basename "$f")"; return 1; }
  done
}

@test "hooks.json present (claude-code supports hooks)" {
  [ -f "$TARGET/hooks.json" ]
  jq empty < "$TARGET/hooks.json"
}
