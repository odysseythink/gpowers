#!/usr/bin/env bats

setup() {
  MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"
}

@test "manifest declares drivers section" {
  jq -e '.drivers.browser' < "$MANIFEST" >/dev/null
}

@test "manifest lists both browser drivers" {
  jq -e '.drivers.browser.available | index("claude-in-chrome")' < "$MANIFEST" >/dev/null
  jq -e '.drivers.browser.available | index("playwright-cli")' < "$MANIFEST" >/dev/null
}

@test "manifest names interface_version 1" {
  v=$(jq -r '.drivers.browser.interface_version' < "$MANIFEST")
  [ "$v" = "1" ]
}
