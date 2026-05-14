---
name: benchmark-models
description: stub fixture for gstack benchmark-models
slash: /benchmark-models
namespace: tools
upstream: gstack@main
---

# benchmark-models

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-benchmark-models` when needed. Cache lives under $(gpowers-path cache).
