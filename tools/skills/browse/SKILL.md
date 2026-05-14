---
name: browse
description: stub fixture for browse
slash: /browse
namespace: tools
upstream: gstack@main
---

# browse


1. Use MCP tool `gpowers-browser open` with URL.
2. Call `gpowers-browser read` for the text content.
3. Close with `gpowers-browser close`.
On non-Claude-Code platforms, fall back: `gpowers-browser` (driver auto-selected).

