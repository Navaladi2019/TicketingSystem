// S17 acceptance (TICK-668): the API steps run-kit.sh needs outside Bruno.
//
//   node kit.mjs pre <evidence-dir>
//     On the S10 build, before the upgrade. admin@tenant-a sets tenant A's
//     incident type to gapless (publishEntityTypeVersion with numbering_mode
//     alone); tenant B's stays gaps (the default). Then admin@tenant-a creates
//     tenant A's incidents and Bob tenant B's, one at a time, until each
//     tenant's last number is INC0000042. Fails if either is not exactly
//     INC0000001..INC0000042. Writes <evidence-dir>/pre-upgrade.json.
//
//   node kit.mjs force <evidence-dir>
//     The forced failure (S17_FORCE_FAIL=1): after the upgrade, one more
//     incident in tenant A, so its next number is INC0000044 and the kit must
//     fail at S17.6. Writes <evidence-dir>/forced-extra-create.json.
//
// Ports come from the same variables as the compose file and api-test.sh.
// Only the dev-only seed users of docs/acceptance/guide/bruno/environments/local.bru.
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';

const [mode, dir] = process.argv.slice(2);
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const USERS = {
  admin_a: ['admin@tenant-a.test', 'dev-only-admin-a'],
  bob: ['bob@tenant-b.test', 'dev-only-bob'],
};
const TARGET = 42;
const inc = (n) => 'INC' + String(n).padStart(7, '0');

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
async function call(tok, method, path, body) {
  const key = `s17-kit-${randomUUID()}`;
  for (let attempt = 1; ; attempt++) {
    const r = await fetch(CORE + path, {
      method,
      headers: { Authorization: 'Bearer ' + tok, 'Content-Type': 'application/json', 'Idempotency-Key': key },
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

async function createUpTo(who, tenant) {
  const tok = await token(who);
  const numbers = [];
  for (let i = 1; i <= TARGET; i++) {
    const r = await call(tok, 'POST', '/api/v1/records/incident', { data: { short_description: `S17 pre-upgrade ${tenant} ${i}` } });
    if (r.status !== 201) throw new Error(`${who} create ${i}: HTTP ${r.status} ${JSON.stringify(r.body)}`);
    numbers.push(r.body.number);
  }
  const want = Array.from({ length: TARGET }, (_, i) => inc(i + 1));
  const ok = JSON.stringify(numbers) === JSON.stringify(want);
  return { tenant, user: USERS[who][0], creates: numbers.length, first: numbers[0], last: numbers[numbers.length - 1], exactly_1_to_42: ok, numbers };
}

if (mode === 'pre') {
  const adminTok = await token('admin_a');
  const set = await call(adminTok, 'POST', '/api/v1/entity-types/incident/versions', { numbering_mode: 'gapless' });
  if (set.status !== 200 && set.status !== 201) throw new Error(`set gapless: HTTP ${set.status} ${JSON.stringify(set.body)}`);
  const typeA = await call(adminTok, 'GET', '/api/v1/entity-types/incident');
  const typeB = await call(await token('bob'), 'GET', '/api/v1/entity-types/incident');
  const [a, b] = await Promise.all([createUpTo('admin_a', 'A'), createUpTo('bob', 'B')]);
  const out = {
    step: 'on the S10 build, before the upgrade',
    set_tenant_a_gapless: { request: 'POST /api/v1/entity-types/incident/versions {"numbering_mode":"gapless"}', status: set.status, numbering_mode: set.body.numbering_mode },
    tenant_a_type: { numbering_mode: typeA.body.numbering_mode, number_prefix: typeA.body.number_prefix },
    tenant_b_type: { numbering_mode: typeB.body.numbering_mode, number_prefix: typeB.body.number_prefix },
    tenant_a: a,
    tenant_b: b,
  };
  writeFileSync(join(dir, 'pre-upgrade.json'), JSON.stringify(out, null, 2) + '\n');
  const ok = a.exactly_1_to_42 && b.exactly_1_to_42 && typeA.body.numbering_mode === 'gapless' && typeB.body.numbering_mode === 'gaps';
  console.log(`s17: before the upgrade: A (${typeA.body.numbering_mode}) last ${a.last}, B (${typeB.body.numbering_mode}) last ${b.last}`);
  if (!ok) { console.error('s17: the counters are not at INC0000042 with A gapless and B gaps'); process.exit(1); }
} else if (mode === 'force') {
  const r = await call(await token('admin_a'), 'POST', '/api/v1/records/incident', { data: { short_description: 'S17 forced failure: one extra create' } });
  writeFileSync(join(dir, 'forced-extra-create.json'), JSON.stringify({ step: 'S17_FORCE_FAIL=1: one extra tenant A create after the upgrade, before Bruno', status: r.status, number: r.body.number }, null, 2) + '\n');
  console.log(`s17: FORCED FAILURE: tenant A took ${r.body.number} before the kit; S17.6 must now fail`);
  if (r.status !== 201) process.exit(1);
} else {
  console.error('usage: node kit.mjs pre|force <evidence-dir>');
  process.exit(2);
}
