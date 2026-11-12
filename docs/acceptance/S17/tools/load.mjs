// S17 acceptance, TICK-778 carry-over checks: the API side of readiness.sh.
//
//   node load.mjs first <out.json>
//     One tenant B (gaps) create, timed, no retry: the first create after a
//     cold restart. Fails unless 201.
//
//   node load.mjs saturate <out.json> <seconds> <clients>
//     <clients> loops of tenant B creates for <seconds>, no retry, to keep
//     every allocator connection busy. Writes a status histogram.
//
// Only the dev-only seed user bob@tenant-b.test of
// docs/acceptance/guide/bruno/environments/local.bru. No token is written.
import { writeFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';

const [mode, out, secs = '20', clients = '40'] = process.argv.slice(2);
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;

async function token() {
  const r = await fetch(`${KC}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username: 'bob@tenant-b.test', password: 'dev-only-bob' }),
  });
  if (r.status !== 200) throw new Error(`sign-in: HTTP ${r.status}`);
  return (await r.json()).access_token;
}

async function create(tok, label) {
  const t0 = performance.now();
  const r = await fetch(`${CORE}/api/v1/records/incident`, {
    method: 'POST',
    headers: { Authorization: 'Bearer ' + tok, 'Content-Type': 'application/json', 'Idempotency-Key': `s17-778-${randomUUID()}` },
    body: JSON.stringify({ data: { short_description: `S17 TICK-778 ${label}` } }),
  });
  const body = await r.json().catch(() => null);
  return { status: r.status, ms: Math.round(performance.now() - t0), number: body?.number, code: body?.code, detail: body?.detail, request: r.headers.get('X-Request-Id') };
}

const tok = await token();
if (mode === 'first') {
  const res = { at: new Date().toISOString(), ...(await create(tok, 'first after cold restart')) };
  writeFileSync(out, JSON.stringify(res, null, 2) + '\n');
  console.log(`first create: HTTP ${res.status} ${res.number ?? res.code} in ${res.ms} ms (request wall time)`);
  process.exit(res.status === 201 ? 0 : 1);
}
if (mode === 'saturate') {
  const end = Date.now() + Number(secs) * 1000;
  const hist = {};
  let n = 0;
  await Promise.all(Array.from({ length: Number(clients) }, async (_, c) => {
    while (Date.now() < end) {
      const r = await create(tok, `saturate ${c}`).catch((e) => ({ status: 'error', detail: String(e) }));
      const k = `${r.status}${r.code ? ' ' + r.code : ''}`;
      hist[k] = (hist[k] || 0) + 1;
      n++;
    }
  }));
  const res = { clients: Number(clients), seconds: Number(secs), requests: n, statuses: hist };
  writeFileSync(out, JSON.stringify(res, null, 2) + '\n');
  console.log(`saturate: ${n} creates from ${clients} clients in ${secs}s: ${JSON.stringify(hist)}`);
}
