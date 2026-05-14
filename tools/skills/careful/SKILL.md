---
name: careful
description: stub fixture for gstack careful
slash: /careful
namespace: tools
upstream: gstack@main
---

# careful

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-careful` when needed. Cache lives under $(gpowers-path cache).
