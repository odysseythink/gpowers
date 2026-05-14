#!/usr/bin/env bash
# lib/runtime-dirs.sh — Defines GPOWERS_* runtime directory env vars.
# Sourced by every CLI and skill. Idempotent. Does not change cwd.

: "${GPOWERS_HOME:=${HOME}/.gpowers}"
: "${GPOWERS_CONFIG:=${GPOWERS_HOME}/config}"
: "${GPOWERS_STATE:=${GPOWERS_HOME}/state}"
: "${GPOWERS_CACHE:=${GPOWERS_HOME}/cache}"
: "${GPOWERS_DATA:=${GPOWERS_HOME}/data}"
: "${GPOWERS_ANALYTICS:=${GPOWERS_HOME}/analytics}"
: "${GPOWERS_TMP:=${GPOWERS_HOME}/tmp}"

export GPOWERS_HOME GPOWERS_CONFIG GPOWERS_STATE GPOWERS_CACHE
export GPOWERS_DATA GPOWERS_ANALYTICS GPOWERS_TMP

# Project root detection.
# Priority: $GPOWERS_PROJECT_DIR env, then walk up cwd for .gpowers, then .git, else empty.
gpowers_detect_project_dir() {
  if [ -n "${GPOWERS_PROJECT_DIR:-}" ]; then
    printf '%s\n' "$GPOWERS_PROJECT_DIR"
    return 0
  fi
  local dir="$PWD"
  # Phase 1: look for .gpowers/
  while [ "$dir" != "/" ] && [ -n "$dir" ]; do
    if [ -d "$dir/.gpowers" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  # Phase 2: look for .git/
  dir="$PWD"
  while [ "$dir" != "/" ] && [ -n "$dir" ]; do
    if [ -e "$dir/.git" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

if _detected="$(gpowers_detect_project_dir)"; then
  GPOWERS_PROJECT_DIR="$_detected"
  GPOWERS_PROJECT_DATA="${GPOWERS_PROJECT_DATA:-${GPOWERS_PROJECT_DIR}/.gpowers}"
  export GPOWERS_PROJECT_DIR GPOWERS_PROJECT_DATA
fi
unset _detected
