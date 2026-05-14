#!/usr/bin/env bash
# lib/platform-paths.sh — Per-platform plugin/skill directory lookup.
# Sourced by install/uninstall. Idempotent.

GPOWERS_PLATFORM_PLUGIN_DIR_claude_code="${HOME}/.claude/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_codex="${HOME}/.codex/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_gemini="${HOME}/.config/gemini/extensions/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_cursor="${HOME}/.cursor/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_opencode="${HOME}/.config/opencode/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_copilot="${HOME}/.config/copilot-cli/plugins/gpowers"
GPOWERS_PLATFORM_PLUGIN_DIR_kimi="${HOME}/.kimi/skills"  # uses prefix mode, not symlink

# Marker directories used to detect "platform is installed on this machine".
GPOWERS_PLATFORM_MARKER_claude_code="${HOME}/.claude"
GPOWERS_PLATFORM_MARKER_codex="${HOME}/.codex"
GPOWERS_PLATFORM_MARKER_gemini="${HOME}/.config/gemini"
GPOWERS_PLATFORM_MARKER_cursor="${HOME}/.cursor"
GPOWERS_PLATFORM_MARKER_opencode="${HOME}/.config/opencode"
GPOWERS_PLATFORM_MARKER_copilot="${HOME}/.config/copilot-cli"
GPOWERS_PLATFORM_MARKER_kimi="${HOME}/.kimi"

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
