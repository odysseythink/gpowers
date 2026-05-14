---
name: context-save
description: stub fixture for gstack context-save
slash: /context-save
namespace: tools
upstream: gstack@main
---

# context-save

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-context-save` when needed. Cache lives under $(gpowers-path cache).
