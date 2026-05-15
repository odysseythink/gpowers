# gpowers driver interface

The driver layer is how gpowers stays cross-platform without binding to any specific MCP server or CLI tool. All browser-using skills speak only nine **abstract verbs**, dispatched at runtime to a driver that implements them.

## The 9 verbs

| Verb | Purpose | Required args | Optional args | Returns |
|---|---|---|---|---|
| `browser.open` | Open URL in new tab | `url` | `viewport` | `{tab_id}` |
| `browser.click` | Click element | `tab_id, selector` | `timeout_ms` | `{ok}` |
| `browser.type` | Fill input | `tab_id, selector, text` | — | `{ok}` |
| `browser.read` | Read text/dom/console | `tab_id, mode` | `selector` | `{content}` |
| `browser.screenshot` | Capture image | `tab_id` | `region` | `{path}` |
| `browser.wait` | Wait for condition | `tab_id, condition` | `timeout_ms` | `{ok}` |
| `browser.eval` | Run JS in page context | `tab_id, code` | — | `{value}` |
| `browser.cookies` | Read/write cookies | `tab_id, op` | `domain, cookies` | `{cookies}` or `{ok}` |
| `browser.close` | Close tab | `tab_id` | — | `{ok}` |

Full schema lives in `tools/drivers/browser/interface.md` — the normative spec.

## Wire format

Every verb is invoked through `gpowers-browser`, which reads `$GPOWERS_BROWSER_DRIVER` and exec's the matching driver script. Arguments come in as a single JSON object on **stdin**. Returns come on **stdout** as JSON. Errors exit non-zero with a `{error, verb, tab_id}` payload on stderr.

```bash
echo '{"url":"https://example.com"}' | gpowers-browser open
# stdout: {"tab_id":"t-claude-in-chrome-1"}

echo '{"tab_id":"t-claude-in-chrome-1","selector":"#submit"}' | gpowers-browser click
# stdout: {"ok":true}
```

## Skill-author contract

Skills MUST:
- Source `tools/drivers/browser/select-driver.sh` in their Preamble (template provided).
- Call only `gpowers-browser <verb>` — never reference `mcp__claude-in-chrome__*` or `playwright` literally.
- Declare `requires-driver: browser` in frontmatter.

The frontmatter declaration lets the installer skip the skill (with a clear hint) on platforms that lack a working driver.

## Built-in drivers

### claude-in-chrome

Translates the 9 verbs to Anthropic's `claude-in-chrome` MCP server. Native on Claude Code. Each verb script emits a `GPOWERS_MCP_INSTRUCTION:` line on stderr that the agent reads and translates to the appropriate `mcp__claude-in-chrome__*` invocation. Tab IDs map to MCP tab handles via the shared tab-registry.

### playwright-cli

Translates the 9 verbs to Playwright's Node API over a long-lived per-tab Node runner. Works on every platform that can install `playwright` (`bun add -g @playwright/test` or `npm install -g playwright`, then `npx playwright install chromium`).

## Selecting at runtime

`tools/drivers/browser/select-driver.sh` is the single point of resolution:

1. If `$GPOWERS_BROWSER_DRIVER` is already set, honor it.
2. If `$GPOWERS_PLATFORM=claude-code`, default to `claude-in-chrome`.
3. Otherwise, if `playwright` is on PATH, default to `playwright-cli`.
4. Otherwise, set `GPOWERS_BROWSER_DRIVER=missing` and print an install hint.

## Adding a new driver

Suppose Anthropic ships a new MCP server `chrome-pro` that you want to wire up:

1. **Create the directory:**
   ```bash
   mkdir -p $GPOWERS_HOME/tools/drivers/browser/chrome-pro
   ```
2. **Write 9 verb scripts.** Each follows the same shape: read JSON args, do the work, return JSON. The shared `_shared/{json-args,tab-registry}.sh` helpers are available; source them.
3. **Add a `capabilities.json`** declaring backend, platform, and feature flags.
4. **Update `select-driver.sh`** to detect when chrome-pro is the right default (or rely on users setting `GPOWERS_BROWSER_DRIVER=chrome-pro` explicitly).
5. **Add a parity test row** in `tests/integration/drivers/parity.bats` for the new driver.

`tools/drivers/browser/_template/` (TODO: ship in a future minor version) will provide a starter scaffold.

## Tab semantics

A `tab_id` returned by `browser.open` is opaque to skills and stable until `browser.close` consumes it. Internal state per tab lives under `$(gpowers-path state)/browser/tabs/<tab_id>/`. Each driver may write its own keys there (the `mcp_tab_id` from claude-in-chrome, the FIFO paths for playwright-cli, etc.).

## Mock mode for testing

Set `GPOWERS_BROWSER_MOCK=1` and every verb returns a canned success without touching a real browser. This is how `tests/integration/tools-browser/browse-skill-smoke.bats` and parity tests stay fast and deterministic.
