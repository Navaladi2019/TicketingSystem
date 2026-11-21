---
type: adr-amendment
id: ADR-006-A2
title: "Amendment A2 to ADR-006: D7 allows one bounded pre-count per filtered field, and a forced per-record probe in the capped count, for lists with a narrowing row ACL whose visible set is small"
status: PROPOSED
date: 2026-11-21
issue: TICK-1022
rfc: none (amendment from a spike; the decision request is the Postgres & Data Modeling Specialist's TICK-1022 comment 01a1fae7-c4a2-7997-b383-d420fedc2224, run 5; critique thread under TICK-1022 comment 01a1faed-e8a8-7a44-acd8-c0f660981740)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D7 (one compiled query) and the D7 static test; implements the count half of [[ADR-006-A1-d13-fanout-bands-350ms]] A1-6 (2)"
source: "TICK-1022 runs 1-5, PR #796: `docs/data/tick-1022-post-backfill-plan.md`, `spike-bounded-d13-server-time.md`; spike branch `TICK-1022/spike-probe` at `5b37843f` (not for merge). Archived in `docs/adr/ADR-006/spikes/a2-bounded-probe/`"
deciders: Solution Architect (author; verdict); Technical Architect (round-1 critique 2026-11-20, comment 01a1faf4: no blocking item, three rev-1 asks A-C, three spikes; "accept once rev 1 adds guards A-C and S-A2-anti and S-A2-skew pass"); Security & Multi-Tenancy Reviewer (comment 01a1faf2: VERDICT APPROVE for A2-2 and A2-6, R13 raised to Low, four compiler-PR conditions); CTO (no tie-break needed: nothing contested)
---

# ADR-006 amendment A2: a bounded pre-count picks the count's shape

> **Status: PROPOSED, rev 1** (2026-11-21). Every critique item is taken as proposed, so nothing is contested and no CTO tie-break is needed. **A2 becomes ACCEPTED when spikes S-A2-anti and S-A2-skew (A2-11) pass**, the Technical Architect's stated condition. S-A2-multi decides whether A2-12's field cap stays at 3. The compiler PR may be opened and reviewed now. It merges only after A2 is ACCEPTED and the A2-8 gate passes.
>
> **Rev 1 changelog (round-1 critique).**
> - **TA finding 4 / ask A:** new guard A2-3a. The probe is forced only when the agent's visible set is at most `K` (capped count on `records`, no side table). This prevents the anti-correlated walk projected at 0.75-1.5 s at 50% visibility.
> - **TA ask B:** the 6% anti-correlated fixture gets a pass rule (A2-8 (3)).
> - **TA ask C, finding 5:** A2-4 now also gates the pre-count's own plan (Index (Only) Scan on `rfv_*_value`, no Seq Scan). Spike S-A2-skew tests it with a value that is broad in another tenant.
> - **TA finding 2:** the plan-shape test also runs on every PostgreSQL minor or major bump.
> - **TA finding 7:** A2-3 names the dependency on `plan_cache_mode = force_custom_plan` for `nava_app`, and A2-4 runs under the role's settings.
> - **TA finding 9:** A2-12 caps pre-counted fields at 3; S-A2-multi measures it.
> - **TA findings 10-12:** the A2-7 assertion matches the rendered SQL (`$2::uuid`). The explainer states the rare-match worst case. A2-10 re-tunes `K` together with `broadCap`.
> - **Security:** R13 is Low with the reviewer's wording, plus one more revisit trigger. A2-8 (3) measures the latency step at 19,999 against 20,000 side rows. The four compiler-PR conditions are A2-13.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1-I9, as ADR-006 and A1 are.

## In plain words

A list request returns one page of records plus a `total`, capped at 1,001. When the user's row ACL hides most records and the filter matches a large share of the type (≥ 40%), PostgreSQL can count that total in two ways. The fast way: for each record the user can see, look up its value. The slow way: read every side-table row of the value (192k here), build a hash table, and spill it to disk at `work_mem` 4 MB. Which way it picks depends on the side table's **visibility map**, meaning how many of its pages vacuum has marked all-visible. That changes with backfills and ordinary edits, and the `rfv_*` tables are shared by every tenant. So the same request took 34 ms on one fresh stack and 217 ms on the next. Analyzing the table doesn't fix it (run 1). A lower `random_page_cost` only moves the threshold (run 3).

The fix takes the choice away from the planner's guess. Before the list, the server counts the filter's side rows, stopping at 20,000. If the count reaches 20,000, the value is broad. The server also counts the records the user can see, stopping at `K`. If both hold, the count statement is written so that PostgreSQL can only do the per-record lookup. That lookup costs at most one probe per visible record, and the second count keeps that number small. The page statement doesn't change. ADR-006 D7 says no other query may read a side table, so this needs an amendment.

## What was measured

Run 5, spike `5b37843f`, on the D13 fixture: an 800k seed with a 400k type, where the ACL agent sees 24,118 records (6%). PostgreSQL 16.10, `work_mem` 4 MB. The host was busy (the no-filter control read 136-193 ms p95 against 73 ms idle), so **server time is the evidence** and p95 is indicative only.

| ACL cells | Count before | Pre-count + count after | Page | 8-client p95 (indicative) |
|---|---|---|---|---|
| 40%, 5 operators | 31.9-114.4 ms, 3 of 5 spill | 10.9-17.8 + 17.5-26.3 ms | unchanged | 145-233 ms (band (ii) limit 350) |
| 2% | 23.6-33.8 ms | 4.6-7.8 + 19.2-41.4 ms | unchanged plan | 223-842 ms (host; see A2-8) |
| 0.25% | 16.3-23.8 ms | 1.0-1.8 + 14.4-34.4 ms | unchanged plan | 138-341 ms |
| 0.01% | 4.0-16.8 ms | 0.3-1.6 + 5.2-17.8 ms | unchanged plan | 75-246 ms |

Kit-shaped repro (48% value, the TICK-1022 cell): the count took 17.0-26.8 ms in **every** visibility-map state, against 102-168 ms before. The pre-count adds 14-22 ms. The 1% value keeps its plan (15-25 ms). Run 4 showed why the probe needs bounds. Without the pre-count, a 2% range probe took 7.7-8.3 s, and the 0.01% ACL cells, where about 2 of 24,118 visible records match, took 91-179 ms. That second number is the walk guard A2-3a bounds.

## Decisions

| # | Decision |
|---|---|
| **A2-1** | **D7 reads, from this amendment on:** "**Compiled list queries.** In a list request, a side table is read in exactly two places. (1) A correlated `EXISTS (… v.tenant_id = r.tenant_id AND v.record_id = r.id …)` inside the list statements that carry the compiled row ACL: the page, and `total`, the existing capped count (1,001). The count may render a broad filter as a forced per-record probe (A2-3); the page never does. Both select the same rows. (2) At most one **bounded pre-count** per filtered side-table field, for at most 3 fields (A2-2, A2-12). Its result only picks the count's shape. There is no other count, facet or progress query on a side table. A static test enforces both places." |
| **A2-2** | **The pre-count.** `SELECT count(*) FROM (SELECT 1 FROM rfv_<type> v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND <the field's comparisons> LIMIT broadCap) c`, where `broadCap = 20000` is a named constant. It runs (a) only when the list has a narrowing row ACL (ACL branches or ACL arguments), never for an actor whose ACL does not narrow; (b) **after** the mask check (400 `masked_field`), the readiness check (422 `field_not_ready`), value validation and the cursor's masked-sort check, so a rejected request never reaches it (order confirmed by Security in `5b37843f`); (c) in the request's transaction (`tenancy.InTenantTx`), as `nava_app`, under RLS, with the tenant from the verified principal and `field_id` from the registry row resolved in that transaction (I1, I2); (d) once per field, with all of that field's comparisons, exactly as the page's `EXISTS` renders them. Its number is never returned, logged, traced as an attribute or used as a metric label. |
| **A2-3** | **The forced probe, in the count only.** When A2-3a allows it, that field's `EXISTS` in the count statement carries `OFFSET 0` and an opaque value term (`v.value \|\| ''`, `v.value + 0`, `v.value + interval '0'`). `OFFSET 0` makes `simplify_EXISTS_query` refuse the sublink (PG 16 `subselect.c`). That closes both the pull-up to a semi join and the hashed-SubPlan alternative, so PostgreSQL probes `rfv_*_pkey` `(tenant_id, record_id, field_id)` per visible record. No `enable_*` setting, no hint and no new session GUC are used (A1-6). The shape depends on the existing `plan_cache_mode = force_custom_plan` for `nava_app` (`db/grants.yaml`): a generic plan would lose the value-specific estimates. That setting is now a named dependency of A2. The shape is carried as `ListQuery.CountWhere`, which uses the same placeholders in the same order as `Where`. The page keeps today's SQL; a first cut that also changed the page read 4.25 s. |
| **A2-3a** | **Visible-set guard (rev 1, TA ask A).** A broad value alone is not enough. If the agent's records rarely carry the value, the probe walks the whole visible set. So the probe is forced only when **both** hold: the pre-count reached `broadCap`, **and** a capped count of the agent's visible set is at most `K`. That count is `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND <compiled row ACL> LIMIT K+1)`. With ACL branches, each branch counts apart and the counts add up, as the capped count does. It reads `records` only, no side table, so it does not touch D7. It runs once per request, only after some pre-count reached `broadCap`. `K` is a named constant, starting at **30,000** (about 200 ms of probes at run-4 rates of 4-7 µs). S-A2-anti calibrates it. Above `K`, the count keeps today's SQL: at that visibility, hashing the side rows is the cheaper plan, and it is what runs today. Its number is never returned, logged or labelled, the same rule as A2-2. |
| **A2-4** | **The gate is the plan, not the trick.** The opaque term is how the spike got there, not the contract. The contract is a plan-shape test that extends CI-1. It runs under `nava_app`'s role settings, on every change to the compiler **and on every PostgreSQL minor or major bump**. For each value type and operator: (1) a broad filter's count `EXPLAIN` reaches the side table through `rfv_*_pkey`, with no Hash or Merge join on `rfv_*`; (2) a selective filter's count is unchanged; (3) **the pre-count's `EXPLAIN` is an Index (Only) Scan on `rfv_*_value` with no Seq Scan**, with the visibility map in states `none` and `vacuum` (TA ask C). The `bool` term `v.value OR false` is constant-folded away, so `bool` relies on `OFFSET 0` alone, probing `rfv_bool_value` with `record_id` as an equality. The test, not the term, proves it. |
| **A2-5** | **Same rows, proven.** A unit test per value type renders `Where` and `CountWhere` for every operator and checks that the placeholders match. An integration test runs both counts on a fixture of edge values: text under `COLLATE "C"`, `±Infinity` for `num`, `-0` against `0`, `±infinity` for `ts`, both bools, and `in` with repeated values. It asserts the same `total`. The opaque functions are not leakproof. That is safe: PostgreSQL evaluates them only after the RLS quals, and none of them can raise an error on a stored value. CI-2's catalog covers comparison operators only and doesn't change. |
| **A2-6** | **Within-tenant timing (I5), accepted as R13 (Low).** The pre-count doesn't apply the row ACL. So whether a value has ≥ 20,000 side rows in the tenant, counting records the user can't read, shows in latency. Security's analysis (comment 01a1faf2) shows the reach. **A range filter lets a caller binary-search the 20,000-row order statistic of hidden values. Once the boundary is tuned, one hidden record crossing it flips the bit.** The response body, `total` and status don't change. The existing API test ("a value only in a row-ACL-denied record gives the same hits and `total` as a non-existent value") stays, and gains a broad variant (A2-13 (4)). Today's D7 shape already leaks a continuous version of the signal: the cost of the hash join scales with every side row of the value, ACL-denied ones included. The leak needs a field the caller is allowed to read, and nothing crosses tenants. **Approved by the Security & Multi-Tenancy Reviewer.** Field-level secrecy needs a mask; a row ACL alone doesn't give aggregate secrecy. |
| **A2-7** | **The D7 static test** (`internal/fieldindex/d7_test.go`) allows exactly one more site: the pre-count function in `internal/server/records_list_index.go`. It checks that the pre-count's **rendered** SQL matches `v\.tenant_id = \$1`, `v\.field_id = \$2(::uuid)?` and `LIMIT <broadCap>`, using a regex on what the function emits. Any other reader still fails. The A2-3a guard reads `records`, so it is outside this test. |
| **A2-8** | **Merge gate for the compiler PR**, on top of review and green CI. (1) A2 is ACCEPTED. (2) Single-client server time on the D13 ACL cells: no cell's pre-counts + guard + count exceeds its run-5 "before" count by more than 10 ms, and band (ii) cells are ≤ 30 ms. (3) **S-A2-anti** (A2-11) passes, including its 6% cell under A2-8 (2)'s rule (TA ask B). The latency step at 19,999 against 20,000 side rows is recorded with both plans, so R13 states a measured number (Security). (4) **S-A2-skew** passes. (5) TICK-1022's own acceptance: the S21 kit's ≥ 40% ACL cell is inside 350 ms p95 on **two** fresh stacks in a row. |
| **A2-9** | **Budgets don't move.** D13 as amended by A1 stays, and band (ii) is still accepted at 350 ms. A2 removes the visibility-map dependency. It does not claim A1-6's 150 ms exit criterion, which needs an idle reference host and the page half of option (a). The 2% band's page cost stays a TICK-992 item. A1's R-a trigger and alert stay. |
| **A2-10** | **Calibration limit.** `broadCap = 20000` is 5% of the measured 400k type. On a much larger type, the same cap marks smaller shares as broad (0.5% of 4M). That widens the walk A2-3a bounds. Both constants are re-measured together when R-a's existing trigger fires ("a tenant passes 400k records → build the adaptive shape"). Neither is a tenant setting. |
| **A2-11** | **Spikes, with pass and fail stated before they run** (TA). The Postgres & Data Modeling Specialist runs them on the run-5 harness. **S-A2-anti:** a value with ≥ 20k side rows that no visible record carries, at 6% and about 50% visibility, plus a partly correlated value (about 500 visible matches, under the 1,001 cap) at 6%. Visibility-map states `none` and `vacuum`. *Pass:* the count with the guard is ≤ "before" + 10 ms in every cell, and every cell is ≤ 30 ms or under 350 ms p95 in band (ii). *Fail:* tune `K` or reshape A2-3a. Option (C) only if no `K` works. **S-A2-skew:** two tenants; the value is broad (≥ 192k) in tenant B and rare (≤ 100) in tenant A. Pre-count as A's agent in states `none` and `vacuum`. *Pass:* Index (Only) Scan on `rfv_*_value`, ≤ 20 ms. *Fail:* reshape the pre-count so the index is the only cheap plan (for example `ORDER BY v.tenant_id, v.field_id, v.value` for non-`in` operators), then rerun. **S-A2-multi:** a three-field filter, all fields broad, as the ACL agent. *Pass:* pre-counts + count ≤ "before" + 10 ms. *Fail:* lower A2-12's cap. |
| **A2-12** | **At most 3 pre-counted fields per request** (TA finding 9). Only the first 3 ready side-table fields in the filter's canonical order (field id) are pre-counted. Further fields keep today's SQL in the count. This is a named constant, so a five-field filter adds at most three pre-counts. S-A2-multi confirms or lowers it. |
| **A2-13** | **Compiler-PR conditions from Security**, each a runnable test, reviewed by the Security & Multi-Tenancy Reviewer. (1) **No pre-count on a rejected or non-narrowing request:** count the `rfv_*` statements (a query hook or a `pg_stat_statements` delta) for a masked filter (400), a backfilling field (422), an invalid value (400), a cursor whose sort key is masked (400), and an admin or other non-narrowing actor. Each shows zero. (2) **Scope:** besides A2-7, an integration test seeds tenant B with 50k side rows of the same value and checks that tenant A's filter stays not broad, with A's count plan the selective one. (3) **No disclosure:** `broad`, the pre-count and the guard's count never appear in logs, span attributes or metric labels. `observeIndexedList` keeps only its existing `narrowing` label, and a test asserts the recorded attributes. (4) **Same rows, ACL-denied, broad:** 20k+ side rows, all ACL-denied, give `total` = 0 with the same status and body shape. |

**Not changed:** RLS, the row-ACL compilation, the masks (D6, R9/CI-2b), the 1,001 count cap, the page statement, D13 numbers, R-a, R-b, R8, R9, R11, R12. I2-I4 and I6-I9 are not touched: no grant, no role, no write path, no workflow.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. The pre-count and the guard run as `nava_app` under the RLS tenant policy (`FORCE ROW LEVEL SECURITY` on `rfv_*` and `records`), and also filter `tenant_id = $1` from the principal. A2 also removes a cross-tenant influence: on broad values, the count's plan no longer depends on statistics pooled across tenants. |
| I2 tenant source | Holds. `$1` is `p.TenantID` from the verified principal. `field_id` comes from the registry row, never from the request. |
| I3 roles | Unchanged. No grant and no `SECURITY DEFINER` (run 1's ANALYZE route was rejected because it needed one). `plan_cache_mode` is an existing setting, now named as a dependency. |
| I5 one authorization layer | Rows and `total` are unchanged. A within-tenant timing signal is accepted as R13 (Low), approved by Security. |
| I8 configuration over code | `broadCap`, `K` and the field cap are platform constants, not tenant knobs. |
| I4, I6, I7, I9 | Not touched (read path only). |

## New residual

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R13 (Low) | For a list with a narrowing row ACL, latency shows whether a filtered value has ≥ 20,000 side rows in the tenant, counting records the user can't read. Range filters reveal the 20,000-row order statistic of hidden values, and one hidden record crossing a tuned boundary flips the bit. The body, `total` and status don't change. It needs a field the caller may read, and nothing crosses tenants. The step size is measured in A2-8 (3). | Security & Multi-Tenancy Reviewer | A tenant requires timing isolation between row-ACL groups. Or a per-value existence check is added anywhere else. Or a tenant puts a field it treats as sensitive (salary, health) under a row ACL only, without a mask. |

## Response to round 1

| Item | Label | Resolution |
|---|---|---|
| TA 1, 3, 6, 8, 12 | confirmed sound | No change, beyond citing the PG 16 source in A2-3 and re-tuning `K` with `broadCap` in A2-10 |
| TA 2: planner internals, not a documented contract | confirmed | **Accepted.** A2-4 runs on every PostgreSQL bump |
| TA 4: the probe walks the visible set when matches are rare | verify-by-spike, predicted to fail | **Accepted as ask A.** New guard A2-3a. S-A2-anti calibrates `K` |
| TA 5: the pre-count's own plan uses pooled statistics | verify-by-spike | **Accepted as ask C.** A2-4 (3) and S-A2-skew |
| TA 7: depends on `force_custom_plan` | confirmed, not stated | **Accepted.** Named in A2-3 |
| TA 9: no cap on pre-counts per request | non-blocking | **Accepted.** A2-12 caps at 3; S-A2-multi |
| TA 10: assertion on the rendered `::uuid` | nit | **Accepted.** A2-7 uses a regex |
| TA 11: explainer's "stop at 1,001" | nit | **Accepted.** The explainer states the worst case and the guard |
| TA B: no pass rule for the 6% fixture | — | **Accepted.** A2-8 (3) |
| Security: R13 severity and wording | APPROVE | **Accepted.** Low, with the reviewer's wording and one more trigger. Register row updated |
| Security: measure the step | — | **Accepted.** A2-8 (3) |
| Security: compiler-PR conditions 1-4 | — | **Accepted.** A2-13 |

## Rejected options

| Option | Why |
|---|---|
| ANALYZE the side table when a field flips to `ready` (the issue's suggestion) | Run 1: 255 ms with it, 202 ms without. The plan follows the visibility map, not the statistics. It would also need a `SECURITY DEFINER` wrapper, which breaks I3. |
| VACUUM at the flip | The worker can't run it: VACUUM doesn't run in a transaction or function, and it needs ownership. It wouldn't hold either: rewriting 5% of the side rows through edits drops `rfv_num` to 1% all-visible (run 2). |
| `random_page_cost = 1.1` for `nava_app` | Run 3: the `in`, `lt` and ts-range 40% counts keep the spilling hash join (79.8-97.5 ms). It moves the threshold and changes every plan the app role makes. |
| Unbounded per-record probe in every ACL count (run 4) | Fixes the broad band and breaks the selective ones: 2% ranges take 7.7-8.3 s, and 0.01% takes 91-179 ms. |
| Forced probe without the visible-set guard (rev 0) | It walks the whole visible set when the agent's records rarely carry the broad value. The projection is 0.75-1.5 s at 50% visibility (TA 4). |
| An ACL-aware pre-count, to close R13 | It costs what A2 is designed to avoid: the pre-count would join the visible set. |
| Forced probe in the page too | 4.25 s in the first cut. The page is already 4-14 ms in this band. |
| Shape picked from the planner's row estimate (A1's wording of option (a)) | The estimate comes from the same pooled statistics that misled the planner. A bounded count of the tenant's own index is exact up to the cap and costs 0.3-18 ms. |
| Raise `work_mem` for `nava_app` | It removes the spill but not the hash of 192k rows, and it multiplies memory per connection for every query. Not measured, not pursued. |
| (C) A D13 exception for the band | The miss is plan instability, not a fixed fan-out. A1-2 grants 350 ms only for fan-out the D7 shape cannot avoid, and A2 shows this one can be avoided. Kept as the fallback if no `K` passes S-A2-anti. |

## Competitive Gate (ADR-008)

**Result:** PASS. X3 (speed) improves: the ≥ 40% ACL count no longer depends on vacuum state. CP-FW-01 ("slow loading at high ticket volume") is reduced, not widened. No new vendor, cost or component.
