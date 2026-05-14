---
name: canary
description: stub fixture for canary
slash: /canary
namespace: tools
upstream: gstack@main
---

# canary

Post-deploy: gpowers-browser open to canary URL, then gpowers-browser eval to check `window.__version`. Non-CC: `gpowers-browser` (driver auto-selected).
