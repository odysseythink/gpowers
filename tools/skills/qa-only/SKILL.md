---
name: qa-only
description: stub fixture for qa-only
slash: /qa-only
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

# qa-only

Use gpowers-browser read and gpowers-browser screenshot for screenshots only. No interaction. Non-CC: `gpowers-browser` (driver auto-selected).
