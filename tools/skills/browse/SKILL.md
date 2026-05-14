---
name: browse
description: stub fixture for browse
slash: /browse
namespace: tools
upstream: gstack@main
requires-driver: browser
requires-driver: browser
---

## Preamble (auto)

Before any browser verb call, source the driver selector:

```bash
source "$GPOWERS_HOME/tools/drivers/browser/select-driver.sh"
```

This exports `GPOWERS_BROWSER_DRIVER`. All browser interactions in this skill use `gpowers-browser <verb>` and never reference a specific MCP server or CLI tool by name.

# browse


1. Use MCP tool `gpowers-browser open` with URL.
2. Call `gpowers-browser read` for the text content.
3. Close with `gpowers-browser close`.
On non-Claude-Code platforms, fall back: `gpowers-browser` (driver auto-selected).

