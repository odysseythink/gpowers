# browse-go Feature Inventory

> Comprehensive structural overview of the Go rewrite of the browse tool.
> Generated from source analysis of `tools/skills/browse-go/`.

---

## 1. CLI Entry Point (`cmd/browse/main.go`)

### Subcommands
| Command | Description |
|---------|-------------|
| `server` | Run the HTTP daemon in the foreground (used internally by `start`) |
| `start` | Spawn a background daemon with lockfile protection, health-check wait |
| `stop` | Gracefully stop the running daemon |
| `status` | Print PID, port, mode, config hash from state file |
| `connect` | Kill existing server, clean Chromium locks, spawn **headed** mode + terminal agent |
| `disconnect` | Stop headed mode, return to headless |
| `pair-agent` | Generate a setup key for remote agent pairing (`--control`, `--admin`, `--client-id`) |
| *(default)* | Send any command to the daemon (auto-starts if not running) |

### Global Flags
| Flag | Description |
|------|-------------|
| `--proxy <url>` | Upstream SOCKS5 proxy URL |
| `--headed` | Launch visible Chrome instead of headless |
| `--tab-id <id>` | Target a specific tab for the command |

### State & Lifecycle
- **State file**: `.gstack/browse.json` — stores PID, Port, Token, ConfigHash, Mode
- **Lockfile**: Prevents double-start races
- **Crash recovery**: Auto-restart on connection error (1 retry max)
- **Config mismatch detection**: Hash of proxy+headed flags; restart if changed
- **Signal handling**: SIGTERM/SIGINT graceful shutdown
- **Process isolation**: `Setpgid` on Unix, `CREATE_NEW_PROCESS_GROUP` on Windows
- **Orphan cleanup**: Reads Chromium `SingletonLock` symlink to kill stale processes

---

## 2. Terminal Agent (`cmd/terminal-agent/main.go` + `pkg/terminal/agent.go`)

WebSocket-to-PTY bridge for the sidebar Terminal pane.

### Endpoints
| Endpoint | Auth | Description |
|----------|------|-------------|
| `/internal/grant` | Internal Bearer token | Create a new PTY session token |
| `/internal/revoke` | Internal Bearer token | Revoke a session token |
| `/ws` | Session token (via `Sec-WebSocket-Protocol` or `gstack_pty` cookie) | WebSocket ↔ PTY bidirectional pipe |

### Features
- Binds an **ephemeral port**; writes port + internal token to atomically-named files
- Spawns `$SHELL` (fallback `/bin/bash`) with `TERM=xterm-256color`, `BROWSE_PORT` env
- Supports terminal resize via JSON text frames
- Binary frames → PTY stdin; PTY output → WebSocket binary frames
- Cleanup on disconnect (signal-based)

---

## 3. Browser Management (`pkg/browser/`)

### `manager.go` — BrowserManager

| Feature | Detail |
|---------|--------|
| **Headless launch** | Stealth args, `--no-sandbox` for Docker/CI, extensions via `BROWSE_EXTENSIONS_DIR` |
| **Headed launch** | Visible Chrome with extension auth bootstrap (`LaunchHeadedOptions{AuthToken, ServerPort}`) |
| **Tabs** | `NewTab(url)`, `CloseTab(id)`, `SwitchTab(id)`, `CloseAllTabs()` |
| **State save/restore** | Cookies, pages, localStorage/sessionStorage, loadedHtml, ownership |
| **Extra headers** | Injected into all requests |
| **Proxy config** | Passed through to Chromium |
| **Dialog auto-accept** | Configurable |
| **Watch mode** | Periodic snapshots every 5s |

### CDP Event Wiring (`wireTabEvents`)
| Event | Destination |
|-------|-------------|
| Console | `consoleBuffer` (circular) |
| Dialog | Auto-accept/dismiss + `dialogBuffer` |
| Network | `networkBuffer` + response body capture |
| Response | `captureResponseBody()` via `Network.getResponseBody` |

### `state.go` — TabSession

| Feature | Detail |
|---------|--------|
| **Element refs** | `@e` (interactive), `@c` (clickable) — validated against DOM, stale-check |
| **Iframe switching** | `SwitchToFrame(targetID)` via `chromedp.WithTargetID`; `SwitchToMainFrame()` cancels frame ctx |
| **Loaded HTML replay** | Stored for restoration |
| **Style history** | Undo stack for `style --undo` (`PushStyleMod/PopStyleMod/ClearStyleHistory`) |

### `buffer.go` — SizeCappedBuffer
- 50MB cap per buffer (console, network, dialog, response capture)

---

## 4. HTTP Server (`pkg/server/server.go`)

### Route Table

#### Health & Command
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/health` | GET | None | Daemon health check |
| `/command` | POST | Bearer | Execute single command |
| `/batch` | POST | Bearer | Execute multiple commands (nested batch guard) |
| `/refs` | POST | Bearer | Resolve `@e`/`@c` refs to selectors |

#### Tab & File
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/tabs` | GET | Bearer | List all tabs |
| `/file` | GET | Bearer | Serve captured files |

#### Server Control
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/stop` | POST | Root only | Graceful shutdown |

#### Audit & Activity
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/audit` | GET | Bearer | Query audit log (filters: command, client, verdict, max) |
| `/audit-stats` | GET | Bearer | Audit summary statistics |
| `/activity/stream` | GET | SSE | Real-time activity streaming |
| `/activity/history` | GET | Bearer | JSON history with `?limit=` |

#### Pairing & Tokens
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/pair` | POST | Root or setup key | Pair a remote agent |
| `/token` | POST | Root only | Mint scoped tokens |
| `/agents` | GET | Root only | List paired agents |

#### Inspector
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/inspector/inspect` | POST | Bearer | CSS cascade inspection |
| `/inspector/apply` | POST | Bearer | Live CSS modification |
| `/inspector/undo` | POST | Bearer | Undo last modification |
| `/inspector/reset` | POST | Bearer | Reset all modifications |
| `/inspector/history` | GET | Bearer | Modification history |
| `/inspector/events` | SSE | Bearer | Live inspector events |

#### Session & Tunnel
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/sse-session` | GET | SSE | Generic SSE session endpoint |
| `/pty-session` | GET | WebSocket | PTY session (terminal agent) |
| `/tunnel/start` | POST | Root only | Start ngrok tunnel |
| `/connect` | GET | Tunnel | Tunnel surface connect |

#### Cookies & Welcome
| Route | Method | Auth | Description |
|-------|--------|------|-------------|
| `/cookie/picker` | GET | Session cookie | Cookie picker web UI |
| `/cookie/auth-code` | GET | One-time code | Auth code validation |
| `/welcome` | GET | None | Project-specific or built-in welcome HTML |

### Middleware Stack
1. **CORS** — Configured for Chrome extension origin
2. **Idle timeout reset** — Auto-shutdown after inactivity
3. **Bearer auth** — Three-tier:
   - **Root token**: All scopes (from state file / env)
   - **Skill token** (`sk_*`): Parsed by `skilltoken.ValidateString`
   - **Registry token**: Scoped, domain-restricted, rate-limited
4. **Header injection**: Sets `X-Browse-Scope` + `X-Client-ID`

### Tunnel Surface (`StartTunnelSurface`)
- Secondary HTTP listener exposing only `/connect` + `/command`
- Commands restricted to `tunnel.TunnelCommands` allowlist (29 commands)
- Separate port, isolated from main API surface

---

## 5. Security Pipeline (`pkg/security/`)

### L1 — Trust Boundary Envelope (`envelope.go`)
- `WrapUntrustedPageContent()`: Escapes inner sentinels, wraps with `═══ BEGIN/END UNTRUSTED WEB CONTENT ═══`
- `WrapUntrusted()`: General-purpose `--- BEGIN/END UNTRUSTED EXTERNAL CONTENT ---`
- `DatamarkContent()`: Invisible Unicode watermark every 3rd sentence
- `EscapeEnvelopeSentinels()`: Splices zero-width spaces into "CONTENT" if literal appears

### L2 — DOM Strip (`dom_strip.go`)
- `MarkHiddenElements()`: Detects and marks elements with:
  - opacity < 0.1, font-size < 1px, off-screen positioning
  - same fg/bg color, clip hiding, visibility hidden
  - ARIA label injection patterns
- `GetCleanText()`: Extracts `innerText` after removing hidden elements + script/style/noscript/svg
- `CleanupHiddenMarkers()`: Removes `data-gstack-hidden` attributes

### L3 — URL Blocklist (`url_blocklist.go` — referenced, not read)
- Blocks known malicious/phishing domains

### L4 — ML Classifiers (`classifier.go`, `classifier_rule.go`)

| Classifier | Layer Name | Basis | Confidence Range |
|------------|-----------|-------|-----------------|
| **RuleBased** | `rule_based` | 12 regex patterns (ignore instructions, act as admin, override, etc.) | 0.40–0.85 |
| **ARIARegex** | `aria_regex` | 7 ARIA-focused injection patterns | 0.45–0.90 |
| **TestSavant** | `testsavant_content` | ONNX model (optional, auto-download) | varies |
| **Deberta** | `deberta_content` | ONNX model (optional, env-gated) | varies |
| **Haiku** | `transcript_classifier` | Slow-path transcript scan | varies |

- `MultiClassifier`: Concurrent load, parallel scan, fail-open on errors
- `ScanPageContent()`: Normalizes HTML → plain text, truncates to 4000 chars

### L5 — Canary (`canary.go` — referenced)
- Per-session random canary string
- Injected into page content; any leak in response → deterministic BLOCK

### L6 — Verdict Ensemble (`verdict.go`)

| Threshold | Action |
|-----------|--------|
| Canary leak ≥ 1.0 | **BLOCK** (deterministic) |
| 2+ block-votes from layers | **BLOCK** |
| Single-layer content ≥ 0.92 | **BLOCK** (or WARN if not tool-output) |
| Max ML ≥ 0.75 | **WARN** |
| Max ML ≥ 0.40 | **LOG_ONLY** |
| Below 0.40 | **safe** |

Verdicts: `safe`, `log_only`, `warn`, `block`, `user_overrode`

### Pipeline Orchestration (`pipeline.go`)
- `SecureTextResult()`: L2 → L3 → L4 → L5 → L6 → L1
- `SecureSnapshotResult()`: L3 → L4 → L5 → L6 → L1 (no DOM strip)
- `ScanTranscript()`: L4b Haiku classifier for user message + tool calls
- `Status()`: Returns `StatusDetail` with per-layer health for shield icon

### Scope System (`scope.go`)
6 scope categories: `all`, `navigate`, `read`, `interact`, `write`, `inspect`, `system`
- 60+ commands mapped to required scopes
- `ScopeSet`: Parsed from comma-separated string; `all` wildcard
- `CheckScope()`: Returns error if missing required scope

### Rate Limiting (`ratelimit.go`)
- Per-key token-bucket (`clientID:command`)
- Env: `BROWSE_RATE_LIMIT=off`, `BROWSE_RATE_LIMIT_RPS=10`, `BROWSE_RATE_LIMIT_BURST=20`
- `Allow(key)`: Refills by elapsed time, rejects if < 1 token

### Audit Logging (`attempt_log.go`)
- JSONL at `~/.gstack/security/attempts.jsonl`
- 10MB rotation, 5 generations
- Fields: TS, URLDomain, PayloadHash (salted SHA-256), Confidence, Layer, Verdict
- `LogAttempt()`: Never throws (fail-safe)

### Session State (`state.go`)
- Cross-process security state: `~/.gstack/security/session-state.json`
- Canary, classifier status, warned domains
- Per-tab decision files (`tab-N.json`): allow/block with reason

---

## 6. Activity Streaming (`pkg/activity/activity.go`)

- **Circular buffer**: 500 entries, thread-safe
- **Entry types**: `command_start`, `command_end`, `navigation`, `error`
- **Fields**: ID, Timestamp, Type, Command, Args, URL, Duration, Error, Meta
- **SSE endpoint** (`/activity/stream`): Sends buffered history then live events
- **JSON endpoint** (`/activity/history`): Query with `?limit=`
- **Backpressure**: Non-blocking send; drops silently if subscriber slow

---

## 7. Token Registry (`pkg/tokenregistry/registry.go`)

### Scope Categories
| Category | Permissions |
|----------|-------------|
| `read` | text, html, links, snapshot, etc. |
| `write` | fill, click, upload, eval, etc. |
| `admin` | token management, pair-agent |
| `meta` | chain command (allows nested execution) |
| `control` | stop, status, tunnel management |

### Token Types
| Type | TTL | Uses | Description |
|------|-----|------|-------------|
| **Session token** | 24h default | Unlimited | Standard agent token |
| **Setup key** | 5 min | 1 use | Pair-agent ceremony |

### Checks
- `CheckScope()`: Root = true; `chain` requires `meta`; others checked against `scopeMap`
- `CheckDomain()`: Glob matching (`*.example.com`) against page URL
- `CheckRate()`: Per-second window counter per ClientID
- `CheckConnectRateLimit()`: Global flood protection (300 attempts / 60s)
- `ExchangeSetupKey()`: Idempotent if 0 commands executed

---

## 8. SOCKS5 Bridge (`pkg/socks/socks.go`)

- **Local SOCKS5** (unauthenticated client) → **upstream authenticated SOCKS5**
- `StartBridge(upstream, port)`: Port 0 = ephemeral
- Handshake: Accepts no-auth locally; offers authNone + authPassword upstream
- `upstreamConnect()`: Sends CONNECT through upstream proxy
- `TestUpstream()`: Verifies connectivity to 1.1.1.1:443 with retries + backoff
- `Close()`: Shuts down listener + all in-flight connections

---

## 9. Cookie Import & Picker (`pkg/cookieimport/import.go` + `pkg/picker/picker.go`)

### Browser Cookie Decryption
- **Platform-specific derived keys**: macOS (Keychain), Linux (Secret Service / libsecret), Windows (DPAPI)
- **SQLite decryption**: Reads Chromium/Firefox cookie databases
- **v20 App-Bound Encryption**: CDP fallback (`Network.getAllCookies`) when local decryption fails
- **Domain validation**: Ensures imported cookies match current page domain
- `--all` mode: Imports all browser cookies with scoping warning

### Web UI Picker
- One-time auth codes (30s TTL) → session cookies (1h)
- Web interface at `/cookie/picker`
- Auth code validation at `/cookie/auth-code`

---

## 10. Xvfb Support (`pkg/xvfb/xvfb.go`)

- `ShouldSpawn()`: True on Linux when `DISPLAY` is unset
- `PickFreeDisplay()`: Scans `:99`–`:120` for free TCP port 6000+d
- `Spawn(display)`: `Xvfb <display> -screen 0 1920x1080x24 -ac +extension GLX +render -noreset`
- `Cleanup(info)`: Verifies PID still matches Xvfb via `/proc/PID/cmdline`, then TERM + KILL

---

## 11. Browser Skills (`pkg/browserskill/`)

### Storage Tiers (first-wins)
| Tier | Path | Writable |
|------|------|----------|
| Project | `<project>/.gstack/browser-skills/<name>/` | Yes |
| Global | `~/.gstack/browser-skills/<name>/` | Yes |
| Bundled | `<install>/browser-skills/<name>/` | No (read-only) |

### Skill Structure
- `SKILL.md` with YAML frontmatter: `name`, `description`, `host`, `triggers`, `trusted`, `version`, `source`
- Executable entrypoints (searched in order): `run`, `script`, `script.sh`, `script.go`, `script.py`
- Test entrypoints: `test`, `run_test`, `script_test.go`

### Command Interface (`skill` command)
| Subcommand | Description |
|------------|-------------|
| `list` | List all skills with resolved tier |
| `show <name>` | Print SKILL.md |
| `run <name> [--arg k=v]... [--timeout=Ns]` | Execute skill with scoped token |
| `test <name>` | Run skill test entrypoint |
| `rm <name> [--global]` | Tombstone a user-tier skill |

### Spawn Security
- **Scoped token**: `skilltoken.Mint()` with `read,navigate,interact,inspect,system` scopes
- **Environment filtering**:
  - Trusted skills: Full env except `GSTACK_TOKEN` (root token never propagated)
  - Untrusted skills: Minimal allowlist (`LANG`, `LC_*`, `TERM`, `TZ`) + minimal `PATH`
  - Secret scrubbing: Regex patterns for TOKEN, KEY, SECRET, PASSWORD, AWS_*, AZURE_*, GCP_*, etc.
- **Stdout cap**: 1MB truncation
- **Timeout**: Default 60s, configurable

---

## 12. Domain Skills (`pkg/domainskill/`)

Per-site notes the agent writes for itself.

### Storage
- Per-project: `~/.gstack/projects/<slug>/learnings.jsonl`
- Global: `~/.gstack/global-domain-skills.jsonl`
- Append-only with `O_APPEND` (POSIX atomic appends < PIPE_BUF)
- Tombstone for deletes; `Compactor` rewrites file dropping superseded rows

### State Machine (T6)
```
quarantined ──(N=3 uses, no flags)──► active ──(manual promote)──► global
```

### Command Interface (`domain-skill` command)
| Subcommand | Description |
|------------|-------------|
| `save` | Save body from stdin or `--from-file` (host from active tab) |
| `list` | List all skills visible to current project |
| `show <host>` | Print skill body |
| `promote-to-global <host>` | Promote active skill to global scope |
| `rollback <host> [--global]` | Restore prior version |
| `rm <host> [--global]` | Tombstone |

### Write Protection
- Classifier score ≥ 0.85 → BLOCK (potential injection)
- Host normalized: strips protocol, path, port, `www.` prefix

---

## 13. Inspector (`pkg/inspector/inspector.go`)

CDP-based CSS inspection and live modification.

### Inspection Output (`Result`)
| Field | Description |
|-------|-------------|
| `selector` | Query selector |
| `tagName`, `id`, `classes`, `attributes` | Element metadata |
| `boxModel` | Content, padding, border, margin dimensions |
| `computed` | Key computed styles (filtered set of ~50 properties) |
| `matchedRules` | All matching CSS rules with specificity, source, line, overridden status |
| `inlineStyles` | Current inline styles |
| `pseudoElements` | `::before`, `::after` rules |

### Modification
- `Apply(selector, property, value)`: Tries CDP `CSS.setStyleTexts` first, falls back to inline `setProperty`
- `Undo(index)`: Reverts by index (or last if < 0)
- `Reset()`: Reverts all modifications
- **Dangerous CSS guard**: Rejects `url(`, `expression(`, `@import`, `javascript:`, `data:`
- **History**: Thread-safe slice with pub/sub events (apply/undo/reset)

### Specificity Computation
- Regex-based {a,b,c} calculation: IDs → a, classes/attrs/pseudo-classes → b, types/pseudo-elements → c
- Rules sorted by specificity descending
- `!important` handling: Important rule overrides non-important even with lower specificity

---

## 14. Tunnel (`pkg/tunnel/tunnel.go`)

ngrok tunnel lifecycle for remote agent access.

### Features
- `Start(localPort, authtoken)`: Spawns `ngrok http` subprocess, polls local API (`:4040/api/tunnels`) for public URL
- `ResolveAuthtoken()`: Reads from `NGROK_AUTHTOKEN` env, `~/.gstack/ngrok.env`, or ngrok config files
- `Close()`: Kills ngrok process
- Stale process cleanup before start (`pkill -f ngrok`)

### Tunnel Allowlist (29 commands)
`connect`, `goto`, `back`, `forward`, `reload`, `text`, `html`, `links`, `forms`, `accessibility`, `snapshot`, `click`, `fill`, `scroll`, `wait`, `screenshot`, `status`, `tabs`, `tab`, `newtab`, `closetab`, `stop`

---

## 15. Command Registry (`pkg/commands/`)

### 15 Categories, 60+ Commands

#### Navigation (`navigation.go`)
`goto`, `back`, `forward`, `reload`, `stop`

#### Tabs (`tabs.go`)
`tabs`, `tab`, `newtab`, `closetab`, `switchtab`

#### Reading (`reading.go`)
`text`, `html`, `links`, `forms`, `accessibility`, `data`, `media`

#### Interaction (`interaction.go`)
`click`, `fill`, `type`, `scroll`, `hover`, `wait`, `select`

#### Visual (`visual.go`)
`screenshot`, `pdf`

#### Write (`write.go`)
`dialog-accept`, `dialog-dismiss`, `cookie`, `cookie-import`, `cookie-import-browser`, `header`, `style`, `cleanup`, `upload`, `download`, `scrape`

#### Inspection (`inspection.go`)
`js`, `eval`, `css`, `attrs`, `console`, `network`, `dialog`, `cookies`, `storage`, `perf`

#### Snapshot (`snapshot.go`)
`snapshot [-i] [-c] [-d N] [-s sel] [--diff]`

#### Meta (`meta.go`)
`diff`, `chain`, `pdf`, `frame`, `state`, `tab-each`, `archive`, `handoff`, `resume`, `connect`, `disconnect`, `focus`, `ux-audit`, `watch`

#### Server (`server.go`)
`status`, `stop`

#### Inbox (`inbox.go`)
`inbox [--clear]`

#### CDP (`cdp.go` + `cdpallowlist.go`)
`cdp <Domain.method> [json-params]` — 25 allowlisted methods only

#### BrowserSkill (`browserskill.go`)
`skill <list|show|run|test|rm>`

#### DomainSkill (`domainskill.go`)
`domain-skill <save|list|show|promote|rollback|rm>`

### Registry Execution Flow (`registry.go`)
1. **Lookup** — Canonicalize name (lowercase, aliases like `setcontent` → `load-html`)
2. **Scope check** — `CheckScope()` against granted `ScopeSet`
3. **Rate limit** — `RateLimiter.Allow(key)`
4. **Get session** — `GetOrCreateSession()` (unless `noSessionCommands`)
5. **Execute** — Handler dispatch
6. **Security post-process** — Full pipeline for `secureCommands`

### Security-Post-Processed Commands
`text`, `html`, `links`, `forms`, `accessibility`, `data`, `media`, `inspect`, `snapshot`, `download`, `scrape`

---

## 16. CDP Allowlist (`pkg/commands/cdpallowlist.go`)

Deny-default posture. Only 25 methods permitted.

| Domain | Methods | Scope | Output | Trust |
|--------|---------|-------|--------|-------|
| **Accessibility** | `getFullAXTree`, `getPartialAXTree`, `getRootAXNode` | tab | untrusted | — |
| **DOM** | `describeNode`, `getBoxModel`, `getNodeForLocation` | tab | mixed | — |
| **CSS** | `getMatchedStylesForNode`, `getComputedStyleForNode`, `getInlineStylesForNode` | tab | trusted | — |
| **Performance** | `getMetrics`, `enable`, `disable` | tab | trusted | — |
| **Tracing** | `start`, `end` | browser | trusted | — |
| **Emulation** | `setDeviceMetricsOverride`, `clearDeviceMetricsOverride`, `setUserAgentOverride` | tab | — | — |
| **Page** | `captureScreenshot`, `printToPDF` | tab | untrusted | — |
| **Page** | `getFrameTree` | tab | trusted | — |
| **Network** | `enable`, `disable` | tab | — | — |
| **Target** | `getTargets`, `attachToTarget`, `detachFromTarget`, `getTargetInfo` | browser | trusted | — |
| **Runtime** | `getProperties` | tab | untrusted | **NO `evaluate`/`callFunctionOn`** |

---

## 17. Configuration (`pkg/config/config.go`)

### BrowseConfig Paths
| Path | Default |
|------|---------|
| `StateDir` | `<project>/.gstack/` or `BROWSE_STATE_FILE` dir |
| `StateFile` | `<StateDir>/browse.json` |
| `ConsoleLog` | `<StateDir>/browse-console.log` |
| `NetworkLog` | `<StateDir>/browse-network.log` |
| `DialogLog` | `<StateDir>/browse-dialog.log` |
| `AuditLog` | `<StateDir>/browse-audit.jsonl` |

### Environment
| Variable | Purpose |
|----------|---------|
| `BROWSE_STATE_FILE` | Override state file path |
| `BROWSE_PROXY` / `--proxy` | Upstream SOCKS5 proxy |
| `BROWSE_HEADED` / `--headed` | Headed mode |
| `BROWSE_SCOPE` | Default scope set |
| `BROWSE_RATE_LIMIT` | `off` to disable |
| `BROWSE_RATE_LIMIT_RPS` | Tokens/sec (default 10) |
| `BROWSE_RATE_LIMIT_BURST` | Bucket size (default 20) |
| `BROWSE_EXTENSIONS_DIR` | Chrome extensions directory |
| `CHROMIUM_PROFILE` | Chromium user data directory |
| `GSTACK_HOME` | Override `~/.gstack` |
| `NGROK_AUTHTOKEN` | ngrok authentication |

### Git Integration
- `GitRoot()`: `git rev-parse --show-toplevel`
- `RemoteSlug()`: `owner-repo` from remote origin URL
- `EnsureStateDir()`: Creates `.gstack/`, adds `.gstack/` to `.gitignore`

---

## 18. Cross-Platform Support

| Platform | Focus | Window Activation | Orphan Cleanup |
|----------|-------|-------------------|----------------|
| **macOS** | `osascript` | `tell application "Google Chrome" to activate` | SingletonLock symlink |
| **Linux** | `xdotool` + Xvfb | `xdotool search --name ... windowactivate` | SingletonLock + Xvfb cleanup |
| **Windows** | PowerShell | PowerShell activation script | `taskkill` (no SingletonLock) |

---

## 19. File Structure Summary

```
cmd/
  browse/           CLI entry, daemon lifecycle, platform-specific process mgmt
  terminal-agent/   WebSocket-to-PTY bridge
pkg/
  activity/         SSE activity streaming
  browser/          BrowserManager, TabSession, SizeCappedBuffer
  browserskill/     Skill storage, frontmatter parsing, spawn execution
  commands/         15 command categories, registry, CDP allowlist
  config/           Path resolution, git integration
  cookieimport/     Browser cookie decryption
  domainskill/      Per-site notes with state machine
  inspector/        CDP CSS inspection & live modification
  picker/           Cookie picker web UI
  security/         L1-L6 pipeline, scopes, rate limits, audit
  server/           HTTP daemon, routes, middleware, tunnel surface
  skilltoken/       Per-spawn scoped tokens
  socks/            SOCKS5 bridge
  terminal/         PTY agent library
  tokenregistry/    Scoped token registry
  tunnel/           ngrok tunnel lifecycle
  xvfb/             Xvfb spawn/cleanup
```
