#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$REPO"
  ADAPTERS="$REPO/platforms/kimi/adapters"
}

@test "every adapter starts with gpowers-" {
  for d in "$ADAPTERS"/*/; do
    name=$(basename "$d")
    case "$name" in
      gpowers|gpowers-*) ;;
      *) echo "non-prefixed adapter: $name"; return 1;;
    esac
  done
}

@test "every adapter has frontmatter naming gpowers-source" {
  for d in "$ADAPTERS"/*/; do
    [ "$(basename "$d")" = "gpowers" ] && continue
    grep -q "^gpowers-source:" "$d/SKILL.md" || { echo "missing source: $(basename "$d")"; return 1; }
  done
}

@test "every adapter inlines using-gpowers preamble (four-module model)" {
  for d in "$ADAPTERS"/*/; do
    [ "$(basename "$d")" = "gpowers" ] && continue
    body=$(awk 'BEGIN{fm=0} /^---$/{fm++; next} fm>=2{print}' "$d/SKILL.md")
    echo "$body" | grep -qF "four-module" || { echo "$(basename "$d") preamble missing four-module"; return 1; }
  done
}

@test "router adapter 'gpowers' exists" {
  [ -d "$ADAPTERS/gpowers" ]
  [ -f "$ADAPTERS/gpowers/SKILL.md" ]
}
