// S18 acceptance (TICK-677): the steps run-kit.sh needs outside Bruno.
//
//   node kit.mjs admin-b
//     Seeds admin@tenant-b.test: a Keycloak user in organization tenant-b and
//     in group /t/<B>/admins, the group the seed records as tenant B's admin
//     group. run-kit.sh then runs one group sync cycle. Dev-only password, the
//     same pattern as the realm's seed users. Idempotent.
//
//   node kit.mjs pre <evidence-dir>
//     On the S17 build, before the upgrade. admin@tenant-a creates the ticket
//     types `visa` (prefix VISA) and `permit` (prefix PMT) and gives each an
//     access policy: tenant A's admins and service-desk groups read and write
//     (the group ids come from the default incident ACL). Then, on the default
//     numbering: VISA0000001 (admin@tenant-a), VISA0000002 (agent@tenant-a,
//     Alice), INC0000001..INC0000002 in tenant A and in tenant B (Bob).
//     Fails on any other number. Writes <evidence-dir>/pre-upgrade.json and
//     <evidence-dir>/pre-ids.env (record ids for Bruno, no credentials).
//
//   node kit.mjs me-b <evidence-dir>
//     After the group sync: admin@tenant-b resolves to tenant B and is in its
//     admins group. Writes <evidence-dir>/admin-b-me.json.
//
// Ports come from the same variables as the compose file and api-test.sh.
// Only dev-only seed users (docs/acceptance/guide/bruno/environments/local.bru).
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';

const [mode, dir] = process.argv.slice(2);
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const A = '00000000-0000-4000-8000-0000000000a1';
const B = '00000000-0000-4000-8000-0000000000b1';
const USERS = {
  admin_a: ['admin@tenant-a.test', 'dev-only-admin-a'],
  alice: ['alice@tenant-a.test', 'dev-only-alice'],
  bob: ['bob@tenant-b.test', 'dev-only-bob'],
  admin_b: ['admin@tenant-b.test', 'dev-only-admin-b'],
};

// The definition of both kit types. The Bruno folder sends it back with
// number_format (AC1, E9) and with one more property (AC4).
export const DEFINITION = {
  schema: {
    $schema: 'https://json-schema.org/draft/2020-12/schema',
    type: 'object',
    additionalProperties: false,
    required: ['short_description', 'state'],
    properties: {
      short_description: { type: 'string', minLength: 1, maxLength: 160 },
      state: { type: 'string', enum: ['new', 'closed'], default: 'new' },
    },
  },
  fixed_columns: { state: 'state' },
  states: { initial: 'new', states: ['new', 'closed'], transitions: [{ from: 'new', to: 'closed' }] },
};

async function token(who) {
  const [username, password] = USERS[who];
  const r = await fetch(`${KC}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username, password }),
  });
  if (r.status !== 200) throw new Error(`sign-in ${who}: HTTP ${r.status}`);
  return (await r.json()).access_token;
}

// One Idempotency-Key per call; a 429 or 503 (rate limit, retry_later) is
// sent again with the same key after its Retry-After, up to 10 times.
async function call(tok, method, path, body, headers = {}) {
  const key = `s18-kit-${randomUUID()}`;
  for (let attempt = 1; ; attempt++) {
    const r = await fetch(CORE + path, {
      method,
      headers: { Authorization: 'Bearer ' + tok, 'Content-Type': 'application/json', 'Idempotency-Key': key, ...headers },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const text = await r.text();
    if ((r.status === 429 || r.status === 503) && attempt < 10) {
      await new Promise((ok) => setTimeout(ok, 1000 * Number(r.headers.get('Retry-After') || 1)));
      continue;
    }
    let json = null;
    try { json = JSON.parse(text); } catch { /* not JSON */ }
    return { status: r.status, body: json ?? text };
  }
}

function must(r, want, what) {
  if (!want.includes(r.status)) throw new Error(`${what}: HTTP ${r.status} ${JSON.stringify(r.body)}`);
  return r.body;
}

async function kcAdmin() {
  const r = await fetch(`${KC}/realms/master/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'password', client_id: 'admin-cli',
      username: process.env.KEYCLOAK_ADMIN || 'admin', password: process.env.KEYCLOAK_ADMIN_PASSWORD || 'dev-only-admin',
    }),
  });
  if (r.status !== 200) throw new Error(`Keycloak admin sign-in: HTTP ${r.status}`);
  const tok = (await r.json()).access_token;
  return async (method, path, body) => {
    const res = await fetch(`${KC}/admin/realms/nava${path}`, {
      method,
      headers: { Authorization: 'Bearer ' + tok, 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const text = await res.text();
    let json = null;
    try { json = JSON.parse(text); } catch { /* empty */ }
    return { status: res.status, body: json ?? text };
  };
}

async function createType(tok, key, name, prefix, aclRules) {
  must(await call(tok, 'POST', '/api/v1/entity-types', { key, name, number_prefix: prefix, definition: DEFINITION }), [201], `create type ${key}`);
  const acl = must(await call(tok, 'PUT', `/api/v1/entity-types/${key}/access`, { rules: aclRules }, { 'If-Match': 'W/"0"' }), [200], `ACL of ${key}`);
  return { key, number_prefix: prefix, acl_version: acl.version, acl_rules: acl.rules.map((r) => r.id) };
}

async function createRecords(who, entity, n, label) {
  const tok = await token(who);
  const out = [];
  for (let i = 1; i <= n; i++) {
    const b = must(await call(tok, 'POST', `/api/v1/records/${entity}`, { data: { short_description: `S18 pre-upgrade ${label} ${i}` } }), [201], `${who} create ${entity} ${i}`);
    out.push({ user: USERS[who][0], number: b.number, id: b.id });
  }
  return out;
}

if (mode === 'admin-b') {
  const kc = await kcAdmin();
  const [username, password] = USERS.admin_b;
  const found = await kc('GET', `/users?username=${encodeURIComponent(username)}&exact=true`);
  let id = Array.isArray(found.body) && found.body[0] && found.body[0].id;
  if (!id) {
    const c = await kc('POST', '/users', {
      username, email: username, emailVerified: true, firstName: 'Admin', lastName: 'B', enabled: true,
      credentials: [{ type: 'password', value: password, temporary: false }],
    });
    if (c.status !== 201) throw new Error(`create ${username}: HTTP ${c.status} ${JSON.stringify(c.body)}`);
    id = (await kc('GET', `/users?username=${encodeURIComponent(username)}&exact=true`)).body[0].id;
  }
  // Organization membership gives the token tenant B's organization claim.
  const m = await kc('POST', `/organizations/${B}/members`, id);
  if (![201, 204, 409].includes(m.status)) throw new Error(`add ${username} to organization tenant-b: HTTP ${m.status} ${JSON.stringify(m.body)}`);
  const g = await kc('GET', `/group-by-path/t/${B}/admins`);
  if (g.status !== 200) throw new Error(`no group /t/${B}/admins: HTTP ${g.status}`);
  const j = await kc('PUT', `/users/${id}/groups/${g.body.id}`);
  if (j.status !== 204) throw new Error(`add ${username} to /t/${B}/admins: HTTP ${j.status}`);
  console.log(`s18: ${username} is in organization tenant-b and in /t/${B}/admins`);
} else if (mode === 'me-b') {
  const me = must(await call(await token('admin_b'), 'GET', '/api/v1/me'), [200], 'admin@tenant-b GET /me');
  const paths = (me.group_paths || []).map((g) => g.path);
  writeFileSync(join(dir, 'admin-b-me.json'), JSON.stringify({ step: 'admin@tenant-b after one group sync', tenant_id: me.tenant_id, group_paths: paths }, null, 2) + '\n');
  if (me.tenant_id !== B || !paths.includes('/admins')) {
    console.error(`s18: admin@tenant-b is not tenant B's admin: tenant ${me.tenant_id}, groups ${paths.join(' ')}`);
    process.exit(1);
  }
  console.log('s18: admin@tenant-b resolves to tenant B, group /admins');
} else if (mode === 'pre') {
  const adminTok = await token('admin_a');
  const inc = must(await call(adminTok, 'GET', '/api/v1/entity-types/incident/access'), [200], 'incident ACL');
  const groupsOf = (id) => (inc.rules.find((r) => r.id === id) || { who: {} }).who.groups || [];
  const rules = [
    { id: 's18-admins', ops: ['read', 'write'], who: { groups: groupsOf('default-admins') } },
    { id: 's18-service-desk', ops: ['read', 'write'], who: { groups: groupsOf('default-service-desk') } },
  ];
  if (!rules[0].who.groups.length || !rules[1].who.groups.length) throw new Error('the default incident ACL has no admins or service-desk group');
  const visa = await createType(adminTok, 'visa', 'Visa application', 'VISA', rules);
  const permit = await createType(adminTok, 'permit', 'Work permit', 'PMT', rules);
  const visaA = await createRecords('admin_a', 'visa', 1, 'visa A admin');
  const visaAlice = await createRecords('alice', 'visa', 1, 'visa A agent');
  const [incA, incB] = await Promise.all([createRecords('admin_a', 'incident', 2, 'incident A'), createRecords('bob', 'incident', 2, 'incident B')]);
  const visas = [...visaA, ...visaAlice];
  const ok = JSON.stringify(visas.map((r) => r.number)) === '["VISA0000001","VISA0000002"]'
    && JSON.stringify(incA.map((r) => r.number)) === '["INC0000001","INC0000002"]'
    && JSON.stringify(incB.map((r) => r.number)) === '["INC0000001","INC0000002"]';
  const out = {
    step: 'on the S17 build, before the upgrade, default numbering',
    types: [visa, permit],
    visa_tenant_a: visas,
    incident_tenant_a: incA,
    incident_tenant_b: incB,
    as_expected: ok,
  };
  writeFileSync(join(dir, 'pre-upgrade.json'), JSON.stringify(out, null, 2) + '\n');
  writeFileSync(join(dir, 'pre-ids.env'), [
    `s18_visa1_id=${visas[0].id}`, `s18_visa2_id=${visas[1].id}`, `s18_inc_a1_id=${incA[0].id}`, `s18_inc_b1_id=${incB[0].id}`,
  ].join('\n') + '\n');
  console.log(`s18: before the upgrade: visa ${visas.map((r) => r.number).join(' ')}; incidents A ${incA.map((r) => r.number).join(' ')}, B ${incB.map((r) => r.number).join(' ')}`);
  if (!ok) { console.error('s18: the pre-upgrade numbers are not the expected ones'); process.exit(1); }
} else {
  console.error('usage: node kit.mjs admin-b | me-b <evidence-dir> | pre <evidence-dir>');
  process.exit(2);
}
