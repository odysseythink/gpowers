#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "docgen skills emits a markdown table" {
  out=$(_gpowers-docgen.sh skills)
  echo "$out" | head -1 | grep -q "^| Module |"
}

@test "docgen commands emits slashes with backticks" {
  out=$(_gpowers-docgen.sh commands)
  echo "$out" | grep -q '`/'
}

@test "docgen platforms emits 7 rows + header + separator" {
  out=$(_gpowers-docgen.sh platforms)
  data=$(echo "$out" | tail -n +3 | wc -l)
  [ "$data" -eq 7 ]
}

@test "docgen unknown kind exits 2" {
  run _gpowers-docgen.sh bogus
  [ "$status" -eq 2 ]
}
