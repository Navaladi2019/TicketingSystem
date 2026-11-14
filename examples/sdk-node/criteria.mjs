// S25 acceptance walk (TICK-830): criteria 1 to 4 of story S25 and the cross-tenant check,
// through the installed @nava/sdk tarball against the running local stack, as two tenants.
// Prints one PASS/FAIL line per check and exits 1 on any failure. Criterion 5 (the CI `sdk`
// drift gate) is a GitHub run, not something this script can show; see docs/acceptance/S25/.
//
//   node criteria.mjs            # after `npm install` in this directory
//
// Never prints a token. Writes the replayed create's id to $S25_REPLAY_ID_OUT when set, so the
// kit can count its audit_log rows in Postgres.

import { readFileSync, writeFileSync } from "node:fs";
import { createClient, NavaProblem, RateLimited, SDK_VERSION, VersionConflict } from "@nava/sdk";
import { devToken } from "./dev-token.mjs";

const CORE = process.env.NAVA_CORE_URL ?? "http://127.0.0.1:18080";
const RUN = crypto.randomUUID().slice(0, 8);
const ALICE = "alice@tenant-a.test"; // tenant A, agent
const ADMIN_A = "admin@tenant-a.test"; // tenant A, admin
const BOB = "bob@tenant-b.test"; // tenant B, agent

let failed = 0;
function check(name, ok, detail = "") {
  if (!ok) failed++;
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? `  (${detail})` : ""}`);
}

/** A client for a seeded user that records every request's method, path, status and Nava-SDK header. */
function clientFor(user) {
  const seen = [];
  const recording = async (req) => {
    const res = await fetch(req);
    seen.push({ method: req.method, path: new URL(req.url).pathname, status: res.status, sdk: req.headers.get("Nava-SDK") });
    return res;
  };
  return { client: createClient({ baseUrl: CORE, getToken: () => devToken(user), fetch: recording }), seen };
}

async function refusal(p) {
  try {
    await p;
  } catch (e) {
    return e;
  }
  return undefined;
}

async function count(client, filter) {
  let n = 0;
  for await (const _ of client.records.list("incident", { filter })) n++;
  return n;
}

console.log(`S25 acceptance walk, run ${RUN}, against ${CORE}`);

// --- Criterion 1: list iterates every page through cursors; get returns record and version.
{
  const { client, seen } = clientFor(ALICE);
  const marker = `s25 ${RUN} list`;
  const made = [];
  for (let i = 0; i < 5; i++) made.push((await client.records.create("incident", { short_description: `${marker} ${i}`, description: marker })).id);
  seen.length = 0;
  const listed = [];
  for await (const r of client.records.list("incident", { filter: { description: marker }, pageSize: 2 })) listed.push(r.id);
  const pages = seen.filter((s) => s.method === "GET" && s.path === "/api/v1/records/incident");
  check("C1 list: 5 records at pageSize 2 arrive in full, each once", JSON.stringify([...listed].sort()) === JSON.stringify([...made].sort()), `${listed.length} listed`);
  check("C1 list: it took 3 page requests (cursor followed twice)", pages.length === 3, `${pages.length} GETs`);

  const { record, version } = await client.records.get("incident", made[0]);
  check("C1 get: returns the record", record.id === made[0] && record.data.short_description === `${marker} 0`, `${record.number}`);
  check("C1 get: returns its version", version === 1 && record.version === 1, `version ${version}`);
}

// --- Criterion 2: create twice with one idempotencyKey makes one record.
let replayId;
{
  const { client } = clientFor(ALICE);
  const key = `s25-${RUN}-replay`;
  const data = { short_description: `s25 ${RUN} replay` };
  const first = await client.records.create("incident", data, { idempotencyKey: key });
  const again = await client.records.create("incident", data, { idempotencyKey: key });
  replayId = first.id;
  check("C2 replay answers the same record", again.id === first.id && again.number === first.number, `${first.number} both times`);
  const n = await count(client, { short_description: data.short_description });
  check("C2 exactly one record exists for the replayed body", n === 1, `${n} found`);
  if (process.env.S25_REPLAY_ID_OUT) writeFileSync(process.env.S25_REPLAY_ID_OUT, `${first.id}\n`);

  // Isolation: the same key in tenant B is a different record; keys are per tenant.
  const b = await clientFor(BOB).client.records.create("incident", data, { idempotencyKey: key });
  check("ISO same idempotencyKey in tenant B makes tenant B's own record", b.id !== first.id, `B got ${b.number}`);
}

// --- Criterion 3: stale version -> VersionConflict; invalid body -> NavaProblem record_invalid; 429 -> retryAfter.
let alicesId;
{
  const { client } = clientFor(ADMIN_A);
  const rec = await clientFor(ALICE).client.records.create("incident", { short_description: `s25 ${RUN} stale` });
  alicesId = rec.id;
  const v1 = (await client.records.get("incident", rec.id)).version;
  const v2 = (await client.records.update("incident", rec.id, { priority: 1 }, { version: v1 })).version;
  const e = await refusal(client.records.update("incident", rec.id, { priority: 4 }, { version: v1 }));
  check("C3 update with the current version succeeds", v2 === v1 + 1, `v${v1} -> v${v2}`);
  check("C3 update with a stale version throws VersionConflict", e instanceof VersionConflict && e instanceof NavaProblem && e.status === 412 && e.code === "version_conflict", e ? `${e.name} ${e.status} ${e.code}` : "no error");
  const after = (await client.records.get("incident", rec.id)).record.data.priority;
  check("C3 the stale update changed nothing", after === 1, `priority ${after}`);
}
{
  const { client } = clientFor(ALICE);
  const marker = `s25 ${RUN} invalid`;
  const e = await refusal(client.records.create("incident", { description: marker, priority: 9 }));
  const fields = e?.fields ?? [];
  check("C3 invalid body throws NavaProblem with code record_invalid", e instanceof NavaProblem && e.status === 422 && e.code === "record_invalid", e ? `${e.name} ${e.status} ${e.code}` : "no error");
  check("C3 the field errors are exposed", fields.length > 0 && fields.every((f) => /^\/data(\/|$)/.test(f.pointer)), fields.map((f) => `${f.pointer}${f.rule ? ` ${f.rule}` : ""}`).join(", "));
  const n = await count(client, { description: marker });
  check("C3 the refused create wrote nothing", n === 0, `${n} found`);
}

// --- Isolation: tenant B cannot read, list or change what tenant A created.
{
  const { client } = clientFor(BOB);
  const g = await refusal(client.records.get("incident", alicesId));
  check("ISO tenant B get of tenant A's record is 404 not_found", g instanceof NavaProblem && g.status === 404 && g.code === "not_found", g ? `${g.status} ${g.code}` : "no error: B read A's record");
  const u = await refusal(client.records.update("incident", alicesId, { priority: 5 }, { version: 2 }));
  check("ISO tenant B update of tenant A's record is 404 not_found", u instanceof NavaProblem && u.status === 404, u ? `${u.status} ${u.code}` : "no error: B changed A's record");
  const n = await count(client, { short_description: `s25 ${RUN} stale` });
  check("ISO tenant A's record is not in tenant B's list", n === 0, `${n} found`);
  const still = (await clientFor(ADMIN_A).client.records.get("incident", alicesId)).record.data.priority;
  check("ISO tenant A's record is unchanged", still === 1, `priority ${still}`);
}

// --- Criterion 4: every request carries Nava-SDK: ts/<version>; the package version is 1.x.
{
  const pkg = JSON.parse(readFileSync(new URL("../package.json", import.meta.resolve("@nava/sdk")), "utf8"));
  check("C4 installed package version is 1.x (API major 1)", /^1\./.test(pkg.version) && SDK_VERSION === pkg.version, `${pkg.name}@${pkg.version}, SDK_VERSION ${SDK_VERSION}`);
  const { client, seen } = clientFor(ALICE);
  const r = await client.records.create("incident", { short_description: `s25 ${RUN} header` });
  await client.records.get("incident", r.id);
  for await (const _ of client.records.list("incident", { pageSize: 1, filter: { short_description: `s25 ${RUN} header` } }));
  await refusal(client.records.update("incident", r.id, { priority: 2 }, { version: 99 }));
  const want = `ts/${pkg.version}`;
  const methods = [...new Set(seen.map((s) => s.method))].sort().join(",");
  check("C4 every request (create, get, list, update) carries Nava-SDK", seen.length >= 4 && seen.every((s) => s.sdk === want), `${seen.length} requests [${methods}], header ${[...new Set(seen.map((s) => s.sdk))].join(" | ")}`);
}

// --- Criterion 3 (last, it spends tenant B's rate-limit bucket): a 429 exposes retryAfter.
{
  const { client } = clientFor(BOB);
  const first = () => client.records.list("incident", { pageSize: 1 })[Symbol.asyncIterator]().next();
  await first(); // fetch the token before the burst
  // 200 at once, not the CI test's 300: on Docker Desktop for Windows the published-port proxy refuses
  // part of 300 simultaneous connects (ECONNREFUSED, never reaching nava-core). 200 still empties the
  // bucket (burst 100, 50/s by default).
  const burst = Number(process.env.S25_BURST ?? 200);
  const results = await Promise.allSettled(Array.from({ length: burst }, first));
  const limited = results.filter((r) => r.status === "rejected" && r.reason instanceof RateLimited).map((r) => r.reason);
  const other = results.filter((r) => r.status === "rejected" && !(r.reason instanceof RateLimited));
  check("C3 a burst past the tenant's bucket throws RateLimited", limited.length > 0 && other.length === 0, `${limited.length} of ${burst} limited, ${other.length} other errors`);
  const ra = [...new Set(limited.map((e) => e.retryAfter))];
  check("C3 RateLimited exposes a positive retryAfter (seconds) and code rate_limited", limited.length > 0 && limited.every((e) => e instanceof NavaProblem && e.status === 429 && e.code === "rate_limited" && typeof e.retryAfter === "number" && e.retryAfter > 0), `retryAfter ${ra.join(",")}`);
}

console.log(failed === 0 ? "ALL PASS" : `${failed} FAILED`);
process.exit(failed === 0 ? 0 : 1);
