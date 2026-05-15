#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/PLATFORMS.md"; }

@test "PLATFORMS.md exists" { [ -f "$F" ]; }

@test "names all 7 platforms" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    grep -qw "$p" "$F" || { echo "missing $p"; return 1; }
  done
}

@test "has at least 4 matrices (table headers)" {
  count=$(grep -cE '^\| [A-Za-z]' "$F")
  [ "$count" -ge 4 ]
}

@test "calls out kimi's flat-prefix namespace" {
  grep -qi "flat.prefix\|gpowers-\*\|gpowers- prefix" "$F"
}
