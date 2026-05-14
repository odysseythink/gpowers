#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  REWRITER="$GPOWERS_REPO/bin/_gpowers-rewrite-browser.py"
  FIX="$GPOWERS_REPO/tests/fixtures/tools-browser/rewriter-snippets"
}

@test "rewriter exists and is executable" {
  [ -x "$REWRITER" ]
}

@test "case 01: tabs_create_mcp + URL → browser.open" {
  out=$("$REWRITER" < "$FIX/input/01-tabs_create.md")
  diff <(echo "$out") "$FIX/expected/01-tabs_create.md"
}

@test "case 02: navigate + read_page → open + read" {
  out=$("$REWRITER" < "$FIX/input/02-navigate.md")
  diff <(echo "$out") "$FIX/expected/02-navigate.md"
}

@test "case 03: form_input → type" {
  out=$("$REWRITER" < "$FIX/input/03-form_input.md")
  diff <(echo "$out") "$FIX/expected/03-form_input.md"
}

@test "case 04: find + click → wait + click" {
  out=$("$REWRITER" < "$FIX/input/04-find-click.md")
  diff <(echo "$out") "$FIX/expected/04-find-click.md"
}

@test "case 05: computer screenshot → screenshot" {
  out=$("$REWRITER" < "$FIX/input/05-screenshot.md")
  diff <(echo "$out") "$FIX/expected/05-screenshot.md"
}

@test "case 06: javascript_tool → eval" {
  out=$("$REWRITER" < "$FIX/input/06-eval.md")
  diff <(echo "$out") "$FIX/expected/06-eval.md"
}

@test "case 07: read_console_messages → read mode console" {
  out=$("$REWRITER" < "$FIX/input/07-console.md")
  diff <(echo "$out") "$FIX/expected/07-console.md"
}

@test "case 08: tabs_close_mcp → close" {
  out=$("$REWRITER" < "$FIX/input/08-close.md")
  diff <(echo "$out") "$FIX/expected/08-close.md"
}

@test "case 09: playwright CLI line → abstract driver reference" {
  out=$("$REWRITER" < "$FIX/input/09-playwright-fallback.md")
  diff <(echo "$out") "$FIX/expected/09-playwright-fallback.md"
}
