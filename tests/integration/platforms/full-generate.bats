#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "gen all completes successfully" {
  run gpowers-platforms gen all
  [ "$status" -eq 0 ]
}

@test "verify all reports OK for every platform" {
  out=$(gpowers-platforms verify all)
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    echo "$out" | grep -q "OK: $p" || { echo "verify missing $p"; return 1; }
  done
}

@test "every non-kimi platform has commands/ populated" {
  for p in claude-code codex gemini cursor opencode copilot; do
    count=$(find "$GPOWERS_HOME/platforms/$p/commands" -name '*.md' 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] || { echo "$p has no commands"; return 1; }
  done
}

@test "kimi adapters count matches total non-core skill count" {
  total=$(find "$GPOWERS_HOME/roles/skills" "$GPOWERS_HOME/tools/skills" \
               "$GPOWERS_HOME/business/skills" -name SKILL.md 2>/dev/null | wc -l)
  # Plus core skills minus using-gpowers, plus 1 router 'gpowers'
  core_count=$(find "$GPOWERS_HOME/core/skills" -name SKILL.md 2>/dev/null | wc -l)
  expected=$(( total + core_count ))   # core_count includes using-gpowers; we skip it but add router
  adapters=$(find "$GPOWERS_HOME/platforms/kimi/adapters" -mindepth 1 -maxdepth 1 -type d | wc -l)
  [ "$adapters" -eq "$expected" ]
}
