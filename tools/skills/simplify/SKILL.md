---
name: simplify
description: stub fixture for gstack simplify
slash: /simplify
namespace: tools
upstream: gstack@main
---

# simplify

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-simplify` when needed. Cache lives under $(gpowers-path cache).
