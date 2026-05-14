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
