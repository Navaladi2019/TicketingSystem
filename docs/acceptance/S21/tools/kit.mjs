// S21 acceptance (TICK-749): the API steps run-kit.sh needs before the seed.
//
//   node kit.mjs setup <evidence-dir>
//     As admin@tenant-a, on a fresh stack (incident at version 1):
//       1. publishes the next incident version with one new optional property,
//          `due_at` (string, format date-time), not flagged, and with every
//          seed-template flag removed (`priority` since TICK-752): Compatible;
//       2. publishes tenant A's incident ACL with its rules unchanged and one
//          mask: `description` is read and written by the admins group only,
//          so it is a masked field for Alice (criterion 3).
//     seed.sql then writes the 400k incidents at that version. Writes
//     <evidence-dir>/setup.json (requests and answers, no credentials).
//
//   node kit.mjs report <evidence-dir>
//     After the Bruno folder and evidence.mjs: the scripted half of the
//     ADR-006-A2 checks (A2-8 (7)), the same rules the .bru tests apply, so a
//     run by hand and run-kit.sh agree. Re-checks the total contract (A2-1) on
//     every list body the evidence recorded (S21.12 to S21.18), that short
//     results answer exact (A2-4), and from S21.19's results: every cell's
//     p95, total_state counts and share of unknown, with the no-filter
//     controls. The >= 40% ACL cell (Alice, filter[priority]=3) must be
//     <= 350 ms p95 with every total_state capped or unknown. Writes
//     <evidence-dir>/p95.txt and exits 1 when a check fails.
//
// Ports come from the same variables as the compose file and api-test.sh.
// Only the dev-only seed users of docs/acceptance/guide/bruno/environments/local.bru.
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
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
  // Since S22 (TICK-752) the seed template flags `priority`. seed.sql writes
  // with session_replication_role = replica, which skips the field-index
  // trigger, so a field flagged before the seed would miss 400k side rows.
  // This publish drops every flag (an S22 removal, over 0 records), and
  // run-kit.sh waits until the removal has deleted the registry rows. S21.4
  // then still sees no flag, and S21.8 flags priority and due_at as before.
  const unflagged = [];
  for (const [k, p] of Object.entries(schema.properties)) {
    if (p['x-nava-filterable']) { delete p['x-nava-filterable']; unflagged.push(k); }
  }
  const pub = await call(tok, 'POST', '/api/v1/entity-types/incident/versions',
    { definition: { schema, fixed_columns: cur.json.fixed_columns, states: cur.json.states } });
  must(pub.status === 201 && pub.json.compat === 'compatible', `publish due_at${unflagged.length ? ' and unflag ' + unflagged.join(',') : ''}: HTTP ${pub.status}`);

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

// The total contract of a list body (ADR-006-A2 A2-1), as lib/s21.js T.contract.
export function contract(b) {
  if (!b || typeof b !== 'object') return 'no body';
  const s = b.total_state;
  if (!['exact', 'capped', 'unknown'].includes(s)) return `total_state ${JSON.stringify(s)}`;
  if (!('total' in b)) return 'no total';
  if (s === 'unknown' && b.total !== null) return `unknown with total ${JSON.stringify(b.total)}`;
  if (s === 'exact' && !(Number.isInteger(b.total) && b.total >= 0 && b.total <= 1000)) return `exact with total ${JSON.stringify(b.total)}`;
  if (s === 'capped' && b.total !== 1000) return `capped with total ${JSON.stringify(b.total)}`;
  if ((b.total_capped === true) !== (s === 'capped')) return `total_capped ${JSON.stringify(b.total_capped)} with ${s}`;
  return null;
}

// The cells of S21.19, read back from its test names (lib/s21.js T.cell via
// collection.py): "<who> <filter> (<band>): only 200s, <n> requests, p50 <a>
// p95 <b> max <c> ms, from <iso>" and "<who> <filter>: every 200 meets the
// total contract; total_state {...}, unknown <x>%".
export function cells(tests) {
  const out = new Map();
  for (const t of tests) {
    let m = /^(\S+) (\S+) \((.*)\): only 200s, (\d+) requests, p50 (\S+) p95 (\S+) max (\S+) ms, from (\S+)$/.exec(t.test);
    if (m) {
      const c = out.get(m[1] + ' ' + m[2]) || {};
      out.set(m[1] + ' ' + m[2], Object.assign(c, { who: m[1], filter: m[2], band: m[3], n: +m[4], p50: +m[5], p95: +m[6], max: +m[7], started_at: m[8], ok200: t.status === 'pass' }));
      continue;
    }
    m = /^(\S+) (\S+): every 200 meets the total contract; total_state (\{.*\}), unknown (\S+)%$/.exec(t.test);
    if (m) {
      const c = out.get(m[1] + ' ' + m[2]) || {};
      out.set(m[1] + ' ' + m[2], Object.assign(c, { states: JSON.parse(m[3]), unknown_share: m[4] === 'null' ? null : +m[4], contract: t.status === 'pass' }));
    }
  }
  return [...out.values()];
}

function report() {
  const read = (n) => {
    const f = join(dir, n + '.json');
    return existsSync(f) ? JSON.parse(readFileSync(f, 'utf8')) : null;
  };
  const fail = [];
  const lines = [];
  // Every list body the evidence keeps, with the total each must show when
  // the result is shorter than the page (A2-4: no count, exact).
  const short = { 'S21-12': 10, 'S21-17': 0, 'S21-18': null };
  for (const n of ['S21-12', 'S21-13', 'S21-14', 'S21-15', 'S21-16', 'S21-17', 'S21-18']) {
    const e = read(n);
    if (!e) { fail.push(`${n}: no evidence`); continue; }
    const b = e.response && e.response.body;
    const bad = contract(b);
    if (bad) fail.push(`${n}: ${bad}`);
    const shortPage = Array.isArray(b && b.items) && !b.next_cursor;
    if (n in short || shortPage) {
      if (b.total_state !== 'exact') fail.push(`${n}: a short result answered ${b.total_state}, not exact`);
      if (short[n] !== null && short[n] !== undefined && b.total !== short[n]) fail.push(`${n}: total ${b.total}, want ${short[n]}`);
    }
    lines.push(`${n}: ${b && b.items ? b.items.length : '-'} items, total ${JSON.stringify(b && b.total)}, total_state ${b && b.total_state}${shortPage ? ' (short result)' : ''}`);
  }
  const p = read('S21-19');
  const cs = p ? cells(p.tests || []) : [];
  if (!cs.length) fail.push('S21-19: no cells in the evidence');
  lines.push('', 'S21.19 cells (client latency, 8 clients, ms):');
  lines.push('who    filter                                                          p50   p95   max     n  total_state                          unknown');
  for (const c of cs) {
    lines.push(`${c.who.padEnd(6)} ${c.filter.padEnd(63)} ${String(c.p50).padStart(4)}  ${String(c.p95).padStart(4)}  ${String(c.max).padStart(4)}  ${String(c.n).padStart(4)}  ${JSON.stringify(c.states || {}).padEnd(36)} ${c.unknown_share}%`);
    if (c.contract === false) fail.push(`S21.19 ${c.who} ${c.filter}: a 200 broke the total contract`);
  }
  const acl = cs.find((c) => c.who === 'Alice' && c.filter === 'filter[priority]=3');
  const ctl = cs.find((c) => c.who === 'admin' && c.filter === '(none)');
  if (!acl) fail.push('S21.19: no >= 40% ACL cell');
  else {
    if (!acl.ok200) fail.push('>= 40% ACL cell: not only 200s');
    if (!(acl.p95 <= 350)) fail.push(`>= 40% ACL cell: p95 ${acl.p95} ms > 350 ms`);
    const other = Object.keys(acl.states || {}).filter((s) => s !== 'capped' && s !== 'unknown');
    if (other.length) fail.push(`>= 40% ACL cell: total_state ${other.join(', ')} (want capped or unknown)`);
    lines.push('', `>= 40% ACL cell (Alice filter[priority]=3): p95 ${acl.p95} ms (limit 350), unknown ${acl.unknown_share}% of ${acl.n}, total_state ${JSON.stringify(acl.states)}, from ${acl.started_at}`);
    // rfv_num's all-visible share at the cell: the last visibility.txt sample
    // taken before the cell started (run-kit.sh samples every 10 s).
    const vf = join(dir, 'visibility.txt');
    const at = Date.parse(acl.started_at);
    const vs = existsSync(vf) ? readFileSync(vf, 'utf8').split('\n').map((l) => l.split('|'))
      .filter((f) => f[1] === 'rfv_num' && Date.parse(f[0]) <= at) : [];
    const v = vs[vs.length - 1];
    if (v) lines.push(`rfv_num at ${v[0]}: relpages ${v[2]}, relallvisible ${v[3]} (${(100 * Number(v[3]) / Math.max(1, Number(v[2]))).toFixed(1)}% all-visible), last autovacuum ${v[5] || 'never'}, last autoanalyze ${v[6] || 'never'}`);
    else lines.push('rfv_num all-visible share at the cell: no visibility.txt sample (a hand run records it in verify.sql only)');
  }
  if (ctl) lines.push(`no-filter control (admin): p50 ${ctl.p50} p95 ${ctl.p95} ms (P3-36's idle host: 73 ms; 1.1x is 80 ms)`);
  const alc = cs.find((c) => c.who === 'Alice' && c.filter === '(none)');
  if (alc) lines.push(`no-filter control (Alice): p50 ${alc.p50} p95 ${alc.p95} ms`);
  // Run validity (acceptance criterion 1): a quiet host at start and the admin
  // no-filter control <= 80 ms p95. A void run neither counts nor resets.
  const hf = join(dir, 'host-check.txt');
  const host = existsSync(hf) ? (/^host_at_start: (\S+)/m.exec(readFileSync(hf, 'utf8')) || [])[1] : undefined;
  const why = [];
  if (host !== 'quiet') why.push(host ? `host ${host} at start (host-check.txt)` : 'no host-check.txt');
  if (!ctl) why.push('admin no-filter control not measured');
  else if (!(ctl.p95 <= 80)) why.push(`admin no-filter control p95 ${ctl.p95} ms > 80 ms`);
  lines.push('', why.length ? `run validity: VOID (${why.join('; ')})` : 'run validity: VALID (quiet host at start, control <= 80 ms)');
  lines.push('', fail.length ? 'kit report: FAIL\n  ' + fail.join('\n  ') : 'kit report: PASS');
  const text = lines.join('\n') + '\n';
  writeFileSync(join(dir, 'p95.txt'), text);
  process.stdout.write(text);
  if (fail.length) process.exit(1);
}

if (mode === 'setup') await setup();
else if (mode === 'report') report();
else { console.error('usage: node kit.mjs setup|report <evidence-dir>'); process.exit(2); }
