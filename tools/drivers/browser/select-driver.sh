# Source me. Exports GPOWERS_BROWSER_DRIVER. Honors pre-set value.
# Detection order:
#   1. $GPOWERS_BROWSER_DRIVER pre-set → use as-is
#   2. $GPOWERS_PLATFORM = claude-code  → claude-in-chrome
#   3. playwright CLI on PATH         → playwright-cli
#   4. otherwise → missing (with install hint to stderr)

if [ -n "${GPOWERS_BROWSER_DRIVER:-}" ]; then
  return 0 2>/dev/null || exit 0
fi

case "${GPOWERS_PLATFORM:-}" in
  claude-code)
    export GPOWERS_BROWSER_DRIVER=claude-in-chrome
    ;;
  *)
    if (command -v playwright >/dev/null 2>&1) \
        || (command -v npx >/dev/null 2>&1 && npx --no-install playwright --version >/dev/null 2>&1); then
      export GPOWERS_BROWSER_DRIVER=playwright-cli
    else
      export GPOWERS_BROWSER_DRIVER=missing
      echo "gpowers: no browser driver available. Install: bun add -g @playwright/test  (or use Claude Code with claude-in-chrome MCP)" >&2
    fi
    ;;
esac
