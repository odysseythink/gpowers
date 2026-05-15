#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  CI="$REPO/.github/workflows/ci.yml"
}

@test "ci.yml exists" { [ -f "$CI" ]; }
@test "ci.yml defines unit job" { grep -q "^  unit:" "$CI"; }
@test "ci.yml defines integration job" { grep -q "^  integration:" "$CI"; }
@test "ci.yml defines shellcheck job" { grep -q "^  shellcheck:" "$CI"; }
@test "ci.yml runs ./tests/run.sh" { grep -q "tests/run.sh" "$CI"; }
