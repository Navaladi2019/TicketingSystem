// S15 acceptance (TICK-443): writes incidents in tenants A (Alice) and B (Bob)
// at a steady rate, so the worker kill happens while both tenants have audit
// entries arriving. Each create writes one audit entry (record/create).
//
// Usage (from the repository root, Node.js 22):
//   node docs/acceptance/S15/tools/load.mjs <seconds> <per-tenant rps> <out.json>
// Ports come from the environment (NAVA_CORE_PORT, KEYCLOAK_PORT), as compose
// reads them. The dev-only seed passwords come from the Bruno environment
// docs/acceptance/guide/bruno/environments/local.bru. Tokens are never written.
import { readFileSync, writeFileSync } from 'node:fs';

const [secs = '40', rps = '15', out = 'load.json'] = process.argv.slice(2);
const core = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const kc = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const env = readFileSync('docs/acceptance/guide/bruno/environments/local.bru', 'utf8');
const envVar = (k) => (env.match(new RegExp(`^\\s*${k}:\\s*(.*)$`, 'm')) || [])[1]?.trim();
const run = Date.now().toString(36);

async function token(user, password) {
  const r = await fetch(`${kc}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username: user, password }),
  });
  if (r.status !== 200) throw new Error(`token for ${user}: HTTP ${r.status}`);
  return (await r.json()).access_token;
}

async function writer(name, user, password) {
  const tok = await token(user, password);
  const res = { tenant: name, created: [], statuses: {}, first_at: null, last_at: null };
  const stop = Date.now() + Number(secs) * 1000;
  const gap = 1000 / Number(rps);
  let i = 0;
  while (Date.now() < stop) {
    const t0 = Date.now();
    const key = `tick443-s15-load-${run}-${name}-${i}`;
    const r = await fetch(`${core}/api/v1/records/incident`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${tok}`, 'Idempotency-Key': key, 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: { short_description: `S15 load ${run} ${name} ${i}`, priority: 3 } }),
    });
    res.statuses[r.status] = (res.statuses[r.status] || 0) + 1;
    if (r.status === 201) {
      const b = await r.json();
      res.created.push(b.id);
      res.first_at ??= new Date().toISOString();
      res.last_at = new Date().toISOString();
      i++;
    } else if (r.status === 429) {
      await new Promise((ok) => setTimeout(ok, 1000 * Number(r.headers.get('retry-after') || 1)));
    } else {
      await r.text();
    }
    const wait = gap - (Date.now() - t0);
    if (wait > 0) await new Promise((ok) => setTimeout(ok, wait));
  }
  return res;
}

const results = await Promise.all([
  writer('A', envVar('alice_username'), envVar('alice_password')),
  writer('B', envVar('bob_username'), envVar('bob_password')),
]);
writeFileSync(out, JSON.stringify({ run, seconds: Number(secs), rps: Number(rps), results }, null, 2));
for (const r of results) console.log(`load: tenant ${r.tenant}: ${r.created.length} incidents created, statuses ${JSON.stringify(r.statuses)}`);
if (results.some((r) => r.created.length === 0 || Object.keys(r.statuses).some((s) => s !== '201' && s !== '429'))) process.exit(1);
