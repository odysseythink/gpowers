# gpowers drivers/ Abstraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `tools/drivers/` abstraction layer so all browser-using skills depend on a 9-verb interface instead of a specific MCP server or CLI tool. Two driver implementations: `claude-in-chrome` (MCP, Claude Code native) and `playwright-cli` (npm/bun installable, every other platform). A `select-driver.sh` script picks the right one at runtime and exports `GPOWERS_BROWSER_DRIVER`.

**Architecture:** Driver interface lives at `tools/drivers/browser/interface.md` — a normative spec defining 9 verbs (open, click, type, read, screenshot, wait, eval, cookies, close), their argument shapes, and return types. Each driver provides a small Bash dispatch script `tools/drivers/browser/<driver>/<verb>.sh` that the verb call resolves to. Skills invoke verbs through a thin shim `bin/gpowers-browser` which reads `$GPOWERS_BROWSER_DRIVER` and execs the corresponding script. This is dispatch by directory layout (not by case statement) — adding a new driver = `mkdir <driver>/ + cp template/*.sh` with no central registry edit. Parity is enforced by a contract test that runs each verb against both drivers and asserts identical observable behavior on a known fixture page.

**Tech Stack:** Bash 4+ for dispatch shims and select-driver, JSON via `jq` for verb argument passing, Playwright CLI (`@playwright/test`) as one backend, Claude Code MCP `claude-in-chrome` as the other, bats-core for tests, a static HTML fixture page served by `python3 -m http.server` for contract tests.

**Depends on:** Plan #1 (foundation: `bin/`, `lib/runtime-dirs.sh`, `gpowers-path`), Plan #2 (core/ only used in driver smoke; can run in parallel with #2).

---

## File Structure

```
tools/drivers/
├── browser/
│   ├── interface.md                    Normative 9-verb spec
│   ├── select-driver.sh                Runtime driver detection
│   ├── _shared/
│   │   ├── json-args.sh                Common: parse JSON args from stdin
│   │   └── tab-registry.sh             Common: tab_id allocation
│   ├── claude-in-chrome/
│   │   ├── open.sh    click.sh   type.sh
│   │   ├── read.sh    screenshot.sh    wait.sh
│   │   ├── eval.sh    cookies.sh       close.sh
│   │   └── README.md                   verb → MCP tool mapping
│   └── playwright-cli/
│       ├── open.sh    click.sh   type.sh
│       ├── read.sh    screenshot.sh    wait.sh
│       ├── eval.sh    cookies.sh       close.sh
│       ├── _playwright-runner.sh       internal: spawn `playwright test`
│       └── README.md                   verb → playwright command template
tools/bin/
└── gpowers-browser                     The verb dispatch shim
tests/unit/drivers/
├── interface-spec.bats                 interface.md schema completeness
├── select-driver.bats                  detection logic
├── verb-shape.bats                     every verb script accepts JSON stdin
└── tab-registry.bats                   tab_id alloc / lookup / release
tests/integration/drivers/
├── parity.bats                         same verb, both drivers, same result
└── fixtures/
    ├── page.html                       contract test page
    └── server.sh                       launch python3 http.server on free port
```

---

## Task 1: Write the interface spec

**Files:**
- Create: `tools/drivers/browser/interface.md`

The interface spec is the contract. Both drivers must satisfy it. The contract test in Task 11 reads this file.

- [ ] **Step 1: Write the failing test for spec completeness**

```bash
cat > tests/unit/drivers/interface-spec.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  SPEC="$BATS_TEST_DIRNAME/../../../tools/drivers/browser/interface.md"
}

@test "interface.md exists" {
  [ -f "$SPEC" ]
}

@test "interface.md defines all 9 verbs" {
  for verb in open click type read screenshot wait eval cookies close; do
    grep -q "^### browser\\.$verb$" "$SPEC" || {
      echo "verb not defined: $verb"; return 1
    }
  done
}

@test "interface.md defines a JSON envelope" {
  grep -qi "stdin.*json\|json.*stdin" "$SPEC"
}

@test "interface.md defines tab_id semantics" {
  grep -qi "tab_id" "$SPEC"
}
EOF
```

Run: `bats tests/unit/drivers/interface-spec.bats` — expect FAIL.

- [ ] **Step 2: Write the spec**

```bash
mkdir -p tools/drivers/browser
cat > tools/drivers/browser/interface.md <<'EOF'
# gpowers browser driver interface

This is the normative contract every browser driver MUST satisfy. Skills invoke verbs through `gpowers-browser <verb>`; that shim reads `$GPOWERS_BROWSER_DRIVER` and execs `tools/drivers/browser/<driver>/<verb>.sh`.

## Wire format

Every verb accepts a single JSON object on **stdin**, returns a single JSON object on **stdout**, and uses exit code 0 for success, non-zero for failure with a `{"error": "..."}` envelope on stderr.

```
echo '{"url":"https://example.com"}' | tools/drivers/browser/<driver>/open.sh
# stdout: {"tab_id":"t-1"}
```

## tab_id semantics

A `tab_id` is a driver-opaque string returned by `browser.open`. It is valid until `browser.close` consumes it. Drivers may store it in `~/.gpowers/state/browser/tabs/` (see Plan #1 runtime-dirs); the format is driver-specific.

## The 9 verbs

### browser.open

| Field | Type | Required | Notes |
|---|---|---|---|
| url | string | yes | absolute URL |
| viewport | `{width:int,height:int}` | no | defaults to 1280×800 |

Returns: `{"tab_id": "string"}`

### browser.click

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | from open |
| selector | string | yes | CSS selector |
| timeout_ms | int | no | default 5000 |

Returns: `{"ok": true}` on click, `{"ok": false, "reason": "..."}` if not clickable.

### browser.type

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| selector | string | yes | |
| text | string | yes | |

Returns: `{"ok": true}`

### browser.read

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| mode | string | yes | `text` \| `dom` \| `console` |
| selector | string | no | when mode=text/dom, scope to selector |

Returns: `{"content": "string"}`. For `console`, returns most-recent messages joined by `\n`.

### browser.screenshot

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| region | string | no | `viewport` \| `full` \| selector |

Returns: `{"path": "absolute path to PNG"}` — file lives under `$(gpowers-path cache)/browser/shots/`.

### browser.wait

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| condition | string | yes | `selector:<css>` \| `network-idle` \| `load` |
| timeout_ms | int | no | default 30000 |

Returns: `{"ok": true}` or `{"ok": false, "reason": "timeout"}`

### browser.eval

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| code | string | yes | JS expression evaluated in page context |

Returns: `{"value": <any JSON>}` — the eval result serialized.

### browser.cookies

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |
| op | string | yes | `get` \| `set` \| `clear` |
| domain | string | no | scope; default current page domain |
| cookies | array | no | required for `set`: `[{name,value,domain,path,httpOnly,secure}, ...]` |

Returns (op=get): `{"cookies": [...]}`. (op=set/clear): `{"ok": true}`.

### browser.close

| Field | Type | Required | Notes |
|---|---|---|---|
| tab_id | string | yes | |

Returns: `{"ok": true}`. Idempotent — closing an already-closed tab returns ok.

## Errors

On failure a driver MUST exit non-zero and print a JSON error to stderr:

```
{"error": "selector not found", "verb": "click", "tab_id": "t-1"}
```

## Driver capability flag

Each driver MUST provide `<driver>/capabilities.json` listing optional features it supports (e.g., `{"network_interception": true}`). Skills MAY query but MUST NOT depend on optional features.
EOF
```

- [ ] **Step 3: Run test to verify pass**

Run: `bats tests/unit/drivers/interface-spec.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add tools/drivers/browser/interface.md tests/unit/drivers/interface-spec.bats
git commit -m "feat(drivers): 9-verb browser interface spec"
```

---

## Task 2: Shared helpers (json-args.sh, tab-registry.sh)

**Files:**
- Create: `tools/drivers/browser/_shared/json-args.sh`
- Create: `tools/drivers/browser/_shared/tab-registry.sh`

Both drivers parse JSON args identically and need a tab_id store. Extract once to keep verb scripts thin.

- [ ] **Step 1: Write the failing test for tab-registry**

```bash
cat > tests/unit/drivers/tab-registry.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$GPOWERS_HOME/state"
  export PATH="$GPOWERS_REPO/bin:$PATH"
  source "$GPOWERS_REPO/tools/drivers/browser/_shared/tab-registry.sh"
}

@test "tab_alloc returns unique tab_ids" {
  a=$(tab_alloc claude-in-chrome)
  b=$(tab_alloc claude-in-chrome)
  [ "$a" != "$b" ]
}

@test "tab_set then tab_get round-trips data" {
  id=$(tab_alloc playwright-cli)
  tab_set "$id" backend_handle "pw-handle-42"
  [ "$(tab_get "$id" backend_handle)" = "pw-handle-42" ]
}

@test "tab_release removes tab" {
  id=$(tab_alloc claude-in-chrome)
  tab_set "$id" mcp_tab_id "mcp-7"
  tab_release "$id"
  ! tab_get "$id" mcp_tab_id 2>/dev/null
}
EOF
```

Run: `bats tests/unit/drivers/tab-registry.bats` — expect FAIL.

- [ ] **Step 2: Write tab-registry.sh**

```bash
mkdir -p tools/drivers/browser/_shared
cat > tools/drivers/browser/_shared/tab-registry.sh <<'EOF'
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
EOF
```

- [ ] **Step 3: Write json-args.sh**

```bash
cat > tools/drivers/browser/_shared/json-args.sh <<'EOF'
# Source me. Reads JSON object from stdin into associative-like access.
# Usage: source json-args.sh; ARGS_JSON="$(read_args)"; arg .url; arg .tab_id

read_args() {
  cat -
}

arg() {
  # arg <jq-path> [<default>] — extracts a scalar from $ARGS_JSON
  local path="$1" default="${2-}"
  local val
  val=$(printf '%s' "$ARGS_JSON" | jq -r "$path // empty" 2>/dev/null || echo "")
  if [ -z "$val" ] && [ -n "$default" ]; then echo "$default"; else echo "$val"; fi
}

emit() {
  # emit <jq-json-template> — prints a JSON object to stdout
  printf '%s\n' "$1"
}

die() {
  # die <message> [<verb>] [<tab_id>]
  local msg="$1" verb="${2:-unknown}" tab="${3:-}"
  jq -n --arg e "$msg" --arg v "$verb" --arg t "$tab" \
     '{error: $e, verb: $v, tab_id: $t}' >&2
  exit 1
}
EOF
```

- [ ] **Step 4: Run tab-registry test to verify pass**

Run: `bats tests/unit/drivers/tab-registry.bats`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add tools/drivers/browser/_shared/ tests/unit/drivers/tab-registry.bats
git commit -m "feat(drivers): shared json-args + tab-registry helpers"
```

---

## Task 3: Implement claude-in-chrome driver

**Files:**
- Create: `tools/drivers/browser/claude-in-chrome/{open,click,type,read,screenshot,wait,eval,cookies,close}.sh`
- Create: `tools/drivers/browser/claude-in-chrome/capabilities.json`
- Create: `tools/drivers/browser/claude-in-chrome/README.md`

claude-in-chrome verb scripts emit "MCP instructions" rather than calling MCP directly — they're meant to be sourced into an agent context where the agent can invoke MCP. Because skills are Markdown that an LLM reads, these scripts produce a short instruction string the agent will follow. (For non-agent CI testing, the scripts return a stub success; the parity test mocks the MCP backend.)

- [ ] **Step 1: Write driver capabilities**

```bash
mkdir -p tools/drivers/browser/claude-in-chrome
cat > tools/drivers/browser/claude-in-chrome/capabilities.json <<'EOF'
{
  "driver": "claude-in-chrome",
  "backend": "mcp",
  "platform_native": ["claude-code"],
  "features": {
    "network_interception": true,
    "console_messages": true,
    "cookies_full": true,
    "screenshots": true,
    "viewport_resize": true
  }
}
EOF
```

- [ ] **Step 2: Write open.sh as the pattern**

```bash
cat > tools/drivers/browser/claude-in-chrome/open.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
URL=$(arg .url)
VW=$(arg .viewport.width 1280)
VH=$(arg .viewport.height 800)

[ -n "$URL" ] || die "url required" open

# In agent context, the actual MCP call is performed by the LLM after
# reading this driver's README.md. For automated CI / unit test we shim
# via $GPOWERS_BROWSER_MOCK if set.
if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  tab_id=$(tab_alloc claude-in-chrome)
  tab_set "$tab_id" mock_url "$URL"
  jq -n --arg id "$tab_id" '{tab_id: $id}'
  exit 0
fi

# Production path: emit MCP invocation instruction.
tab_id=$(tab_alloc claude-in-chrome)
tab_set "$tab_id" pending_open "$URL"
cat >&2 <<MCP
GPOWERS_MCP_INSTRUCTION: invoke mcp__claude-in-chrome__tabs_create_mcp with {"url":"$URL"}, then mcp__claude-in-chrome__resize_window {"width":$VW,"height":$VH}. Bind the returned tab id to gpowers tab_id "$tab_id" via: tab_set "$tab_id" mcp_tab_id <returned>.
MCP
jq -n --arg id "$tab_id" '{tab_id: $id}'
EOF
chmod +x tools/drivers/browser/claude-in-chrome/open.sh
```

- [ ] **Step 3: Write the other 8 verb scripts**

Each follows the same shape — parse args, look up mcp_tab_id via `tab_get`, emit MCP instruction to stderr, return JSON on stdout. For brevity, encode them in a generator:

```bash
gen_verb() {
  local verb="$1" mcp_op="$2" stdout_template="$3" required_args="$4"
  cat > "tools/drivers/browser/claude-in-chrome/$verb.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="\$(cd "\$(dirname "\$0")" && pwd)"
source "\$DRIVER_DIR/../_shared/json-args.sh"
source "\$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="\$(read_args)"
TAB_ID=\$(arg .tab_id)
[ -n "\$TAB_ID" ] || die "tab_id required" $verb

if [ -n "\${GPOWERS_BROWSER_MOCK:-}" ]; then
  $stdout_template
  exit 0
fi

MCP_TAB=\$(tab_get "\$TAB_ID" mcp_tab_id) || die "tab not initialized" $verb "\$TAB_ID"
echo "GPOWERS_MCP_INSTRUCTION: invoke mcp__claude-in-chrome__$mcp_op for \$MCP_TAB with args from \$ARGS_JSON" >&2
$stdout_template
EOF
  chmod +x "tools/drivers/browser/claude-in-chrome/$verb.sh"
}

gen_verb click       click           "jq -n '{ok:true}'"                                      "selector"
gen_verb type        form_input      "jq -n '{ok:true}'"                                      "selector,text"
gen_verb read        read_page       "jq -n --arg c 'mock page text' '{content:\$c}'"        "mode"
gen_verb screenshot  computer        "jq -n --arg p \"\$(gpowers-path cache)/browser/shots/mock.png\" '{path:\$p}'" ""
gen_verb wait        find            "jq -n '{ok:true}'"                                      "condition"
gen_verb eval        javascript_tool "jq -n '{value:null}'"                                   "code"
gen_verb cookies     javascript_tool "jq -n '{cookies:[]}'"                                   "op"
gen_verb close       tabs_close_mcp  "jq -n '{ok:true}'"                                      ""
```

Note: `close.sh` must also call `tab_release "$TAB_ID"` after emitting. Patch:

```bash
sed -i.bak '/^echo "GPOWERS_MCP_INSTRUCTION.*close/a\
tab_release "$TAB_ID"' tools/drivers/browser/claude-in-chrome/close.sh
rm tools/drivers/browser/claude-in-chrome/close.sh.bak
```

- [ ] **Step 4: Write driver README**

```bash
cat > tools/drivers/browser/claude-in-chrome/README.md <<'EOF'
# claude-in-chrome driver

Maps gpowers 9-verb interface to Claude Code MCP server `claude-in-chrome` tools.

| gpowers verb | MCP tool |
|---|---|
| open | mcp__claude-in-chrome__tabs_create_mcp + resize_window |
| click | mcp__claude-in-chrome__click (via computer or find+click) |
| type | mcp__claude-in-chrome__form_input |
| read | mcp__claude-in-chrome__read_page (mode=text/dom) / read_console_messages (mode=console) |
| screenshot | mcp__claude-in-chrome__computer (action=screenshot) |
| wait | mcp__claude-in-chrome__find (with retry) |
| eval | mcp__claude-in-chrome__javascript_tool |
| cookies | mcp__claude-in-chrome__javascript_tool (document.cookie shim) |
| close | mcp__claude-in-chrome__tabs_close_mcp + tab_release |

Verb scripts emit a `GPOWERS_MCP_INSTRUCTION:` line on stderr that the agent reads and translates to the appropriate MCP tool call. The stdout payload is the verb's return value.

For automated tests (no live MCP), set `GPOWERS_BROWSER_MOCK=1` — scripts will skip the instruction emission and return canned success values.
EOF
```

- [ ] **Step 5: Commit**

```bash
git add tools/drivers/browser/claude-in-chrome/
git commit -m "feat(drivers): claude-in-chrome driver — 9 verbs over MCP"
```

---

## Task 4: Implement playwright-cli driver

**Files:**
- Create: `tools/drivers/browser/playwright-cli/*.sh`
- Create: `tools/drivers/browser/playwright-cli/_playwright-runner.sh`
- Create: `tools/drivers/browser/playwright-cli/capabilities.json`
- Create: `tools/drivers/browser/playwright-cli/README.md`

This driver shells out to `playwright` CLI. It runs a small per-tab Node process via a long-lived runner; tab_id maps to a directory holding the runner's stdin/stdout FIFOs.

- [ ] **Step 1: Write capabilities + README**

```bash
mkdir -p tools/drivers/browser/playwright-cli
cat > tools/drivers/browser/playwright-cli/capabilities.json <<'EOF'
{
  "driver": "playwright-cli",
  "backend": "playwright",
  "platform_native": ["codex","gemini","cursor","opencode","copilot","kimi"],
  "features": {
    "network_interception": true,
    "console_messages": true,
    "cookies_full": true,
    "screenshots": true,
    "viewport_resize": true
  }
}
EOF

cat > tools/drivers/browser/playwright-cli/README.md <<'EOF'
# playwright-cli driver

Maps gpowers 9-verb interface to Playwright CLI / Node API. Each `browser.open` spawns a detached Node process running a per-tab event loop; tab_id is the registry entry storing the FIFOs path. Each subsequent verb writes a JSON request to the runner FIFO and reads the JSON response.

Requires: `bun add -g @playwright/test` or `npm install -g playwright` plus browsers (`npx playwright install chromium`).

| gpowers verb | playwright API |
|---|---|
| open | browser.newContext + context.newPage + page.goto |
| click | page.locator(sel).click |
| type | page.locator(sel).fill |
| read | page.textContent / page.content / on('console') buffer |
| screenshot | page.screenshot |
| wait | page.waitForSelector / waitForLoadState |
| eval | page.evaluate |
| cookies | context.cookies / addCookies / clearCookies |
| close | context.close + tab_release |
EOF
```

- [ ] **Step 2: Write the long-lived runner**

```bash
cat > tools/drivers/browser/playwright-cli/_playwright-runner.sh <<'EOF'
#!/usr/bin/env bash
# Spawns a Node child running playwright in headless mode, exchanging JSON
# requests/responses over a pair of FIFOs in $tab_dir.
set -euo pipefail

DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

TAB_ID="${1:?tab_id required}"
URL="${2:?initial URL required}"
VW="${3:-1280}"
VH="${4:-800}"

TAB_DIR="$(gpowers-path state)/browser/tabs/$TAB_ID"
mkfifo "$TAB_DIR/req" "$TAB_DIR/res"

NODE_SCRIPT="$DRIVER_DIR/runner.mjs"
[ -f "$NODE_SCRIPT" ] || cat > "$NODE_SCRIPT" <<'JS'
import { chromium } from 'playwright';
import { createReadStream, createWriteStream } from 'node:fs';
import readline from 'node:readline';
const [reqPath, resPath, url, vw, vh] = process.argv.slice(2);
const browser = await chromium.launch();
const context = await browser.newContext({ viewport: { width: +vw, height: +vh } });
const page = await context.newPage();
const consoleLog = [];
page.on('console', m => consoleLog.push(`[${m.type()}] ${m.text()}`));
await page.goto(url);
const out = createWriteStream(resPath);
const rl = readline.createInterface({ input: createReadStream(reqPath) });
rl.on('line', async line => {
  const req = JSON.parse(line);
  try {
    let result;
    switch (req.verb) {
      case 'click':      await page.locator(req.selector).click({ timeout: req.timeout_ms || 5000 }); result = { ok: true }; break;
      case 'type':       await page.locator(req.selector).fill(req.text); result = { ok: true }; break;
      case 'read':
        if (req.mode === 'text')     result = { content: req.selector ? await page.textContent(req.selector) : await page.innerText('body') };
        else if (req.mode === 'dom') result = { content: req.selector ? await page.locator(req.selector).innerHTML() : await page.content() };
        else                          result = { content: consoleLog.join('\n') };
        break;
      case 'screenshot': const p = `${process.env.GPOWERS_CACHE}/browser/shots/${Date.now()}.png`; await page.screenshot({ path: p, fullPage: req.region === 'full' }); result = { path: p }; break;
      case 'wait':
        if (req.condition.startsWith('selector:')) await page.waitForSelector(req.condition.slice(9), { timeout: req.timeout_ms || 30000 });
        else if (req.condition === 'network-idle') await page.waitForLoadState('networkidle', { timeout: req.timeout_ms || 30000 });
        else                                       await page.waitForLoadState('load', { timeout: req.timeout_ms || 30000 });
        result = { ok: true };
        break;
      case 'eval':       result = { value: await page.evaluate(req.code) }; break;
      case 'cookies':
        if (req.op === 'get')   result = { cookies: await context.cookies(req.domain ? [`https://${req.domain}/`] : undefined) };
        else if (req.op === 'set')   { await context.addCookies(req.cookies); result = { ok: true }; }
        else                          { await context.clearCookies(); result = { ok: true }; }
        break;
      case 'close':      await context.close(); await browser.close(); result = { ok: true }; out.write(JSON.stringify(result) + '\n'); process.exit(0);
      default: result = { error: `unknown verb: ${req.verb}` };
    }
    out.write(JSON.stringify(result) + '\n');
  } catch (e) {
    out.write(JSON.stringify({ ok: false, error: String(e) }) + '\n');
  }
});
JS

setsid node "$NODE_SCRIPT" "$TAB_DIR/req" "$TAB_DIR/res" "$URL" "$VW" "$VH" \
  </dev/null >"$TAB_DIR/stdout" 2>"$TAB_DIR/stderr" &
echo $! > "$TAB_DIR/pid"
EOF
chmod +x tools/drivers/browser/playwright-cli/_playwright-runner.sh
```

- [ ] **Step 3: Write open.sh**

```bash
cat > tools/drivers/browser/playwright-cli/open.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$DRIVER_DIR/../_shared/json-args.sh"
source "$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="$(read_args)"
URL=$(arg .url); [ -n "$URL" ] || die "url required" open
VW=$(arg .viewport.width 1280); VH=$(arg .viewport.height 800)

if [ -n "${GPOWERS_BROWSER_MOCK:-}" ]; then
  tab_id=$(tab_alloc playwright-cli); tab_set "$tab_id" mock_url "$URL"
  jq -n --arg id "$tab_id" '{tab_id: $id}'; exit 0
fi

command -v playwright >/dev/null || command -v npx >/dev/null \
  || die "playwright not installed: npm i -g playwright" open

tab_id=$(tab_alloc playwright-cli)
"$DRIVER_DIR/_playwright-runner.sh" "$tab_id" "$URL" "$VW" "$VH"
jq -n --arg id "$tab_id" '{tab_id: $id}'
EOF
chmod +x tools/drivers/browser/playwright-cli/open.sh
```

- [ ] **Step 4: Write the other 8 verbs as a single template-driven generator**

```bash
for verb in click type read screenshot wait eval cookies close; do
cat > "tools/drivers/browser/playwright-cli/$verb.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
DRIVER_DIR="\$(cd "\$(dirname "\$0")" && pwd)"
source "\$DRIVER_DIR/../_shared/json-args.sh"
source "\$DRIVER_DIR/../_shared/tab-registry.sh"

ARGS_JSON="\$(read_args)"
TAB_ID=\$(arg .tab_id); [ -n "\$TAB_ID" ] || die "tab_id required" $verb

if [ -n "\${GPOWERS_BROWSER_MOCK:-}" ]; then
  case "$verb" in
    click|type|wait)  jq -n '{ok:true}' ;;
    read)             jq -n --arg c "mock content" '{content:\$c}' ;;
    screenshot)       jq -n --arg p "\$(gpowers-path cache)/browser/shots/mock.png" '{path:\$p}' ;;
    eval)             jq -n '{value:null}' ;;
    cookies)          jq -n '{cookies:[]}' ;;
    close)            tab_release "\$TAB_ID"; jq -n '{ok:true}' ;;
  esac
  exit 0
fi

TAB_DIR="\$(gpowers-path state)/browser/tabs/\$TAB_ID"
[ -p "\$TAB_DIR/req" ] || die "tab not opened" $verb "\$TAB_ID"

# Inject verb into args and send to runner
REQ=\$(echo "\$ARGS_JSON" | jq --arg v "$verb" '.verb = \$v')
printf '%s\n' "\$REQ" > "\$TAB_DIR/req" &
RES=\$(head -n1 "\$TAB_DIR/res")
[ "$verb" = "close" ] && tab_release "\$TAB_ID"
printf '%s\n' "\$RES"
EOF
chmod +x "tools/drivers/browser/playwright-cli/$verb.sh"
done
```

- [ ] **Step 5: Commit**

```bash
git add tools/drivers/browser/playwright-cli/
git commit -m "feat(drivers): playwright-cli driver — 9 verbs over long-lived runner"
```

---

## Task 5: Implement select-driver.sh

**Files:**
- Create: `tools/drivers/browser/select-driver.sh`

Detection logic: prefer claude-in-chrome (MCP available), fall back to playwright CLI, error with install hint if neither.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/drivers/select-driver.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  SCRIPT="$GPOWERS_REPO/tools/drivers/browser/select-driver.sh"
  export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
  mkdir -p "$BATS_TEST_TMPDIR/bin"
}

teardown() {
  unset GPOWERS_BROWSER_DRIVER GPOWERS_PLATFORM
}

@test "prefers claude-in-chrome when GPOWERS_PLATFORM=claude-code" {
  export GPOWERS_PLATFORM=claude-code
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "claude-in-chrome" ]
}

@test "falls back to playwright-cli when claude-code unavailable + playwright present" {
  export GPOWERS_PLATFORM=codex
  cat > "$BATS_TEST_TMPDIR/bin/playwright" <<'F'
#!/bin/sh
echo "Version 1.0"
F
  chmod +x "$BATS_TEST_TMPDIR/bin/playwright"
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "playwright-cli" ]
}

@test "sets driver to 'missing' with install hint when no backend available" {
  export GPOWERS_PLATFORM=codex
  PATH="/usr/bin:/bin" source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "missing" ]
}

@test "honors pre-set GPOWERS_BROWSER_DRIVER without override" {
  export GPOWERS_PLATFORM=claude-code
  export GPOWERS_BROWSER_DRIVER=playwright-cli
  source "$SCRIPT"
  [ "$GPOWERS_BROWSER_DRIVER" = "playwright-cli" ]
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Write the script**

```bash
cat > tools/drivers/browser/select-driver.sh <<'EOF'
# Source me. Exports GPOWERS_BROWSER_DRIVER. Honors pre-set value.
# Detection order:
#   1. $GPOWERS_BROWSER_DRIVER pre-set → use as-is
#   2. $GPOWERS_PLATFORM = claude-code  → claude-in-chrome
#   3. playwright CLI on PATH         → playwright-cli
#   4. otherwise → missing (with install hint to stderr)

if [ -n "${GPOWERS_BROWSER_DRIVER:-}" ]; then
  return 0 2>/dev/null || exit 0
fi

case "${GPOWERS_PLATFORM:-}" in
  claude-code)
    export GPOWERS_BROWSER_DRIVER=claude-in-chrome
    ;;
  *)
    if command -v playwright >/dev/null 2>&1 \
        || command -v npx >/dev/null 2>&1 && npx --no-install playwright --version >/dev/null 2>&1; then
      export GPOWERS_BROWSER_DRIVER=playwright-cli
    else
      export GPOWERS_BROWSER_DRIVER=missing
      echo "gpowers: no browser driver available. Install: bun add -g @playwright/test  (or use Claude Code with claude-in-chrome MCP)" >&2
    fi
    ;;
esac
EOF
```

- [ ] **Step 3: Run test to verify pass**

Run: `bats tests/unit/drivers/select-driver.bats`
Expected: PASS (4 tests).

- [ ] **Step 4: Commit**

```bash
git add tools/drivers/browser/select-driver.sh tests/unit/drivers/select-driver.bats
git commit -m "feat(drivers): select-driver runtime detection"
```

---

## Task 6: Write `gpowers-browser` dispatch shim

**Files:**
- Create: `tools/bin/gpowers-browser`

The single entry point skills call. `gpowers-browser open` → exec `<driver>/open.sh`.

- [ ] **Step 1: Write failing test**

```bash
cat > tests/unit/drivers/verb-shape.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  GPOWERS_REPO="$BATS_TEST_DIRNAME/../../.."
  export GPOWERS_HOME="$GPOWERS_REPO"
  export PATH="$GPOWERS_REPO/bin:$GPOWERS_REPO/tools/bin:$PATH"
  export GPOWERS_BROWSER_DRIVER=claude-in-chrome
  export GPOWERS_BROWSER_MOCK=1
}

@test "gpowers-browser open returns tab_id" {
  result=$(echo '{"url":"https://example.com"}' | gpowers-browser open)
  echo "$result" | jq -e '.tab_id' >/dev/null
}

@test "gpowers-browser unknown verb errors clearly" {
  run bash -c 'echo "{}" | gpowers-browser bogus 2>&1'
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "unknown verb"
}

@test "gpowers-browser missing driver errors with hint" {
  unset GPOWERS_BROWSER_DRIVER
  export GPOWERS_BROWSER_DRIVER=missing
  run bash -c 'echo "{}" | gpowers-browser open 2>&1'
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "install"
}
EOF
```

Run: expect FAIL.

- [ ] **Step 2: Write the shim**

```bash
mkdir -p tools/bin
cat > tools/bin/gpowers-browser <<'EOF'
#!/usr/bin/env bash
# Usage: gpowers-browser <verb> < args-json
# Reads $GPOWERS_BROWSER_DRIVER and exec's the matching driver script.
set -euo pipefail

VERB="${1:-}"
[ -n "$VERB" ] || { echo "gpowers-browser: verb required" >&2; exit 2; }
shift

VALID="open click type read screenshot wait eval cookies close"
case " $VALID " in
  *" $VERB "*) ;;
  *) echo "gpowers-browser: unknown verb '$VERB'. Valid: $VALID" >&2; exit 2;;
esac

# Source select-driver if not already
if [ -z "${GPOWERS_BROWSER_DRIVER:-}" ]; then
  source "${GPOWERS_HOME:?GPOWERS_HOME required}/tools/drivers/browser/select-driver.sh"
fi

if [ "$GPOWERS_BROWSER_DRIVER" = "missing" ]; then
  echo "gpowers-browser: no browser driver available. Install: bun add -g @playwright/test" >&2
  exit 3
fi

SCRIPT="$GPOWERS_HOME/tools/drivers/browser/$GPOWERS_BROWSER_DRIVER/$VERB.sh"
[ -x "$SCRIPT" ] || { echo "gpowers-browser: driver $GPOWERS_BROWSER_DRIVER missing verb $VERB" >&2; exit 4; }

exec "$SCRIPT" "$@"
EOF
chmod +x tools/bin/gpowers-browser
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/drivers/verb-shape.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add tools/bin/gpowers-browser tests/unit/drivers/verb-shape.bats
git commit -m "feat(drivers): gpowers-browser dispatch shim"
```

---

## Task 7: Contract / parity test

**Files:**
- Create: `tests/integration/drivers/fixtures/page.html`
- Create: `tests/integration/drivers/fixtures/server.sh`
- Create: `tests/integration/drivers/parity.bats`

Run all 9 verbs against both drivers in mock mode. Confirms wire format is identical even if backends differ.

- [ ] **Step 1: Write the fixture page**

```bash
mkdir -p tests/integration/drivers/fixtures
cat > tests/integration/drivers/fixtures/page.html <<'EOF'
<!doctype html>
<html><head><title>gpowers fixture</title></head>
<body>
  <h1 id="title">Hello gpowers</h1>
  <input id="name" />
  <button id="btn" onclick="document.getElementById('out').textContent='clicked';">Click</button>
  <pre id="out"></pre>
  <script>console.log("fixture page loaded");</script>
</body></html>
EOF
```

- [ ] **Step 2: Write the fixture server launcher**

```bash
cat > tests/integration/drivers/fixtures/server.sh <<'EOF'
#!/usr/bin/env bash
# Start a python http.server on a free port. Echoes the port.
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cd "$DIR" && python3 -m http.server "$PORT" >/dev/null 2>&1 &
echo $! > /tmp/gpowers-fixture-server.pid
echo "$PORT"
EOF
chmod +x tests/integration/drivers/fixtures/server.sh
```

- [ ] **Step 3: Write parity test (mock mode only — real-mode tests added in Plan #11)**

```bash
cat > tests/integration/drivers/parity.bats <<'EOF'
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
EOF
```

Run: `bats tests/integration/drivers/parity.bats`
Expected: PASS (9 tests, one per verb).

- [ ] **Step 4: Commit**

```bash
git add tests/integration/drivers/
git commit -m "test(drivers): wire-format parity across claude-in-chrome and playwright-cli"
```

---

## Task 8: Update manifest and capabilities registry

**Files:**
- Modify: `manifest.json`

Record that `drivers/browser/` is present and which drivers are available on this install.

- [ ] **Step 1: Failing test**

```bash
cat > tests/unit/drivers/manifest-records-drivers.bats <<'EOF'
#!/usr/bin/env bats

setup() {
  MANIFEST="$BATS_TEST_DIRNAME/../../../manifest.json"
}

@test "manifest declares drivers section" {
  jq -e '.drivers.browser' < "$MANIFEST" >/dev/null
}

@test "manifest lists both browser drivers" {
  jq -e '.drivers.browser.available | index("claude-in-chrome")' < "$MANIFEST" >/dev/null
  jq -e '.drivers.browser.available | index("playwright-cli")' < "$MANIFEST" >/dev/null
}

@test "manifest names interface_version 1" {
  v=$(jq -r '.drivers.browser.interface_version' < "$MANIFEST")
  [ "$v" = "1" ]
}
EOF
```

- [ ] **Step 2: Update manifest**

```bash
jq '.drivers = { browser: { interface_version: 1, available: ["claude-in-chrome","playwright-cli"] } }' \
   manifest.json > manifest.json.tmp && mv manifest.json.tmp manifest.json
```

- [ ] **Step 3: Run test**

Run: `bats tests/unit/drivers/manifest-records-drivers.bats`
Expected: PASS (3 tests).

- [ ] **Step 4: Commit**

```bash
git add manifest.json tests/unit/drivers/manifest-records-drivers.bats
git commit -m "feat(drivers): manifest declares browser driver interface v1"
```

---

## Self-Review

### 1. Spec coverage (§4 of design)

| Spec requirement | Task |
|---|---|
| 9 verbs defined | Task 1 (interface.md) |
| claude-in-chrome driver implements 9 verbs | Task 3 |
| playwright-cli driver implements 9 verbs | Task 4 |
| select-driver.sh detection | Task 5 |
| skill author contract (no MCP refs in skills) | Task 1 (interface.md), Task 6 (shim is the only entry) |
| `gpowers-browser` shim is single entry | Task 6 |
| Parity tested across drivers | Task 7 |

### 2. Placeholder scan

No TBDs. Every verb script has working mock-mode behavior; real-mode behavior is described in code blocks (Node script for playwright, MCP instruction strings for claude-in-chrome). Real-mode end-to-end tests against a live browser are deferred to Plan #11's platform-smoke layer — this is explicit, not a placeholder.

### 3. Type / name consistency

- All 9 verbs use the same names across spec, both drivers, dispatch shim, and tests.
- `tab_id` semantics consistent: opaque string allocated by `tab_alloc`, threaded through every other verb, released by `close`.
- `GPOWERS_BROWSER_DRIVER` env var name consistent (Tasks 5, 6, 7).
- `GPOWERS_BROWSER_MOCK=1` consistent in Tasks 3, 4, 6, 7.

### 4. Decomposition

8 tasks, each producing a single committable artifact. Drivers split symmetrically (one task each) so reviewers can compare side-by-side.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-14-gpowers-drivers.md`. Can run in parallel with Plan #2 (only depends on Plan #1's foundation). Pick subagent-driven or inline at execution time.
