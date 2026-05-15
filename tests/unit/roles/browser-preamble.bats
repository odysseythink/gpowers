#!/usr/bin/env bats

setup() { SKILL="$BATS_TEST_DIRNAME/../../../roles/skills/design-review/SKILL.md"; }

@test "design-review declares requires-driver: browser" {
  grep -q "^requires-driver: browser$" "$SKILL"
}

@test "design-review has browser preamble" {
  grep -q "source \"\$GPOWERS_HOME/tools/drivers/browser/select-driver.sh\"" "$SKILL"
}

@test "design-review uses gpowers-browser verbs, not mcp__ refs" {
  ! grep -q "mcp__claude-in-chrome__" "$SKILL"
}

@test "design-review has no literal npx playwright commands" {
  ! grep -q "npx playwright" "$SKILL"
}
