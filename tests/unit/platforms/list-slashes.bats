#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  HELPER="$REPO/bin/_gpowers-list-slashes.sh"
}

@test "helper exists and is executable" { [ -x "$HELPER" ]; }

@test "lists at least 4 columns per row" {
  out=$("$HELPER" | head -1)
  fields=$(echo "$out" | awk -F'\t' '{print NF}')
  [ "$fields" -ge 4 ]
}

@test "output has rows for tools, roles, business skills" {
  out=$("$HELPER")
  echo "$out" | awk -F'\t' '{print $2}' | grep -q "^tools$"
  echo "$out" | awk -F'\t' '{print $2}' | grep -q "^roles$"
  echo "$out" | awk -F'\t' '{print $2}' | grep -q "^business$"
}

@test "design-review row has requires_driver = browser" {
  out=$("$HELPER")
  echo "$out" | awk -F'\t' '$1=="/design-review" {print $4}' | grep -q "^browser$"
}
