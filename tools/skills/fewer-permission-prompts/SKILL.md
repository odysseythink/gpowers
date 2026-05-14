---
name: fewer-permission-prompts
description: stub fixture for gstack fewer-permission-prompts
slash: /fewer-permission-prompts
namespace: tools
upstream: gstack@main
---

# fewer-permission-prompts

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-fewer-permission-prompts` when needed. Cache lives under $(gpowers-path cache).
