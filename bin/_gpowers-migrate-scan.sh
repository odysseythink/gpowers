#!/usr/bin/env bash
# Emits a JSON scan report to stdout. Reads $HOME.
set -euo pipefail
: "${HOME:?HOME required}"

GSTACK_ROOT="$HOME/.gstack"
SP_ROOT="$HOME/.config/superpowers"

gstack_present=false
[ -d "$GSTACK_ROOT" ] && gstack_present=true

sp_present=false
[ -d "$SP_ROOT" ] && sp_present=true

# gstack details
g_profiles=()
[ -f "$GSTACK_ROOT/builder-profile" ] && g_profiles+=("builder-profile")
[ -f "$GSTACK_ROOT/developer-profile" ] && g_profiles+=("developer-profile")
[ -f "$GSTACK_ROOT/gbrain-repo-policy" ] && g_profiles+=("gbrain-repo-policy")

g_projects=()
if [ -d "$GSTACK_ROOT/projects" ]; then
  while IFS= read -r dir; do
    g_projects+=("$(basename "$dir")")
  done < <(find "$GSTACK_ROOT/projects" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
fi

g_has_compact=false
[ -f "$HOME/.config/gstack/compact.toml" ] && g_has_compact=true

# superpowers details
sp_worktrees=()
if [ -d "$SP_ROOT/worktrees" ]; then
  while IFS= read -r dir; do
    # Format: <project>/<branch>
    rel="${dir#"$SP_ROOT/worktrees/"}"
    sp_worktrees+=("$rel")
  done < <(find "$SP_ROOT/worktrees" -mindepth 2 -maxdepth 2 -type d 2>/dev/null)
fi

# Emit JSON
jq -n \
  --argjson gp "$gstack_present" \
  --argjson sp "$sp_present" \
  --argjson compact "$g_has_compact" \
  --arg gstack_root "$GSTACK_ROOT" \
  --arg sp_root "$SP_ROOT" \
  --argjson profiles "$(printf '%s\n' "${g_profiles[@]+"${g_profiles[@]}"}" | jq -R . | jq -s .)" \
  --argjson projects "$(printf '%s\n' "${g_projects[@]+"${g_projects[@]}"}" | jq -R . | jq -s .)" \
  --argjson worktrees "$(printf '%s\n' "${sp_worktrees[@]+"${sp_worktrees[@]}"}" | jq -R . | jq -s .)" \
  '{
     gstack: {
       present: $gp, root: $gstack_root,
       profiles: $profiles, projects: $projects, has_compact: $compact
     },
     superpowers: {
       present: $sp, root: $sp_root, worktrees: $worktrees
     }
   }'
