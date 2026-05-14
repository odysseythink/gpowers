---
name: browse
description: stub fixture for browse
slash: /browse
---

# browse


1. Use MCP tool `mcp__claude-in-chrome__tabs_create_mcp` with URL.
2. Call `mcp__claude-in-chrome__read_page` for the text content.
3. Close with `mcp__claude-in-chrome__tabs_close_mcp`.
On non-Claude-Code platforms, fall back: `npx playwright open https://example.com`.

