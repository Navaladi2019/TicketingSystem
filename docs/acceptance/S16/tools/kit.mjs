// S16 acceptance (TICK-448, P5-43): walks the five S16 acceptance steps that
// need a downloaded file, a second concurrent request or the offline
// verifier, and writes the evidence to docs/acceptance/S16/evidence/.
//
//   AC1  auditor@tenant-a exports the log filtered to one incident;
//        `nava audit verify` says ok (exit 0). The same file with one line
//        deleted, and with one byte edited, says tampered (exit 1).
//   AC2  A second export started while one is streaming gets 409
//        export_in_progress; each export that started is an audit/export
//        entry in the log, and the refused one wrote none.
//   AC3  The same filtered export as CSV (format=csv): no cell starts with
//        = + - @ tab or CR unprefixed, and `nava audit verify` accepts it.
//   AC4  POST /api/v1/audit/chain:reseed: 409 chain_alive while sealing runs.
//   AC5  Erasure (TICK-449 grooming): the S13 erased and pending samples give
//        their documented verdicts; the on-stack evidence is P5-42's CI-A7
//        job. No on-stack erasure until Epic 30.
//   ISO  Bob (tenant B) gets 403 on the export and on the reseed; tenant A's
//        whole export has no tenant-B entry and not Bob's incident.
//
// Usage, from the repository root, Node.js 22, after `task build` (bin/nava)
// and docs/acceptance/S15/tools/upgrade-setup.sh:
//   node docs/acceptance/S16/tools/kit.mjs
// Ports come from the environment (NAVA_CORE_PORT, KEYCLOAK_PORT), as compose
// reads them; run-kit.sh loads .env. The dev-only seed passwords come from
// docs/acceptance/guide/bruno/environments/local.bru. Tokens are never
// written: the evidence holds requests without the Authorization header.
import { readFileSync, writeFileSync, mkdirSync, existsSync, rmSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import http from 'node:http';

const EV = 'docs/acceptance/S16/evidence';
mkdirSync(EV, { recursive: true });
const CORE_PORT = Number(process.env.NAVA_CORE_PORT || 18080);
const core = `http://127.0.0.1:${CORE_PORT}`;
const kc = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const env = readFileSync('docs/acceptance/guide/bruno/environments/local.bru', 'utf8');
const envVar = (k) => (env.match(new RegExp(`^\\s*${k}:\\s*(.*)$`, 'm')) || [])[1]?.trim();
const NAVA = ['bin/nava.exe', 'bin/nava'].find((p) => existsSync(p));
const A = '00000000-0000-4000-8000-0000000000a1';
const run = Date.now().toString(36);
const LOAD = Number(process.env.S16_LOAD || 600); // incidents with a 30,000-character description

const results = [];
function check(id, name, ok, detail) {
  results.push({ id, name, ok: !!ok, detail });
  console.log(`${ok ? 'PASS' : 'FAIL'} ${id} ${name}${detail ? ': ' + detail : ''}`);
}

async function token(user, password) {
  const r = await fetch(`${kc}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username: user, password }),
  });
  if (r.status !== 200) throw new Error(`token for ${user}: HTTP ${r.status}`);
  return (await r.json()).access_token;
}

// Tokens live a few minutes; the load step outlives one.
const users = {
  auditor: ['auditor_a_username', 'auditor_a_password'],
  alice: ['alice_username', 'alice_password'],
  bob: ['bob_username', 'bob_password'],
};
const tokens = {};
async function tok(who) {
  const t = tokens[who];
  if (t) {
    const exp = JSON.parse(Buffer.from(t.split('.')[1], 'base64url').toString()).exp * 1000;
    if (exp - Date.now() > 60000) return t;
  }
  tokens[who] = await token(envVar(users[who][0]), envVar(users[who][1]));
  return tokens[who];
}

// call: one request; the evidence file gets the request (no token) and the
// response (a body over 64 KB is cut, its size recorded).
async function call(who, method, path, { body, headers = {}, save } = {}) {
  const h = { ...headers };
  if (who) h.Authorization = `Bearer ${await tok(who)}`;
  if (body !== undefined) h['Content-Type'] = 'application/json';
  const r = await fetch(core + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  const out = { status: r.status, headers: Object.fromEntries(r.headers), text };
  try { out.json = JSON.parse(text); } catch { /* not JSON */ }
  if (save) {
    const shown = { ...h };
    if (shown.Authorization) shown.Authorization = `Bearer <${who}'s access token>`;
    writeFileSync(`${EV}/${save}.json`, JSON.stringify({
      request: { method, url: path, as: who, headers: shown, body },
      response: { status: r.status, headers: out.headers, bytes: Buffer.byteLength(text), body: out.json ?? (text.length > 65536 ? text.slice(0, 65536) + '\n<cut>' : text) },
    }, null, 2) + '\n');
  }
  return out;
}

function verify(file) {
  const r = spawnSync(NAVA, ['audit', 'verify', file], { encoding: 'utf8' });
  return { code: r.status, out: (r.stdout + r.stderr).trim() };
}
// verifyFile writes the verifier's output next to the file, as the S13 kit's
// expected/*.txt do: the exit code, then stdout.
function verifyFile(file, txt) {
  const v = verify(file);
  writeFileSync(txt, `$ nava audit verify ${file.split('/').pop()}\nexit ${v.code}\n${v.out}\n`);
  return v;
}

const sleep = (ms) => new Promise((ok) => setTimeout(ok, ms));
const lines = (s) => s.split('\n').filter((l) => l.length);

async function createIncident(who, data, key) {
  const r = await call(who, 'POST', '/api/v1/records/incident', { body: { data }, headers: { 'Idempotency-Key': key } });
  if (r.status !== 201) throw new Error(`create incident as ${who}: HTTP ${r.status} ${r.text}`);
  return r.json.id;
}

async function waitSealed(subject, limitS = 330) {
  const t0 = Date.now();
  for (;;) {
    const r = await call('auditor', 'GET', `/api/v1/audit/entries?subject_id=${subject}&limit=10`);
    const e = (r.json?.items || []).find((x) => x.action === 'create');
    if (e?.seq) return { seq: e.seq, waited_s: Math.round((Date.now() - t0) / 1000) };
    if (Date.now() - t0 > limitS * 1000) throw new Error(`entry of ${subject} not sealed after ${limitS} s`);
    await sleep(2000);
  }
}

async function exportEntries() {
  const r = await call('auditor', 'GET', '/api/v1/audit/entries?domain=audit&subject_type=audit_export&limit=100');
  if (r.status !== 200) throw new Error(`list export entries: HTTP ${r.status}`);
  return r.json.items.filter((e) => e.action === 'export');
}

// ---------------------------------------------------------------- setup
console.log(`s16: run ${run}, core ${core}, verifier ${NAVA}`);
if (!NAVA) throw new Error('bin/nava not found: run task build first');
const incident = await createIncident('alice', {
  short_description: `=HYPERLINK("http://example.invalid","S16 ${run}")`,
  description: '@SUM(1+1) +1 -1 S16 acceptance: formula-looking text, which must never execute in a spreadsheet',
  priority: 2,
}, `tick448-s16-${run}-incident`);
const bobIncident = await createIncident('bob', { short_description: `S16 ${run} tenant B`, priority: 3 }, `tick448-s16-${run}-bob`);
console.log(`s16: Alice's incident ${incident}, Bob's ${bobIncident}`);
const sealed = await waitSealed(incident);
console.log(`s16: Alice's incident sealed at seq ${sealed.seq} (polled ${sealed.waited_s} s)`);

// ---------------------------------------------------------------- AC1
const filtered = await call('auditor', 'GET', `/api/v1/audit/entries:export?subject_id=${incident}`);
const F = `${EV}/ac1-export-filtered.jsonl`;
writeFileSync(F, filtered.text);
const fl = lines(filtered.text).map((l) => JSON.parse(l));
check('AC1', 'filtered export: 200, application/x-ndjson', filtered.status === 200 && /application\/x-ndjson/.test(filtered.headers['content-type']), `HTTP ${filtered.status}, ${filtered.headers['content-type']}, ${fl.length} lines`);
const full = fl.filter((l) => l.kind === 'entry');
check('AC1', 'every full line is the incident; the rest of the range are stubs',
  full.length >= 1 && full.every((l) => l.envelope.subject_id === incident) && fl.some((l) => l.kind === 'stub'),
  `${full.length} full, ${fl.filter((l) => l.kind === 'stub').length} stubs`);
check('AC1', 'the header records the filter', fl[0]?.kind === 'header' && JSON.stringify(fl[0]).includes(incident));
let v = verifyFile(F, `${EV}/ac1-verify-ok.txt`);
check('AC1', 'nava audit verify: ok, exit 0', v.code === 0 && v.out.startsWith('ok: '), v.out.split('\n')[0]);

// One line deleted: the incident's own line, so the range has a gap.
const raw = lines(filtered.text);
const idx = raw.findIndex((l) => l.includes(`"subject_id":"${incident}"`) && l.startsWith('{"kind":"entry"'));
const del = raw.filter((_, i) => i !== idx).join('\n') + '\n';
writeFileSync(`${EV}/ac1-export-line-deleted.jsonl`, del);
v = verifyFile(`${EV}/ac1-export-line-deleted.jsonl`, `${EV}/ac1-verify-line-deleted.txt`);
check('AC1', 'one line deleted: tampered, exit 1', v.code === 1 && v.out.startsWith('tampered: '), v.out);

// One byte edited: the first hex digit of the incident entry's request_id,
// as in S13's export-byte-edited sample.
const line = raw[idx];
const m = line.match(/"request_id":"([0-9a-f])/);
const at = m.index + m[0].length - 1;
const flipped = line.slice(0, at) + (m[1] === '0' ? '1' : '0') + line.slice(at + 1);
writeFileSync(`${EV}/ac1-export-byte-edited.jsonl`, raw.map((l, i) => (i === idx ? flipped : l)).join('\n') + '\n');
v = verifyFile(`${EV}/ac1-export-byte-edited.jsonl`, `${EV}/ac1-verify-byte-edited.txt`);
check('AC1', 'one byte edited: tampered, exit 1', v.code === 1 && v.out.startsWith('tampered: '), v.out);

// ---------------------------------------------------------------- AC2
// A whole-log export large enough that it is still streaming when the
// second request arrives: Alice writes LOAD incidents with a 30,000-character
// description, then the first export is opened and read by a paused client,
// so the server blocks on a full socket and keeps the lease.
const pad = 'S16 load '.padEnd(30000, 'x');
for (let i = 0; i < LOAD; i += 10) {
  await Promise.all(Array.from({ length: Math.min(10, LOAD - i) }, (_, j) =>
    createIncident('alice', { short_description: `S16 ${run} load ${i + j}`, description: pad, priority: 4 }, `tick448-s16-${run}-load-${i + j}`)));
}
console.log(`s16: ${LOAD} large incidents written`);
const before = await exportEntries();
const auditorTok = await tok('auditor');
const slow = await new Promise((ok, fail) => {
  const req = http.get({ host: '127.0.0.1', port: CORE_PORT, path: '/api/v1/audit/entries:export', headers: { Authorization: `Bearer ${auditorTok}` } }, (res) => {
    res.pause();
    ok({ req, res });
  });
  req.on('error', fail);
});
check('AC2', 'first export: 200 and streaming', slow.res.statusCode === 200, `HTTP ${slow.res.statusCode}`);
await sleep(2000);
const second = await call('auditor', 'GET', `/api/v1/audit/entries:export?subject_id=${incident}`, { save: 'ac2-second-export-409' });
check('AC2', 'second export while the first streams: 409 export_in_progress', second.status === 409 && second.json?.code === 'export_in_progress', `HTTP ${second.status} ${second.json?.code}`);
// Drain the first export to the end, so it releases the lease.
let bytes = 0;
const firstText = await new Promise((ok) => {
  const chunks = [];
  slow.res.on('data', (c) => { bytes += c.length; chunks.push(c); });
  slow.res.on('end', () => ok(Buffer.concat(chunks).toString('utf8')));
  slow.res.resume();
});
const firstLines = lines(firstText).map((l) => JSON.parse(l));
check('AC2', 'the first export finished with a trailer', firstLines.at(-1)?.kind === 'trailer', `${bytes} bytes, ${firstLines.length} lines`);
writeFileSync(`${EV}/ac2-first-export-summary.json`, JSON.stringify({
  bytes, lines: firstLines.length, header: firstLines[0], trailer: firstLines.at(-1),
}, null, 2) + '\n');
writeFileSync(`${EV}/.ac2-first-export.jsonl`, firstText);
v = verifyFile(`${EV}/.ac2-first-export.jsonl`, `${EV}/ac2-verify-first-export.txt`);
rmSync(`${EV}/.ac2-first-export.jsonl`); // tens of MB: the summary and the verdict are the evidence
check('AC2', 'the whole-log export verifies: ok, exit 0', v.code === 0, v.out.split('\n')[0]);
const after = await exportEntries();
const added = after.filter((e) => !before.some((b) => b.id === e.id));
writeFileSync(`${EV}/ac2-export-entries.json`, JSON.stringify({ before: before.length, after: after.length, added }, null, 2) + '\n');
check('AC2', 'the streamed export is one audit/export entry; the refused one wrote none', added.length === 1, `${added.length} new audit/export entries`);
check('AC2', 'the entry records the request', added.length === 1 && JSON.stringify(added[0].diff ?? '').includes('request'), JSON.stringify(added[0]?.diff ?? null).slice(0, 200));
// The filtered export of AC1 is an entry too.
check('AC2', "AC1's filtered export is an audit/export entry with its filter", before.some((e) => JSON.stringify(e.diff ?? '').includes(incident)));

// ---------------------------------------------------------------- AC3
const csv = await call('auditor', 'GET', `/api/v1/audit/entries:export?subject_id=${incident}&format=csv`);
const C = `${EV}/ac3-export-filtered.csv`;
writeFileSync(C, csv.text);
const ctype = csv.headers['content-type'] || '';
check('AC3', 'format=csv: 200, text/csv', csv.status === 200 && /text\/csv/.test(ctype), `HTTP ${csv.status}, ${ctype}`);
if (/text\/csv/.test(ctype)) {
  // RFC 4180 cells, CRLF rows; a quoted cell may hold commas, quotes and LF.
  const cells = [];
  let cell = '', q = false;
  for (let i = 0; i < csv.text.length; i++) {
    const ch = csv.text[i];
    if (q) {
      if (ch === '"' && csv.text[i + 1] === '"') { cell += '"'; i++; } else if (ch === '"') q = false; else cell += ch;
    } else if (ch === '"') q = true;
    else if (ch === ',') { cells.push(cell); cell = ''; } else if (ch === '\r' && csv.text[i + 1] === '\n') { cells.push(cell); cell = ''; i++; } else cell += ch;
  }
  const bad = cells.filter((c) => /^[=+\-@\t\r]/.test(c));
  const prefixed = cells.filter((c) => /^'[=+\-@\t\r']/.test(c));
  check('AC3', 'no cell starts with = + - @ tab or CR unprefixed', bad.length === 0, `${bad.length} unprefixed, ${prefixed.length} prefixed with '`);
  v = verifyFile(C, `${EV}/ac3-verify-csv.txt`);
  check('AC3', 'nava audit verify on the CSV: ok, exit 0', v.code === 0, v.out.split('\n')[0]);
} else {
  writeFileSync(`${EV}/ac3-format-csv-response.txt`, `GET /api/v1/audit/entries:export?subject_id=${incident}&format=csv\nHTTP ${csv.status}\ncontent-type: ${ctype}\n\n${csv.text.slice(0, 600)}\n`);
}

// ---------------------------------------------------------------- AC4
const reseed = await call('auditor', 'POST', '/api/v1/audit/chain:reseed', { headers: { 'Idempotency-Key': `tick448-s16-${run}-reseed` }, save: 'ac4-reseed-409' });
check('AC4', 'chain:reseed while sealing runs: 409 chain_alive', reseed.status === 409 && reseed.json?.code === 'chain_alive', `HTTP ${reseed.status} ${reseed.json?.code}`);

// ---------------------------------------------------------------- AC5
// TICK-449: the evidence is P5-42's CI-A7 job plus S13's samples.
for (const [file, code] of [['export-erased-pending', 3], ['export-erased-without-erase-entry', 1]]) {
  const want = readFileSync(`docs/acceptance/S13/expected/${file}.txt`, 'utf8').trim();
  const got = verify(`docs/acceptance/S13/${file}.jsonl`);
  const text = `exit ${got.code}\n${got.out}`;
  writeFileSync(`${EV}/ac5-${file}.txt`, `$ nava audit verify docs/acceptance/S13/${file}.jsonl\n${text}\n`);
  check('AC5', `S13 sample ${file}: exit ${code}, as documented`, got.code === code && text === want, text.split('\n')[1]);
}

// ---------------------------------------------------------------- isolation
const bobExport = await call('bob', 'GET', '/api/v1/audit/entries:export', { save: 'iso-bob-export-403' });
check('ISO', "Bob (tenant B, no audit.export) cannot export: 403", bobExport.status === 403 && bobExport.json?.code === 'forbidden', `HTTP ${bobExport.status}`);
const bobReseed = await call('bob', 'POST', '/api/v1/audit/chain:reseed', { headers: { 'Idempotency-Key': `tick448-s16-${run}-bob-reseed` }, save: 'iso-bob-reseed-403' });
check('ISO', 'Bob cannot reseed: 403', bobReseed.status === 403 && bobReseed.json?.code === 'forbidden', `HTTP ${bobReseed.status}`);
const tenants = new Set(firstLines.filter((l) => l.envelope).map((l) => l.envelope.tenant_id));
check('ISO', "tenant A's whole export holds only tenant A's entries", tenants.size === 1 && tenants.has(A), [...tenants].join(','));
check('ISO', "Bob's incident is nowhere in tenant A's export", !firstText.includes(bobIncident) && !filtered.text.includes(bobIncident));

writeFileSync(`${EV}/kit-summary.json`, JSON.stringify({ run, at: new Date().toISOString(), incident, bob_incident: bobIncident, sealed, load: LOAD, results }, null, 2) + '\n');
const failed = results.filter((r) => !r.ok);
console.log(failed.length ? `s16 kit: FAILED ${failed.length} of ${results.length}` : `s16 kit: PASS (${results.length} checks)`);
process.exit(failed.length ? 1 : 0);
