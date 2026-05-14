---
name: context-restore
description: stub fixture for gstack context-restore
slash: /context-restore
namespace: tools
upstream: gstack@main
---

# context-restore

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-context-restore` when needed. Cache lives under $(gpowers-path cache).
