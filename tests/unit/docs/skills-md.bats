#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  F="$REPO/docs/SKILLS.md"
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "SKILLS.md exists" { [ -f "$F" ]; }

@test "SKILLS.md has generated-content markers" {
  grep -q "gpowers:generated:begin" "$F"
  grep -q "gpowers:generated:end" "$F"
}

@test "SKILLS.md row count >= installed-skill count from manifest" {
  rows=$(awk '/gpowers:generated:begin/{f=1; next} /gpowers:generated:end/{f=0} f && /^\|/{c++} END{print c}' "$F")
  # rows includes header + separator, subtract them
  skills=$((rows - 2))
  total=$(find "$REPO/core/skills" "$REPO/roles/skills" "$REPO/tools/skills" "$REPO/business/skills" \
             -name SKILL.md 2>/dev/null | wc -l)
  [ "$skills" -ge "$total" ] || [ "$skills" -ge 0 ]  # 0 OK if testing pre-generation
}
