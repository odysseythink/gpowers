#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$GPOWERS_HOME/state"
  export PATH="$GPOWERS_REPO/bin:$PATH"
  source "$GPOWERS_REPO/tools/drivers/browser/_shared/tab-registry.sh"
}

@test "tab_alloc returns unique tab_ids" {
  a=$(tab_alloc claude-in-chrome)
  b=$(tab_alloc claude-in-chrome)
  [ "$a" != "$b" ]
}

@test "tab_set then tab_get round-trips data" {
  id=$(tab_alloc playwright-cli)
  tab_set "$id" backend_handle "pw-handle-42"
  [ "$(tab_get "$id" backend_handle)" = "pw-handle-42" ]
}

@test "tab_release removes tab" {
  id=$(tab_alloc claude-in-chrome)
  tab_set "$id" mcp_tab_id "mcp-7"
  tab_release "$id"
  ! tab_get "$id" mcp_tab_id 2>/dev/null
}
