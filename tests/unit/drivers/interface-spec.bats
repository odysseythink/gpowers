#!/usr/bin/env bats

setup() {
  SPEC="$BATS_TEST_DIRNAME/../../../tools/drivers/browser/interface.md"
}

@test "interface.md exists" {
  [ -f "$SPEC" ]
}

@test "interface.md defines all 9 verbs" {
  for verb in open click type read screenshot wait eval cookies close; do
    grep -q "^### browser\\.$verb$" "$SPEC" || {
      echo "verb not defined: $verb"; return 1
    }
  done
}

@test "interface.md defines a JSON envelope" {
  grep -qi "stdin.*json\|json.*stdin" "$SPEC"
}

@test "interface.md defines tab_id semantics" {
  grep -qi "tab_id" "$SPEC"
}
