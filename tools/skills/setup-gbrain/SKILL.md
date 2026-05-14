---
name: setup-gbrain
description: stub fixture for setup-gbrain
slash: /setup-gbrain
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

# setup-gbrain

gpowers-browser open to gbrain.app/onboard, gpowers-browser type for email, gpowers-browser wait (condition: selector:<css>) + click submit. The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
