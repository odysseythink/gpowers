---
name: land-and-deploy
description: stub fixture for gstack land-and-deploy
slash: /land-and-deploy
namespace: tools
upstream: gstack@main
---

# land-and-deploy

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-land-and-deploy` when needed. Cache lives under $(gpowers-path cache).
