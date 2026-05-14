---
name: make-pdf
description: stub fixture for gstack make-pdf
slash: /make-pdf
namespace: tools
upstream: gstack@main
---

# make-pdf

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-make-pdf` when needed. Cache lives under $(gpowers-path cache).
