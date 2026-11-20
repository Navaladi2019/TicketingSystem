---
type: adr-amendment
id: ADR-006-A2
title: "Amendment A2 to ADR-006: D7 allows one bounded pre-count per filtered field, and a forced per-record probe in the capped count, for lists with a narrowing row ACL"
status: PROPOSED
date: 2026-11-21
issue: TICK-1022
rfc: none (amendment from a spike; the decision request is the Postgres & Data Modeling Specialist's TICK-1022 comment 01a1fae7-c4a2-7997-b383-d420fedc2224, run 5)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D7 (one compiled query) and the D7 static test; implements the count half of [[ADR-006-A1-d13-fanout-bands-350ms]] A1-6 (2)"
source: "TICK-1022 runs 1-5, PR #796: `docs/data/tick-1022-post-backfill-plan.md`, `spike-bounded-d13-server-time.md`; spike branch `TICK-1022/spike-probe` at `5b37843f` (not for merge). Archived in `docs/adr/ADR-006/spikes/a2-bounded-probe/`"
deciders: Solution Architect (author; verdict after critique); Technical Architect (critique pending); Security & Multi-Tenancy Reviewer (I5 timing question, pending); CTO (tie-break only if contested)
---

# ADR-006 amendment A2: a bounded pre-count picks the count's shape

> **Status: PROPOSED, rev 0** (2026-11-21). Waiting on the Technical Architect's critique and the Security & Multi-Tenancy Reviewer's answer on A2-6. The compiler PR may be opened and reviewed now. It merges only after A2 is ACCEPTED.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1-I9, as ADR-006 and A1 are.

## In plain words

A list request returns one page of records plus a `total`, capped at 1,001. When the user's row ACL hides most records and the filter matches a large share of the type (≥ 40%), PostgreSQL computes that total in one of two ways. Fast: for each record the user can see, look up its value. Slow: read every side-table row of the value (192k here), build a hash table, and spill it to disk at `work_mem` 4 MB. Which one it picks depends on the side table's **visibility map**: how many of its pages vacuum has marked all-visible. That changes with backfills and ordinary edits, and `rfv_*` tables are shared by every tenant. So the same request took 34 ms on one fresh stack and 217 ms on the next. Analyzing the table does not fix it (run 1), and a lower `random_page_cost` only moves the threshold (run 3).

The fix stops asking the planner. Before the list, the server counts the filter's side rows up to 20,000 (cheap: it stops at 20,000). If it hits 20,000, the value is broad, and the count statement is written so PostgreSQL can only do the per-record lookup. The page statement does not change. ADR-006 D7 says no other query may read a side table, so this needs an amendment.

## What was measured

Run 5, spike `5b37843f`, D13 fixture (800k seed, 400k type, ACL agent sees 24,118 records = 6%), PostgreSQL 16.10, `work_mem` 4 MB. The host was busy (no-filter control p95 136-193 ms against 73 ms idle), so **server time is the evidence** and p95 is indicative only.

| ACL cells | Count before | Pre-count + count after | Page | 8-client p95 (indicative) |
|---|---|---|---|---|
| 40%, 5 operators | 31.9-114.4 ms, 3 of 5 spill | 10.9-17.8 + 17.5-26.3 ms | unchanged | 145-233 ms (band (ii) limit 350) |
| 2% | 23.6-33.8 ms | 4.6-7.8 + 19.2-41.4 ms | unchanged plan | 223-842 ms (host; see A2-8) |
| 0.25% | 16.3-23.8 ms | 1.0-1.8 + 14.4-34.4 ms | unchanged plan | 138-341 ms |
| 0.01% | 4.0-16.8 ms | 0.3-1.6 + 5.2-17.8 ms | unchanged plan | 75-246 ms |

Kit-shaped repro (48% value, the TICK-1022 cell): count 17.0-26.8 ms in **every** visibility-map state, against 102-168 ms before; the pre-count adds 14-22 ms. The 1% value keeps its plan (15-25 ms). Run 4 (the probe without the bound) showed why the bound is needed: a 2% range probe took 7.7-8.3 s.

## Decisions

| # | Decision |
|---|---|
| **A2-1** | **D7 reads, from this amendment on:** "**Compiled list queries.** In a list request, a side table is read in exactly two places. (1) A correlated `EXISTS (… v.tenant_id = r.tenant_id AND v.record_id = r.id …)` inside the list statements that carry the compiled row ACL: the page, and `total`, the existing capped count (1,001). The count may render a broad filter as a forced per-record probe (A2-3); the page never does. Both select the same rows. (2) At most one **bounded pre-count** per filtered side-table field (A2-2). Its result only picks the count's shape. There is no other count, facet or progress query on a side table. A static test enforces both places." |
| **A2-2** | **The pre-count.** `SELECT count(*) FROM (SELECT 1 FROM rfv_<type> v WHERE v.tenant_id = $1 AND v.field_id = $2 AND <the field's comparisons> LIMIT broadCap) c`, with `broadCap = 20000`, a named constant. It runs (a) only when the list has a narrowing row ACL (ACL branches or ACL arguments); never for an actor whose ACL does not narrow; (b) **after** the mask check (D6, 400 `masked_field`), the readiness check (422 `field_not_ready`) and value validation, so a rejected filter never reaches it; (c) in the request's transaction, as `nava_app`, under RLS, with the tenant from the verified principal (I1, I2); (d) once per field, with all of that field's comparisons, exactly as the page's `EXISTS` renders them. Its number is never returned, logged, traced as an attribute or used as a metric label. |
| **A2-3** | **The forced probe, in the count only.** When the pre-count reaches `broadCap`, that field's `EXISTS` in the count statement carries `OFFSET 0` and an opaque value term (`v.value \|\| ''`, `v.value + 0`, `v.value + interval '0'`). PostgreSQL then probes `rfv_*_pkey` `(tenant_id, record_id, field_id)` per visible record and cannot build the hash join. No `enable_*` setting, no hint, no session GUC (A1-6). It is carried as `ListQuery.CountWhere`, which uses the same placeholders in the same order as `Where`. The page keeps today's SQL: a first cut that also changed the page read 4.25 s. |
| **A2-4** | **What the gate is: the plan, not the trick.** The opaque term is how the spike got there, not the contract. The contract is a plan-shape test (extends CI-1): for each value type and operator, a broad filter's count `EXPLAIN` shows the side table reached through `rfv_*_pkey` with no Hash or Merge join on `rfv_*`, and a selective filter's count is unchanged. The `bool` term `v.value OR false` is constant-folded away by the planner, so `bool` relies on `OFFSET 0` alone; the test, not the term, proves it. If a PostgreSQL upgrade breaks the shape, this test fails before production does. |
| **A2-5** | **Same rows, proven.** A unit test per value type renders `Where` and `CountWhere` for every operator and checks that the placeholders match. An integration test runs both counts on a fixture with NULL-free edge values (text collation, `±Infinity` for `num`, `infinity` for `ts`, both bools, `in` with repeated values) and asserts the same `total`. The opaque functions are not leakproof; that is safe, because PostgreSQL evaluates them only after the RLS quals, and none of them can raise an error on a stored value (`+ 0`, `\|\| ''`, `+ interval '0'`). CI-2's catalog is about comparison operators and is unchanged. |
| **A2-6** | **Within-tenant timing (I5).** The pre-count does not apply the row ACL, so whether a value has ≥ 20,000 side rows in the tenant, including records the user cannot read, now shows as roughly 15-40 ms of latency. The response body, `total` and status are unchanged (the existing API test "a value only in a row-ACL-denied record gives the same hits and `total` as a non-existent value" stays). Today's D7 shape already leaks more: the hash join's cost scales with **every** side row of the value, ACL-denied ones included, and the plan itself depends on statistics pooled across tenants. A2 replaces a continuous, cross-tenant-influenced signal with one coarse bit that depends on the user's own tenant only. **My position: acceptable residual R13, no new tenant boundary is crossed (I1 holds).** The Security & Multi-Tenancy Reviewer rules on it; a blocking finding there overrides this amendment. |
| **A2-7** | **The D7 static test** (`internal/fieldindex/d7_test.go`) allows exactly one more site: the pre-count function in `internal/server/records_list_index.go`. It asserts that the pre-count's SQL contains `LIMIT` with `broadCap`, `v.tenant_id = $1` and `v.field_id = $2`. Any other reader still fails. |
| **A2-8** | **Merge gate for the compiler PR** (beyond review and green CI). (1) A2 is ACCEPTED. (2) Server time, single client, on the D13 ACL cells: no cell's pre-count + count exceeds its run-5 "before" count by more than 10 ms, and band (ii) cells are ≤ 30 ms. (3) The adversarial fixtures A1-6 (3) asked for: a broad value (≥ 20k side rows) **anti-correlated** with the agent's visible set, so the probe walks the whole visible set and the count returns 0, at 6% visibility and at about 50% visibility. Each is recorded with its plan. If the 50% case's count exceeds the "before" hash plan's time, A2 is revised before merge (a second guard on visible-set size), not shipped. (4) TICK-1022's own acceptance: the S21 kit's ≥ 40% ACL cell inside 350 ms p95 on **two** fresh stacks in a row. |
| **A2-9** | **Budgets do not move.** D13 as amended by A1 stays: band (ii) is still accepted at 350 ms. A2 removes the visibility-map dependency; it does not claim A1-6's 150 ms exit criterion, which needs an idle reference host and the page half of option (a). The 2% band's page cost stays TICK-992's item. A1's R-a trigger and alert stay. |
| **A2-10** | **Calibration limit.** `broadCap` = 20,000 is 5% of the measured 400k type. On a much larger type the same cap marks smaller shares as broad (0.5% of 4M). That is inside R-a's existing trigger ("a tenant passes 400k records → build the adaptive shape"), so the cap is re-measured when R-a fires; it is not a tenant setting. |

**Not changed:** RLS, the row-ACL compilation, the masks (D6, R9/CI-2b), the 1,001 count cap, the page statement, D13 numbers, R-a, R-b, R8, R9, R11, R12. I2-I4 and I6-I9 are not touched: no grant, no role, no write path, no workflow.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. The pre-count runs as `nava_app` under the RLS tenant policy, and also filters `tenant_id = $1` from the principal. |
| I2 tenant source | Holds. `$1` is `p.TenantID` from the verified principal, never a request value. |
| I3 roles | Unchanged. No grant, no `SECURITY DEFINER` (run 1's ANALYZE route was rejected because it needed one). |
| I5 one authorization layer | Rows and `total` are unchanged. One coarse within-tenant timing bit is added (A2-6, R13), smaller than the signal it replaces. Pending the Security reviewer. |
| I8 configuration over code | The cap is a platform constant, not a tenant knob. No tenant code reaches the database. |
| I4, I6, I7, I9 | Not touched (read path only). |

## New residual

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R13 | A list's latency shows whether a filtered value has ≥ 20,000 side rows in the tenant, counting records the user cannot read (A2-6). Body, `total` and status are unchanged | Security & Multi-Tenancy Reviewer | A tenant requires within-tenant timing isolation between row-ACL groups, or the platform adds a per-value existence check anywhere else |

## Rejected options

| Option | Why |
|---|---|
| ANALYZE the side table when a field flips to `ready` (the issue's suggestion) | Run 1: 255 ms with it, 202 ms without. The plan follows the visibility map, not the statistics. It also needs a `SECURITY DEFINER` wrapper (breaks I3 / CI-1 as written). |
| VACUUM at the flip | The worker cannot: VACUUM does not run in a transaction or function and needs ownership. It would not hold either: rewriting 5% of side rows by edits drops `rfv_num` to 1% all-visible (run 2). |
| `random_page_cost = 1.1` for `nava_app` | Run 3: the `in`, `lt` and ts-range 40% counts keep the spilling hash join (79.8-97.5 ms). It moves the threshold and changes every plan the app role makes. |
| Unbounded per-record probe in every ACL count (run 4) | Fixes the broad band, breaks the selective ones: 2% ranges 7.7-8.3 s, 0.01% 91-179 ms. |
| Forced probe in the page too | 4.25 s in the first cut. The page is already 4-14 ms in this band. |
| Shape picked from the planner's row estimate (A1's wording of option (a)) | The estimate comes from the same pooled statistics that misled the planner; a bounded count of the tenant's own index is exact up to the cap and costs 0.3-18 ms. |
| Raise `work_mem` for `nava_app` | Removes the spill, not the hash of 192k rows, and multiplies memory per connection for every query. Not measured; not pursued. |
| (C) A D13 exception for the band | The miss is plan instability, not a fixed fan-out. A1-2 grants 350 ms only for fan-out the D7 shape cannot avoid, and A2 shows it can be avoided. Kept as the fallback if A2-8 (3) fails. |

## Competitive Gate (ADR-008)

**Result:** PASS. X3 (speed) improves: the ≥ 40% ACL count no longer depends on vacuum state. CP-FW-01 ("slow loading at high ticket volume") is reduced, not widened. No new vendor, cost or component.
