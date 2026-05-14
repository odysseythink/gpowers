---
name: setup-browser-cookies
description: stub fixture for setup-browser-cookies
slash: /setup-browser-cookies
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

# setup-browser-cookies

gpowers-browser open, run gpowers-browser eval to set document.cookie. Non-CC: `gpowers-browser` (driver auto-selected).
