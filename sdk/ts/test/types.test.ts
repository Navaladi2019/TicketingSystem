import test from "node:test";
import assert from "node:assert/strict";
import { createRawClient } from "../src/index.ts";
import type { components } from "../src/index.ts";

type RecordList = components["schemas"]["RecordList"];

const body: RecordList = { items: [], next_cursor: null, total: 0, total_state: "exact" };
const fetchStub = async () =>
  new Response(JSON.stringify(body), { headers: { "content-type": "application/json" } });

test("records list for incident returns the generated RecordList type", async () => {
  const client = createRawClient("http://sdk.invalid", fetchStub);
  const { data } = await client.GET("/api/v1/records/{entity}", {
    params: { path: { entity: "incident" } },
  });
  const typed: RecordList | undefined = data;
  assert.equal(typed?.total, 0);
  assert.equal(typed?.total_state, "exact");
});

// Compile-time only: total is nullable and total_state is the three-value enum.
export const unknownTotal: RecordList = { items: [], next_cursor: null, total: null, total_state: "unknown" };
export const totalType: RecordList["total"] extends number | null ? true : never = true;
export const stateType: RecordList["total_state"] extends "exact" | "capped" | "unknown" ? true : never = true;

// Compile-time only: tsc fails if the misspelt path type-checks (the directive would be unused).
export function misspeltPathFailsTsc() {
  const client = createRawClient("http://sdk.invalid", fetchStub);
  // @ts-expect-error unknown path must not type-check
  return client.GET("/api/v1/recordz/{entity}", { params: { path: { entity: "incident" } } });
}
