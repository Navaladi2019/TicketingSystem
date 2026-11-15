import http from 'node:http';
import { readFileSync } from 'node:fs';
import { chromium, firefox, webkit } from '@playwright/test';
const CSP = "script-src 'self'; style-src 'self'";
const srv = http.createServer((q, r) => {
  const p = q.url.split('?')[0];
  const h = { 'Content-Security-Policy': CSP };
  if (p === '/bundle.js' || p === '/probe.js') { h['Content-Type'] = 'text/javascript'; r.writeHead(200, h); r.end(readFileSync(p === '/probe.js' ? 'probe.js' : 'dist/bundle.js')); }
  else { h['Content-Type'] = 'text/html'; r.writeHead(200, h); r.end(readFileSync('index.html')); }
}).listen(0, '127.0.0.1');
await new Promise(r => srv.once('listening', r)); const port = srv.address().port;
const attempts = (process.argv[2] || '1,2').split(',');
let failed = false;
for (const attempt of attempts) for (const [name, bt] of Object.entries({ chromium, firefox, webkit }).filter(([n]) => !process.argv[3] || process.argv[3].split(',').includes(n))) {
  const b = await bt.launch(); const pg = await b.newPage(); pg.setDefaultTimeout(4000); pg.setDefaultNavigationTimeout(30000);
  const cons = []; pg.on('console', m => m.type() === 'error' && cons.push(m.text()));
  await pg.goto(`http://127.0.0.1:${port}/?attempt=${attempt}`);
  await pg.waitForTimeout(800);
  const R = {};
  const inShadow = sel => pg.evaluate(s => !!document.querySelector('nava-records').shadowRoot.querySelector(s), sel);
  R.rendered = await inShadow('select');
  R.oneOfOptions = await pg.evaluate(() => [...document.querySelector('nava-records').shadowRoot.querySelectorAll('select')].map(s => [...s.options].map(o => o.label || o.text).join('|')));
  const rr = () => pg.evaluate(() => [...document.querySelector('nava-records').shadowRoot.querySelectorAll('label')].some(l => l.textContent.includes('Rejection reason') && l.getClientRects().length > 0));
  R.hiddenBefore = !(await rr());
  try {
    const state = pg.locator('nava-records select').nth(1);
    await state.selectOption('rejected'); await pg.waitForTimeout(200);
    R.shownAfter = await rr();
    await state.selectOption('approved'); await pg.waitForTimeout(200);
    R.hiddenAgain = !(await rr());
  } catch (e) { R.ruleError = e.message.split('\n')[0]; }
  R.selectStyled = await pg.evaluate(() => { const s = document.querySelector('nava-records').shadowRoot.querySelector('select'); return s ? getComputedStyle(s).borderTopColor : null; });
  R.bodyChildren = await pg.evaluate(() => [...document.body.children].map(e => e.tagName).join(','));
  R.violations = await pg.evaluate(() => window.__v); R.pageErrors = await pg.evaluate(() => window.__errs); R.consoleErrors = cons.slice(0, 3);
  const pass = R.violations.length === 0 && R.rendered && R.hiddenBefore && R.shownAfter && R.hiddenAgain && R.selectStyled === 'rgb(0, 128, 0)';
  console.log(`attempt ${attempt} ${name}: ${pass ? 'PASS' : 'FAIL'}`, JSON.stringify(R));
  if (!pass) failed = true;
  await b.close();
}
srv.close(); process.exit(failed ? 1 : 0);
