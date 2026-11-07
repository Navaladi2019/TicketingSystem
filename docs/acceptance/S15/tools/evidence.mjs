// S15 acceptance (TICK-443): turns the Bruno JSON reports run-kit.sh copied
// into <evidence>/.bruno-<folder>.json into one evidence file per request,
// <evidence>/<request>.json, with the request and the response and no
// credentials: Authorization headers, form bodies (passwords) and token
// responses are left out. Writes <evidence>/run-summary.json with every test.
//
// Usage: node docs/acceptance/S15/tools/evidence.mjs docs/acceptance/S15/evidence
import { readdirSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';

const dir = process.argv[2];
const summary = [];
let failed = 0;
for (const f of readdirSync(dir).filter((n) => /^\.bruno-.*\.json$/.test(n)).sort()) {
  const raw = JSON.parse(readFileSync(join(dir, f), 'utf8'));
  const results = (Array.isArray(raw) ? raw.flatMap((it) => it.results || []) : raw.results) || [];
  for (const r of results) {
    const file = (r.test && r.test.filename) || r.name || 'request';
    const name = file.replace(/^.*[\\/]/, '').replace(/\.bru$/, '');
    const tests = [...(r.testResults || []), ...(r.assertionResults || [])].map((t) => ({ test: t.description, status: t.status, ...(t.error ? { error: t.error } : {}) }));
    failed += tests.filter((t) => t.status !== 'pass').length;
    summary.push({ request: name, status: r.response && r.response.status, tests });
    const isToken = /openid-connect\/token/.test((r.request && r.request.url) || '');
    if (isToken) continue; // sign-ins: a password in, a token out
    const req = r.request || {};
    const headers = Object.fromEntries(Object.entries(req.headers || {}).filter(([k]) => k.toLowerCase() !== 'authorization'));
    const out = {
      request: { method: req.method, url: req.url, headers, body: req.data ?? null },
      response: { status: r.response && r.response.status, body: r.response && r.response.data },
      tests,
    };
    writeFileSync(join(dir, name + '.json'), JSON.stringify(out, null, 2) + '\n');
  }
  rmSync(join(dir, f));
}
writeFileSync(join(dir, 'run-summary.json'), JSON.stringify({ requests: summary.length, tests: summary.reduce((n, s) => n + s.tests.length, 0), failed, results: summary }, null, 2) + '\n');
console.log(`evidence: ${summary.length} requests, ${failed} failed tests`);
if (failed) process.exit(1);
