#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  SCRIPT="$GPOWERS_REPO/tools/drivers/browser/select-driver.sh"
  export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
  mkdir -p "$BATS_TEST_TMPDIR/bin"
}

teardown() {
  unset GPOWERS_BROWSER_DRIVER GPOWERS_PLATFORM
}

@test "prefers claude-in-chrome when GPOWERS_PLATFORM=claude-code" {
  export GPOWERS_PLATFORM=claude-code
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "claude-in-chrome" ]
}

@test "falls back to playwright-cli when claude-code unavailable + playwright present" {
  export GPOWERS_PLATFORM=codex
  cat > "$BATS_TEST_TMPDIR/bin/playwright" <<'F'
#!/bin/sh
echo "Version 1.0"
F
  chmod +x "$BATS_TEST_TMPDIR/bin/playwright"
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "playwright-cli" ]
}

@test "sets driver to 'missing' with install hint when no backend available" {
  export GPOWERS_PLATFORM=codex
  PATH="/usr/bin:/bin" source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "missing" ]
}

@test "honors pre-set GPOWERS_BROWSER_DRIVER without override" {
  export GPOWERS_PLATFORM=claude-code
  export GPOWERS_BROWSER_DRIVER=playwright-cli
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "playwright-cli" ]
}
