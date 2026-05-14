---
name: benchmark
description: stub fixture for benchmark
slash: /benchmark
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

# benchmark

gpowers-browser open, then gpowers-browser eval with `JSON.stringify(performance.getEntriesByType("navigation"))`. The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
