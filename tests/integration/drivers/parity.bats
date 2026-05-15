#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
  export GPOWERS_BROWSER_MOCK=1
}

run_verb() {
  local driver="$1" verb="$2" args="$3"
  GPOWERS_BROWSER_DRIVER="$driver" bash -c "echo '$args' | gpowers-browser '$verb'"
}

@test "open returns tab_id from both drivers" {
  cic=$(run_verb claude-in-chrome open '{"url":"http://x"}')
  pw=$(run_verb playwright-cli  open '{"url":"http://x"}')
  echo "$cic" | jq -e '.tab_id' >/dev/null
  echo "$pw"  | jq -e '.tab_id' >/dev/null
}

@test "click returns {ok:true} from both drivers" {
  cic=$(run_verb claude-in-chrome click '{"tab_id":"t-a","selector":"#x"}')
  pw=$(run_verb  playwright-cli  click '{"tab_id":"t-b","selector":"#x"}')
  [ "$(echo "$cic" | jq -r .ok)" = "true" ]
  [ "$(echo "$pw"  | jq -r .ok)" = "true" ]
}

@test "read returns .content from both drivers" {
  cic=$(run_verb claude-in-chrome read '{"tab_id":"t","mode":"text"}')
  pw=$(run_verb  playwright-cli  read '{"tab_id":"t","mode":"text"}')
  echo "$cic" | jq -e '.content' >/dev/null
  echo "$pw"  | jq -e '.content' >/dev/null
}

@test "screenshot returns .path from both drivers" {
  cic=$(run_verb claude-in-chrome screenshot '{"tab_id":"t"}')
  pw=$(run_verb  playwright-cli  screenshot '{"tab_id":"t"}')
  echo "$cic" | jq -e '.path' >/dev/null
  echo "$pw"  | jq -e '.path' >/dev/null
}

@test "eval returns .value field from both drivers" {
  cic=$(run_verb claude-in-chrome eval '{"tab_id":"t","code":"1+1"}')
  pw=$(run_verb  playwright-cli  eval '{"tab_id":"t","code":"1+1"}')
  echo "$cic" | jq -e 'has("value")' >/dev/null
  echo "$pw"  | jq -e 'has("value")' >/dev/null
}

@test "cookies get returns .cookies array from both drivers" {
  cic=$(run_verb claude-in-chrome cookies '{"tab_id":"t","op":"get"}')
  pw=$(run_verb  playwright-cli  cookies '{"tab_id":"t","op":"get"}')
  echo "$cic" | jq -e '.cookies | type == "array"' >/dev/null
  echo "$pw"  | jq -e '.cookies | type == "array"' >/dev/null
}

@test "close returns {ok:true} from both drivers" {
  cic=$(run_verb claude-in-chrome close '{"tab_id":"t"}')
  pw=$(run_verb  playwright-cli  close '{"tab_id":"t"}')
  [ "$(echo "$cic" | jq -r .ok)" = "true" ]
  [ "$(echo "$pw"  | jq -r .ok)" = "true" ]
}

@test "wait returns .ok field from both drivers" {
  cic=$(run_verb claude-in-chrome wait '{"tab_id":"t","condition":"load"}')
  pw=$(run_verb  playwright-cli  wait '{"tab_id":"t","condition":"load"}')
  echo "$cic" | jq -e 'has("ok")' >/dev/null
  echo "$pw"  | jq -e 'has("ok")' >/dev/null
}

@test "type returns .ok field from both drivers" {
  cic=$(run_verb claude-in-chrome type '{"tab_id":"t","selector":"#x","text":"hi"}')
  pw=$(run_verb  playwright-cli  type '{"tab_id":"t","selector":"#x","text":"hi"}')
  echo "$cic" | jq -e 'has("ok")' >/dev/null
  echo "$pw"  | jq -e 'has("ok")' >/dev/null
}
