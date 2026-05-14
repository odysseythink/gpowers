#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
  export GPOWERS_BROWSER_DRIVER=claude-in-chrome
  export GPOWERS_BROWSER_MOCK=1
}

@test "gpowers-browser open returns tab_id" {
  result=$(echo '{"url":"https://example.com"}' | gpowers-browser open)
  echo "$result" | jq -e '.tab_id' >/dev/null
}

@test "gpowers-browser unknown verb errors clearly" {
  run bash -c 'echo "{}" | gpowers-browser bogus 2>&1'
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "unknown verb"
}

@test "gpowers-browser missing driver errors with hint" {
  unset GPOWERS_BROWSER_DRIVER
  export GPOWERS_BROWSER_DRIVER=missing
  run bash -c 'echo "{}" | gpowers-browser open 2>&1'
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "install"
}
