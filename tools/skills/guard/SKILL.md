---
name: guard
description: stub fixture for gstack guard
slash: /guard
namespace: tools
upstream: gstack@main
---

# guard

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-guard` when needed. Cache lives under $(gpowers-path cache).
