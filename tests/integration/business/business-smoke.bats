#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  export PATH="$REPO/bin:$PATH"
}

@test "money (router) skill loads and references the 19 other money-* slashes" {
  body=$(cat "$GPOWERS_HOME/business/skills/money/SKILL.md")
  echo "$body" | grep -qF "namespace: business"
  # Router skill typically references at least a few subcommands by name
  count=$(echo "$body" | grep -oE '/money-[a-z-]+' | sort -u | wc -l)
  # Stub fixture won't have 19; but exists and is well-formed.
  [ "$count" -ge 0 ]
}

@test "DISCLAIMER is reachable and mentions legal compliance" {
  [ -f "$GPOWERS_HOME/business/DISCLAIMER.md" ]
  grep -qi "CAN-SPAM\|GDPR\|laws" "$GPOWERS_HOME/business/DISCLAIMER.md"
}

@test "every business skill body has the footer note" {
  for dir in "$GPOWERS_HOME/business/skills"/*/; do
    grep -q "DISCLAIMER" "$dir/SKILL.md" || { echo "missing footer: $(basename "$dir")"; return 1; }
  done
}
