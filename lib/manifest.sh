#!/usr/bin/env bash
# lib/manifest.sh — Read and write manifest.json.

gpowers_manifest_set_installed() {
  local manifest_path="$1"
  local location="$2"
  shift 2
  local modules=("$@")
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local modules_json
  modules_json="$(printf '%s\n' "${modules[@]}" | jq -R . | jq -s .)"
  local tmp
  tmp="$(mktemp)"
  jq --arg loc "$location" \
     --arg ts "$now" \
     --argjson mods "$modules_json" \
     '.installed_at = $ts | .install_location = $loc | .installed_modules = $mods' \
     "$manifest_path" > "$tmp"
  mv "$tmp" "$manifest_path"
}
