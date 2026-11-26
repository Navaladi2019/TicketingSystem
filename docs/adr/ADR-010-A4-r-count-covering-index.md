---
type: adr-amendment
id: ADR-010-A4
title: "Amendment A4 to ADR-010: shape R's list count reads a covering records_list_updated (INCLUDE org_unit_id), with its text unchanged"
status: PROPOSED
date: 2026-11-26
issue: TICK-1143
rfc: "RFC-023, rev 0 and rev 1, archived at docs/adr/ADR-010/A4/rfc/ (TICK-1143 comments 01a21460, rev 1 in reply to 01a21546)"
supersedes: none
amends: "[[ADR-010-A3-records-driven-region-shape]] A3-1 (the count's index), A3-5 (bound), A3-R2 (rewritten), A2-R4 as extended by A3"
evidence: "docs/adr/ADR-010/A4/spikes/ (Technical Architect round 1 and re-check; Postgres & Data Modeling Specialist plan checks; Security & Multi-Tenancy Reviewer plan checks; the TICK-1104 bench)"
deciders: "Solution Architect (author); Technical Architect (round 1 CHANGES B1, B2; re-check CHANGES B2); Postgres & Data Modeling Specialist (round 1 CHANGES P1 to P4; re-check CHANGES guard, P3); Security & Multi-Tenancy Reviewer (round 1 CHANGES S1 to S3; re-check APPROVE); CTO (tie-break, round 2 closed, comment 01a2154d)"
---

# ADR-010 amendment A4: shape R's count reads a covering index

> **Status: PROPOSED.** The verdict comes from the Solution Architect after this PR, under the merge policy set by vknavaladi. The CTO's tie-break (TICK-1143, comment 01a2154d) closed round 2 with five rulings, and all five are folded in below. No reviewer dissents from the index.
>
> **Invariants source.** The project description is empty, so the invariant check is incomplete. Checked against ADR-001 I1 to I9, as A2 and A3 were.

**What changes:** the index the default ticket list already uses, `records_list_updated`, also stores each ticket's org unit. The region count can then be answered from the index alone, without opening the ticket rows.
**Why:** where one customer holds about half of all tickets or more, the count for a small ticket type under a large part of the organisation could read every ticket of every customer. Page plus count then took 162 to 178 ms of SQL against a 150 ms list budget.
**What you'll see when it's done:** the same tickets and the same total. The count takes about 17 to 28 ms with a warm cache and 18 to 25 ms after a restart, whatever the table's size.
**Codes used below:**
- **R** is ADR-010-A3's shape that reads the ticket type's own records.
- **IOS** is an Index Only Scan, a read answered from the index without visiting the table.
- **N̂_e** is the estimated number of records of the type, **|S|** the units in the subtree, and **m** the matching records.
- **A2-4** is ADR-006-A2's first-page shortcut, which skips the count on a short first page. It is on the S21 branch, not yet in `main`.
- **W1** is the Technical Architect's test table: 250,000 records of 1 KB, one tenant, N̂_e = 33,366.

## Decision

| # | Rule |
|---|---|
| **A4-1** | **Covering index.** A new NO TRANSACTION migration:<br/>• builds `records_list_updated_v2 ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id)` with `CREATE INDEX CONCURRENTLY`;<br/>• drops `records_list_updated` CONCURRENTLY;<br/>• renames v2 to `records_list_updated`.<br/>On a re-run it drops an INVALID v2, as `20261109140000_records_list_updated.sql` does. `org_unit_id` is a fixed column, so PLAN-003 D25 holds (no data or expression index). |
| **A4-2** | **The count text does not change:** A3-1's, for one arm and for `UNION ALL`. It has no `ORDER BY` and no Go change. |
| **A4-3** | **The page arm is unchanged.** Its text and its pin (`Index Scan records_list_updated`) stay. It reads every column, so it cannot be an IOS. |
| **A4-4** | **Fallback, decided now:** if a CI pin in A4-5 shows the un-negated R count is not an IOS of `records_list_updated`, the implementation adds `ORDER BY r.updated_at DESC, r.id DESC` inside that R arm, before its own `LIMIT 1001` (RFC-023 rev 0, D2, D3 and D5). The task reports this on its issue; no new RFC is needed. **If D4 applies, the order is added only to counts with at most one R arm. A count with two or more R arms keeps A3-1's text.** Under the fallback:<br/>• a bitmap + Sort of the type's slice is a pass, and only a Sort over a Seq Scan triggers anything;<br/>• a branched count passes if it crosses no 150 ms line that it passes today, and loses no more than the unbranched count of the same world and m. |
| **A4-5** | **Plan pins, for the CI worlds only.** In production the count may also be a heap index read or a bitmap under a filter the index does not hold, or a Seq Scan in small tables.<br/>• **R count (CI-3, CI-4, and the un-negated R arm in CI-5):** `Index Only Scan` on `records_list_updated`, filter `(hashed SubPlan n) IS TRUE`, no `Sort`, no `Gather`. Its `Index Cond` starts with `tenant_id`, and rows read, (`Actual Rows` + `Rows Removed by Filter`) × `Loops`, are at most A's rows of the type.<br/>• **A negated R arm in CI-5** (requester first, then `under`) is pinned only as no worse than today's plan class.<br/>• **CI-4 guard:** re-plan with `enable_indexonlyscan = off`. Compare only the `records` scan node's run cost (total minus startup). The runner-up must cost at least 1.3× the IOS, or the test fails as "world at a tie".<br/>• **Mutant `r_count_no_covering`** (CI-4 built with today's index): the count is a Seq Scan, and rows read are more than A's rows.<br/>• **The page pin is unchanged.** TICK-1104's class pin for the count is replaced. |
| **A4-6** | **CI-4's world:** 200,000 empty rows, N̂_e = 33,366, 50,000 rows of other tenants (A's share is 75 %), \|S\| = 10,111 and 20,222, m = 0.<br/>• Today the count there is a Seq Scan, 53 to 54 % cheaper than the next plan, so it is red before A4 and not a tie.<br/>• With the index it is an IOS in 4 of 4 loads; the mutant is a Seq Scan in 4 of 4.<br/>• Node run cost: IOS 1,724 against Seq Scan 8,905, a ratio of 5.2. |
| **A4-7** | **A2-R4, count part, reworded:** "R's count plans with the statement's values (`QueryExecModeExec`; `plan_cache_mode = force_custom_plan` on `nava_app`, `db/grants.yaml`). Before A4, under `force_generic_plan` the count was a Seq Scan in four of five measured cells. With A4 the generic plan is reported, not pinned; the report must show an IOS, an index walk or a bitmap of the type's slice." The trigger is unchanged. The generic-plan IOS is not measured. |
| **A4-8** | **A3-5 bound restored:** R's unfiltered count reads at most the type's index entries (≤ about 33,400) plus visibility-map pages, and no row of another tenant. |
| **A4-9** | **Security.** A new register row, AR-TICK-1143-01 (below), at 🟡 Low. AR-TICK-1040-01 is unchanged: its three bits do not depend on other tenants' rows (Security, confirmed). |
| **A4-10** | **TICK-1105** gains the A3-R2 cell at m = 500 beside m = 0, and records the count's plan. Once A2-4 is in `main`, the m = 0 cell runs no count. |

**AR-TICK-1143-01** (a list count's time can include other tenants' rows) · 🟡 Low · ADR-010-A4.
- **The signal:** when PostgreSQL reads `records` with a Seq Scan for a list's count, the time also covers other tenants' rows. It can hint at how many records all other tenants hold together; with two tenants, at the other one's size.
- **What does not cross:** no row, total or shape choice crosses the tenant wall. The tenant test runs first, so the filter never evaluates another tenant's row content.
- **Removed:** A4 removes it for R's unfiltered count, which reads the covering index `records_list_updated` (an Index Only Scan whose range starts at the tenant).
- **Remains:**
  - in small tables;
  - under a request filter the index does not hold, where it is as before A4 and can reach the whole table (measured: a default-valued data filter, 400,000 rows, 34,490 other tenants' rows read);
  - in an R arm that follows another branch of the policy, whose negation names a column the index does not hold;
  - in counts that are not R.
- **Accepted** by the Security & Multi-Tenancy Reviewer (TICK-1143), who owns the row.
- **Trigger:**
  - a requirement that a tenant's total record volume is confidential from other tenants;
  - CI-3, CI-4, CI-5 or TICK-1105 shows a Seq Scan for an unfiltered R count;
  - an unfiltered R count reads more rows of other tenants than N̂_e;
  - a request-filtered list count does so in a production slow-query report.

The R-arm clause is the CTO's addition. Security confirms it on this PR.

## Context it rests on

| Fact | Evidence | Known / inferred |
|---|---|---|
| Without the index, the unordered R count can be a Seq Scan of the whole table. That depends on table rows / N̂_e and on the tenant's share, and several cells are near a cost tie | TICK-1104 bench; Postgres round 1 (lead 3.7 to 5.5 % at 420,000 rows, −0.3 to 1.3 % at a 54 % share) | Known |
| With the index, the count is an IOS for both texts (with or without an order) | TA round 1 (W1, W2); Postgres (three cells, CI-4's world 4 of 4) | Known |
| Count, ms. Warm: today 86–110 (W1) and 145–170 (W2); ordered 61–80 and 64; **index 17.3–27.6 and 16.9–18.0**. Cold, m = 500: today 1,122–1,654; ordered 8,958–10,231; **index 18–25**. After 5 % updated with no vacuum: 23.0–28.1, still an IOS | TA round 1, on a host 1.3 to 1.8× slower than the Postgres specialist's quiet window | Known |
| Index cost: 77.8 → 95.6 bytes a row (+23 %), +2.3 % across `records` indexes, about 62 → 77 MB at 800,000 rows. WAL +31 bytes per update (+1.0 to 1.5 %). HOT is already lost to `updated_at`. Built in 1.3 s at 800,000 rows | Postgres round 1; TA round 1 | Known; build time at production size **not measured** |
| A negated R arm names `requester_id`, which the index does not hold, so it is a Seq Scan or a bitmap, as today | Postgres re-check | Known |

## Consequences

- **One `core-dev` task against `main`** (the Technical Lead plans it):
  - the migration (A4-1);
  - `internal/db/dbtest/records_pipeline_test.go`'s index list, with `INCLUDE (org_unit_id)` on `records_list_updated`;
  - the pins of A4-5 and A4-6 in `internal/records/pipeline/acl_under_records_integration_test.go`.

  The AR-TICK-1143-01 row is already in `docs/security/accepted-risks.md`, added on this design PR, as the repo's rule asks for a newly accepted risk.

  It changes no Go in `internal/authz`, `internal/records/store` or the handler, so it does not conflict with S21's `list.go` unless A4-4 fires.
- **Merge order against S21 (ADR-006-A2 A2-4)** is the Technical Lead's call at plan time. If A4 lands first, the plan says so and cites A4-R3's first sentence.
- **Then TICK-1105** runs, with A4-10's cell.
- **Handbook:** `core-acl`, the region page and `records` indexes (`docs/handbook/src/database/records.md`) change. This is the Technical Lead's handbook task.

**Tests that prove A4:**

| # | Test | Catches |
|---|---|---|
| CI-1 | The `records` index list, with `INCLUDE (org_unit_id)` | The index missing or redefined |
| CI-2 | `TestNoTransactionMigrationsAreIdempotent` covers the new file; a failed build's INVALID v2 is dropped on re-run | A half-built swap |
| CI-3 | Shared world (as today), 10,111 and 20,222 units: A4-5's count pin; the page pin unchanged | A changed plan |
| CI-4 | A4-6's world: the count pin, the 1.3× node guard, mutant `r_count_no_covering` | Red before A4; AR-TICK-1143-01; drift to a tie |
| CI-5 | `under` first then requester: the R arm is an IOS. Requester first then `under`: the negated R arm is no worse than today's class. Two reads of `records` are accepted | `UNION ALL` |
| CI-6 | Agreement suite with R forced; 1,000, 1,001 and 1,002 matches through the handler | Rows or totals (A2-6, A3-4) |
| CI-7 | A3's five cross-tenant items | I1 |
| CI-8 | Generic-plan report (A4-7) | A2-R4 |

**API test scenarios:** no operation, status or body changes. `listRecords` 200 and 500 keep the tests from TICK-1104. The operation is a read, so there is no invalid-write or idempotent-replay case. Cross-tenant is CI-7.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. No new statement. The IOS range starts at `tenant_id`, and RLS is unchanged. |
| I5 one authorization layer | Holds. The text and rows are unchanged; an index cannot change `count(*)`. |
| I8 configuration over code | Not applicable: a fixed index, no setting. |
| I2 to I4, I6, I7, I9 | Not touched. |

## Competitive Gate (ADR-008)

**Result:** PASS.
1. **Axis moved:** X3 (speed). The R count moves from 86–170 ms warm and 1.1–1.7 s cold to 17–28 ms in both.
2. **Pain points:** none added; no setting, limit or refusal.
3. **Measured by:** the TA spike (round 1 and re-check), then TICK-1105.
4. **Parity:** I1 to I9 unchanged.

## Rejected options

| Option | Why not |
|---|---|
| `ORDER BY` on the count, without the index (RFC-023 rev 0) | Cold, the count took 8,958–10,231 ms against 1,122–1,654 today. The order makes about 24,400 random heap reads, while today's Seq Scan reads the heap in order. It also needs a Go change and loses 16 to 37 ms on branched counts in low-ratio tables. |
| Index and order together | With the index, both texts plan as an IOS, so the order adds nothing that was measured. Under a filter the index does not hold, it brings back the random cold reads, and it adds the Go change. Kept as the A4-4 fallback. |
| Reject; keep A3-1 as it is | The count's worst case grows with the whole table, breaks A3-5's bound and times other tenants' rows. |
| Order every negated R arm now (P3 option 1) | It brings back the 9 to 11 s cold walk on exactly those counts, which A4 exists to remove. A negated arm plans as today under A4, so no cell gets worse. |
| `INCLUDE` every column a negation can name (P3 option 2) | One more column per policy branch, and it still cannot cover a data filter. |
| `SET LOCAL enable_seqscan = off` around the count | Round trips on every R list, a setting to undo before the page, and it reaches the other `UNION ALL` arms. |
| Extended statistics on `(tenant_id, entity_definition_id)` | Not needed: RLS counts the tenant once (Postgres). It would change plans across the product. |
| A whole-plan 1.3× guard | With IOS off, the hashed SubPlan's closure read also loses its IOS. That adds about 1,600 to every runner-up, so the mutant passed at 2.13× although both plans were the same Seq Scan (Postgres re-check). |

## Recorded dissent

None on the decision. On P3 the Postgres & Data Modeling Specialist named three options and left the choice open. The CTO decided option 3, a residual with a trigger (comment 01a2154d).

## Risks and residuals

| # | Residual | Owner | Trigger to re-decide |
|---|---|---|---|
| A4-R1 | AR-TICK-1143-01, Low | Security & Multi-Tenancy Reviewer | In the row |
| A4-R2 | A filter the index does not hold (`state IN (…)`, a data filter) brings back heap reads and today's plan choice, Seq Scan included | Postgres & Data Modeling Specialist | A3-R2 triggers 2 and 3 on a filtered list |
| A4-R3 | **Until ADR-006-A2 A2-4 is in `main`, A4 makes the first cold R list with at most `limit` matches slower: 1.45–1.95 s today (the Seq Scan count warms the heap for the page) against 8.8–11.2 s with the covering index (W1, m = 0, measured). After A2-4 the count is skipped there, and the page's cold walk is the same with or without A4.** The R page's cold walk is the page arm's, which is not reopened | Solution Architect, as A3's owner | A region list p95 above 1 s in the first 10 minutes after a database restart, or the first installation whose `records` heap is larger than RAM. The page arm is then reopened. |
| A4-R4 | The index is +23 % and WAL +1.0 to 1.5 % on every `records` write. The swap's build time on a production-sized table is not measured | Postgres & Data Modeling Specialist | The migration takes over 10 minutes in the stage rehearsal, or the `records` write p95 rises by more than 5 % |
| A4-R5 | **A negated R arm is not an IOS.** An R arm after another policy branch (requester, assignee, group) carries that branch's negation. It names a column the index does not hold, so it plans as today: a Seq Scan or a bitmap. In CI-4's world it reads 50,000 rows of other tenants | Postgres & Data Modeling Specialist | CI-5, TICK-1105 or a production slow-query report shows a negated R arm above the 150 ms budget, or reading more other tenants' rows than N̂_e. The fix is then A4-4's order on that arm, already decided. |
| A3-R2 (rewritten) | R's count reads outside the type's index range: a Seq Scan, `records_org_unit`, or a heap read of the whole slice. With A4, the count is an IOS in every measured cell | Postgres & Data Modeling Specialist | 1. CI-3, CI-4 or CI-5 fails, including CI-4's guard; A4-4 then applies. 2. TICK-1105, either A3-R2 cell: the count is not an IOS of `records_list_updated`, or handler p95 is above 150 ms, or the count is slower than CC in the same run. 3. Production: record-list p95 above 100 ms while `authz_under_list_shape_total{shape="records"}` rises (A3-R5). 4. `records` autovacuum falls behind (`n_dead_tup` above 20 % of `n_live_tup`, or the last autovacuum older than 24 h) while trigger 3 holds. |
| A2-R4 | Reworded in A4-7 | Postgres & Data Modeling Specialist | Unchanged |
