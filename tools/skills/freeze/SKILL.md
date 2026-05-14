---
name: freeze
description: stub fixture for gstack freeze
slash: /freeze
namespace: tools
upstream: gstack@main
---

# freeze

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-freeze` when needed. Cache lives under $(gpowers-path cache).
