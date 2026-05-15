#!/usr/bin/env bash
set -euo pipefail
: "${HOME:?HOME required}"
: "${GPOWERS_HOME:?GPOWERS_HOME required}"
HERE="$(cd "$(dirname "$0")" && pwd)"

source "$HERE/../lib/migration-rules.sh"

mappings='[]'

resolve_project() {
  local slug="$1"
  local repo
  repo=$("$HERE/_gpowers-find-project-by-slug.sh" "$slug")
  if [ -z "$repo" ]; then
    echo "$GPOWERS_HOME/data/legacy-projects/$slug"
  else
    echo "$repo"
  fi
}

for rule in "${GPOWERS_MIGRATION_RULES[@]}"; do
  IFS='|' read -r type src_tmpl dst_tmpl comment <<<"$rule"

  case "$type" in
    file|dir)
      src=$(eval echo "$src_tmpl")
      dst=$(eval echo "$dst_tmpl")
      if [ -e "$src" ]; then
        mappings=$(echo "$mappings" | jq --arg s "$src" --arg d "$dst" --arg c "$comment" \
                   '. += [{type:"file_or_dir", src:$s, dst:$d, comment:$c}]')
      fi
      ;;
    project-glob)
      src_pattern=$(echo "$src_tmpl" | sed 's|<slug>|*|')
      while IFS= read -r match; do
        [ -e "$match" ] || continue
        slug=$(echo "$match" | sed "s|$HOME/.gstack/projects/||; s|/.*||")
        repo=$(resolve_project "$slug")
        dst_tail=$(echo "$dst_tmpl" | sed 's|^<project_repo>|REPO|')
        dst=${dst_tail/REPO/$repo}
        # Substitute ${slug} if used in dst_tmpl (legacy-projects fallback case)
        dst=${dst//\$\{slug\}/$slug}
        # If repo is the legacy fallback, prepend $GPOWERS_HOME path is already absolute
        mappings=$(echo "$mappings" | jq --arg s "$match" --arg d "$dst" --arg c "$comment" --arg slug "$slug" \
                   '. += [{type:"project", src:$s, dst:$d, comment:$c, slug:$slug}]')
      done < <(eval echo "$src_pattern" 2>/dev/null)
      ;;
    worktree-glob)
      src_pattern=$(echo "$src_tmpl" | sed 's|<project>|*|; s|<branch>|*|')
      while IFS= read -r match; do
        [ -e "$match" ] || continue
        rel=${match#"$HOME/.config/superpowers/worktrees/"}
        proj=$(echo "$rel" | cut -d/ -f1)
        branch=$(echo "$rel" | cut -d/ -f2)
        dst="$GPOWERS_HOME/state/worktrees/$proj/$branch"
        mappings=$(echo "$mappings" | jq --arg s "$match" --arg d "$dst" --arg c "$comment" \
                   '. += [{type:"worktree", src:$s, dst:$d, comment:$c}]')
      done < <(eval echo "$src_pattern" 2>/dev/null)
      ;;
  esac
done

# Compute conflicts: dst already exists
conflicts=$(echo "$mappings" | jq '[.[] | select(.dst as $d | (env.HOME // "") + "/" + $d | test("^/")) ] | length')

echo "$mappings" | jq --argjson cnt "$conflicts" \
  '{mappings: ., total: length, will_create_dirs: (map(.dst) | map(split("/")[:-1] | join("/")) | unique | length)}'
