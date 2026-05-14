---
name: qa
description: stub fixture for qa
slash: /qa
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

# qa


- Navigate: gpowers-browser open
- Type: gpowers-browser type
- Click: gpowers-browser wait (condition: selector:<css>) then click
- Screenshot: gpowers-browser screenshot (action: screenshot)
- Console: gpowers-browser read (mode: console)
Non-CC: use `gpowers-browser` (driver auto-selected) with custom script.

