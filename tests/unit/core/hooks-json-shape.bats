#!/usr/bin/env bats

setup() {
  HOOKS_JSON="$BATS_TEST_DIRNAME/../../../core/hooks/hooks.json"
}

@test "hooks.json is valid JSON" {
  jq empty < "$HOOKS_JSON"
}

@test "hooks.json registers SessionStart" {
  jq -e '.hooks[] | select(.event == "SessionStart")' < "$HOOKS_JSON" >/dev/null
}

@test "hooks.json SessionStart command points to session-start" {
  cmd=$(jq -r '.hooks[] | select(.event == "SessionStart") | .command' "$HOOKS_JSON")
  case "$cmd" in *session-start*) :;; *) echo "got: $cmd"; return 1;; esac
}

@test "hooks.json declares Windows variant via run-hook.cmd" {
  jq -e '.hooks[] | select(.event == "SessionStart") | .windows' "$HOOKS_JSON" >/dev/null
}
