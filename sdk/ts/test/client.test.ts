import test from "node:test";
import assert from "node:assert/strict";
import { createClient, NavaProblem, RateLimited, SDK_VERSION, VersionConflict } from "../src/index.ts";

const json = (status: number, body: unknown, headers: Record<string, string> = {}) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": status >= 400 ? "application/problem+json" : "application/json", ...headers },
  });

const rec = (id: string, version = 1) => ({
  id, number: "INC1", version, schema_version: 1, state: "new", data: {},
  created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
});
const problem = (status: number, code: string, extra: object = {}) => ({
  type: `urn:nava:problem:${code}`, title: code, status, code, detail: "d", instance: "urn:nava:request:1", ...extra,
});

function harness(responses: Array<() => Response>) {
  const calls: Request[] = [];
  let tokens = 0;
  const fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const req = input instanceof Request ? input : new Request(input, init);
    calls.push(req);
    const next = responses.shift();
    assert.ok(next, "unexpected extra request");
    return next();
  };
  const client = createClient({
    baseUrl: "http://sdk.invalid",
    getToken: async () => `tok${++tokens}`,
    fetch: fetch as typeof globalThis.fetch,
  });
  return { client, calls, tokenCalls: () => tokens };
}

test("list follows cursors until next_cursor is null and sends filters", async () => {
  const h = harness([
    () => json(200, { items: [rec("a"), rec("b")], next_cursor: "c1", total: 3 }),
    () => json(200, { items: [rec("c")], next_cursor: null, total: 3 }),
  ]);
  const ids: string[] = [];
  for await (const r of h.client.records.list("incident", { filter: { state: "new", p: ["1", "2"] }, pageSize: 2 })) ids.push(r.id);
  assert.deepEqual(ids, ["a", "b", "c"]);
  assert.equal(h.calls.length, 2);
  const u1 = new URL(h.calls[0]!.url);
  assert.equal(u1.searchParams.get("filter[state]"), "new");
  assert.equal(u1.searchParams.get("filter[p][in]"), "1,2");
  assert.equal(u1.searchParams.get("limit"), "2");
  assert.equal(new URL(h.calls[1]!.url).searchParams.get("cursor"), "c1");
});

test("list of an empty page ends", async () => {
  const h = harness([() => json(200, { items: [], next_cursor: null, total: 0 })]);
  const out = [];
  for await (const r of h.client.records.list("incident")) out.push(r);
  assert.equal(out.length, 0);
});

test("every request carries the Nava-SDK header and a fresh bearer token", async () => {
  const h = harness([() => json(200, rec("a", 4), { ETag: 'W/"4"' })]);
  const { record, version } = await h.client.records.get("incident", "a");
  assert.equal(record.id, "a");
  assert.equal(version, 4);
  assert.equal(h.calls[0]!.headers.get("Nava-SDK"), `ts/${SDK_VERSION}`);
  assert.equal(h.calls[0]!.headers.get("Authorization"), "Bearer tok1");
});

test("401 calls getToken once more and retries once with the same Idempotency-Key", async () => {
  const h = harness([() => json(401, problem(401, "unauthenticated")), () => json(201, rec("n"))]);
  const r = await h.client.records.create("incident", { a: 1 });
  assert.equal(r.id, "n");
  assert.equal(h.tokenCalls(), 2);
  assert.equal(h.calls.length, 2);
  const k = h.calls[0]!.headers.get("Idempotency-Key");
  assert.match(k!, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  assert.equal(h.calls[1]!.headers.get("Idempotency-Key"), k);
  assert.equal(h.calls[1]!.headers.get("Authorization"), "Bearer tok2");
  assert.deepEqual(await h.calls[1]!.json(), { data: { a: 1 } });
});

test("a second 401 throws and does not retry again", async () => {
  const h = harness([() => json(401, problem(401, "unauthenticated")), () => json(401, problem(401, "unauthenticated"))]);
  await assert.rejects(h.client.records.get("incident", "a"), (e: unknown) => e instanceof NavaProblem && e.status === 401);
  assert.equal(h.tokenCalls(), 2);
  assert.equal(h.calls.length, 2);
});

test("create uses the given idempotency key and generates distinct keys otherwise", async () => {
  const h = harness([() => json(201, rec("a")), () => json(201, rec("b")), () => json(201, rec("c"))]);
  await h.client.records.create("incident", {}, { idempotencyKey: "my-key" });
  await h.client.records.create("incident", {});
  await h.client.records.create("incident", {});
  assert.equal(h.calls[0]!.headers.get("Idempotency-Key"), "my-key");
  assert.notEqual(h.calls[1]!.headers.get("Idempotency-Key"), h.calls[2]!.headers.get("Idempotency-Key"));
});

test("update sends If-Match from the version and a merge-patch body", async () => {
  const h = harness([() => json(200, rec("a", 6))]);
  const r = await h.client.records.update("incident", "a", { x: null }, { version: 5 });
  assert.equal(r.version, 6);
  const c = h.calls[0]!;
  assert.equal(c.method, "PATCH");
  assert.equal(c.headers.get("If-Match"), 'W/"5"');
  assert.ok(c.headers.get("Idempotency-Key"));
  assert.match(c.headers.get("Content-Type")!, /application\/merge-patch\+json/);
  assert.deepEqual(await c.json(), { data: { x: null } });
});

test("412 becomes VersionConflict", async () => {
  const h = harness([() => json(412, problem(412, "version_conflict"), { ETag: 'W/"9"' })]);
  await assert.rejects(h.client.records.update("incident", "a", {}, { version: 1 }), (e: unknown) => {
    assert.ok(e instanceof VersionConflict);
    assert.ok(e instanceof NavaProblem);
    assert.equal(e.code, "version_conflict");
    return true;
  });
});

test("422 record_invalid becomes NavaProblem with fields", async () => {
  const errors = [{ pointer: "/data/title", rule: "required" }];
  const h = harness([() => json(422, problem(422, "record_invalid", { errors }))]);
  await assert.rejects(h.client.records.create("incident", {}), (e: unknown) => {
    assert.ok(e instanceof NavaProblem && !(e instanceof VersionConflict));
    assert.equal(e.status, 422);
    assert.equal(e.code, "record_invalid");
    assert.deepEqual(e.fields, errors);
    return true;
  });
});

test("429 becomes RateLimited with retryAfter seconds", async () => {
  const h = harness([() => json(429, problem(429, "rate_limited"), { "Retry-After": "7" })]);
  await assert.rejects(h.client.records.get("incident", "a"), (e: unknown) => {
    assert.ok(e instanceof RateLimited);
    assert.equal(e.retryAfter, 7);
    return true;
  });
});

test("429 without Retry-After leaves retryAfter undefined", async () => {
  const h = harness([() => json(429, problem(429, "rate_limited"))]);
  await assert.rejects(h.client.records.get("incident", "a"), (e: unknown) => e instanceof RateLimited && e.retryAfter === undefined);
});

test("a non-JSON error body still throws NavaProblem", async () => {
  const h = harness([() => new Response("<html>bad gateway</html>", { status: 502 })]);
  await assert.rejects(h.client.records.get("incident", "a"), (e: unknown) => e instanceof NavaProblem && e.status === 502 && e.code === "unknown");
});
