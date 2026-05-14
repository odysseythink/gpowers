---
name: sync-gbrain
description: stub fixture for sync-gbrain
slash: /sync-gbrain
namespace: tools
upstream: gstack@main
requires-driver: browser
---

## Preamble (auto)

Before any browser verb call, source the driver selector:

```bash
source "$GPOWERS_HOME/tools/drivers/browser/select-driver.sh"
```

This exports `GPOWERS_BROWSER_DRIVER`. All browser interactions in this skill use `gpowers-browser <verb>` and never reference a specific MCP server or CLI tool by name.

# sync-gbrain

Periodic: gpowers-browser open, gpowers-browser eval window.__sync(). The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
