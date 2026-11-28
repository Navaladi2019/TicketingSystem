// S22 acceptance (TICK-754): the API step run-kit.sh needs before the seed.
//
//   node kit.mjs setup <evidence-dir>
//     As admin@tenant-a, on a fresh stack: creates the ticket type `s22_asset`
//     (prefix SAST) with `name` and an optional `s22_code` (string), not
//     flagged, and an access policy letting tenant A's admins group read and
//     write it. seed.sql then writes 20,000 records at that version. Then it
//     publishes the next version of the seed type `incident` with one more
//     optional field, `s22_note` (string), never flagged: S22.6 filters on it,
//     so its 400 does not depend on what other suites flagged on a shared
//     stack (the CI run 46107246042 found short_description already flagged).
//     Writes <evidence-dir>/setup.json (requests and answers, no credentials).
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
  if (method !== 'GET') h['Idempotency-Key'] = 's22-setup-' + randomUUID();
  const r = await fetch(CORE + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  let json = null;
  try { json = JSON.parse(text); } catch { /* not JSON */ }
  const { Authorization, ...shown } = h;
  log.push({ request: { method, path, headers: shown, body: body ?? null }, response: { status: r.status, body: json ?? text } });
  return { status: r.status, json };
}

function must(cond, msg) {
  if (!cond) {
    writeFileSync(join(dir, 'setup.json'), JSON.stringify(log, null, 2) + '\n');
    throw new Error('s22 setup: ' + msg);
  }
}

// The groups of one default policy (ACL format 2: rules.policies[].when.all[].any[].member_of.groups).
function policyGroups(acl, name) {
  const p = ((acl && acl.rules && acl.rules.policies) || []).find((x) => x.name === name);
  const out = [];
  for (const a of (p && p.when && p.when.all) || []) for (const b of a.any || []) if (b.member_of) out.push(...(b.member_of.groups || []));
  return out;
}

async function setup() {
  const tok = await token('admin@tenant-a.test', 'dev-only-admin-a');
  const exists = await call(tok, 'GET', '/api/v1/entity-types/s22_asset');
  must(exists.status === 404, 's22_asset already exists: the kit needs a fresh stack');
  const created = await call(tok, 'POST', '/api/v1/entity-types', {
    key: 's22_asset', name: 'S22 asset', number_prefix: 'SAST',
    definition: {
      schema: {
        type: 'object', additionalProperties: false, required: ['name', 'state'],
        properties: {
          name: { type: 'string', title: 'Name', minLength: 1, maxLength: 160 },
          state: { type: 'string', enum: ['new', 'closed'], default: 'new' },
          s22_code: { type: 'string', title: 'Code', maxLength: 16 },
        },
      },
      fixed_columns: { state: 'state' },
      states: { initial: 'new', states: ['new', 'closed'], transitions: [{ from: 'new', to: 'closed' }] },
    },
  });
  must(created.status === 201, `create s22_asset: HTTP ${created.status}`);

  // The admins group reads and writes it (its group ids from the default incident ACL, as S18's kit does).
  const inc = await call(tok, 'GET', '/api/v1/entity-types/incident/access');
  must(inc.status === 200, `read the incident access policy: HTTP ${inc.status}`);
  const groups = policyGroups(inc.json, 'default_admins');
  must(groups.length > 0, 'the default incident ACL has no admins group');
  const put = await call(tok, 'PUT', '/api/v1/entity-types/s22_asset/access',
    { rules: { format: 2, conditions: [], policies: [{ name: 's22_admins', ops: ['read', 'write'], when: { all: [{ any: [{ member_of: { groups } }] }] } }] }, masks: [] },
    { 'If-Match': 'W/"0"' });
  must(put.status === 200, `publish the s22_asset access policy: HTTP ${put.status}`);

  // S22.6's field: an optional incident field the kit adds and nobody flags.
  const cur = await call(tok, 'GET', '/api/v1/entity-types/incident');
  must(cur.status === 200, `read the incident type: HTTP ${cur.status}`);
  const schema = cur.json.schema;
  must(!schema.properties.s22_note, 'incident already has s22_note: the kit needs a stack S22 has not run on');
  schema.properties.s22_note = { type: 'string', title: 'S22 note', maxLength: 64 };
  const pub = await call(tok, 'POST', '/api/v1/entity-types/incident/versions',
    { definition: { schema, fixed_columns: cur.json.fixed_columns, states: cur.json.states } });
  must(pub.status === 201, `publish incident with s22_note: HTTP ${pub.status}`);
  must(!(pub.json.schema.properties.s22_note || {})['x-nava-filterable'], 'incident s22_note came back flagged');
}

if (mode === 'setup') {
  await setup();
  writeFileSync(join(dir, 'setup.json'), JSON.stringify(log, null, 2) + '\n');
} else {
  console.error('usage: node kit.mjs setup <evidence-dir>');
  process.exit(2);
}
