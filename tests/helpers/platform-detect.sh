# Source me. Provides: platform_present <name> → exit 0 if present.
platform_present() {
  case "$1" in
    claude-code) command -v claude >/dev/null;;
    codex)       command -v codex >/dev/null;;
    gemini)      command -v gemini >/dev/null;;
    cursor)      command -v cursor-cli >/dev/null || command -v cursor >/dev/null;;
    opencode)    command -v opencode >/dev/null;;
    copilot)     command -v gh >/dev/null && gh copilot --version >/dev/null 2>&1;;
    kimi)        command -v kimi >/dev/null;;
    kimi-code)   command -v kimi-code >/dev/null;;
    *) return 1;;
  esac
}

platforms_present() {
  for p in claude-code codex gemini cursor opencode copilot kimi kimi-code; do
    if platform_present "$p"; then echo "$p"; fi
  done
}
