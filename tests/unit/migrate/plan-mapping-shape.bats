#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export HOME="$REPO/tests/fixtures/migrate/fake-home-gstack"
  export GPOWERS_HOME="$HOME/.gpowers"
  PLAN="$REPO/bin/_gpowers-migrate-plan.sh"
}

@test "plan emits valid JSON" {
  "$PLAN" | jq empty
}

@test "plan has 'mappings' array" {
  out=$("$PLAN")
  echo "$out" | jq -e '.mappings | type == "array"' >/dev/null
}

@test "each mapping has src/dst/comment fields" {
  out=$("$PLAN")
  echo "$out" | jq -e 'all(.mappings[]; has("src") and has("dst") and has("comment"))' >/dev/null
}

@test "plan maps builder-profile to gpowers/config/" {
  out=$("$PLAN")
  echo "$out" | jq -e '.mappings[] | select(.src | endswith("builder-profile")) | .dst | endswith("config/builder-profile")' >/dev/null
}

@test "plan-handles-project-slug — proj-alpha resolves to fallback" {
  out=$("$PLAN")
  # proj-alpha has no .repo-path, no matching dir — should land in legacy-projects
  echo "$out" | jq -e '.mappings[] | select(.src | contains("projects/proj-alpha")) | .dst | contains("legacy-projects/proj-alpha")' >/dev/null
}
