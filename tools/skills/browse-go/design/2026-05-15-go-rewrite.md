# browse Go Rewrite — Design Document

**Status:** Phase A-D complete (Security Stack + Scoped Tokens + Rate Limiting + Audit Logging + Activity Streaming + Performance + Domain Skills + SOCKS5 Bridge + Browser Skills)

## Goal
Replace the TypeScript/Bun/Playwright browse stack with a pure Go/chromedp implementation. Single static binary, zero Node/Bun runtime dependency.

## Why chromedp (not playwright-go)
- browse uses **Chromium only** — zero Firefox/WebKit references in production code
- chromedp is **pure Go + direct CDP**, no Node bridge
- 12.5k stars, actively maintained by Google ecosystem
- playwright-go is community-maintained and "looking for maintainers"

## Architecture Overview

```
┌─────────────────┐     HTTP/JSON      ┌─────────────────────────────┐
│   browse CLI    │ ◄────────────────► │   browse server (Go)        │
│   (cmd/cli)     │                    │   (net/http)                │
└─────────────────┘                    │                             │
                                       │  ┌─────────────────────┐    │
                                       │  │  Command Router     │    │
                                       │  │  (pkg/commands)     │    │
                                       │  └──────────┬──────────┘    │
                                       │             │               │
                                       │  ┌──────────▼──────────┐    │
                                       │  │  Security Pipeline  │    │
                                       │  │  (pkg/security)     │    │
                                       │  │  L1-L6 + Audit      │    │
                                       │  └──────────┬──────────┘    │
                                       │             │               │
                                       │  ┌──────────▼──────────┐    │
                                       │  │  BrowserManager     │    │
                                       │  │  (pkg/browser)      │    │
                                       │  │  chromedp contexts  │    │
                                       │  └──────────┬──────────┘    │
                                       │             │               │
                                       │  ┌──────────▼──────────┐    │
                                       │  │  Chromium (CDP)     │    │
                                       │  └─────────────────────┘    │
                                       └─────────────────────────────┘
```

## Completed Features

### Phase 1: Bootstrap ✅
- `go mod init browse-go`
- Dependencies: `chromedp/chromedp`, `sergi/go-diff`, `x/net`
- Directory skeleton
- Ported `platform.ts`, `config.ts`, `error-handling.ts`
- Ported `CircularBuffer` from `buffers.ts`

### Phase 2: Browser Core ✅
- `BrowserManager` with chromedp
- Launch / stop / restart Chromium
- Tab management (create, close, switch, list)
- Context isolation (cookies, storage, userAgent)
- Stealth injection (`navigator.webdriver` mask + launch args)
- Dialog auto-accept
- Screenshot + PDF
- Network capture (request/response buffers)

### Phase 3: Read Commands ✅
- text, html, links, forms, accessibility, attrs
- console, network, cookies, storage, perf
- dialog, is, inspect, media, data

### Phase 4: Write Commands ✅
- goto, back, forward, reload, load-html
- click, fill, select, hover, type, press, scroll, wait
- viewport, cookie, **cookie-import**, header, useragent
- upload, dialog-accept, dialog-dismiss
- style, cleanup, prettyscreenshot
- download, scrape, archive

### Phase 5: Meta Commands ✅
- tabs, tab, tab-each, newtab, closetab
- screenshot, pdf, responsive
- chain, diff
- url, snapshot
- connect, disconnect, focus
- inbox, watch, state, frame
- ux-audit, cdp

### Phase A: Security Stack (L1-L6) ✅
| Layer | Feature | File |
|-------|---------|------|
| L1 | Content envelope with zero-width watermark | `pkg/security/envelope.go` |
| L2 | Hidden element stripping (opacity/font-size/off-screen/clip/ARIA) | `pkg/security/dom_strip.go` |
| L3 | URL blacklist filter (10 exfiltration domains) | `pkg/security/url_filter.go` |
| L4 | ML classifier interface + rule-based fallback + ONNX bridge stub | `pkg/security/classifier*.go` |
| L5 | Canary token generation/injection/detection | `pkg/security/canary.go` |
| L6 | Ensemble verdict combiner (2-of-N + SO-FP mitigation) | `pkg/security/verdict.go` |
| — | Security pipeline orchestrator | `pkg/security/pipeline.go` |
| — | Session state + decision files | `pkg/security/state.go` |
| — | Attack audit log (JSONL + rotation + device salt) | `pkg/security/attempt_log.go` |
| — | Python ONNX inference script | `scripts/onnx_classifier.py` |

### Phase B: Scoped Token + Rate Limit + Audit Logging ✅
| Feature | Description |
|---------|-------------|
| Scoped Token Registry | 6 scope categories (`navigate`, `read`, `interact`, `write`, `inspect`, `system`) — all 69+ commands mapped |
| Rate Limiting | Token-bucket per `(clientID, command)`, configurable via env |
| Audit Logging | Full command audit (`~/.gstack/security/audit.jsonl`), queryable via `/audit` + `/audit-stats` |

### Phase C: API Alignment + Performance + Activity Streaming ✅
| Feature | Description |
|---------|-------------|
| cookie-import | Import cookies from JSON file (Playwright format) |
| Activity Streaming | SSE `/activity/stream` + REST `/activity/history` for Chrome extension Side Panel |
| URL Cache | Cached in `BrowserManager` via CDP nav events — eliminates round-trips |
| Reduced CurrentURL() calls | Single capture per command in registry |

### Phase D: Domain Skills + SOCKS5 Bridge + Browser Skills ✅
| Feature | Description |
|---------|-------------|
| domain-skill | Per-site notes with 3-state lifecycle (quarantined → active → global), JSONL append-only storage |
| SOCKS5 Bridge | Local unauthenticated SOCKS5 listener relaying through authenticated upstream proxy |
| skill | Browser skills — 3-tier lookup (project > global > bundled), SKILL.md frontmatter parsing |
| Command registry | `domain-skill` and `skill` commands wired into HTTP API |

## Module Mapping

| TS Source | Lines | Go Package | Status |
|-----------|-------|------------|--------|
| `browser-manager.ts` | ~600 | `pkg/browser` | ✅ Complete |
| `server.ts` | ~800 | `pkg/server` | ✅ Core routes + activity + audit |
| `cli.ts` | ~400 | `cmd/browse` | ✅ Complete |
| `read-commands.ts` | ~544 | `pkg/commands/reading.go` | ✅ Complete |
| `write-commands.ts` | ~1433 | `pkg/commands/write.go` | ✅ Complete |
| `meta-commands.ts` | ~1151 | `pkg/commands/meta.go` + `snapshot.go` + `visual.go` + `inspection.go` | ✅ Complete |
| `snapshot.ts` | ~300 | `pkg/commands/snapshot.go` | ✅ Complete |
| `cdp-bridge.ts` | ~200 | `pkg/commands/cdp.go` | ✅ Complete |
| `cdp-allowlist.ts` | ~150 | `pkg/commands/cdpallowlist.go` | ✅ Complete |
| `network-capture.ts` | ~179 | `pkg/browser/network.go` | ✅ Complete |
| `content-security.ts` | ~200 | `pkg/security/*.go` (Phase A) | ✅ Complete |
| `token-registry.ts` | ~150 | `pkg/security/scope.go` + `ratelimit.go` (Phase B) | ✅ Complete |
| `audit.ts` | ~200 | `pkg/security/audit.go` (Phase B) | ✅ Complete |
| `activity.ts` | ~180 | `pkg/activity/activity.go` (Phase C) | ✅ Complete |
| `security.ts` | ~300 | `pkg/security/*.go` (Phase A) | ✅ Complete |
| `security-classifier.ts` | ~400 | `pkg/security/classifier*.go` (Phase A) | ✅ Complete (TestSavant ONNX + Haiku CLI bridge) |

## Remaining Gaps (not yet ported)

All originally-identified gaps have been implemented. The Go port is feature-complete relative to the TS upstream.

| Feature | Status | Notes |
|---------|--------|-------|
| cookie-import-browser | ✅ Complete | Cross-platform decryption + CDP fallback + picker UI |
| handoff (headless→headed) | ✅ Complete | `handoff`, `resume`, `connect`, `disconnect`, `focus` |
| domain-skill | ✅ Complete | 3-state lifecycle with JSONL storage |
| skill (browser skills) | ✅ Complete | 3-tier lookup (project > global > bundled) |
| SOCKS bridge | ✅ Complete | Local unauth → upstream auth SOCKS5 relay |
| XVFB support | ✅ Complete | Linux virtual display auto-spawn |
| Cookie picker UI | ✅ Complete | `/cookie-picker` HTTP endpoint with HTML UI |
| Tunnel server | ✅ Complete | ngrok integration + tunnel surface with allowlist |
| Pair agent | ✅ Complete | `/pair`, `/token`, `/agents` endpoints |
| network capture flags | ✅ Complete | `--capture`, `--export`, `--bodies`, `--filter` |
| PDF TOC | ✅ Complete | JS-based TOC injection via `injectToc`/`cleanupToc` |
| Windows terminal agent | ✅ Complete | ConPTY-based WebSocket PTY (Windows 10 1809+) |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BROWSE_SCOPE` | `all` | Default command scopes (comma-separated) |
| `BROWSE_RATE_LIMIT` | `on` | `off` to disable rate limiting |
| `BROWSE_RATE_LIMIT_RPS` | `10` | Tokens per second |
| `BROWSE_RATE_LIMIT_BURST` | `20` | Burst capacity |
| `BROWSE_CONTENT_FILTER` | `warn` | URL filter mode: `off`/`warn`/`block` |
| `GSTACK_SECURITY_OFF` | `0` | `1` to disable ML classifiers |

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Server health + security status + rate limit config |
| `/command` | POST | Yes | Execute single command |
| `/batch` | POST | Yes | Execute multiple commands |
| `/refs` | GET | Yes | Current page ref map |
| `/tabs` | GET | Yes | List all tabs |
| `/file` | GET | Yes | Serve temp file |
| `/stop` | POST | Yes | Shutdown server |
| `/audit` | GET | Yes | Query audit log |
| `/audit-stats` | GET | Yes | Audit statistics |
| `/activity/stream` | GET | Yes | SSE activity stream |
| `/activity/history` | GET | Yes | Activity history (REST) |

## Test Coverage

- **pkg/security**: 68 tests (all layers + Haiku classifier)
- **pkg/commands**: ~82 tests
- **pkg/server**: ~35 tests
- **pkg/activity**: 6 tests
- **pkg/browser**: ~12 tests
- **pkg/domainskill**: 24 tests
- **pkg/socks**: 21 tests
- **pkg/browserskill**: 17 tests
- **pkg/buffers**: 2 tests
- **pkg/terminal**: 3 tests
- **tests/e2e**: 8 e2e tests (launches real Chromium)
- **Total**: ~320 Test* functions, all passing

## Line Count

| Category | Lines |
|----------|-------|
| Source (pkg/) | ~10,800 |
| Tests | ~3,300 |
| Scripts (Python ONNX) | ~100 |
| **Total** | **~14,200** |

## Risks — Updated

| Risk | Status |
|------|--------|
| chromedp API gaps | Mitigated — custom implementations where needed |
| Cookie decryption complexity | ✅ Complete — cross-platform (macOS/Windows/Linux) |
| Security classifier | ✅ Complete — L1-L6 all layers + TestSavant ONNX + Haiku CLI |
| Upstream sync loss | Hard fork confirmed — manual porting required |
| Regression in edge cases | 8 e2e tests + ~320 unit tests covering core paths |

## Decision

**Hard fork accepted.** The Go rewrite is a permanent divergence from the TS upstream. Manual porting of upstream features will be required going forward. The tradeoff is justified by:
- Single static binary (no Node/Bun runtime)
- Better memory efficiency
- Native cross-compilation
- Direct CDP access without Playwright bridge overhead
