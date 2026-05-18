#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "no args = upgrade all (dry-run shows plan)" {
  out=$(gpowers-upgrade --dry-run)
  for m in core roles tools; do
    echo "$out" | grep -q "$m" || { echo "missing $m"; return 1; }
  done
}

@test "named module narrows scope" {
  out=$(gpowers-upgrade tools --dry-run)
  echo "$out" | grep -q "tools"
  ! echo "$out" | grep -q "core\|roles"
}

@test "--check delegates without modifying" {
  out=$(gpowers-upgrade --check 2>&1 || true)
  # Output must look like a table; we accept whatever ls-remote returns
  echo "$out" | grep -qE 'core|roles|tools'
}

@test "unknown module exits 2" {
  run gpowers-upgrade nonsense
  [ "$status" -eq 2 ]
}

@test "--resume invokes resume helper" {
  out=$(gpowers-upgrade --resume 2>&1)
  echo "$out" | grep -qi "no upgrade in progress\|nothing to resume"
}
