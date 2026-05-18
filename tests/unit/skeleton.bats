#!/usr/bin/env bats

@test "repo has all top-level module placeholders" {
  for dir in core roles tools platforms; do
    [ -e "${BATS_TEST_DIRNAME}/../../${dir}/.placeholder" ]
  done
}

@test "repo has LICENSE, README, manifest, upstream-sources" {
  [ -f "${BATS_TEST_DIRNAME}/../../LICENSE" ]
  [ -f "${BATS_TEST_DIRNAME}/../../README.md" ]
  [ -f "${BATS_TEST_DIRNAME}/../../manifest.json" ]
  [ -f "${BATS_TEST_DIRNAME}/../../upstream-sources.json" ]
}

@test "manifest.json is valid JSON with required fields" {
  result=$(jq -r '.version, .installed_modules | type' "${BATS_TEST_DIRNAME}/../../manifest.json")
  [ "$(echo "$result" | head -1)" != "null" ]
  [ "$(echo "$result" | tail -1)" = "array" ]
}
