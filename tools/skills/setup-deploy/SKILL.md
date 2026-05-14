---
name: setup-deploy
description: stub fixture for gstack setup-deploy
slash: /setup-deploy
namespace: tools
upstream: gstack@main
---

# setup-deploy

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-setup-deploy` when needed. Cache lives under $(gpowers-path cache).
