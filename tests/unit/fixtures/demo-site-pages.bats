#!/usr/bin/env bats

setup() { D="$BATS_TEST_DIRNAME/../../../tests/fixtures/demo-site"; }

@test "qa-form.html exists and contains the form" {
  [ -f "$D/qa-form.html" ]
  grep -q "id=\"form\"" "$D/qa-form.html"
  grep -q "id=\"submit\"" "$D/qa-form.html"
}

@test "canary-version.html exposes window.__version" {
  grep -q "window.__version" "$D/canary-version.html"
}

@test "benchmark.html measures something" {
  grep -q "performance.now" "$D/benchmark.html"
}
