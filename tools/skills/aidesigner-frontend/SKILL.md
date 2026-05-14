---
name: aidesigner-frontend
description: stub fixture for aidesigner-frontend
slash: /aidesigner-frontend
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

# aidesigner-frontend

Full design+ship pipeline: gpowers-browser open, gpowers-browser type prompt, gpowers-browser wait + gpowers-browser click generate, gpowers-browser screenshot, gpowers-browser eval result, close. The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
