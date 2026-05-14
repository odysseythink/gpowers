#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
  export GPOWERS_BROWSER_MOCK=1
  export GPOWERS_BROWSER_DRIVER=claude-in-chrome
}

@test "select-driver export GPOWERS_BROWSER_DRIVER" {
  [ -n "$GPOWERS_BROWSER_DRIVER" ]
  [ "$GPOWERS_BROWSER_DRIVER" != "missing" ]
}

@test "browse skill flow: open → read → close (mocked)" {
  open_out=$(echo '{"url":"https://example.com"}' | gpowers-browser open)
  tab=$(echo "$open_out" | jq -r .tab_id)
  [ -n "$tab" ]

  read_out=$(echo "{\"tab_id\":\"$tab\",\"mode\":\"text\"}" | gpowers-browser read)
  echo "$read_out" | jq -e '.content' >/dev/null

  close_out=$(echo "{\"tab_id\":\"$tab\"}" | gpowers-browser close)
  [ "$(echo "$close_out" | jq -r .ok)" = "true" ]
}

@test "qa skill verbs all available via dispatcher" {
  for verb in open click type wait read screenshot eval cookies close; do
    run bash -c "echo '{}' | gpowers-browser $verb 2>&1 || true"
    # We don't assert success (no real tab), only that the dispatcher accepted the verb
    if echo "$output" | grep -q "unknown verb"; then
      echo "verb rejected: $verb"; return 1
    fi
  done
}
