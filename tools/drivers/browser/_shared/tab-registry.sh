# Source me. Provides: tab_alloc, tab_set, tab_get, tab_release.
# State lives under $(gpowers-path state)/browser/tabs/<tab_id>/<key>.

_tab_root() {
  echo "$(gpowers-path state)/browser/tabs"
}

tab_alloc() {
  # tab_alloc <driver-name> → echoes unique tab_id
  local driver="${1:?driver name required}"
  local root; root="$(_tab_root)"
  mkdir -p "$root"
  local seq_file="$root/.seq"
  local n=1
  if [ -f "$seq_file" ]; then n=$(cat "$seq_file"); n=$((n + 1)); fi
  printf '%s\n' "$n" > "$seq_file"
  local id="t-${driver}-${n}"
  mkdir -p "$root/$id"
  printf '%s\n' "$driver" > "$root/$id/.driver"
  echo "$id"
}

tab_set() {
  # tab_set <tab_id> <key> <value>
  local id="${1:?}" key="${2:?}" value="${3:?}"
  local dir; dir="$(_tab_root)/$id"
  [ -d "$dir" ] || { echo "tab_set: unknown tab $id" >&2; return 1; }
  printf '%s\n' "$value" > "$dir/$key"
}

tab_get() {
  local id="${1:?}" key="${2:?}"
  local file; file="$(_tab_root)/$id/$key"
  [ -f "$file" ] || return 1
  cat "$file"
}

tab_release() {
  local id="${1:?}"
  local dir; dir="$(_tab_root)/$id"
  [ -d "$dir" ] && rm -rf "$dir"
}
