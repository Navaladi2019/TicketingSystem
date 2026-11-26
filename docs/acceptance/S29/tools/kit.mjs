// S29 acceptance (TICK-754, formerly TICK-928): the API steps run-kit.sh
// needs around the seed.
//
//   node kit.mjs setup <evidence-dir>
//     As admin@tenant-a, on a fresh stack: publishes the next incident
//     version with `priority`'s template default removed and one new optional
//     property `s29_tier` (string), so the seeded records lack both and the
//     kit's own publish (Bruno S29.10) is the one that adds a default.
//
//   node kit.mjs held <evidence-dir>
//     After seed.sql: 50 tenant A incidents created through the API, each in
//     the service-desk group (Alice reads them), 25 with priority 3 and 25 with
//     priority 1; and, as admin@tenant-b, 5 tenant B incidents with priority 1.
//     Writes <evidence-dir>/held.env: s29_held_3 (the priority-3 ids) and
//     s29_sample_held (5 of each), for Bruno as --env-var. No credentials.
//
// Ports come from the same variables as the compose file and api-test.sh.
// Only the dev-only seed users of docs/acceptance/guide/bruno/environments/local.bru.
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';

const [mode, dir] = process.argv.slice(2);
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;

async function token(username, password) {
  const r = await fetch(`${KC}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username, password }),
  });
  if (r.status !== 200) throw new Error(`sign-in ${username}: HTTP ${r.status}`);
  return (await r.json()).access_token;
}

const log = [];
async function call(tok, method, path, body) {
  const h = { Authorization: 'Bearer ' + tok };
  if (body !== undefined) h['Content-Type'] = 'application/json';
  if (method !== 'GET') h['Idempotency-Key'] = 's29-setup-' + randomUUID();
  const r = await fetch(CORE + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  let json = null;
  try { json = JSON.parse(text); } catch { /* not JSON */ }
  const { Authorization, ...shown } = h;
  log.push({ request: { method, path, headers: shown, body: body ?? null }, response: { status: r.status, body: json ?? text } });
  return { status: r.status, json };
}

function must(cond, msg, file) {
  if (!cond) {
    writeFileSync(join(dir, file), JSON.stringify(log, null, 2) + '\n');
    throw new Error('s29: ' + msg);
  }
}

async function setup() {
  const tok = await token('admin@tenant-a.test', 'dev-only-admin-a');
  const cur = await call(tok, 'GET', '/api/v1/entity-types/incident');
  must(cur.status === 200, `read incident: HTTP ${cur.status}`, 'setup.json');
  const schema = cur.json.schema;
  must(!schema.properties.s29_tier, 'incident already has s29_tier: the kit needs a fresh stack', 'setup.json');
  delete schema.properties.priority.default;
  schema.properties.s29_tier = { type: 'string', title: 'Tier', maxLength: 32 };
  const pub = await call(tok, 'POST', '/api/v1/entity-types/incident/versions',
    { definition: { schema, fixed_columns: cur.json.fixed_columns, states: cur.json.states } });
  must(pub.status === 201, `publish without priority's default, with s29_tier: HTTP ${pub.status}`, 'setup.json');
  must(!pub.json.backfill, 'a publish that adds no default started a backfill job (ADR-009 D1)', 'setup.json');
}

async function held() {
  const tok = await token('admin@tenant-a.test', 'dev-only-admin-a');
  const acl = await call(tok, 'GET', '/api/v1/entity-types/incident/access');
  must(acl.status === 200, `read the incident ACL: HTTP ${acl.status}`, 'held.json');
  const sd = (((acl.json.rules || []).find((r) => r.id === 'default-service-desk') || { who: {} }).who.groups || [])[0];
  must(sd, 'the default incident ACL has no service-desk group', 'held.json');
  const ids = { 3: [], 1: [] };
  for (let i = 0; i < 50; i++) {
    const p = i < 25 ? 3 : 1;
    const r = await call(tok, 'POST', '/api/v1/records/incident',
      { data: { short_description: `S29 held ${i + 1}`, priority: p, assignment_group: sd } });
    must(r.status === 201, `create held incident ${i + 1}: HTTP ${r.status}`, 'held.json');
    ids[p].push(r.json.id);
  }
  const btok = await token('admin@tenant-b.test', 'dev-only-admin-b');
  for (let i = 0; i < 5; i++) {
    const r = await call(btok, 'POST', '/api/v1/records/incident', { data: { short_description: `S29 tenant B ${i + 1}`, priority: 1 } });
    must(r.status === 201, `create tenant B incident ${i + 1}: HTTP ${r.status}`, 'held.json');
  }
  writeFileSync(join(dir, 'held.env'),
    `s29_held_3=${ids[3].join(',')}\ns29_sample_held=${[...ids[3].slice(0, 5), ...ids[1].slice(0, 5)].join(',')}\n`);
  writeFileSync(join(dir, 'held.json'), JSON.stringify(log.map((l) => ({ request: l.request.path, status: l.response.status })), null, 2) + '\n');
}

if (mode === 'setup') {
  await setup();
  writeFileSync(join(dir, 'setup.json'), JSON.stringify(log, null, 2) + '\n');
} else if (mode === 'held') {
  await held();
} else {
  console.error('usage: node kit.mjs setup|held <evidence-dir>');
  process.exit(2);
}
