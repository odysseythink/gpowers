# playwright-cli driver

Maps gpowers 9-verb interface to Playwright CLI / Node API. Each `browser.open` spawns a detached Node process running a per-tab event loop; tab_id is the registry entry storing the FIFOs path. Each subsequent verb writes a JSON request to the runner FIFO and reads the JSON response.

Requires: `bun add -g @playwright/test` or `npm install -g playwright` plus browsers (`npx playwright install chromium`).

| gpowers verb | playwright API |
|---|---|
| open | browser.newContext + context.newPage + page.goto |
| click | page.locator(sel).click |
| type | page.locator(sel).fill |
| read | page.textContent / page.content / on('console') buffer |
| screenshot | page.screenshot |
| wait | page.waitForSelector / waitForLoadState |
| eval | page.evaluate |
| cookies | context.cookies / addCookies / clearCookies |
| close | context.close + tab_release |
