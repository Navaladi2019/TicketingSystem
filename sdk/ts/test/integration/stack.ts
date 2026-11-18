// Helpers for the SDK integration tests (TICK-829, RFC-017 CI-E8): the running
// local stack's URLs and tokens for its seeded users. Nothing here mocks the API:
// every request goes to the real nava-core, Postgres and Keycloak.
//
// The stack case deploy/compose/test/cases/sdk_ts.sh sets the variables below
// after tools/seed.sh; the defaults are the compose ports of a laptop stack.
// The passwords are the realm's dev-only seed values (also in
// docs/acceptance/guide/bruno/environments/local.bru), never real credentials.

import { createClient, type NavaClient } from "../../src/index.ts";

const env = (k: string, d: string) => process.env[k] || d;

export const CORE_URL = env("NAVA_CORE_URL", "http://127.0.0.1:18080");
export const KEYCLOAK_URL = env("NAVA_KEYCLOAK_URL", "http://127.0.0.1:18081");
const REALM = env("NAVA_REALM", "nava");

export interface SeededUser {
  username: string;
  password: string;
}

export const ALICE: SeededUser = { username: "alice@tenant-a.test", password: env("NAVA_ALICE_PASSWORD", "dev-only-alice") };
export const ADMIN_A: SeededUser = { username: "admin@tenant-a.test", password: env("NAVA_ADMIN_A_PASSWORD", "dev-only-admin-a") };
export const BOB: SeededUser = { username: "bob@tenant-b.test", password: env("NAVA_BOB_PASSWORD", "dev-only-bob") };

/** A password-grant access token from realm nava, client nava-web (as the acceptance guide signs in). */
export async function passwordToken(u: SeededUser): Promise<string> {
  const res = await fetch(`${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ grant_type: "password", client_id: "nava-web", scope: "openid", username: u.username, password: u.password }),
  });
  if (res.status !== 200) throw new Error(`token for ${u.username}: Keycloak answered HTTP ${res.status}`);
  const body = (await res.json()) as { access_token?: unknown };
  if (typeof body.access_token !== "string") throw new Error(`token for ${u.username}: no access_token`);
  return body.access_token;
}

/** Counts every request the client sends, by method and path, and the getToken calls. */
export interface Probe {
  requests: Array<{ method: string; path: string; status: number }>;
  tokenCalls: number;
}

export interface ClientFor {
  client: NavaClient;
  probe: Probe;
}

/**
 * An SDK client for a seeded user. One token is fetched and cached per client; `firstTokens`
 * are handed out before it, so a test can make the first request carry a token the API refuses.
 */
export function clientFor(u: SeededUser, opts: { firstTokens?: string[] } = {}): ClientFor {
  const probe: Probe = { requests: [], tokenCalls: 0 };
  const queued = [...(opts.firstTokens ?? [])];
  let cached: Promise<string> | undefined;
  const getToken = async () => {
    probe.tokenCalls++;
    const next = queued.shift();
    if (next !== undefined) return next;
    cached ??= passwordToken(u);
    return cached;
  };
  const counting: typeof fetch = async (input, init) => {
    const req = input instanceof Request ? input : new Request(input, init);
    const res = await fetch(req);
    probe.requests.push({ method: req.method, path: new URL(req.url).pathname, status: res.status });
    return res;
  };
  return { client: createClient({ baseUrl: CORE_URL, getToken, fetch: counting }), probe };
}

/** The record's change history (create and update audit rows), read with a raw request: the SDK has no history call in v1. */
export async function history(u: SeededUser, entity: string, id: string): Promise<Array<{ action: string; source: string }>> {
  const res = await fetch(`${CORE_URL}/api/v1/records/${entity}/${id}/history`, {
    headers: { Authorization: `Bearer ${await passwordToken(u)}` },
  });
  if (res.status !== 200) throw new Error(`history of ${entity}/${id}: HTTP ${res.status}`);
  const body = (await res.json()) as { items: Array<{ action: string; source: string }>; next_cursor?: string };
  if (body.next_cursor) throw new Error(`history of ${entity}/${id}: more than one page`);
  return body.items;
}

/** A short random marker that keeps this run's records apart from every other run's on a shared stack. */
export const RUN = globalThis.crypto.randomUUID().slice(0, 8);

/** The cap on the wait for a new field index to finish its backfill. */
const INDEX_WAIT_MS = 60_000;

/**
 * Publishes a new version of an entity type with `props` flagged `x-nava-filterable` (ADR-006), as
 * `u`, a tenant admin. Raw requests: the SDK has no entity-type calls in v1. Touches only the named
 * properties and publishes nothing when they are all flagged already, so a rerun on the same stack
 * is a no-op. The index is `backfilling` after this; see waitFilterable.
 */
export async function flagFilterable(u: SeededUser, entity: string, props: string[]): Promise<void> {
  const headers = { Authorization: `Bearer ${await passwordToken(u)}`, "Content-Type": "application/json" };
  const base = `${CORE_URL}/api/v1/entity-types/${entity}`;
  const cur = await fetch(base, { headers });
  if (cur.status !== 200) throw new Error(`entity type ${entity}: HTTP ${cur.status}`);
  const type = (await cur.json()) as {
    schema: { properties: Record<string, Record<string, unknown>> };
    fixed_columns: unknown;
    states: unknown;
  };

  const missing = props.filter((p) => type.schema.properties[p]?.["x-nava-filterable"] !== true);
  if (missing.length === 0) return;
  for (const p of missing) {
    if (!type.schema.properties[p]) throw new Error(`entity type ${entity} has no property ${p}`);
    type.schema.properties[p]["x-nava-filterable"] = true;
  }
  const res = await fetch(`${base}/versions`, {
    method: "POST",
    headers: { ...headers, "Idempotency-Key": `sdk-it-${RUN}-flag-${missing.join("-")}` },
    body: JSON.stringify({ definition: { schema: type.schema, fixed_columns: type.fixed_columns, states: type.states } }),
  });
  if (res.status !== 201) throw new Error(`publish ${entity} with ${missing.join(", ")} flagged: HTTP ${res.status} ${await res.text()}`);
}

/**
 * Waits, up to INDEX_WAIT_MS, until a list filtered on each prop is served as `u` (no 422
 * field_not_ready). A filter on an unflagged data field is a 400 field_not_filterable.
 */
export async function waitFilterable(u: SeededUser, entity: string, props: string[]): Promise<void> {
  const headers = { Authorization: `Bearer ${await passwordToken(u)}` };
  const deadline = Date.now() + INDEX_WAIT_MS;
  for (const p of props) {
    for (;;) {
      const res = await fetch(`${CORE_URL}/api/v1/records/${entity}?limit=1&filter[${p}]=${encodeURIComponent(`sdk-it-${RUN}-ready`)}`, { headers });
      if (res.status === 200) break;
      if (res.status !== 422) throw new Error(`filter on ${entity}.${p}: HTTP ${res.status} ${await res.text()}`);
      if (Date.now() > deadline) throw new Error(`the index of ${entity}.${p} is not ready after ${INDEX_WAIT_MS} ms`);
      await new Promise((r) => setTimeout(r, 500));
    }
  }
}
