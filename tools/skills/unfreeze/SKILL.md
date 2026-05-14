---
name: unfreeze
description: stub fixture for gstack unfreeze
slash: /unfreeze
namespace: tools
upstream: gstack@main
---

# unfreeze

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-unfreeze` when needed. Cache lives under $(gpowers-path cache).
