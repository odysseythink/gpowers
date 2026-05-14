---
name: qa
description: stub fixture for qa
slash: /qa
namespace: tools
upstream: gstack@main
---

# qa


- Navigate: gpowers-browser open
- Type: gpowers-browser type
- Click: gpowers-browser wait (condition: selector:<css>) then click
- Screenshot: gpowers-browser screenshot (action: screenshot)
- Console: gpowers-browser read (mode: console)
Non-CC: use `gpowers-browser` (driver auto-selected) with custom script.

