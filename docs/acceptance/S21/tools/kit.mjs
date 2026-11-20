// S21 acceptance (TICK-749): the API steps run-kit.sh needs before the seed.
//
//   node kit.mjs setup <evidence-dir>
//     As admin@tenant-a, on a fresh stack (incident at version 1):
//       1. publishes the next incident version with one new optional property,
//          `due_at` (string, format date-time), not flagged: Compatible;
//       2. publishes tenant A's incident ACL with its rules unchanged and one
//          mask: `description` is read and written by the admins group only,
//          so it is a masked field for Alice (criterion 3).
//     seed.sql then writes the 400k incidents at that version. Writes
//     <evidence-dir>/setup.json (requests and answers, no credentials).
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
async function call(tok, method, path, body, headers = {}) {
  const h = { Authorization: 'Bearer ' + tok, ...headers };
  if (body !== undefined) h['Content-Type'] = 'application/json';
  if (method !== 'GET') h['Idempotency-Key'] = 's21-setup-' + randomUUID();
  const r = await fetch(CORE + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  let json = null;
  try { json = JSON.parse(text); } catch { /* not JSON */ }
  const { Authorization, ...shown } = h;
  log.push({ request: { method, path, headers: shown, body: body ?? null }, response: { status: r.status, etag: r.headers.get('etag'), body: json ?? text } });
  return { status: r.status, json, etag: r.headers.get('etag') };
}

function must(cond, msg) {
  if (!cond) {
    writeFileSync(join(dir, 'setup.json'), JSON.stringify(log, null, 2) + '\n');
    throw new Error('s21 setup: ' + msg);
  }
}

async function setup() {
  const tok = await token('admin@tenant-a.test', 'dev-only-admin-a');
  const cur = await call(tok, 'GET', '/api/v1/entity-types/incident');
  must(cur.status === 200, `read incident: HTTP ${cur.status}`);
  const schema = cur.json.schema;
  must(!schema.properties.due_at, 'incident already has due_at: the kit needs a fresh stack');
  schema.properties.due_at = { type: 'string', format: 'date-time', title: 'Due at' };
  const pub = await call(tok, 'POST', '/api/v1/entity-types/incident/versions',
    { definition: { schema, fixed_columns: cur.json.fixed_columns, states: cur.json.states } });
  must(pub.status === 201 && pub.json.compat === 'compatible', `publish due_at: HTTP ${pub.status}`);

  const me = await call(tok, 'GET', '/api/v1/me');
  const admins = me.json.group_paths.find((g) => g.path === '/admins');
  must(admins, 'admin@tenant-a is not in /admins (run tools/seed.sh)');
  const pol = await call(tok, 'GET', '/api/v1/entity-types/incident/access');
  must(pol.status === 200, `read access policy: HTTP ${pol.status}`);
  const mask = { field: 'description', read: { groups: [admins.id] }, write: { groups: [admins.id] } };
  const put = await call(tok, 'PUT', '/api/v1/entity-types/incident/access',
    { rules: pol.json.rules, masks: [mask] }, { 'If-Match': pol.etag });
  must(put.status === 200, `publish mask: HTTP ${put.status}`);
  writeFileSync(join(dir, 'setup.json'), JSON.stringify(log, null, 2) + '\n');
  console.log(`s21 setup: incident version ${pub.json.version} has due_at; ACL version ${put.json.version} masks description (admins only)`);
}

if (mode === 'setup') await setup();
else { console.error('usage: node kit.mjs setup <evidence-dir>'); process.exit(2); }
