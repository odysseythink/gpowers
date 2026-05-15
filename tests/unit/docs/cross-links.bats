#!/usr/bin/env bats

setup() { REPO="$BATS_TEST_DIRNAME/../../.."; }

check_links_in() {
  local file="$1"
  local fdir; fdir=$(dirname "$file")
  # Extract [name](target) for local targets only (skip http/https/mailto)
  while read -r target; do
    [ -z "$target" ] && continue
    case "$target" in http*|mailto:*) continue;; esac
    # Strip fragment
    bare="${target%%#*}"
    [ -z "$bare" ] && continue
    if [ -e "$fdir/$bare" ] || [ -e "$REPO/$bare" ]; then
      :
    else
      echo "BROKEN: $file → $target"
      return 1
    fi
  done < <(awk 'match($0, /\[[^]]*\]\(([^)]*)\)/, m){print m[1]}' "$file")
}

@test "README.md links all resolve" { check_links_in "$REPO/README.md"; }
@test "docs/ARCHITECTURE.md links all resolve" { check_links_in "$REPO/docs/ARCHITECTURE.md"; }
@test "docs/INSTALL.md links all resolve" { check_links_in "$REPO/docs/INSTALL.md"; }
@test "docs/PLATFORMS.md links all resolve" { check_links_in "$REPO/docs/PLATFORMS.md"; }
@test "docs/SKILLS.md links all resolve" { check_links_in "$REPO/docs/SKILLS.md"; }
@test "docs/COMMANDS.md links all resolve" { check_links_in "$REPO/docs/COMMANDS.md"; }
@test "docs/UPGRADING.md links all resolve" { check_links_in "$REPO/docs/UPGRADING.md"; }
@test "docs/DRIVERS.md links all resolve" { check_links_in "$REPO/docs/DRIVERS.md"; }
@test "docs/RUNTIME_LAYOUT.md links all resolve" { check_links_in "$REPO/docs/RUNTIME_LAYOUT.md"; }
@test "docs/CONTRIBUTING.md links all resolve" { check_links_in "$REPO/docs/CONTRIBUTING.md"; }
