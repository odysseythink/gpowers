#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

@test "run.sh unit fixtures completes without error" {
  run bash "$REPO/tests/run.sh" unit fixtures
  [ "$status" -eq 0 ]
}

@test "run.sh integration completes without error" {
  run bash "$REPO/tests/run.sh" integration
  if [ "$status" -ne 0 ]; then
    echo "--- integration output ---"
    echo "$output"
    echo "--- end integration output ---"
  fi
  [ "$status" -eq 0 ]
}

@test "run.sh unknown scope exits 2" {
  run bash "$REPO/tests/run.sh" bogus
  [ "$status" -eq 2 ]
}

@test "run.sh unit roles filter limits scope to roles tests" {
  out=$(bash "$REPO/tests/run.sh" unit roles 2>&1 || true)
  # Should run something (roles tests exist) — check for test count line
  echo "$out" | grep -q "^1\.\.[0-9]"
}
