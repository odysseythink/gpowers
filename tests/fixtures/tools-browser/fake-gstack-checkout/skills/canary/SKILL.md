---
name: canary
description: stub fixture for canary
slash: /canary
---

# canary

Post-deploy: mcp__claude-in-chrome__navigate to canary URL, then mcp__claude-in-chrome__javascript_tool to check `window.__version`. Non-CC: `npx playwright test canary.spec.js`.
