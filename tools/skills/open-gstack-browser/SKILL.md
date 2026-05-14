---
name: open-gstack-browser
description: stub fixture for open-gstack-browser
slash: /open-gstack-browser
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

# open-gpowers-browser

Starts persistent Chromium via gpowers-browser open with profile dir $(gpowers-path cache)/chromium-profile. Non-CC: `gpowers-browser` (driver auto-selected).
