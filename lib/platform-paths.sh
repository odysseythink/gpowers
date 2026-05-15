#!/usr/bin/env bash
# lib/platform-paths.sh — Per-platform plugin/skill directory lookup.
# Sourced by install/uninstall. Idempotent.

# Cross-platform home directory resolution.
# On Windows (Git Bash, MSYS, Cygwin), $HOME may point to a POSIX home
# (e.g. /d/mb/home) while Python/Node/Electron apps use the Windows
# user profile (e.g. C:\Users\xxx via USERPROFILE). We prefer
# USERPROFILE when available so platform plugin paths match what the
# actual tools search.
if [ -n "${USERPROFILE:-}" ] && [ -d "${USERPROFILE:-}" ]; then
  _gp_user_home="${USERPROFILE//\\//}"
else
  _gp_user_home="$HOME"
fi

GPOWERS_PLATFORM_PLUGIN_DIR_claude_code="${_gp_user_home}/.claude/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_codex="${_gp_user_home}/.codex/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_gemini="${_gp_user_home}/.config/gemini/extensions/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_cursor="${_gp_user_home}/.cursor/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_opencode="${_gp_user_home}/.config/opencode/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_copilot="${_gp_user_home}/.config/copilot-cli/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_kimi="${_gp_user_home}/.kimi/skills"  # uses prefix mode, not symlink

# Marker directories used to detect "platform is installed on this machine".
GPOWERS_PLATFORM_MARKER_claude_code="${_gp_user_home}/.claude"
GPOWERS_PLATFORM_MARKER_codex="${_gp_user_home}/.codex"
GPOWERS_PLATFORM_MARKER_gemini="${_gp_user_home}/.config/gemini"
GPOWERS_PLATFORM_MARKER_cursor="${_gp_user_home}/.cursor"
GPOWERS_PLATFORM_MARKER_opencode="${_gp_user_home}/.config/opencode"
GPOWERS_PLATFORM_MARKER_copilot="${_gp_user_home}/.config/copilot-cli"
GPOWERS_PLATFORM_MARKER_kimi="${_gp_user_home}/.kimi"

GPOWERS_ALL_PLATFORMS="claude_code codex gemini cursor opencode copilot kimi"

export GPOWERS_ALL_PLATFORMS \
  GPOWERS_PLATFORM_PLUGIN_DIR_claude_code GPOWERS_PLATFORM_PLUGIN_DIR_codex \
  GPOWERS_PLATFORM_PLUGIN_DIR_gemini GPOWERS_PLATFORM_PLUGIN_DIR_cursor \
  GPOWERS_PLATFORM_PLUGIN_DIR_opencode GPOWERS_PLATFORM_PLUGIN_DIR_copilot \
  GPOWERS_PLATFORM_PLUGIN_DIR_kimi \
  GPOWERS_PLATFORM_MARKER_claude_code GPOWERS_PLATFORM_MARKER_codex \
  GPOWERS_PLATFORM_MARKER_gemini GPOWERS_PLATFORM_MARKER_cursor \
  GPOWERS_PLATFORM_MARKER_opencode GPOWERS_PLATFORM_MARKER_copilot \
  GPOWERS_PLATFORM_MARKER_kimi
