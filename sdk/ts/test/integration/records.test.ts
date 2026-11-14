// SDK integration tests against the running stack (TICK-829, PLAN-008 P8-24; ADR-007 D10,
// RFC-017 CI-E8 for the SDK). Real nava-core, Postgres and Keycloak; tenants A and B from
// tools/seed.sh. Run by the stack case sdk_ts (make sdk-integration); never by `npm test`.
//
// The cases run in file order, one at a time: the forced 429 is last, so it cannot starve the
// others, and it spends tenant B's bucket, never tenant A's, which later stack cases use.

import assert from "node:assert/strict";
import { writeFileSync } from "node:fs";
import { after, before, describe, test } from "node:test";
import { NavaProblem, RateLimited, VersionConflict, type NavaRecord } from "../../src/index.ts";
import { ADMIN_A, ALICE, BOB, RUN, clientFor, history } from "./stack.ts";

const ENTITY = "incident";
const LIST_PATH = `/api/v1/records/${ENTITY}`;

async function rejection(p: Promise<unknown>): Promise<unknown> {
  try {
    await p;
  } catch (e) {
    return e;
  }
  assert.fail("expected the call to be refused");
}

describe(`SDK against the stack (run ${RUN})`, { concurrency: 1 }, () => {
  let alicesRecord: NavaRecord;

  before(async () => {
    const { client } = clientFor(ALICE);
    alicesRecord = await client.records.create(ENTITY, { short_description: `sdk-it ${RUN} get`, priority: 2 });
  });

  test("list follows next_cursor across two pages", async () => {
    const { client, probe } = clientFor(ALICE);
    const marker = `sdk-it ${RUN} list`;
    const made: string[] = [];
    for (let i = 0; i < 3; i++) made.push((await client.records.create(ENTITY, { short_description: `${marker} ${i}`, description: marker })).id);

    probe.requests.length = 0;
    const seen: string[] = [];
    for await (const r of client.records.list(ENTITY, { filter: { description: marker }, pageSize: 2 })) seen.push(r.id);

    assert.deepEqual([...seen].sort(), [...made].sort(), "every record of the run, each once");
    const pages = probe.requests.filter((r) => r.method === "GET" && r.path === LIST_PATH);
    assert.equal(pages.length, 2, "3 records at pageSize 2 are two list requests");
  });

  test("get returns the record and its version", async () => {
    const { client } = clientFor(ALICE);
    const { record, version } = await client.records.get(ENTITY, alicesRecord.id);
    assert.equal(record.id, alicesRecord.id);
    assert.equal(record.number, alicesRecord.number);
    assert.equal(version, 1);
    assert.equal(record.data.short_description, `sdk-it ${RUN} get`);
  });

  test("create twice with one Idempotency-Key: one record, one audit entry (CI-E8)", async () => {
    const { client } = clientFor(ALICE);
    const key = `sdk-it-${RUN}-replay`;
    const data = { short_description: `sdk-it ${RUN} replay` };
    const first = await client.records.create(ENTITY, data, { idempotencyKey: key });
    const again = await client.records.create(ENTITY, data, { idempotencyKey: key });

    assert.equal(again.id, first.id, "the replay answers the stored record");
    assert.equal(again.number, first.number);
    const h = await history(ALICE, ENTITY, first.id);
    assert.deepEqual(h.map((e) => e.action), ["create"], "exactly one audit entry");

    const list = clientFor(ALICE).client.records.list(ENTITY, { filter: { short_description: data.short_description } });
    let n = 0;
    for await (const _ of list) n++;
    assert.equal(n, 1, "exactly one record carries the replayed body");

    // The stack case counts audit_log rows for this id in Postgres too.
    if (process.env.SDK_IT_OUT) writeFileSync(process.env.SDK_IT_OUT, `${first.id}\n`);
  });

  test("a 401 is retried once with a fresh token and the same Idempotency-Key", async () => {
    // The hand-written retry (client.ts authedFetch): the API refuses the first token, the SDK
    // asks getToken again and resends a clone. Without that retry this create throws a 401.
    const { client, probe } = clientFor(ALICE, { firstTokens: ["not-a-token"] });
    const key = `sdk-it-${RUN}-401`;
    const r = await client.records.create(ENTITY, { short_description: `sdk-it ${RUN} 401` }, { idempotencyKey: key });

    const posts = probe.requests.filter((q) => q.method === "POST" && q.path === LIST_PATH).map((q) => q.status);
    assert.deepEqual(posts, [401, 201], "one refused send, one retried send");
    assert.equal(probe.tokenCalls, 2, "getToken is called again after the 401");
    const h = await history(ALICE, ENTITY, r.id);
    assert.deepEqual(h.map((e) => e.action), ["create"], "the retry wrote one record, once");
  });

  test("an update with a stale version is a VersionConflict (412)", async () => {
    const { client } = clientFor(ADMIN_A);
    const record = await clientFor(ALICE).client.records.create(ENTITY, { short_description: `sdk-it ${RUN} stale` });
    const v1 = (await client.records.get(ENTITY, record.id)).version;
    const updated = await client.records.update(ENTITY, record.id, { priority: 1 }, { version: v1 });
    assert.equal(updated.version, v1 + 1);

    const e = await rejection(client.records.update(ENTITY, record.id, { priority: 4 }, { version: v1 }));
    assert.ok(e instanceof VersionConflict, `want VersionConflict, got ${String(e)}`);
    assert.equal(e.status, 412);
    assert.equal(e.code, "version_conflict");
    assert.equal((await client.records.get(ENTITY, record.id)).record.data.priority, 1, "the stale write changed nothing");
  });

  test("an invalid body is a NavaProblem record_invalid (422) with field pointers", async () => {
    const { client } = clientFor(ALICE);
    const marker = `sdk-it ${RUN} invalid`;
    const e = await rejection(client.records.create(ENTITY, { description: marker, priority: 9 }));
    assert.ok(e instanceof NavaProblem, `want NavaProblem, got ${String(e)}`);
    assert.equal(e.status, 422);
    assert.equal(e.code, "record_invalid");
    assert.ok(e.fields && e.fields.length > 0, "errors[] is kept as fields");
    for (const f of e.fields) assert.match(f.pointer, /^\/data(\/|$)/);

    let n = 0;
    for await (const _ of client.records.list(ENTITY, { filter: { description: marker } })) n++;
    assert.equal(n, 0, "the refused create wrote nothing");
  });

  test("another tenant's record id is a 404 not_found", async () => {
    const { client } = clientFor(BOB);
    const e = await rejection(client.records.get(ENTITY, alicesRecord.id));
    assert.ok(e instanceof NavaProblem, `want NavaProblem, got ${String(e)}`);
    assert.equal(e.status, 404);
    assert.equal(e.code, "not_found");

    let n = 0;
    for await (const _ of client.records.list(ENTITY, { filter: { short_description: `sdk-it ${RUN} get` } })) n++;
    assert.equal(n, 0, "tenant A's record is not in tenant B's list");
  });

  describe("forced 429", () => {
    after(async () => {
      // Let tenant B's bucket refill (burst 100 at 50/s by default) before the next stack case.
      await new Promise((r) => setTimeout(r, 3000));
    });

    test("a burst past the tenant's bucket is RateLimited with retryAfter", async () => {
      // More requests at once than tenant B's bucket holds (NAVA_RATE_LIMIT_BURST, 100 by default).
      const burst = Number(process.env.SDK_IT_BURST || 300);
      const { client } = clientFor(BOB);
      await client.records.list(ENTITY, { pageSize: 1 })[Symbol.asyncIterator]().next(); // the token, before the burst
      const results = await Promise.allSettled(
        Array.from({ length: burst }, () => client.records.list(ENTITY, { pageSize: 1 })[Symbol.asyncIterator]().next()),
      );
      const limited = results.flatMap((r) => (r.status === "rejected" && r.reason instanceof RateLimited ? [r.reason] : []));
      const other = results.flatMap((r) => (r.status === "rejected" && !(r.reason instanceof RateLimited) ? [r.reason] : []));

      assert.deepEqual(other.map(String), [], "no failure other than 429");
      assert.ok(limited.length > 0, `want at least one 429 out of ${burst}`);
      for (const e of limited) {
        assert.equal(e.status, 429);
        assert.equal(e.code, "rate_limited");
        assert.equal(typeof e.retryAfter, "number", "Retry-After is parsed into retryAfter");
        assert.ok((e.retryAfter as number) > 0, `retryAfter ${e.retryAfter} is a positive wait`);
      }
    });
  });
});
