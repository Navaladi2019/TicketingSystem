// ADR-007 S4: preflight + GET/POST/PATCH from a page at an allowed origin,
// with the five D3 headers, reading status and body of 200/401/403/429.
// usage: node run.mjs <apiBase> <label>
import http from 'node:http';
import { chromium, firefox, webkit } from 'playwright';

const api = process.argv[2] ?? 'http://127.0.0.1:8087';
const label = process.argv[3] ?? '';
const PAGE_ORIGIN = 'http://127.0.0.1:9001'; // listed in the seed's allowed_origins
const BAD_ORIGIN = 'http://127.0.0.1:9002';

// Real pages on loopback (a page faked by request interception is classed as a
// public address space and Chromium's Local Network Access blocks it, which is a
// harness artefact, not a CORS result).
for (const port of [9001, 9002]) {
  http.createServer((_, res) => { res.setHeader('Content-Type', 'text/html'); res.end('<html></html>'); }).listen(port, '127.0.0.1');
}

const cases = [
  { name: '200', token: 'tok-ok', origin: PAGE_ORIGIN, status: 200 },
  { name: '401', token: 'tok-nope', origin: PAGE_ORIGIN, status: 401 },
  { name: '403 origin_not_allowed', token: 'tok-ok', origin: BAD_ORIGIN, status: 403, code: 'origin_not_allowed' },
  { name: '403 client_not_enabled', token: 'tok-off', origin: PAGE_ORIGIN, status: 403, code: 'client_not_enabled' },
  { name: '429', token: 'tok-limited', origin: PAGE_ORIGIN, status: 429, retryAfter: '7' },
];
const methods = ['GET', 'POST', 'PATCH'];

// The page runs in the browser; it is served at a fake origin through route interception.
async function probe(page, base, c, method) {
  return page.evaluate(async ({ base, c, method }) => {
    const headers = {
      Authorization: 'Bearer ' + c.token, 'Content-Type': 'application/json',
      'Idempotency-Key': 'k-1', 'If-Match': '"v1"', 'Nava-SDK': 'ts/0.0.0',
    };
    try {
      const r = await fetch(base + '/api/v1/records/loan', {
        method, headers, body: method === 'GET' ? undefined : '{"a":1}',
      });
      let body = null; try { body = await r.json(); } catch {}
      return { ok: true, status: r.status, body, retryAfter: r.headers.get('Retry-After'), etag: r.headers.get('ETag') };
    } catch (e) { return { ok: false, error: `${e.name}: ${e.message}` }; }
  }, { base, c, method });
}

const results = [];
let failures = 0;
for (const [name, type] of [['chromium', chromium], ['firefox', firefox], ['webkit', webkit]]) {
  const browser = await type.launch();
  for (const c of cases) {
    // The page's origin is the case's Origin.
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto(c.origin + '/');
    for (const m of methods) {
      const r = await probe(page, api, c, m);
      let pass = r.ok && r.status === (m === 'POST' && c.status === 200 ? 201 : c.status);
      if (pass && c.code) pass = r.body?.code === c.code;
      if (pass && c.retryAfter) pass = r.retryAfter === c.retryAfter;
      if (!pass) failures++;
      results.push({ engine: name, case: c.name, method: m, pass, status: r.status, code: r.body?.code, retryAfter: r.retryAfter, error: r.error });
    }
    await ctx.close();
  }
  await browser.close();
}
console.log(`mode=${label}`);
console.log('| engine | case | method | result | status | code | Retry-After | error |');
console.log('|---|---|---|---|---|---|---|---|');
for (const r of results) {
  console.log(`| ${r.engine} | ${r.case} | ${r.method} | ${r.pass ? 'PASS' : 'FAIL'} | ${r.status ?? '-'} | ${r.code ?? '-'} | ${r.retryAfter ?? '-'} | ${r.error ?? '-'} |`);
}
console.log(`\n${results.length - failures}/${results.length} pass`);
process.exit(failures ? 1 : 0);
