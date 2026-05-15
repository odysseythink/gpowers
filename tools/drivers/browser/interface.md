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
