import { chromium } from 'playwright';
import { createReadStream, createWriteStream } from 'node:fs';
import readline from 'node:readline';

const [reqPath, resPath, url, vw, vh] = process.argv.slice(2);
const browser = await chromium.launch();
const context = await browser.newContext({ viewport: { width: +vw, height: +vh } });
const page = await context.newPage();
const consoleLog = [];
page.on('console', m => consoleLog.push(`[${m.type()}] ${m.text()}`));
await page.goto(url);

const out = createWriteStream(resPath);
const rl = readline.createInterface({ input: createReadStream(reqPath) });

rl.on('line', async line => {
  const req = JSON.parse(line);
  try {
    let result;
    switch (req.verb) {
      case 'click':
        await page.locator(req.selector).click({ timeout: req.timeout_ms || 5000 });
        result = { ok: true };
        break;
      case 'type':
        await page.locator(req.selector).fill(req.text);
        result = { ok: true };
        break;
      case 'read':
        if (req.mode === 'text')
          result = { content: req.selector ? await page.textContent(req.selector) : await page.innerText('body') };
        else if (req.mode === 'dom')
          result = { content: req.selector ? await page.locator(req.selector).innerHTML() : await page.content() };
        else
          result = { content: consoleLog.join('\n') };
        break;
      case 'screenshot': {
        const p = `${process.env.GPOWERS_CACHE}/browser/shots/${Date.now()}.png`;
        await page.screenshot({ path: p, fullPage: req.region === 'full' });
        result = { path: p };
        break;
      }
      case 'wait':
        if (req.condition.startsWith('selector:'))
          await page.waitForSelector(req.condition.slice(9), { timeout: req.timeout_ms || 30000 });
        else if (req.condition === 'network-idle')
          await page.waitForLoadState('networkidle', { timeout: req.timeout_ms || 30000 });
        else
          await page.waitForLoadState('load', { timeout: req.timeout_ms || 30000 });
        result = { ok: true };
        break;
      case 'eval':
        result = { value: await page.evaluate(req.code) };
        break;
      case 'cookies':
        if (req.op === 'get')
          result = { cookies: await context.cookies(req.domain ? [`https://${req.domain}/`] : undefined) };
        else if (req.op === 'set') {
          await context.addCookies(req.cookies);
          result = { ok: true };
        } else {
          await context.clearCookies();
          result = { ok: true };
        }
        break;
      case 'close':
        await context.close();
        await browser.close();
        result = { ok: true };
        out.write(JSON.stringify(result) + '\n');
        process.exit(0);
      default:
        result = { error: `unknown verb: ${req.verb}` };
    }
    out.write(JSON.stringify(result) + '\n');
  } catch (e) {
    out.write(JSON.stringify({ ok: false, error: String(e) }) + '\n');
  }
});
