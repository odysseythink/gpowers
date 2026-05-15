#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  TEMPLATE="$REPO/templates/review-alias-command.md"
}

@test "alias template exists" { [ -f "$TEMPLATE" ]; }

@test "alias template prints deprecation banner" {
  grep -qi "deprecated\|renamed\|/pr-review" "$TEMPLATE"
}

@test "alias template references the 6-month window" {
  grep -qi "6 month\|six month\|6-month" "$TEMPLATE"
}

@test "every platform commands/ dir has a review.md after install" {
  for p in claude-code codex gemini cursor opencode copilot; do
    [ -f "$REPO/platforms/$p/commands/review.md" ] || { echo "missing review.md for $p"; return 1; }
  done
}
