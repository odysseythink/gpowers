---
name: health
description: stub fixture for gstack health
slash: /health
namespace: tools
upstream: gstack@main
---

# health

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-health` when needed. Cache lives under $(gpowers-path cache).
