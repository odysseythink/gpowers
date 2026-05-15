#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  ADAPTERS="$REPO/platforms/kimi/adapters"
  MANIFEST="$REPO/platforms/kimi/kimi-skills.json"
}

@test "every dir is exactly 'gpowers' or 'gpowers-<x>'" {
  for d in "$ADAPTERS"/*/; do
    case "$(basename "$d")" in
      gpowers|gpowers-?*) ;;
      *) echo "bad name: $(basename "$d")"; return 1;;
    esac
  done
}

@test "manifest adapters list matches directory listing" {
  manifest_names=$(jq -r '.adapters[]' < "$MANIFEST" | sort)
  dir_names=$(find "$ADAPTERS" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
  [ "$manifest_names" = "$dir_names" ]
}

@test "no double-prefix gpowers-gpowers-*" {
  ! find "$ADAPTERS" -mindepth 1 -maxdepth 1 -type d -name 'gpowers-gpowers-*' | grep .
}
