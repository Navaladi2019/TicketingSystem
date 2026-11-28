// S29 acceptance (TICK-754): the four stack checks the Technical Lead carried
// over from TICK-920 (P10-05) to the fresh-stack run (TICK-754, 2026-11-26
// 23:01Z). run-kit.sh runs them after the Bruno folder and verify.sql, on the
// same stack, because they start more jobs (one of them in tenant B).
//
//   node extras.mjs <evidence-dir>
//
//   E1 CI-4, the wait bound, measured on the server (Technical Lead,
//      2026-11-28 09:26Z): 4 live writers edit seeded incidents for 15 s with
//      no job (the baseline), then while a job adds `s29_x1` to all ~2,050,
//      each writer paced to about 3.5 writes a second (28 requests a second
//      in all, under the 50/s tenant rate limit). Each write: read, then
//      PATCH description with If-Match; a 412 (the job rewrote the record in
//      between: a real interleave) re-reads and retries. For the whole job a
//      sampler inside PostgreSQL (one psql session, a loop that commits after
//      every sample, so the backfill's snapshot barrier never waits on it)
//      reads pg_locks and pg_stat_activity: every ungranted lock of a
//      live-write backend (role nava_app), its waitstart and
//      pg_blocking_pids, and every open transaction of a backfill backend
//      (role nava_worker) with its xact_start. Asserted, the pass rule as
//      ruled:
//        - no live-write backend is ever blocked by more than one backfill
//          backend at once;
//        - no single wait on a backfill backend lasts longer than the p99
//          duration of one backfill transaction measured in the same run
//          (both come from the same samples, so both are lower bounds at
//          the same resolution; the sampling interval is reported);
//        - every 412 retried to success; no lost update (each record holds
//          the writer's last description and the job's value).
//      Client PATCH latencies (baseline and during the job) are `info` lines
//      only; the earlier +250 ms client bound is dropped.
//   E2 CI-11, a real worker kill: a job adds `s29_x2`; once it has rewritten
//      at least 300 records, `docker compose kill nava-worker` mid-page, 3 s,
//      `docker compose start nava-worker`. Asserted: the job ends, and every
//      record it rewrote has exactly one audit entry for it (the per-record
//      count of schema-backfill entries rises by exactly 1; synchronous_commit
//      is off in the rewrite, D7).
//   E3 NAVA_SCHEMA_BACKFILL_CONCURRENCY=1: the worker restarted with the
//      override; tenant A (s29_x3 on incident) and tenant B (s29_x3 on its
//      incident, 2,000 seeded records) publish a default 1 s apart. Asserted:
//      the two jobs' [started_at, finished_at] do not overlap. The worker is
//      restarted without the override after. Known to fail until TICK-1208
//      (cap running jobs across the pool, ADR-009 D7) merges: while
//      E3_PENDING names it, an E3 failure is written `FAIL (pending
//      TICK-1208)` and does not fail the run; an E3 pass while it is still
//      set prints a reminder to clear it.
//   E4 R9 watch item: the history length of every schema.backfill workflow in
//      the run (Temporal visibility through the UI API), the largest against
//      Temporal's 51,200-event limit. Recorded; a history over half the limit
//      fails the step (the trigger to revisit D2's start-only rule).
//
// Writes <evidence-dir>/extras.json and extras.txt (`check|...|ok` lines, as
// verify.sql). No credentials in either. Ports from the compose variables.
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { execFileSync, spawn } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';

const dir = process.argv[2];
const CORE = `http://127.0.0.1:${process.env.NAVA_CORE_PORT || 18080}`;
const KC = `http://127.0.0.1:${process.env.KEYCLOAK_PORT || 18081}`;
const TUI = `http://127.0.0.1:${process.env.ACCEPT_TEMPORAL_UI_PORT || 18233}`;
const A = '00000000-0000-4000-8000-0000000000a1';
const B = '00000000-0000-4000-8000-0000000000b1';
// E3 stays failing until this task merges (Technical Lead, 2026-11-28 09:26Z).
// Clear it in the change that re-runs E3 after TICK-1208.
const E3_PENDING = 'TICK-1208';
const HISTORY_LIMIT = 51200;
const out = { started_at: new Date().toISOString() };
const checks = [];
const check = (name, value, ok, pending) => { checks.push(`check|${name}|${value}|${ok ? 'ok' : pending ? `FAIL (pending ${pending})` : 'FAIL'}`); console.log(checks[checks.length - 1]); };
const info = (name, value) => { checks.push(`info|${name}|${value}`); console.log(checks[checks.length - 1]); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const md5uuid = (s) => { const h = createHash('md5').update(s).digest('hex'); return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`; };
const compose = (...args) => execFileSync('docker', ['compose', ...(process.env.COMPOSE_FILE_ARGS ? process.env.COMPOSE_FILE_ARGS.split(' ') : []), ...args], { stdio: ['ignore', 'pipe', 'inherit'] }).toString();
const psql = (sql) => execFileSync('docker', ['compose', 'exec', '-T', 'postgres', 'sh', '-c',
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -At -v ON_ERROR_STOP=1 -U pgadmin -d nava'], { input: sql, stdio: ['pipe', 'pipe', 'inherit'] }).toString().trim();

const tokens = {};
async function tok(user, pass) {
  const t = tokens[user];
  if (t && t.exp - Date.now() > 90000) return t.v;
  const r = await fetch(`${KC}/realms/nava/protocol/openid-connect/token`, {
    method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username: user, password: pass }),
  });
  if (r.status !== 200) throw new Error(`sign-in ${user}: HTTP ${r.status}`);
  const j = await r.json();
  tokens[user] = { v: j.access_token, exp: Date.now() + j.expires_in * 1000 };
  return j.access_token;
}
const ADMIN_A = ['admin@tenant-a.test', 'dev-only-admin-a'];
const ADMIN_B = ['admin@tenant-b.test', 'dev-only-admin-b'];
async function call(who, method, path, body, headers = {}) {
  const h = { Authorization: 'Bearer ' + (await tok(...who)), ...headers };
  if (body !== undefined && !h['Content-Type']) h['Content-Type'] = 'application/json';
  if (method !== 'GET' && !h['Idempotency-Key']) h['Idempotency-Key'] = 's29x-' + randomUUID();
  const t0 = Date.now();
  const r = await fetch(CORE + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  let json = null;
  try { json = JSON.parse(text); } catch { /* not JSON */ }
  return { status: r.status, json, ms: Date.now() - t0 };
}
const backfill = async (who) => (await call(who, 'GET', '/api/v1/entity-types/incident')).json.backfill;
async function publish(who, mutate, what) {
  const cur = await call(who, 'GET', '/api/v1/entity-types/incident');
  const schema = cur.json.schema;
  mutate(schema);
  const r = await call(who, 'POST', '/api/v1/entity-types/incident/versions', { definition: { schema, fixed_columns: cur.json.fixed_columns, states: cur.json.states } });
  if (r.status !== 201) throw new Error(`${what}: HTTP ${r.status} ${JSON.stringify(r.json)}`);
  return r.json.version;
}
async function waitJob(who, version, limitMs = 900000, onSample) {
  const t0 = Date.now(), ss = [];
  for (;;) {
    const b = await backfill(who);
    ss.push({ at_s: Math.round((Date.now() - t0) / 1000), state: b && b.state, target: b && b.target_version, rewritten: b && b.rewritten, remaining: b && b.remaining });
    if (onSample) await onSample(b);
    if (b && b.target_version === version && ['done', 'failed_records', 'superseded'].includes(b.state)) return { job: b, samples: ss };
    if (Date.now() - t0 > limitMs) throw new Error(`job of version ${version} did not end in ${limitMs / 1000} s: ${JSON.stringify(b)}`);
    await sleep(1000);
  }
}
const pct = (xs, p) => { const s = [...xs].sort((a, b) => a - b); return s.length ? s[Math.min(s.length - 1, Math.ceil((p / 100) * s.length) - 1)] : null; };
const perRecordCounts = () => Object.fromEntries(psql(`SELECT r.id || '|' || (SELECT count(*) FROM audit_log l WHERE l.tenant_id = r.tenant_id AND l.subject_id = r.id AND l.actor_system = 'schema-backfill')
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '${A}' AND d.key = 'incident';`).split('\n').filter(Boolean).map((l) => { const [id, n] = l.split('|'); return [id, Number(n)]; }));

// The E1 sampler: one psql session until the stop table has a row. Every
// iteration is its own short transaction (COMMIT in the loop), so it is never
// in the set the backfill's snapshot barrier waits on. Prints one JSON line.
const SAMPLER_SQL = `SET synchronous_commit = off;
CREATE TEMP TABLE e1_wait (it bigint, ts timestamptz, pid int, waitstart timestamptz, blockers int, bf_blockers int, locktype text);
CREATE TEMP TABLE e1_tx (it bigint, ts timestamptz, pid int, xact_start timestamptz);
CREATE TEMP TABLE e1_run (n bigint, t0 timestamptz, t1 timestamptz);
DO $sampler$
DECLARE n bigint := 0; t0 timestamptz := clock_timestamp();
BEGIN
  LOOP
    EXIT WHEN EXISTS (SELECT 1 FROM s29x_e1_stop) OR clock_timestamp() > t0 + interval '20 minutes';
    n := n + 1;
    INSERT INTO e1_wait
    SELECT n, clock_timestamp(), l.pid, l.waitstart, cardinality(bp.b),
           (SELECT count(*) FROM pg_stat_activity w WHERE w.pid = ANY (bp.b) AND w.usename = 'nava_worker'), l.locktype
      FROM pg_locks l JOIN pg_stat_activity a ON a.pid = l.pid
     CROSS JOIN LATERAL (SELECT pg_blocking_pids(l.pid) AS b) bp
     WHERE NOT l.granted AND a.usename = 'nava_app' AND a.datname = 'nava';
    INSERT INTO e1_tx
    SELECT n, clock_timestamp(), pid, xact_start FROM pg_stat_activity
     WHERE usename = 'nava_worker' AND datname = 'nava' AND xact_start IS NOT NULL AND backend_type = 'client backend';
    COMMIT;
    PERFORM pg_sleep(0.0005);
  END LOOP;
  INSERT INTO e1_run VALUES (n, t0, clock_timestamp());
  COMMIT;
END $sampler$;
WITH w AS (SELECT pid, waitstart, max(bf_blockers) AS bf, extract(epoch FROM max(ts) - waitstart) * 1000 AS ms
             FROM e1_wait WHERE waitstart IS NOT NULL GROUP BY pid, waitstart),
     x AS (SELECT pid, xact_start, extract(epoch FROM max(ts) - xact_start) * 1000 AS ms
             FROM e1_tx GROUP BY pid, xact_start)
SELECT json_build_object(
  'samples', r.n,
  'seconds', round(extract(epoch FROM r.t1 - r.t0)::numeric, 1),
  'avg_interval_ms', round((extract(epoch FROM r.t1 - r.t0) * 1000 / greatest(r.n, 1))::numeric, 3),
  'max_backfill_blockers_at_once', coalesce((SELECT max(bf_blockers) FROM e1_wait), 0),
  'ungranted_lock_samples', (SELECT count(*) FROM e1_wait),
  'waits_on_backfill', (SELECT count(*) FROM w WHERE bf > 0),
  'waits_on_other', (SELECT count(*) FROM w WHERE bf = 0),
  'max_wait_on_backfill_ms', (SELECT round(max(ms)::numeric, 3) FROM w WHERE bf > 0),
  'max_wait_on_other_ms', (SELECT round(max(ms)::numeric, 3) FROM w WHERE bf = 0),
  'longest_waits_on_backfill_ms', (SELECT coalesce(json_agg(round(ms::numeric, 3) ORDER BY ms DESC), '[]') FROM (SELECT ms FROM w WHERE bf > 0 ORDER BY ms DESC LIMIT 20) t),
  'lock_types', (SELECT coalesce(json_object_agg(locktype, c), '{}') FROM (SELECT locktype, count(*) AS c FROM e1_wait GROUP BY locktype) t),
  'backfill_tx', (SELECT count(*) FROM x),
  'backfill_tx_p50_ms', (SELECT round((percentile_cont(0.5) WITHIN GROUP (ORDER BY ms))::numeric, 3) FROM x),
  'backfill_tx_p99_ms', (SELECT round((percentile_cont(0.99) WITHIN GROUP (ORDER BY ms))::numeric, 3) FROM x),
  'backfill_tx_max_ms', (SELECT round(max(ms)::numeric, 3) FROM x))
FROM e1_run r;
`;
function startSampler() {
  psql('DROP TABLE IF EXISTS s29x_e1_stop; CREATE TABLE s29x_e1_stop (at timestamptz); SELECT 1;');
  const p = spawn('docker', ['compose', 'exec', '-T', 'postgres', 'sh', '-c',
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -At -q -v ON_ERROR_STOP=1 -U pgadmin -d nava'], { stdio: ['pipe', 'pipe', 'inherit'] });
  let stdout = '';
  p.stdout.on('data', (d) => { stdout += d; });
  const done = new Promise((resolve, reject) => p.on('close', (code) => {
    if (code !== 0) return reject(new Error(`E1 sampler: psql exit ${code}`));
    const line = stdout.trim().split('\n').filter((l) => l.startsWith('{')).pop();
    if (!line) return reject(new Error('E1 sampler: no summary line'));
    resolve(JSON.parse(line));
  }));
  p.stdin.end(SAMPLER_SQL);
  return {
    stop: async () => {
      psql('INSERT INTO s29x_e1_stop VALUES (now()); SELECT 1;');
      try { return await done; } finally { psql('DROP TABLE IF EXISTS s29x_e1_stop; SELECT 1;'); }
    },
  };
}

// ---------------------------------------------------------------- E1 CI-4
async function e1() {
  await publish(ADMIN_A, (s) => { s.properties.s29_x1 = { type: 'string', maxLength: 16 }; s.properties.s29_x2 = { type: 'string', maxLength: 16 }; s.properties.s29_x3 = { type: 'string', maxLength: 16 }; }, 'add s29_x1..x3');
  // Each writer owns 50 seeded records (ordinals 101..300): never the poison (2000).
  const owned = [0, 1, 2, 3].map((w) => Array.from({ length: 50 }, (_, i) => md5uuid('s29a/' + (101 + w * 50 + i))));
  const last = {};
  const stats = { baseline: [], during: [], s412: 0, retried_ok: 0, failed: [] };
  let phase = 'baseline', stop = false;
  const writer = async (w) => {
    let k = 0;
    while (!stop) {
      const id = owned[w][k % 50];
      const desc = `s29x writer ${w} write ${k}`;
      const started = Date.now();
      for (let attempt = 0; ; attempt++) {
        const cur = await call(ADMIN_A, 'GET', '/api/v1/records/incident/' + id);
        const r = await call(ADMIN_A, 'PATCH', '/api/v1/records/incident/' + id, { data: { description: desc } },
          { 'Content-Type': 'application/merge-patch+json', 'If-Match': `W/"${cur.json.version}"` });
        if (r.status === 200) { if (attempt > 0) stats.retried_ok++; stats[phase].push(r.ms); break; }
        if (r.status === 412 && attempt < 5) { stats.s412++; continue; }
        stats.failed.push({ id, status: r.status, code: r.json && r.json.code });
        break;
      }
      last[id] = desc;
      k++;
      // Paced: each write is a GET and a PATCH; 4 writers at about 3.5 writes a
      // second each stay near 28 requests a second, under the tenant's 50/s
      // rate limit (a 429 would be the limiter, not the backfill).
      await sleep(Math.max(0, 285 - (Date.now() - started)));
    }
  };
  const ws = [0, 1, 2, 3].map(writer);
  await sleep(15000);
  phase = 'during';
  const sampler = startSampler();
  await sleep(500);
  let job, samples, server, control;
  try {
    // Positive control, before the job: a superuser session holds the row
    // lock of one seeded incident (ordinal 400, owned by no writer) for
    // 4 s while the kit PATCHes it. The sampler must see that wait, under
    // "anything else" (the blocker is not nava_worker), or it proves nothing.
    const ctl = md5uuid('s29a/400');
    const holder = spawn('docker', ['compose', 'exec', '-T', 'postgres', 'sh', '-c',
      'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -At -q -v ON_ERROR_STOP=1 -U pgadmin -d nava'], { stdio: ['pipe', 'ignore', 'inherit'] });
    const held = new Promise((resolve) => holder.on('close', resolve));
    holder.stdin.end(`BEGIN; SELECT 1 FROM records WHERE tenant_id = '${A}' AND id = '${ctl}' FOR UPDATE; SELECT pg_sleep(4) /* s29x control */; COMMIT;`);
    // docker compose exec takes about a second to start: PATCH only once the
    // holder is asleep with the lock taken.
    let locked = false;
    for (let i = 0; i < 20 && !locked; i++) {
      locked = psql(`SELECT count(*) FROM pg_stat_activity WHERE usename = 'pgadmin' AND wait_event = 'PgSleep' AND query LIKE '%s29x control%' AND pid <> pg_backend_pid();`) === '1';
      if (!locked) await sleep(100);
    }
    const cur = await call(ADMIN_A, 'GET', '/api/v1/records/incident/' + ctl);
    const r = await call(ADMIN_A, 'PATCH', '/api/v1/records/incident/' + ctl, { data: { description: 's29x control' } },
      { 'Content-Type': 'application/merge-patch+json', 'If-Match': `W/"${cur.json.version}"` });
    control = { locked_before_patch: locked, status: r.status, ms: r.ms, holder_exit: await held };
    await sleep(500);
    const v = await publish(ADMIN_A, (s) => { s.properties.s29_x1.default = 'a'; }, 'default s29_x1');
    ({ job, samples } = await waitJob(ADMIN_A, v));
  } finally {
    server = await sampler.stop().catch((e) => ({ error: e.message }));
    stop = true;
  }
  await Promise.all(ws);
  const lost = [];
  for (const [id, desc] of Object.entries(last)) {
    const r = await call(ADMIN_A, 'GET', '/api/v1/records/incident/' + id);
    if (r.json.data.description !== desc || r.json.data.s29_x1 !== 'a') lost.push({ id, description: r.json.data.description, s29_x1: r.json.data.s29_x1 });
  }
  const stored = Number(psql(`SELECT count(*) FROM records WHERE tenant_id = '${A}' AND id = ANY (ARRAY[${Object.keys(last).map((i) => `'${i}'::uuid`).join(',')}]) AND data->>'s29_x1' = 'a';`));
  const bmax = Math.max(...stats.baseline), dmax = Math.max(...stats.during);
  const sv = server || { error: 'no sampler result' };
  out.e1 = { job, samples, server: sv, control, writes: { baseline: stats.baseline.length, during: stats.during.length },
    client_info_only: { baseline_ms: { p50: pct(stats.baseline, 50), p99: pct(stats.baseline, 99), max: bmax },
      during_ms: { p50: pct(stats.during, 50), p99: pct(stats.during, 99), max: dmax } },
    s412: stats.s412, retried_ok: stats.retried_ok, failed: stats.failed, lost };
  check('E1 CI-4 writes during the job (4 writers)', stats.during.length, stats.during.length >= 40);
  check('E1 CI-4 server sampler covered the job',
    sv.error || `${sv.samples} samples over ${sv.seconds} s (every ${sv.avg_interval_ms} ms on average); ${sv.backfill_tx} backfill transactions seen`,
    !sv.error && sv.samples >= 1000 && sv.backfill_tx >= 100);
  check('E1 CI-4 positive control: the sampler sees a held row lock (4 s, not a backfill backend)',
    sv.error || `lock held first: ${control && control.locked_before_patch}; PATCH ${control && control.status} in ${control && control.ms} ms; ${sv.waits_on_other} wait(s) on anything else, longest ${sv.max_wait_on_other_ms} ms`,
    !sv.error && control && control.locked_before_patch && control.status === 200 && control.holder_exit === 0 && sv.waits_on_other >= 1 && Number(sv.max_wait_on_other_ms) >= 500);
  check('E1 CI-4 no live write blocked by more than one backfill backend at once',
    sv.error || `max ${sv.max_backfill_blockers_at_once} at once; ${sv.waits_on_backfill} waits on a backfill backend, ${sv.waits_on_other} on anything else`,
    !sv.error && sv.max_backfill_blockers_at_once <= 1);
  check('E1 CI-4 longest wait on a backfill backend <= p99 of one backfill transaction, same run',
    sv.error || `${sv.max_wait_on_backfill_ms ?? 0} ms vs p99 ${sv.backfill_tx_p99_ms} ms (p50 ${sv.backfill_tx_p50_ms}, max ${sv.backfill_tx_max_ms})`,
    !sv.error && sv.backfill_tx_p99_ms != null && Number(sv.max_wait_on_backfill_ms ?? 0) <= Number(sv.backfill_tx_p99_ms));
  info('E1 CI-4 client PATCH latency, no job (information only)', `p50 ${pct(stats.baseline, 50)} / p99 ${pct(stats.baseline, 99)} / max ${bmax} ms`);
  info('E1 CI-4 client PATCH latency, during the job (information only)', `p50 ${pct(stats.during, 50)} / p99 ${pct(stats.during, 99)} / max ${dmax} ms`);
  check('E1 CI-4 every 412 retried to success, no write failed', `${stats.s412} 412s, ${stats.retried_ok} retried, ${stats.failed.length} failed`, stats.failed.length === 0 && stats.retried_ok <= stats.s412);
  check('E1 CI-4 no lost update (last description and the job value both stored)', `${lost.length} lost of ${Object.keys(last).length}; ${stored} stored s29_x1`, lost.length === 0 && stored === Object.keys(last).length);
  check('E1 CI-4 the job ended', job.state, ['done', 'failed_records'].includes(job.state));
}

// ---------------------------------------------------------------- E2 CI-11
async function e2() {
  const before = perRecordCounts();
  const v = await publish(ADMIN_A, (s) => { s.properties.s29_x2.default = 'b'; }, 'default s29_x2');
  let killed = null;
  const { job, samples } = await waitJob(ADMIN_A, v, 900000, async (b) => {
    if (!killed && b && b.target_version === v && b.state === 'running' && b.rewritten >= 300) {
      killed = { at: new Date().toISOString(), rewritten: b.rewritten, remaining: b.remaining };
      compose('kill', 'nava-worker');
      await sleep(3000);
      compose('start', 'nava-worker');
      killed.restarted_at = new Date().toISOString();
    }
  });
  const after = perRecordCounts();
  const poison = md5uuid('s29a/2000');
  const deltas = {};
  for (const [id, n] of Object.entries(after)) { const d = n - (before[id] || 0); deltas[d] = (deltas[d] || 0) + 1; }
  const wrong = Object.entries(after).filter(([id, n]) => id !== poison && n - (before[id] || 0) !== 1).length;
  out.e2 = { job, samples, killed, deltas, wrong };
  check('E2 CI-11 nava-worker killed mid-job', killed ? `at ${killed.rewritten} rewritten, ${killed.remaining} remaining` : 'never', !!killed && killed.remaining > 1);
  check('E2 CI-11 the job ended after the kill', job.state, ['done', 'failed_records'].includes(job.state));
  check('E2 CI-11 exactly one new schema-backfill audit entry per rewritten record', `deltas ${JSON.stringify(deltas)}, ${wrong} wrong`, wrong === 0 && (after[poison] || 0) === (before[poison] || 0));
}

// ---------------------------------------------------------------- E3 concurrency 1
async function e3() {
  // Tenant B: 2,000 incidents with no s29_x3, and the field on its type.
  psql(`SET session_replication_role = replica;
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
                     requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
SELECT d.tenant_id, md5('s29b/' || n)::uuid, 1, d.id, d.current_version, 'INC8' || lpad(n::text, 6, '0'), 'new', p.id, NULL, NULL, p.id,
       timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute' - interval '1 day', timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute',
       jsonb_build_object('short_description', 'S29 tenant B seed ' || n, 'state', 'new'), 0
  FROM entity_definitions d JOIN principals p ON p.tenant_id = d.tenant_id AND p.email = 'bob@tenant-b.test', generate_series(1, 2000) n
 WHERE d.tenant_id = '${B}' AND d.key = 'incident';
SET session_replication_role = origin;
SELECT 1;`);
  await publish(ADMIN_B, (s) => { s.properties.s29_x3 = { type: 'string', maxLength: 16 }; }, 'tenant B: add s29_x3');
  const files = ['-f', 'compose.yaml', '-f', 'docs/acceptance/S29/tools/compose.concurrency1.yaml'];
  execFileSync('docker', ['compose', ...files, 'up', '-d', '--no-deps', '--wait', 'nava-worker'], { stdio: 'inherit' });
  const env = compose('exec', '-T', 'nava-worker', 'env').split('\n').find((l) => l.startsWith('NAVA_SCHEMA_BACKFILL_CONCURRENCY=')) || 'unset';
  const va = await publish(ADMIN_A, (s) => { s.properties.s29_x3.default = 'c'; }, 'tenant A: default s29_x3');
  await sleep(1000);
  const vb = await publish(ADMIN_B, (s) => { s.properties.s29_x3.default = 'c'; }, 'tenant B: default s29_x3');
  const [ra, rb] = await Promise.all([waitJob(ADMIN_A, va), waitJob(ADMIN_B, vb)]);
  execFileSync('docker', ['compose', 'up', '-d', '--no-deps', '--wait', 'nava-worker'], { stdio: 'inherit' });
  const ja = ra.job, jb = rb.job;
  const [a0, a1, b0, b1] = [ja.started_at, ja.finished_at, jb.started_at, jb.finished_at].map((t) => Date.parse(t));
  const overlap = Math.max(0, Math.min(a1, b1) - Math.max(a0, b0));
  out.e3 = { worker_env: env, a: ja, b: jb, overlap_ms: overlap, samples_a: ra.samples, samples_b: rb.samples };
  check('E3 the worker ran with the override', env, env === 'NAVA_SCHEMA_BACKFILL_CONCURRENCY=1');
  check('E3 both tenants\' jobs ended', `A ${ja.state}, B ${jb.state}`, ['done', 'failed_records'].includes(ja.state) && jb.state === 'done');
  check('E3 concurrency 1: the two tenants\' jobs ran one after the other, not in parallel', `A ${ja.started_at}..${ja.finished_at}, B ${jb.started_at}..${jb.finished_at}, overlap ${overlap} ms`, overlap === 0, E3_PENDING);
  if (overlap === 0 && E3_PENDING) console.log(`s29 extras: E3 passes while E3_PENDING is still ${E3_PENDING}: clear it in extras.mjs`);
}

// ---------------------------------------------------------------- E4 R9
async function e4() {
  const q = encodeURIComponent("WorkflowType='schema.backfill'");
  const list = await (await fetch(`${TUI}/api/v1/namespaces/nava/workflows?query=${q}`)).json();
  const lens = [];
  for (const e of list.executions || []) {
    const id = e.execution.workflowId;
    let n = 0, token = '';
    do {
      const r = await (await fetch(`${TUI}/api/v1/namespaces/nava/workflows/${encodeURIComponent(id)}/history?maximumPageSize=1000${token ? '&nextPageToken=' + encodeURIComponent(token) : ''}`)).json();
      n += ((r.history && r.history.events) || []).length;
      token = r.nextPageToken || '';
    } while (token);
    lens.push({ workflow_id: id.replace(/^t:[0-9a-f-]{36}:/, 't:<tenant>:'), status: e.status, events: n });
  }
  lens.sort((x, y) => y.events - x.events);
  out.e4 = { limit: HISTORY_LIMIT, histories: lens };
  const top = lens[0] || { events: 0 };
  check('E4 R9 schema.backfill workflows found', lens.length, lens.length >= 5);
  check(`E4 R9 largest schema.backfill history (limit ${HISTORY_LIMIT}; fails over half)`, `${top.events} events (${(100 * top.events / HISTORY_LIMIT).toFixed(1)}%) in ${top.workflow_id}`, top.events > 0 && top.events <= HISTORY_LIMIT / 2);
}

let failed = null;
for (const [name, f] of [['E1', e1], ['E2', e2], ['E3', e3], ['E4', e4]]) {
  try { console.log(`s29 extras: ${name} (${new Date().toISOString()})`); await f(); } catch (e) { failed = failed || e; check(`${name} ran to the end`, String(e.message).slice(0, 200), false); }
}
out.finished_at = new Date().toISOString();
writeFileSync(join(dir, 'extras.json'), JSON.stringify(out, null, 2) + '\n');
writeFileSync(join(dir, 'extras.txt'), checks.join('\n') + '\n');
process.exit(checks.some((c) => c.endsWith('|FAIL')) ? 1 : 0);
