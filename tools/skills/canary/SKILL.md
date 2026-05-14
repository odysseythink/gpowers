---
name: canary
description: stub fixture for canary
slash: /canary
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

# canary

Post-deploy: gpowers-browser open to canary URL, then gpowers-browser eval to check `window.__version`. Non-CC: `gpowers-browser` (driver auto-selected).
