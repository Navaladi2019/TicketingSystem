import test from "node:test";
import assert from "node:assert/strict";
import { describeTotal } from "../src/index.ts";
import type { components, TotalView } from "../src/index.ts";

type RecordList = components["schemas"]["RecordList"];

const page = (total: number | null, total_state: RecordList["total_state"]) => ({ total, total_state });

test("exact totals render with thousands separators", () => {
  assert.deepEqual(describeTotal(page(0, "exact")), { text: "0", state: "exact", value: 0 });
  assert.deepEqual(describeTotal(page(1234, "exact")), { text: "1,234", state: "exact", value: 1234 });
});

test("capped renders 1,000+", () => {
  assert.deepEqual(describeTotal(page(1000, "capped")), { text: "1,000+", state: "capped", value: 1000 });
});

test("unknown shows N+ from listed, else many, never 0", () => {
  assert.equal(describeTotal(page(null, "unknown"), 50).text, "50+");
  assert.equal(describeTotal(page(null, "unknown")).text, "many");
  assert.equal(describeTotal(page(null, "unknown"), 0).text, "many");
  assert.equal(describeTotal(page(null, "unknown"), 50).value, null);
  assert.equal(describeTotal(page(null, "unknown"), 50).state, "unknown");
});

test("a best-known total survives a later unknown page", () => {
  const first = describeTotal(page(1000, "capped"));
  assert.deepEqual(describeTotal(page(null, "unknown"), 100, first), first);
  const exact = describeTotal(page(640, "exact"));
  assert.deepEqual(describeTotal(page(null, "unknown"), 100, exact), exact);
});

test("a later exact page replaces an earlier unknown", () => {
  const first = describeTotal(page(null, "unknown"), 50);
  assert.equal(describeTotal(page(640, "exact"), 100, first).text, "640");
});

test("an earlier unknown view is kept when listed is unknown", () => {
  const first = describeTotal(page(null, "unknown"));
  assert.deepEqual(describeTotal(page(null, "unknown"), undefined, first), first);
});

test("listed takes priority over an earlier unknown view", () => {
  const first = describeTotal(page(null, "unknown"), 50);
  assert.equal(first.text, "50+");
  assert.equal(describeTotal(page(null, "unknown"), 100, first).text, "100+");
});

test("exact/capped with a null total is a contract error", () => {
  assert.throws(() => describeTotal(page(null, "exact")), TypeError);
  assert.throws(() => describeTotal(page(null, "capped")), TypeError);
});

// Compile-time only: the generated RecordList is accepted directly, not a copy of it.
export function acceptsGeneratedRecordList(list: RecordList): TotalView {
  return describeTotal(list);
}
