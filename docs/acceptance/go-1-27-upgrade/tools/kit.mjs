// TICK-961 acceptance (Go 1.27 upgrade): the API and database checks that
// run-kit.sh runs on a fresh, seeded stack.
//
//   node kit.mjs <evidence-dir>
//
// As admin@tenant-a and admin@tenant-b (dev-only seed users,
// docs/acceptance/guide/bruno/environments/local.bru):
//   AC2 ids:       each tenant creates an entity type (with an ACL), publishes
//                  a workflow definition on it and creates a number format.
//                  Every returned id, the step-id map and every groups row are
//                  checked for UUID version 7 (character 15 of the text form).
//   AC3 strict:    every operation of docs/api/strict-json-bodies.md gets a
//                  body with a duplicate member name and one with invalid
//                  UTF-8 (a raw 0xFF byte, and a lone "\ud800" escape):
//                  422, the documented code, one errors entry with a /body
//                  pointer and rule. The tenant's rows are counted before and
//                  after: nothing is written.
//   AC4 audit:     admin@tenant-a writes a type name and a record holding
//                  U+2028 and U+2029, waits until the entries are sealed and
//                  saves the export (export-tenant-a.jsonl, verified by
//                  run-kit.sh with `nava audit verify`).
//   AC5 isolation: admin@tenant-b reads and writes tenant A's new rows by key
//                  and id: the same 404 as a made-up one; tenant A's rows are
//                  unchanged, and tenant B's export names none of them.
// Writes one JSON file per request (no credentials) and result.json.
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';

const dir = process.argv[2];
if (!dir) { console.error('usage: node kit.mjs <evidence-dir>'); process.exit(2); }
mkdirSync(dir, { recursive: true });
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const A = '00000000-0000-4000-8000-0000000000a1';
const B = '00000000-0000-4000-8000-0000000000b1';
const RUN = Date.now().toString(36);
const USERS = {
  admin_a: ['admin@tenant-a.test', 'dev-only-admin-a'],
  admin_b: ['admin@tenant-b.test', 'dev-only-admin-b'],
};
const results = [];
let seq = 0;
function check(id, what, ok, detail) {
  results.push({ id, what, ok: !!ok, detail });
  console.log(`${ok ? 'PASS' : 'FAIL'} ${id} ${what}${ok ? '' : ' :: ' + JSON.stringify(detail)}`);
}
const v7 = (id) => typeof id === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(id);

function sql(q) {
  return execFileSync('docker', ['compose', 'exec', '-T', 'postgres', 'sh', '-c',
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -v ON_ERROR_STOP=1 -At -F "|" -c "$1"', 'sh', q],
  { encoding: 'utf8' }).trim();
}

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

// raw: a Buffer sent as is (the strict-JSON bodies); otherwise JSON.stringify.
async function call(name, who, tok, method, path, body, headers = {}, raw) {
  const key = `t961-${RUN}-${randomUUID()}`;
  for (let attempt = 1; ; attempt++) {
    const r = await fetch(CORE + path, {
      method,
      headers: { Authorization: 'Bearer ' + tok, 'Content-Type': 'application/json', 'Idempotency-Key': key, ...headers },
      body: raw !== undefined ? raw : body === undefined ? undefined : JSON.stringify(body),
    });
    const text = await r.text();
    if ((r.status === 429 || r.status === 503) && attempt < 10) {
      await new Promise((ok) => setTimeout(ok, 1000 * Number(r.headers.get('Retry-After') || 1)));
      continue;
    }
    let json = null;
    try { json = JSON.parse(text); } catch { /* not JSON */ }
    const out = { status: r.status, body: json ?? text, headers: { 'content-type': r.headers.get('content-type') } };
    if (name) {
      seq++;
      writeFileSync(join(dir, `${String(seq).padStart(2, '0')}-${name}.json`), JSON.stringify({
        step: name, user: USERS[who][0], request: { method, path, headers: { 'If-Match': headers['If-Match'] },
          body: raw !== undefined ? { raw_hex: raw.toString('hex'), raw_latin1: raw.toString('latin1') } : body },
        response: out,
      }, null, 2) + '\n');
    }
    return out;
  }
}
function must(r, want, what) {
  if (!want.includes(r.status)) throw new Error(`${what}: HTTP ${r.status} ${JSON.stringify(r.body)}`);
  return r.body;
}

const SCHEMA = {
  $schema: 'https://json-schema.org/draft/2020-12/schema',
  type: 'object',
  additionalProperties: false,
  required: ['short_description', 'state'],
  properties: {
    short_description: { type: 'string', minLength: 1, maxLength: 160 },
    state: { type: 'string', enum: ['new', 'closed'], default: 'new' },
  },
};
const DEFINITION = {
  schema: SCHEMA,
  fixed_columns: { state: 'state' },
  states: { initial: 'new', states: ['new', 'closed'], transitions: [{ from: 'new', to: 'closed' }] },
};
const workflow = (entity) => ({
  format: 'nava.workflow/v1',
  trigger: { on: 'record.created', entity },
  start: 'wait',
  steps: [
    { key: 'wait', type: 'timer', duration: 'PT1H', next: 'mark' },
    { key: 'mark', type: 'update_record', set: { short_description: 'triaged' }, next: 'done' },
    { key: 'done', type: 'end' },
  ],
});

// AC2: one tenant's new rows, all through the API.
async function createRows(who, tag, tok, nameSuffix = '') {
  const t = tag.toLowerCase();
  const key = `t961_${t}_${RUN}`;
  const inc = must(await call(`${t}-incident-acl`, who, tok, 'GET', '/api/v1/entity-types/incident/access'), [200], 'incident ACL');
  const groupsOf = (id) => (inc.rules.find((r) => r.id === id) || { who: {} }).who.groups || [];
  const type = must(await call(`${t}-create-type`, who, tok, 'POST', '/api/v1/entity-types',
    { key, name: `T961 ${tag}${nameSuffix}`, number_prefix: `TN${tag}`, definition: DEFINITION }), [201], `create type ${key}`);
  const acl = must(await call(`${t}-put-acl`, who, tok, 'PUT', `/api/v1/entity-types/${key}/access`,
    { rules: [{ id: 't961-admins', ops: ['read', 'write'], who: { groups: groupsOf('default-admins') } }] }, { 'If-Match': 'W/"0"' }), [200], `ACL of ${key}`);
  const wf = must(await call(`${t}-publish-workflow`, who, tok, 'POST', `/api/v1/workflow-definitions/${key}/versions`, workflow(key)), [201], `workflow ${key}`);
  const nf = must(await call(`${t}-create-number-format`, who, tok, 'POST', '/api/v1/number-formats',
    { key: `t961_${t}_${RUN}`, document: { template: '{prefix}-{seq:6}', overflow: 'refuse' } }), [201], `number format ${key}`);
  const rec = must(await call(`${t}-create-record`, who, tok, 'POST', `/api/v1/records/${key}`,
    { data: { short_description: `T961 ${tag} record${nameSuffix}` } }), [201], `record ${key}`);
  return { key, type, acl, wf, nf, rec };
}

const tokA = await token('admin_a');
const tokB = await token('admin_b');

// ---------- AC2 + AC4 setup: new rows in both tenants ----------
// Tenant A's type name and record text hold U+2028 and U+2029 (AC4).
const SEP = ' line sep para sep';
const rowsA = await createRows('admin_a', 'A', tokA, SEP);
const rowsB = await createRows('admin_b', 'B', tokB);

for (const [tag, r] of [['A', rowsA], ['B', rowsB]]) {
  const ids = {
    'number format id': r.nf.id,
    'record id': r.rec.id,
  };
  for (const [what, id] of Object.entries(ids)) check(`AC2.${tag}`, `tenant ${tag} ${what} ${id} is v7`, v7(id), { id });
  const tenant = tag === 'A' ? A : B;
  const db = sql(`SELECT d.id, v.id FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id
                  WHERE d.tenant_id = '${tenant}' AND d.key = '${r.key}'`).split('\n');
  for (const line of db) {
    const [d, v] = line.split('|');
    check(`AC2.${tag}`, `tenant ${tag} entity type ${r.key}: definition id ${d} and version id ${v} are v7`, v7(d) && v7(v), { d, v });
  }
  const wf = sql(`SELECT w.id, v.id, v.step_ids::text FROM workflow_definitions w JOIN workflow_definition_versions v ON v.tenant_id = w.tenant_id AND v.definition_id = w.id
                  WHERE w.tenant_id = '${tenant}' AND w.key = '${r.key}'`);
  const [wid, wvid, steps] = wf.split('|');
  const stepIds = Object.values(JSON.parse(steps || '{}'));
  check(`AC2.${tag}`, `tenant ${tag} workflow definition ${wid}, version ${wvid} and ${stepIds.length} step ids are v7`,
    v7(wid) && v7(wvid) && stepIds.length === 3 && stepIds.every(v7), { wid, wvid, stepIds });
}
const groups = sql(`SELECT tenant_id, id, path FROM groups ORDER BY tenant_id, path`).split('\n').filter(Boolean).map((l) => l.split('|'));
check('AC2.groups', `all ${groups.length} groups rows (minted by nava-groupsync on this build) are v7`,
  groups.length > 0 && groups.every((g) => v7(g[1])), groups.filter((g) => !v7(g[1])));
const nonV7 = sql(`SELECT 'entity_definitions', id FROM entity_definitions WHERE key NOT IN ('incident','request','change') AND substr(id::text,15,1) <> '7'
  UNION ALL SELECT 'workflow_definitions', id FROM workflow_definitions WHERE substr(id::text,15,1) <> '7'
  UNION ALL SELECT 'workflow_definition_versions', id FROM workflow_definition_versions WHERE substr(id::text,15,1) <> '7'
  UNION ALL SELECT 'number_formats', id FROM number_formats WHERE substr(id::text,15,1) <> '7'
  UNION ALL SELECT 'groups', id FROM groups WHERE substr(id::text,15,1) <> '7'`);
check('AC2.db', 'no row minted on this stack (seed templates incident/request/change excluded) has a non-v7 id', nonV7 === '', nonV7);
writeFileSync(join(dir, 'ids-db.txt'), sql(`SELECT 'entity_definitions', tenant_id, key, id FROM entity_definitions
  UNION ALL SELECT 'workflow_definitions', tenant_id, key, id FROM workflow_definitions
  UNION ALL SELECT 'number_formats', tenant_id, key, id FROM number_formats
  UNION ALL SELECT 'groups', tenant_id, path, id FROM groups ORDER BY 1, 2, 3`) + '\n');

// ---------- AC3: strict JSON bodies ----------
const counts = (t) => sql(`SELECT
  (SELECT count(*) FROM entity_definitions WHERE tenant_id='${t}'),
  (SELECT count(*) FROM entity_definition_versions WHERE tenant_id='${t}'),
  (SELECT count(*) FROM acl_versions WHERE tenant_id='${t}'),
  (SELECT count(*) FROM workflow_definition_versions WHERE tenant_id='${t}'),
  (SELECT count(*) FROM number_formats WHERE tenant_id='${t}'),
  (SELECT count(*) FROM number_format_versions WHERE tenant_id='${t}'),
  (SELECT count(*) FROM records WHERE tenant_id='${t}'),
  (SELECT count(*) FROM audit_log WHERE tenant_id='${t}'),
  (SELECT count(*) FROM outbox WHERE tenant_id='${t}')`);
const COUNT_COLS = 'entity_definitions|entity_definition_versions|acl_versions|workflow_definition_versions|number_formats|number_format_versions|records|audit_log|outbox';
const tzNow = must(await call('a-get-time-zone', 'admin_a', tokA, 'GET', '/api/v1/tenant/settings/time-zone'), [200], 'time zone');
const k = rowsA.key;
// Each body is valid apart from the one defect at @@, so a lenient decoder
// would write. [operation, method, path, headers, code, template, member, value]
const wfBody = JSON.stringify(workflow(k)).replace('"start":"wait"', '@@');
const OPS = [
  ['createEntityType', 'POST', '/api/v1/entity-types', {}, 'invalid_definition',
    `{"key":"t961_strict_${RUN}",@@,"number_prefix":"TS","definition":${JSON.stringify(DEFINITION)}}`, 'name', 'Strict'],
  ['publishEntityTypeVersion', 'POST', `/api/v1/entity-types/${k}/versions`, {}, 'invalid_definition',
    `{"definition":{"schema":${JSON.stringify(SCHEMA)},@@,"states":${JSON.stringify(DEFINITION.states)}}}`, 'fixed_columns', null],
  ['putEntityTypeAccess', 'PUT', `/api/v1/entity-types/${k}/access`, { 'If-Match': `W/"${rowsA.acl.version}"` }, 'invalid_access_policy',
    '{"rules":[{@@,"ops":["read"],"who":{"groups":[]}}]}', 'id', 'x'],
  ['publishWorkflowDefinitionVersion', 'POST', `/api/v1/workflow-definitions/${k}/versions`, {}, 'workflow_definition_invalid',
    wfBody, 'start', 'wait'],
  ['createNumberFormat', 'POST', '/api/v1/number-formats', {}, 'number_format_invalid',
    `{"key":"t961_strict_${RUN}","document":{"template":"{prefix}{seq:5}",@@}}`, 'overflow', 'refuse'],
  ['putNumberFormat', 'PUT', `/api/v1/number-formats/${rowsA.nf.id}`, { 'If-Match': `W/"${rowsA.nf.version}"` }, 'number_format_invalid',
    '{"document":{"template":"{prefix}{seq:7}",@@}}', 'overflow', 'refuse'],
  ['putTenantTimeZone', 'PUT', '/api/v1/tenant/settings/time-zone', {}, 'invalid_time_zone',
    '{@@}', 'time_zone', 'Europe/Berlin'],
  ['createRecord', 'POST', `/api/v1/records/${k}`, {}, 'record_invalid',
    '{"data":{@@}}', 'short_description', 'strict'],
  ['patchRecord', 'PATCH', `/api/v1/records/${k}/${rowsA.rec.id}`, { 'If-Match': `W/"${rowsA.rec.version ?? 1}"`, 'Content-Type': 'application/merge-patch+json' }, 'record_invalid',
    '{"data":{@@}}', 'short_description', 'strict'],
];
// value null: the member's value is the definition's fixed_columns object.
const LONE = String.raw`\ud800`;
const DEFECTS = [
  ['duplicate', 'duplicate', (m, v) => { const j = v === null ? JSON.stringify(DEFINITION.fixed_columns) : JSON.stringify(v); return `"${m}":${j},"${m}":${j}`; }],
  // The 0xFF byte: U+00FF in a JS string, sent as one raw byte (the body is encoded latin1).
  ['invalid-utf8-byte', 'invalid_utf8', (m, v) => v === null ? `"${m}":{"stateÿ":"state"}` : `"${m}":"${v}ÿ"`],
  // Six ASCII characters: backslash, u, d, 8, 0, 0.
  ['lone-surrogate', 'invalid_utf8', (m, v) => v === null ? `"${m}":{"state":"${LONE}"}` : `"${m}":"${LONE}"`],
];
const beforeA = counts(A);
for (const [op, method, path, hdr, code, tpl, member, value] of OPS) {
  for (const [dname, rule, defect] of DEFECTS) {
    const text = tpl.replace('@@', defect(member, value));
    const raw = Buffer.from(text, 'latin1');
    const contentType = hdr['Content-Type'];
    const r = await call(`strict-${op}-${dname}`, 'admin_a', tokA, method, path, undefined, { ...hdr, ...(contentType ? { 'Content-Type': contentType } : {}) }, raw);
    const b = r.body || {};
    const e = Array.isArray(b.errors) ? b.errors : [];
    const ok = r.status === 422 && b.code === code && b.status === 422 && String(r.headers['content-type']).startsWith('application/problem+json')
      && e.length === 1 && e[0].rule === rule && typeof e[0].pointer === 'string' && e[0].pointer.startsWith('/body')
      && !/[ÿ�]/.test(JSON.stringify(b));
    check('AC3', `${op} with ${dname}: 422 ${code}, errors[0].rule=${rule}`, ok, { status: r.status, body: b });
  }
}
const afterA = counts(A);
writeFileSync(join(dir, 'strict-counts.txt'), `tenant A row counts around the ${OPS.length * DEFECTS.length} refused bodies\n${COUNT_COLS}\nbefore ${beforeA}\nafter  ${afterA}\n`);
check('AC3.nothing-written', `tenant A row counts unchanged after ${OPS.length * DEFECTS.length} refused bodies (${COUNT_COLS})`, beforeA === afterA, { beforeA, afterA });
const tzAfter = must(await call('a-get-time-zone-after', 'admin_a', tokA, 'GET', '/api/v1/tenant/settings/time-zone'), [200], 'time zone');
check('AC3.nothing-written', 'tenant A time zone unchanged', JSON.stringify(tzNow.time_zone) === JSON.stringify(tzAfter.time_zone), { tzNow, tzAfter });

// ---------- AC5: isolation ----------
const madeUp = '0192f0c0-0000-7000-8000-000000000961';
const bodyOf = (r) => JSON.stringify({ ...(r.body || {}), instance: undefined });
async function same404(name, pathA, pathFake, method = 'GET', body, hdr = {}) {
  const ra = await call(`iso-${name}-tenant-a`, 'admin_b', tokB, method, pathA, body, hdr);
  const rf = await call(`iso-${name}-made-up`, 'admin_b', tokB, method, pathFake, body, hdr);
  const leak = [A, rowsA.key, rowsA.nf.id, rowsA.rec.id].some((s) => JSON.stringify(ra.body).includes(s) && !pathA.includes(s));
  check('AC5', `admin@tenant-b ${method} ${name} of tenant A: ${ra.status}, same as a made-up one (${rf.status})`,
    ra.status === 404 && rf.status === 404 && bodyOf(ra).replace(pathA, '') === bodyOf(rf).replace(pathFake, '') && !leak,
    { a: ra, fake: rf });
}
await same404('entity-type', `/api/v1/entity-types/${rowsA.key}`, `/api/v1/entity-types/t961_none_${RUN}`);
await same404('entity-type-access', `/api/v1/entity-types/${rowsA.key}/access`, `/api/v1/entity-types/t961_none_${RUN}/access`);
await same404('workflow-definition', `/api/v1/workflow-definitions/${rowsA.key}`, `/api/v1/workflow-definitions/t961_none_${RUN}`);
await same404('number-format', `/api/v1/number-formats/${rowsA.nf.id}`, `/api/v1/number-formats/${madeUp}`);
await same404('number-format-put', `/api/v1/number-formats/${rowsA.nf.id}`, `/api/v1/number-formats/${madeUp}`, 'PUT',
  { document: { template: '{prefix}{seq:9}', overflow: 'refuse' } }, { 'If-Match': 'W/"1"' });
await same404('record', `/api/v1/records/${rowsA.key}/${rowsA.rec.id}`, `/api/v1/records/t961_none_${RUN}/${madeUp}`);
await same404('record-patch', `/api/v1/records/${rowsA.key}/${rowsA.rec.id}`, `/api/v1/records/t961_none_${RUN}/${madeUp}`, 'PATCH',
  { data: { short_description: 'B was here' } }, { 'If-Match': 'W/"1"', 'Content-Type': 'application/merge-patch+json' });
const listB = must(await call('iso-number-formats-list-b', 'admin_b', tokB, 'GET', '/api/v1/number-formats'), [200], 'B list formats');
check('AC5', "tenant B's number-format list does not contain tenant A's format", !JSON.stringify(listB).includes(rowsA.nf.id), listB);
const typesB = must(await call('iso-entity-types-list-b', 'admin_b', tokB, 'GET', '/api/v1/entity-types'), [200], 'B list types');
check('AC5', "tenant B's entity-type list does not contain tenant A's type", !JSON.stringify(typesB).includes(rowsA.key), typesB);
// B publishing a workflow under A's key creates B's own definition, never A's.
const wfA = sql(`SELECT v.id, v.version FROM workflow_definitions w JOIN workflow_definition_versions v ON v.definition_id = w.id AND v.tenant_id = w.tenant_id
                 WHERE w.tenant_id = '${A}' AND w.key = '${rowsA.key}' ORDER BY v.version`);
const recA = sql(`SELECT id, data::text, version FROM records WHERE tenant_id = '${A}' AND id = '${rowsA.rec.id}'`);
const wfBPub = await call('iso-workflow-publish-under-a-key', 'admin_b', tokB, 'POST', `/api/v1/workflow-definitions/${rowsA.key}/versions`, workflow(rowsA.key));
check('AC5', `admin@tenant-b publishing a workflow under tenant A's type key is refused (no such type in tenant B): ${wfBPub.status}`,
  wfBPub.status >= 400 && wfBPub.status < 500 && !JSON.stringify(wfBPub.body).includes(A), wfBPub);
const wfA2 = sql(`SELECT v.id, v.version FROM workflow_definitions w JOIN workflow_definition_versions v ON v.definition_id = w.id AND v.tenant_id = w.tenant_id
                  WHERE w.tenant_id = '${A}' AND w.key = '${rowsA.key}' ORDER BY v.version`);
const recA2 = sql(`SELECT id, data::text, version FROM records WHERE tenant_id = '${A}' AND id = '${rowsA.rec.id}'`);
const nfA = sql(`SELECT id, current_version FROM number_formats WHERE id = '${rowsA.nf.id}'`);
check('AC5', "tenant A's workflow versions, record and number format are unchanged after tenant B's attempts",
  wfA === wfA2 && recA === recA2 && nfA.endsWith('|1'), { wfA, wfA2, recA, recA2, nfA });
writeFileSync(join(dir, 'isolation-db.txt'), `tenant A rows after tenant B's attempts\nworkflow versions: ${wfA2}\nrecord: ${recA2}\nnumber format (id|current_version): ${nfA}\n`);

// ---------- AC4: exports for run-kit.sh ----------
// Wait until tenant A's U+2028 type and record entries are sealed.
const sealed = () => sql(`SELECT count(*) FILTER (WHERE seq IS NULL) FROM audit_log WHERE tenant_id IN ('${A}', '${B}')`);
for (let i = 0; i < 120 && sealed() !== '0'; i++) await new Promise((ok) => setTimeout(ok, 5000));
check('AC4.sealed', 'every audit entry of both tenants is sealed before the export', sealed() === '0', sealed());
for (const [tag, who, tok] of [['a', 'admin_a', tokA], ['b', 'admin_b', tokB]]) {
  const r = await fetch(`${CORE}/api/v1/audit/entries:export`, { headers: { Authorization: 'Bearer ' + tok } });
  const buf = Buffer.from(await r.arrayBuffer());
  writeFileSync(join(dir, `export-tenant-${tag}.jsonl`), buf);
  check('AC4.export', `${USERS[who][0]} export: HTTP ${r.status}, ${buf.length} bytes`, r.status === 200 && buf.length > 0, r.status);
  if (tag === 'a') {
    const raw2028 = buf.includes(Buffer.from(' ', 'utf8')), raw2029 = buf.includes(Buffer.from(' ', 'utf8'));
    const esc = buf.includes(Buffer.from('\\u2028'));
    check('AC4.export', `tenant A export holds U+2028 and U+2029 (raw: ${raw2028 && raw2029}, escaped \\u2028: ${esc})`, (raw2028 && raw2029) || esc, { raw2028, raw2029, esc });
  } else {
    const t = buf.toString('utf8');
    check('AC5', "tenant B's export names none of tenant A's new rows", ![A, rowsA.key, rowsA.nf.id, rowsA.rec.id, rowsA.type.id].filter(Boolean).some((s) => t.includes(s)), null);
  }
}

writeFileSync(join(dir, 'result.json'), JSON.stringify({ run: RUN, passed: results.filter((r) => r.ok).length, failed: results.filter((r) => !r.ok).length, results }, null, 2) + '\n');
const failed = results.filter((r) => !r.ok);
console.log(`t961: ${results.length - failed.length}/${results.length} checks pass`);
process.exit(failed.length ? 1 : 0);
