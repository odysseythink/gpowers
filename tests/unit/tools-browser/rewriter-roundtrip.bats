#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  REWRITER="$GPOWERS_REPO/bin/_gpowers-rewrite-browser.py"
  FIX_BASE="$GPOWERS_REPO/tests/fixtures/tools-browser/fake-gstack-checkout/skills"
}

@test "every stub produces zero mcp__ refs after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    if echo "$out" | grep -q "mcp__claude-in-chrome"; then
      echo "$name retained mcp__claude-in-chrome ref after rewrite"
      return 1
    fi
  done
}

@test "every stub produces zero literal playwright command refs after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    if echo "$out" | grep -qE '`(npx )?playwright[^`]*`'; then
      echo "$name retained playwright CLI ref after rewrite"
      return 1
    fi
  done
}

@test "every stub gains at least one gpowers-browser ref after rewrite" {
  for dir in "$FIX_BASE"/*/; do
    name=$(basename "$dir")
    out=$("$REWRITER" < "$dir/SKILL.md")
    echo "$out" | grep -q "gpowers-browser" || {
      echo "$name has no gpowers-browser ref after rewrite"; return 1
    }
  done
}
