---
name: design-review
description: post-implementation visual review (gstack)
slash: /design-review
namespace: roles
upstream: gstack@main
requires-driver: browser
---

## Preamble (auto)

Before any browser verb call, source the driver selector:

```bash
source "$GPOWERS_HOME/tools/drivers/browser/select-driver.sh"
```

This exports `GPOWERS_BROWSER_DRIVER`. All browser interactions in this skill use `gpowers-browser <verb>` and never reference a specific MCP server or CLI tool by name.

# design-review

Visual walkthrough:
1. Use `gpowers-browser open` to the staging URL.
2. `gpowers-browser screenshot` for screenshots.
3. `gpowers-browser read` for DOM check.
On non-CC: `gpowers-browser` (driver auto-selected).

State: $(gpowers-path data)/design-review/<slug>.md
