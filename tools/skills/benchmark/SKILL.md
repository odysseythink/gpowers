---
name: benchmark
description: stub fixture for benchmark
slash: /benchmark
namespace: tools
upstream: gstack@main
---

# benchmark

gpowers-browser open, then gpowers-browser eval with `JSON.stringify(performance.getEntriesByType("navigation"))`. The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh).
