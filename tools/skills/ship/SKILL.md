---
name: ship
description: stub fixture for gstack ship
slash: /ship
namespace: tools
upstream: gstack@main
---

# ship

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-ship` when needed. Cache lives under $(gpowers-path cache).
