# browse-go

Headless browser daemon for AI agents — Go rewrite of the gstack browse tool.

**Zero Node/Bun dependency.** Single static binary powered by [chromedp](https://github.com/chromedp/chromedp) (direct Chrome DevTools Protocol).

## Quick Start

```bash
# Build
go build -o browse ./cmd/browse

# Start daemon
./browse server

# Or run a command directly
./browse goto https://example.com
./browse text
./browse snapshot
```

## Architecture

```
┌─────────────┐      HTTP/JSON      ┌─────────────────────────────┐
│ browse CLI  │  ◄──────────────►   │   browse server (Go)        │
└─────────────┘                     │   (net/http on localhost)   │
                                    │                             │
                                    │  ┌─────────────────────┐    │
                                    │  │  Command Registry   │    │
                                    │  │  Scope + Rate Limit │    │
                                    │  └──────────┬──────────┘    │
                                    │             │               │
                                    │  ┌──────────▼──────────┐    │
                                    │  │  Security Pipeline  │    │
                                    │  │  L1-L6 + Audit Log  │    │
                                    │  └──────────┬──────────┘    │
                                    │             │               │
                                    │  ┌──────────▼──────────┐    │
                                    │  │  BrowserManager     │    │
                                    │  │  (chromedp/CDP)     │    │
                                    │  └──────────┬──────────┘    │
                                    │             │               │
                                    │  ┌──────────▼──────────┐    │
                                    │  │  Chromium           │    │
                                    │  └─────────────────────┘    │
                                    └─────────────────────────────┘
```

## Commands

### Navigation
- `goto <url>` — Navigate to URL
- `back`, `forward`, `reload` — History
- `newtab [url]`, `closetab [id]` — Tab management

### Reading
- `text` — Cleaned page text (with hidden element stripping)
- `html [selector]` — innerHTML
- `links` — All links
- `forms` — Form fields as JSON
- `accessibility` — ARIA tree
- `media` — Images, videos, audio
- `data` — JSON-LD, Open Graph, meta tags

### Interaction
- `click <sel>`, `fill <sel> <text>`, `type <text>`
- `select <sel> <value>`, `hover <sel>`
- `press <key>`, `scroll <sel>`, `wait [ms]`
- `upload <sel> <file>`

### Inspection
- `console`, `network`, `dialog` — Event buffers
- `cookies`, `storage` — Browser state
- `perf` — Page load timings
- `js <expr>`, `eval <file>` — JavaScript execution
- `css <sel> <prop>`, `attrs <sel>` — Element inspection
- `is <prop> <sel>` — State check (visible, enabled, etc.)
- `cdp <Domain.method> [params]` — Raw CDP

### Visual
- `screenshot [file]`, `prettyscreenshot [file]`
- `pdf [file]`, `responsive <w> <h>`

### Meta
- `snapshot [-i]` — Accessibility snapshot with @refs
- `status`, `tabs`, `help`
- `watch`, `state`, `chain`
- `download <url>`, `scrape <url>`, `archive`
- `ux-audit` — UX analysis

### Domain Skills
Per-site notes with a 3-state lifecycle (quarantined → active → global):

```bash
# Save notes for the current site (host derived from active tab)
echo "# Login flow\nClick the blue button" | ./browse domain-skill save

# List all skills
./browse domain-skill list

# Show a specific skill
./browse domain-skill show example.com

# Promote to global scope
./browse domain-skill promote-to-global example.com

# Rollback to prior version
./browse domain-skill rollback example.com

# Delete (tombstone)
./browse domain-skill rm example.com
```

Storage: `~/.gstack/projects/<slug>/learnings.jsonl` (project) and `~/.gstack/global-domain-skills.jsonl` (global).

### Browser Skills
3-tier skill lookup (project > global > bundled):

```bash
# List all skills
./browse skill list

# Show SKILL.md
./browse skill show my-skill

# Remove a user-tier skill
./browse skill rm my-skill --global
```

### SOCKS5 Bridge
Local unauthenticated SOCKS5 listener that relays through an authenticated upstream proxy:

```go
bridge, err := socks.StartBridge(socks.UpstreamConfig{
    Host:     "proxy.example.com",
    Port:     1080,
    UserID:   "user",
    Password: "pass",
}, 0) // 0 = ephemeral port

// Use in Chromium launch args:
// --proxy-server=socks5://127.0.0.1:<bridge.Port>

bridge.Close()
```

## Security Stack (L1-L6)

Every command that returns external content passes through the security pipeline:

| Layer | Defense |
|-------|---------|
| L1 | **Content Envelope** — `═══ BEGIN UNTRUSTED WEB CONTENT ═══` markers with zero-width space watermarking |
| L2 | **Hidden Element Stripping** — Removes opacity<0.1, font-size<1px, off-screen, clip-hiding, ARIA-injected elements |
| L3 | **URL Blacklist** — Blocks 10 known exfiltration domains (webhook.site, ngrok, etc.) |
| L4 | **ML Classifier** — Rule-based injection detection + ONNX bridge stub for BERT-small |
| L5 | **Canary Tokens** — Per-session 48-bit hex tokens injected into system prompt |
| L6 | **Ensemble Verdict** — 2-of-N block rule, canary leak = instant block |

## Scoped Tokens

Restrict command access via the `X-Browse-Scope` HTTP header:

```bash
curl -H "X-Browse-Scope: read" \
     -H "Authorization: Bearer <token>" \
     -d '{"command":"text"}' \
     http://localhost:PORT/command
```

Scopes: `navigate`, `read`, `interact`, `write`, `inspect`, `system`

Default: `all` (no restriction).

## Rate Limiting

Per-client token-bucket rate limiting is enabled by default:

```bash
# Configure
export BROWSE_RATE_LIMIT_RPS=10
export BROWSE_RATE_LIMIT_BURST=20

# Disable
export BROWSE_RATE_LIMIT=off
```

## Activity Streaming

Real-time command feed for the Chrome extension Side Panel:

```bash
# Server-Sent Events
curl -H "Authorization: Bearer <token>" \
     http://localhost:PORT/activity/stream

# REST fallback (last 50 entries)
curl -H "Authorization: Bearer <token>" \
     "http://localhost:PORT/activity/history?limit=50"
```

## Audit Logging

All command executions are logged to `~/.gstack/security/audit.jsonl`:

```bash
# Query recent commands
curl -H "Authorization: Bearer <token>" \
     "http://localhost:PORT/audit?command=text&max=10"

# Statistics
curl -H "Authorization: Bearer <token>" \
     http://localhost:PORT/audit-stats
```

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Health, security status, rate limit config |
| `/command` | POST | Yes | Execute command |
| `/batch` | POST | Yes | Batch commands |
| `/refs` | GET | Yes | Element ref map |
| `/tabs` | GET | Yes | Tab list |
| `/file` | GET | Yes | Serve temp file |
| `/stop` | POST | Yes | Shutdown |
| `/audit` | GET | Yes | Audit log query |
| `/audit-stats` | GET | Yes | Audit statistics |
| `/activity/stream` | GET | Yes | SSE activity stream |
| `/activity/history` | GET | Yes | Activity history |

## Testing

```bash
# All tests (~186 tests)
go test ./...

# Unit tests only
go test ./pkg/...

# E2E tests (requires Chromium)
go test ./tests/e2e/...

# With verbose output
go test ./... -v
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BROWSE_SCOPE` | `all` | Default command scopes |
| `BROWSE_RATE_LIMIT` | `on` | `off` to disable |
| `BROWSE_RATE_LIMIT_RPS` | `10` | Rate limit tokens/sec |
| `BROWSE_RATE_LIMIT_BURST` | `20` | Burst capacity |
| `BROWSE_CONTENT_FILTER` | `warn` | URL filter: `off`/`warn`/`block` |
| `GSTACK_SECURITY_OFF` | `0` | `1` to disable ML classifiers |
| `GSTACK_HAIKU_OFF` | `0` | `1` to disable Haiku transcript classifier |
| `GSTACK_HAIKU_TIMEOUT_MS` | `45000` | Haiku CLI timeout (milliseconds) |

## License

Same as upstream gstack.
