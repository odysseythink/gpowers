#!/usr/bin/env bash
# Usage: _gpowers-import-tool.sh <src-skill-dir> <dst-skill-dir> [<upstream-tag>]
# Default upstream-tag = gstack@main.
set -euo pipefail

SRC="${1:?src skill dir required}"
DST="${2:?dst skill dir required}"
TAG="${3:-gstack@main}"

[ -d "$SRC" ] || { echo "src missing: $SRC" >&2; exit 1; }
[ -f "$SRC/SKILL.md" ] || { echo "src missing SKILL.md: $SRC" >&2; exit 1; }
mkdir -p "$DST"

awk -v tag="$TAG" '
  BEGIN { in_fm = 0; fm_done = 0; injected = 0 }
  NR == 1 && /^---$/ { in_fm = 1; print; next }
  in_fm && /^---$/ {
    if (!injected) { print "namespace: tools"; print "upstream: " tag; injected = 1 }
    in_fm = 0; fm_done = 1; print; next
  }
  in_fm { print; next }
  fm_done {
    # Rewrite gstack paths and CLI names in body
    gsub(/~\/\.claude\/skills\/gstack\/bin\//, "$(gpowers-path home)/bin/")
    gsub(/\.claude\/skills\/gstack\/bin\//, "$(gpowers-path home)/bin/")
    gsub(/~\/\.claude\/skills\/gstack\//, "$(gpowers-path home)/")
    gsub(/\.claude\/skills\/gstack\//, "$(gpowers-path home)/")
    # ~/.gstack/ variants: tilde-prefix specific dirs first, then catch-all
    gsub(/~\/\.gstack\/state/, "$(gpowers-path state)")
    gsub(/~\/\.gstack\/config/, "$(gpowers-path config)")
    gsub(/~\/\.gstack\/cache/, "$(gpowers-path cache)")
    gsub(/~\/\.gstack\/data/, "$(gpowers-path data)")
    gsub(/~\/\.gstack\/analytics/, "$(gpowers-path analytics)")
    gsub(/~\/\.gstack\/logs/, "$(gpowers-path logs)")
    gsub(/~\/\.gstack\/tmp/, "$(gpowers-path tmp)")
    gsub(/~\/\.gstack\//, "$(gpowers-path home)/")
    # $HOME/.gstack/ and ${GSTACK_HOME:-...} variants
    gsub(/\$HOME\/\.gstack\//, "$(gpowers-path home)/")
    gsub(/\$\{GSTACK_HOME:-\$HOME\/\.gstack\}/, "$(gpowers-path home)")
    gsub(/GSTACK_HOME/, "GPOWERS_HOME")
    # project-local .gstack/ and .gpowers/ → $(gpowers-path project)/
    # Named subdirs first so catch-all doesn't obscure them
    gsub(/\.gstack\/qa-reports/, "$(gpowers-path project)/evals")
    gsub(/\.gstack\/canary-reports/, "$(gpowers-path project)/canary")
    gsub(/\.gstack\/benchmark-reports/, "$(gpowers-path project)/benchmark")
    gsub(/\.gstack\/security-reports/, "$(gpowers-path project)/security")
    gsub(/\.gstack\/deploy-reports/, "$(gpowers-path project)/canary")
    gsub(/\.gstack\/browse\.json/, "$(gpowers-path project)/browse.json")
    gsub(/\.gstack\/no-test-bootstrap/, "$(gpowers-path project)/no-test-bootstrap")
    gsub(/\$\(git rev-parse --show-toplevel\)\/\.gstack\//, "$(gpowers-path project)/")
    gsub(/\$\(git rev-parse --show-toplevel 2>\/dev\/null\)\/\.gstack\//, "$(gpowers-path project)/")
    gsub(/\.gstack\//, "$(gpowers-path project)/")
    gsub(/gstack-/, "gpowers-")
    print; next
  }
  { print }
' "$SRC/SKILL.md" > "$DST/SKILL.md"

# Copy any additional files unchanged (e.g. references/, examples/)
find "$SRC" -mindepth 1 -maxdepth 1 -not -name SKILL.md -print0 \
  | xargs -0 -I {} cp -R {} "$DST/" 2>/dev/null || true
