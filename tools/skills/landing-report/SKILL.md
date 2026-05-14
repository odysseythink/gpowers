---
name: landing-report
description: stub fixture for gstack landing-report
slash: /landing-report
namespace: tools
upstream: gstack@main
---

# landing-report

This skill writes state to $(gpowers-path state) and reads from $(gpowers-path config).
It invokes `gpowers-landing-report` when needed. Cache lives under $(gpowers-path cache).
