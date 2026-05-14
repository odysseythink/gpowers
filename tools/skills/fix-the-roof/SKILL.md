---
name: fix-the-roof
description: stub fixture for gstack fix-the-roof
slash: /fix-the-roof
namespace: tools
upstream: gstack@main
---

# fix-the-roof

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-fix-the-roof` when needed. Cache lives under $(gpowers-path cache).
