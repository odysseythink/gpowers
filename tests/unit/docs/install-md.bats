#!/usr/bin/env bats

setup() { F="$BATS_TEST_DIRNAME/../../../docs/INSTALL.md"; }

@test "INSTALL.md exists" { [ -f "$F" ]; }
@test "names every supported platform" {
  for p in claude-code codex gemini cursor opencode copilot kimi; do
    grep -qw "$p" "$F" || { echo "missing $p"; return 1; }
  done
}
@test "documents --with-business flag" { grep -q "\\--with-business" "$F"; }
@test "documents --core-only flag" { grep -q "\\--core-only" "$F"; }
@test "documents --platforms= flag" { grep -q "\\--platforms=" "$F"; }
