import test from "node:test";
import assert from "node:assert/strict";
import { createRawClient } from "../src/index.ts";
import type { components } from "../src/index.ts";

type RecordList = components["schemas"]["RecordList"];

const body: RecordList = { items: [], next_cursor: null, total: 0 };
const fetchStub = async () =>
  new Response(JSON.stringify(body), { headers: { "content-type": "application/json" } });

test("records list for incident returns the generated RecordList type", async () => {
  const client = createRawClient("http://sdk.invalid", fetchStub);
  const { data } = await client.GET("/api/v1/records/{entity}", {
    params: { path: { entity: "incident" } },
  });
  const typed: RecordList | undefined = data;
  assert.equal(typed?.total, 0);
});

// Compile-time only: tsc fails if the misspelt path type-checks (the directive would be unused).
export function misspeltPathFailsTsc() {
  const client = createRawClient("http://sdk.invalid", fetchStub);
  // @ts-expect-error unknown path must not type-check
  return client.GET("/api/v1/recordz/{entity}", { params: { path: { entity: "incident" } } });
}
