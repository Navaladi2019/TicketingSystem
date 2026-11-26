---
type: rfc
id: RFC-023
title: "Shape R's list count reads a covering index (proposed amendment A4 to ADR-010)"
status: REVISED (rev 1, the one revision; for the Technical Architect's re-check of B1 and B2, Security's of S1 to S3, Postgres's of P1 to P4)
date: 2026-11-26
issue: TICK-1143 (rev 0: comment 01a21460; critiques: TA 01a21545, Security 01a21488, Postgres 01a214ad; CTO 01a21546)
amends: "[[ADR-010-A3-records-driven-region-shape]] A3-1 (the count's index), A3-5 (bound), A3-R2 (trigger), A2-R4 as extended by A3"
---

# RFC-023 rev 1: how shape R's list count stops reading the whole ticket table

> **Invariants source.** The project description is empty, so the invariant check is incomplete. Checked against ADR-001 I1 to I9.
> **Numbering (assumed):** RFC-023; the amendment would be ADR-010-A4.
> **Rev 1 changes the recommendation.** Rev 0 recommended `ORDER BY` on the count. With the measured index numbers, rev 1 recommends **the covering index and no change to the count text**. Rev 0 is kept below its Response for the record. Where they differ, rev 1 wins.

**What changes:** the index that the default ticket list already uses also stores each ticket's org unit. The region count can then be answered from the index alone, without opening the ticket rows.
**Why:** where one customer holds about half of all tickets or more, the count can read every ticket of every customer. After a database restart, the order-based fix proposed first made the count take 9 to 11 s.
**What you'll see when it's done:** the same tickets and the same total. The count takes about 17 to 28 ms whether the cache is warm or cold, and whatever the table's size.
**Codes used below:** as in rev 0. **IOS** = Index Only Scan, a read that answers from the index without visiting the table. **VM** = the visibility map, which tells PostgreSQL which table pages it can skip.

## 1. Summary (rev 1)

- **Recommendation: the covering index, without the order.** Replace `records_list_updated` with the same key plus `INCLUDE (org_unit_id)`. Keep the name. A3-1's count text does not change.
- **Why it wins:** it is the only option whose count holds in every measured cell, warm and cold, with no list getting slower (table in section R.1). It needs no Go change, so B1 does not arise.
- **What it costs:** that index +23 % (+2.3 % across all `records` indexes), WAL +31 bytes per update (+1.0 to 1.5 %), and one online index swap. HOT updates are already lost to `updated_at`.
- **What it does not fix:** a request filter outside the index (`state IN (…)`, a data filter) brings back heap reads, and plan choice is as today. The R page's own cold walk is unchanged (8.7 to 11.2 s at m = 0). The page arm is out of scope, so that is recorded as a residual, not decided.
- **The order becomes the pre-decided fallback**, used only if the CI pins show the IOS is not picked (D4).

## Response — round 1

### R.1 The comparison the CTO asked for

Counts in ms. **TA** = Technical Architect's host (1.3 to 1.8× slower than the specialist's). **PG** = the specialist's plans and quiet window.

| | Today | Order only | Covering index only | Both |
|---|---|---|---|---|
| Warm count, Seq Scan cells (W1, W2) | 86–110, 145–170 (TA) | 61–80, 64 (TA) | **17–28, 17–18 (TA)** | IOS for both texts, so the same as the index (TA) |
| Cold count, m = 500 (W1) | 1,122–1,654 | **8,958–10,231** | **18–25** | 18–25 (IOS) |
| After 5 % of the type updated, no vacuum | – | – | 23–28, still IOS (TA) | same |
| Branched, low ratio (B1 cells) | baseline | +16 to +37 | **not measured**; the text is unchanged, so the same plan class as today with one cheaper path | as the index |
| Filter outside the index (`state IN`, data filter) | as today | ordered walk with heap reads; bounded by the type's slice; cold reads at random | as today: heap reads come back (cost 2,820–7,380, PG); Seq Scan possible | as order only |
| Other tenants' rows read (AR-TICK-1143-01) | up to the whole table | none where the walk is picked; Seq Scan remains in small tables and under filters | none where IOS is picked (range on `tenant_id`); same remainder | same |
| Write and size cost | 0 | 0 | +23 % that index, +1.5 % WAL | same |
| Go change | – | `acl.go`, `list.go`, handler (D5 of rev 0) | **none** | both |
| Migration | – | none | `CREATE INDEX CONCURRENTLY`, drop, rename | same |

**Decision rule, written before choosing:** pick the option with no measured cell worse than today. Among those, pick the one with fewer moving parts.
- **Order only fails it:** cold, it is 6 to 9× today's count.
- **The index passes it:** no measured cell is worse.
- **"Both" adds nothing measured:** with the index, both texts plan as an IOS. The order would matter only under a filter outside the index. There it brings back B2's random cold reads, and it adds the Go change and B1's branched losses.
- **"Both" stays the fallback** if a CI pin shows the IOS is not picked.

### R.2 Decisions, rev 1 (replace rev 0's D1 to D10)

| # | Decision | Why |
|---|---|---|
| **D1** (item 1) | **Covering index.** A new NO TRANSACTION migration builds `records_list_updated_v2 ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id)` CONCURRENTLY. It drops `records_list_updated` CONCURRENTLY, then renames v2. It drops an INVALID v2 on re-run, as `20261109140000` does. `org_unit_id` is a fixed column, so PLAN-003 D25 (no data or expression index) holds. | R.1 |
| **D2** (item 2) | **Count text: A3-1's, unchanged,** one arm and `UNION ALL`. No `ORDER BY`. | The IOS is picked for both texts (TA, Postgres). No Go change, so no branched risk. |
| **D3** | **The page arm's text and plan pin are unchanged.** The page keeps `Index Scan records_list_updated`: it reads every column, so it cannot be an IOS. | The CTO's scope. The same index name keeps the page's pin true. |
| **D4** | **Fallback, pre-decided:** if CI-3, CI-4 or CI-5 shows the R count is not an IOS on `records_list_updated`, the implementation adds rev 0's order (rev 0 D2, D3, D5), with B1's rule (R.3) and P3's pass (R.6). The task reports this on the issue; no new RFC. | It keeps the bound if the planner prefers another path. |
| **D5** (item 3) | **Plan pins.** **Count:** `Index Only Scan` on `records_list_updated`, filter `(hashed SubPlan n) IS TRUE`, no `Sort`, no `Gather`, in CI-3, CI-4 and CI-5. **Page:** unchanged. The class pin of TICK-1104 is replaced. **These pins hold for the CI worlds only.** In production the count may also be a heap read, bitmap or Seq Scan in small tables or under a filter (P2). | P2 |
| **D6** (item 4) | **A2-R4, count part:** "R's count plans with the statement's values (`QueryExecModeExec`; `plan_cache_mode = force_custom_plan`). Under `force_generic_plan` the unordered count was a Seq Scan in four of five measured cells before A4. With A4 the generic plan is reported, not pinned, and the report must show an IOS, an index walk or a bitmap of the type's slice." The trigger is unchanged. | Postgres D7 note. The generic-plan IOS is **not measured**: CI-8 reports it. |
| **D7** (item 5) | **A3-R2 rewritten** (R.8). | |
| **D8** | **AR-TICK-1143-01** added to `docs/security/accepted-risks.md` in the A4 PR, at Low (R.7). | S1 |
| **D9** | **TICK-1105 gains the A3-R2 cell at m = 500** beside m = 0, and records the count's plan. | Unchanged from rev 0 D10 |
| **D10** | **`records_pipeline_test.go`'s index list** gains `INCLUDE (org_unit_id)` for `records_list_updated`. | It is the test that would otherwise fail. |

### R.3 B1 (Technical Architect, blocking): the +15 ms rule

**Replaced.** The new rule for a branched count is section 4's: no list crosses 150 ms that passes it today. A branched arm is also flagged if it loses more than the **unbranched** count of the same world and m, beyond the A/A spread.

**Why +15 ms was wrong before the data:**
- A1 existed to catch one thing: PostgreSQL planning an R arm inside `UNION ALL` differently from the same arm alone.
- Measured against today, +15 ms adds that effect to the L3 and L4 losses that D1 had already accepted for unbranched counts. So the rule could fire on a loss rev 0 had already declared acceptable, and it could not isolate the union effect.
- Comparing with the unbranched count of the same world removes the accepted loss and leaves only the union effect.

**Applied to the Technical Architect's cells:**

| World | Branched loss | Unbranched loss in the same world | Crosses 150 ms only when ordered? | Result |
|---|---|---|---|---|
| W5 (ratio 2) | +16.3 to +23.8 | +17.3 / +28.6 (m = 500) | no (TA) | within the unbranched loss: pass |
| W6 (share 4 %) | +26.0 / +36.7 | +24.7 / +32.1 (m = all) | no | +1.3 / +4.6 ms over the unbranched loss. Pass, unless the A/A spread in `out-run*.log` is below that; the TA confirms on re-check |
| W3, W4 | +12.0 to +22.2 | no unbranched loss quoted | no | pass under section 4. The second test needs the unbranched figure, which the TA has in `out-run*.log` |
| W1, W2, W7 | −23 to −156 | – | – | gain |

Under rev 1 this rule governs only the D4 fallback, since D2 changes no text.

### R.4 B2 (Technical Architect, blocking): the cold cache

- **Accepted as measured, not reopened.** Rev 0's L6 ("the order moves the cold cost between the statements") was wrong: today's Seq Scan reads the heap in order and warms it for the page. The ordered walk makes about 24,400 random reads.
- **Resolution:** the index removes the count's cold cost (18 to 25 ms). The order is no longer adopted, so the 9 to 11 s **count** is not an accepted risk. Under D4's fallback the order ships only beside the index, where both texts plan as an IOS.
- **What remains is the page's, and it is recorded, not decided** (page arm out of scope):

| # | Residual | Owner | Trigger |
|---|---|---|---|
| A4-R3 (rewritten) | **R page, cold, m = 0:** the first R list after a restart walks the type's records at random, **8.7 to 11.2 s** (TA, W1, `main` order, IOS count). After S21 the m = 0 page still walks. A4 does not change it. | Solution Architect, as A3's owner | A region list p95 above 1 s in the first 10 minutes after a database restart, or the first installation whose `records` heap is larger than RAM. Then the page arm is reopened (prewarm, or a page read through the covering index). |

### R.5 P1 and S3 (blocking): a CI world that is red today and not a tie

**CI-4 becomes the Postgres specialist's world:** 200,000 empty rows, N̂_e = 33,366, 50,000 rows of other tenants (share 75 %), |S| = 10,111, m = 0.
- **Red today:** the unordered count is a Seq Scan, leading the next plan by 53 to 54 % (Postgres). That is far from a tie. The 420,000-row world (3.7 to 5.5 %) and the 54 % share world (−0.3 to 1.3 %) are dropped.
- **It carries other tenants' rows,** so it is the security test S3 asks for.
- **Pins in CI-4:**
  1. The count is `Index Only Scan records_list_updated` under the hashed SubPlan.
  2. **Rows of other tenants read = 0.** On an IOS the index range starts at `tenant_id`, so none are read. The test asserts that `Rows Removed by Filter` holds none of them.
  3. **Not a tie, asserted by the test itself:** `EXPLAIN` again with `enable_indexonlyscan = off`, and require the runner-up's total cost ≥ 1.3× the IOS's. Below that, the test fails as "world at a tie".
- **Mutant `r_count_no_covering`:** build the world with today's `records_list_updated` (no INCLUDE). The count must be a Seq Scan with a non-zero count of other tenants' rows removed. This is the red proof, and it replaces `r_count_unordered`.
- **Not run in the repository's suite yet** (Postgres, Security). If the IOS does not win by 1.3× there, D4 applies.

### R.6 P2, P3, P4 (Postgres, blocking)

| # | Resolution |
|---|---|
| **P2** | **Accepted.** The statement is corrected in D5. The ordered count (fallback only) has three plans: index walk; bitmap + Sort over the type's slice (A3-5 holds); Sort over a Seq Scan in small tables. The covering-index count has, by the same logic, IOS; a heap index read or bitmap under a filter outside the index; a Seq Scan in small tables or under a selective filter. Pins are for CI worlds only. |
| **P3** | **Accepted, for the D4 fallback.** A1's outcome for a bitmap + Sort is a **pass**. Only a Sort over a **Seq Scan** triggers anything. **CI-5 is built on the requester policy** (`under` or requester self; estimate 3,483), not on two R arms. Its checker accepts two reads of `records`: the R arm's IOS and the requester arm's own index. |
| **P4** | **Accepted.** Rev 0's "36–50 ms" per walk is replaced by **36–67 ms**. Page plus ordered count reaches about 119 ms, and the 80 ms line had 13 ms of room. That only strengthens rev 1's change: with the index the count is 17 to 28 ms, so page plus count is about 61 to 95 ms (page 44 to 67). Also corrected: L3's crossover is a guess, because its 0.15 µs per row includes the hash build; and section 7's shares are "at 400,000 empty rows". |

### R.7 S1, S2 (Security, blocking): AR-TICK-1143-01

- **S1, accepted:** Low, with its own row, added in the A4 PR. AR-TICK-1040-01 stays as written.
- **S2, accepted:** "not provably gone" is replaced by "still present in small tables and under a selective filter". Security's ruling that a Sort over a Seq Scan is accepted at Low means A5 no longer holds acceptance.
- **Trigger: the Postgres wording, as the CTO asked.** Security's "rows / N̂_e of 4 or more" is already met at ratio 5 in one-tenant cells, which hold no other tenants' rows. It measures the plan, not the signal. "Other tenants' rows read exceed N̂_e" measures the signal itself: how much of other tenants' data a count's time carries. It was never exceeded in 59 cells plus Security's filtered cells.

**Row, Security's text with the A4 sentence changed for the index, for Security's re-check:**

> **AR-TICK-1143-01** (a list count's time can include other tenants' rows) · 🟡 Low · ADR-010-A4. When PostgreSQL reads `records` with a Seq Scan for a list's count, the time also covers other tenants' rows. It can hint at how many records all other tenants hold together; with two tenants, at the other one's size. No row, total or shape choice crosses the tenant wall, and the tenant test runs first, so the filter never evaluates another tenant's row content. A4 removes it where R's count reads the covering index `records_list_updated` (an Index Only Scan, whose range starts at the tenant). It remains in small tables and under a selective filter, where a Seq Scan is the cheaper full read, and in counts that are not R. · Accepted (Security & Multi-Tenancy Reviewer, TICK-1143) · Owner: Security & Multi-Tenancy Reviewer · Trigger: a requirement that a tenant's total record volume is confidential from other tenants; or CI-3, CI-4, CI-5 or TICK-1105 shows a Seq Scan for an R count; or any R count reads more rows of other tenants than N̂_e.

- **S4, taken:** L5 now names a Sort over a Seq Scan and Gather Merge too.
- **S5:** only relevant to the D4 fallback. If the order ships, `Records` and `CountOrdered` fall under A3-7's never-exposed list, and the existing leak test covers them.

### R.8 A3-R2, rewritten (item 5)

| | Text |
|---|---|
| **Risk** | R's count reads outside the type's index range: a Seq Scan, `records_org_unit`, or a heap read of the whole slice. Before A4, the Seq Scan depended on the ratio and on the share. At 400,000 empty rows, it appeared at shares of 54 % and 91 % and not at 10 %. With one tenant and N̂_e = 33,366, it appeared up to a ratio of at least 24 (empty rows) and 7.5 (1 KB rows). Several of these cells are near a cost tie (Postgres P1). With A4, the count is an IOS in every cell measured (TA W1 and W2, three Postgres cells). |
| **Owner** | Postgres & Data Modeling Specialist |
| **Triggers** | 1. CI-3, CI-4 or CI-5 fails, including CI-4's 1.3× check; D4 then applies.<br/>2. TICK-1105, either A3-R2 cell (m = 0, m = 500): the count is not an IOS of `records_list_updated`, or handler p95 is above 150 ms, or the count is slower than CC in the same run.<br/>3. Production: record-list p95 above 100 ms while `authz_under_list_shape_total{shape="records"}` is rising (A3-R5).<br/>4. New, for the IOS: `records` autovacuum falls behind (`n_dead_tup` above 20 % of `n_live_tup`, or the last autovacuum older than 24 h) while trigger 3 holds. Newest-first index entries then point at pages not yet marked all-visible. |

### R.9 Tests, rev 1 (replace rev 0's section 6)

| # | Test | What it catches |
|---|---|---|
| CI-1 | `records_pipeline_test.go`: the index list, with `INCLUDE (org_unit_id)` | The index missing or redefined |
| CI-2 | Migration: `TestNoTransactionMigrationsAreIdempotent` covers the new file; after a failed build, the re-run drops an INVALID v2 | A half-built swap |
| CI-3 | Plan, shared world (as today): D5's count pin at 10,111 and 20,222 units; the page pin unchanged | A changed plan |
| CI-4 | Plan, R.5's world: IOS, zero other tenants' rows, the 1.3× check; mutant `r_count_no_covering` gives a Seq Scan | Red before A4; AR-TICK-1143-01; a world drifting to a tie |
| CI-5 | Plan, `under` or requester self: the R arm is an IOS; the requester arm its own index; two reads accepted | `UNION ALL` |
| CI-6 | Existing agreement suite with R forced; 1,000 / 1,001 / 1,002 through the handler | Rows or totals (A2-6, A3-4) |
| CI-7 | Existing five cross-tenant items | I1 |
| CI-8 | Generic-plan report (D6) | A2-R4 |

**API test scenarios:** no operation, status or body changes. `listRecords` 200 and 500 keep TICK-1104's tests.

### R.10 Residuals, rev 1 (replace rev 0's section 10)

| # | Residual | Owner | Trigger |
|---|---|---|---|
| A4-R1 | AR-TICK-1143-01 (R.7), Low | Security & Multi-Tenancy Reviewer | In the row |
| A4-R2 | A filter outside the index brings back heap reads, and today's plan choice, Seq Scan included | Postgres & Data Modeling Specialist | A3-R2 triggers 2 and 3 on a filtered list |
| A4-R3 | R page cold walk (R.4) | Solution Architect | R.4 |
| A4-R4 | Index +23 % and WAL +1.0 to 1.5 % on every `records` write; the swap's build time on a production-sized table is not measured | Postgres & Data Modeling Specialist | The migration takes over 10 minutes in the stage rehearsal, or `records` write p95 rises more than 5 % |
| A3-R2 | R.8 | Postgres & Data Modeling Specialist | R.8 |

### R.11 Sequencing, rev 1

1. TICK-1104 (PR #891) merges as it is.
2. A4 is one `core-dev` task against `main`: one migration, `records_pipeline_test.go`, the pins of R.9. No change to `internal/authz`, `internal/records/store` or the handler, so **no conflict with S21's `list.go`** unless D4 fires. The S21 merge-order question then only applies to the fallback, and it still goes to the Technical Lead.
3. TICK-1105 runs after it, with D9's cell.

### R.12 Items not changed by rev 1

Security's confirmations stand: I1 and I5 hold, and AR-TICK-1040-01's three bits do not depend on other tenants' rows. Postgres's confirmations stand: D2 and D3's texts for the fallback, and L4 with the tenant counted once. Technical Architect: A2 to A6 and N1 to N4 are taken as measured. A2's rule result, "adopt, and cost the covering index", is what rev 1 does, except that the costing showed the order is not needed beside the index.

---

# Rev 0 (superseded where rev 1 differs)


# RFC-023: should shape R's list count read in page order?

> **Invariants source.** The project description is empty, so the invariant check is incomplete. Checked against ADR-001 I1 to I9, as A2 and A3 were.
>
> **Numbering.** RFC-023 is assumed: RFC-022 is the highest in `main`. The amendment would be ADR-010-A4.

**What changes:** the query that counts tickets for one kind of region list reads them newest-updated first, so the database uses an index and stops scanning the whole ticket table.
**Why:** where one customer holds about half of all tickets or more, the count can read every ticket of every customer.
**What you'll see when it's done:** the same tickets and the same total. The count's time follows the ticket type's size, not the table's.
**Codes used below:** **R** = ADR-010-A3's shape that reads the ticket type's own records. **|S|** = units in the subtree. **N̂_e** = estimated records of the type. **m** = matching records. **Ratio** = rows in `records` / N̂_e. **Share** = the tenant's part of `records`. **CC** = A2's count. **K\*** = A2's density threshold. Evidence: the Postgres & Data Modeling Specialist's review on TICK-1104 (`r-count-bench.zip`, PostgreSQL 16.10, `nava_app` under FORCE RLS, literals, warm cache, m = 0, |S| = 10,111).

## 1. Summary

- **Recommendation: adopt.** Every count arm whose region leaf is R gets `ORDER BY r.updated_at DESC, r.id DESC` before its own `LIMIT 1001`. No other arm, no outer query and no index changes.
- **The reason is a bound.** Today PostgreSQL may take a Seq Scan and price only the small part it expects to read before 1,001 matches. When it is wrong it reads the whole table, including other tenants' rows. With the order, a Seq Scan needs a Sort, and a Sort must read all its input, so a Seq Scan is priced at its full cost. A3-5 says "R reads at most the type's records". The Seq Scan breaks that sentence, and the order restores it wherever the index is picked.
- **The trade-off:** +3 and +8 ms in two measured multi-tenant cells, and some cells in section 4 that could lose more. Each is bounded by one walk of the type's records, at most about 33,400. The page already makes that walk at m = 0.
- **The benefit is smaller once S21 lands.** ADR-006-A2 A2-4 (PR #827, on `TICK-739/filterable-fields`, not in `main`) skips the count on a first page with at most `limit` rows, for every list. The benchmarked m = 0 cell is that case. After S21 the exposure is first pages with m between 51 and 1,000, and later pages. I still recommend adopting (section 3).
- **What would change this:** section 5.

## 2. Decisions

| # | Decision | Why |
|---|---|---|
| **D1** (item 1) | **Adopt the ordered count.** A covering index stays the next step if D1 fails; it is not built now. | Measured 284 → 95, 92 → 49 and 148 → 50 ms in the three one-tenant cells of the second run. It adds no index and no write cost. |
| **D2** (item 2) | **Exact text, one arm:**<br/>`SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND <count where> ORDER BY r.updated_at DESC, r.id DESC LIMIT 1001) c` | One text. The `DESC` default, `NULLS FIRST`, is the order of `records_list_updated`, as on the page. |
| **D3** (item 2) | **Exact text, `UNION ALL`:** the order is inside each R arm's own parentheses. Arms that are not R keep today's text. The outer `LIMIT 1001` gets no order.<br/>`SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE … AND <R branch> ORDER BY r.updated_at DESC, r.id DESC LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE … AND <other branch> LIMIT 1001)) u LIMIT 1001) c` | An outer order would sort the union and stop the early end across arms. A driven (CC) arm reads by unit, so an order would force a Sort. |
| **D4** | **The order is always `updated_at`, whatever the list sorts by.** "Page order" holds for the default sort only. | One count text per policy and one plan to pin. The total does not depend on order. A list sorted by `created_at` still reads the same heap pages in its page. |
| **D5** | **Contract.** `authz.ListRead` gains `Records []bool`, one per term like `Materialize`, set to `records(k)` in `readList`. `store.ListQuery` gains `CountOrdered []bool`, copied by the handler (`internal/server/records_list.go:483`). `countArm` adds the order only when its entry is true. An arm is never both driven and ordered (unit test). | `authz` knows which term is R and `store` owns the text. Neither inspects the other's SQL. |
| **D6** (item 3) | **Plan test** (section 6). The count pin narrows from "an index of the class, plain or bitmap" to: `Index Scan` on `records_list_updated`, filter `(hashed SubPlan n) IS TRUE`, no `Sort`, no `Gather`, at 10,111 and 20,222 units. A one-tenant world of 420,000 rows (N̂_e = 20,000) is added. | Today's class pin cannot show that the order works. The one-tenant world is a measured cell where today's count is a Seq Scan, so it is red before the change. The specialist's reason for a class, several indexes tied in cost, goes away because one index alone gives the order. |
| **D7** (item 4) | **A2-R4, count part, reworded:** "R's page and count plan with the statement's values (`QueryExecModeExec`; `plan_cache_mode = force_custom_plan` on `nava_app`, `db/grants.yaml`). Under `force_generic_plan` the unordered count was a Seq Scan in four of five measured cells. The ordered count was an index scan in four and a bitmap in one (50 ms against 46). So a lost custom plan no longer gives a Seq Scan in a measured cell. The generic-plan run stays a report, not a pin." The trigger is unchanged: a change to cached or prepared statements re-runs it. | The order removes most of the generic-plan risk, not the reason for the two guards. |
| **D8** (item 5) | **A3-R2 rewritten** (section 7). | The Seq Scan depends on ratio and share, not on one row count. |
| **D9** | **The timing note is named** (section 8). No text of AR-TICK-1040-01 changes. | Security's request on TICK-1104. |
| **D10** | **TICK-1105 gains one cell:** the A3-R2 cell with m = 500 spread over `updated_at`, beside m = 0. | Once S21 is in `main`, A2-4 skips the m = 0 count, and the A3-R2 cell would pass without measuring it. |

**Not reopened:** R's page arm, the R leaf, the selector, α, U_cert, U_hash, the 1,001 cap, the CC, C3 and W0 counts, the A2-5a count.

## 3. Why adopt although A2-4 will skip the m = 0 count

| Request | Count runs | Cost today in a Seq Scan cell | Ordered |
|---|---|---|---|
| First page, m ≤ 50, `main` today | yes | page 44–52 + count 118–126 = 162–178 ms | about 90–105 ms (**inferred**: the m = 0 count makes the page's own walk) |
| First page, m ≤ 50, after S21 | **no** | 0 | 0 |
| First page, 51 ≤ m ≤ 1,000 | yes; it never reaches the cap, so it reads everything | 118–126 ms for the count (**inferred** from m = 0) | 36–50 ms (**inferred**) |
| Later page, m ≤ 1,000 | yes, on every page | the same | the same |
| m ≥ 1,001 | yes, stops at the cap | depends on density | section 4, L2 |

These are SQL times. The folded probe (about 5.5 to 11 ms, A3-5) and the handler come on top. With m = 500, about 125 ms of count leaves almost nothing of 150 ms; about 50 ms leaves room. R is chosen for lists that may hold no match or a few hundred, so this band is ordinary use. ADR-006's count budget (140 ms on the S21 branch) does not help: it covers only lists with a data filter.

## 4. Cells where the ordered count could lose

Only L1 is measured. The others are **inferred**; the bound in each is one walk of the type's records.

| # | Cell | Today | Ordered | Size and bound |
|---|---|---|---|---|
| **L1** | Multi-tenant, m = 0. 620,000 rows, 3 tenants, N̂_e = 20,000. 400,000 × 1 KB, 20 tenants, N̂_e = 33,366 | bitmap: 16 ms, 47 ms | index: 24 ms, 50 ms | **+8 and +3 ms, measured.** The index walk fetches heap pages in `updated_at` order; a bitmap reads each page once in physical order. |
| **L2** | **Large m in a wide subtree, with the matches among the oldest `updated_at`** (a closed region, an imported archive) | `LIMIT 1001` can stop early, since heap order often follows insertion | walks newest first, through all the non-matches | At most the m = 0 walk: 36–50 ms. Two walks, page plus count, are about 90–105 ms. Where matches are spread evenly, both stop after about 1,001 / density records. |
| **L3** | **Low ratio (below about 7), one tenant**, for example a first installation with one main type | a Seq Scan is the cheaper read: about 0.15 µs per empty row, so 33,366 rows ≈ 5 ms plus the hash build | index walk ≈ 36 ms | Up to about +30 ms. The total stays near 90 ms. The crossover is 36 ms against 118 ms at 800,000 rows, so about 240,000 rows. This is the cell where the Seq Scan is right, and the order gives it up. |
| **L4** | **Small share.** If PostgreSQL treats tenant and type as independent, it expects N̂_e × share × 0.5 matches. Below 1,001 (share below about 6 % at N̂_e = 33,366, or 10 % at 20,000) the `LIMIT` gives no discount | bitmap; at large m it stops after 1,001 matches | may become bitmap + top-N Sort, which reads the type's whole slice at any m | At large m it loses the early stop: at most the m = 0 bitmap count, 31–47 ms. The one hint is the 20-tenant generic plan, a bitmap under the ordered text. |
| **L5** | **A selective request filter** (`state`, `assignment_group_id`, `requester_id`, a data filter) that PostgreSQL believes is broad | the filter's own index | the known `ORDER BY … LIMIT` mistake: the ordering index instead of the filter's | At most one walk, 36–50 ms, where the filter's index might take 1 ms. |
| **L6** | **Cold cache** | a bitmap can prefetch heap pages; a Seq Scan reads the table sequentially | up to N̂_e single-page reads | Not measured; every number is warm. The page reads the same pages at m = 0, so the ordered count moves the cold cost between the two statements. At 51 ≤ m ≤ 1,000 it is an added cost. |

**Rule used:** the order is worth these losses as long as none of them takes a list over 150 ms that passes today. By the bounds above, none does. Only a measurement shows it.

## 5. Assumptions not verified, and the result that would change the recommendation

The Technical Architect spikes A1 to A3 in the critique. Times are net medians in one run, ordered against today, on the same cell.

| # | Assumption | Pass | Result that changes the recommendation, and to what |
|---|---|---|---|
| **A1** | **`UNION ALL`.** PostgreSQL plans each R arm alone, so the arm walks `records_list_updated` with no Sort, and later arms are never executed once the outer 1,001 is reached | Each R arm is `Index Scan records_list_updated`, no `Sort`, and the total equals today's | An R arm gets a `Sort` over a Seq Scan, or the branched count is more than 15 ms slower than today: **narrow D3 to unbranched counts.** The total differs: **reject** (not expected; order cannot change a count under the same cap). |
| **A2** | **The real statement through the handler.** `$n` with casts under `QueryExecModeExec` gets the literals' plan | Same plan as the literal text; handler p95 ≤ 150 ms in the 250,000 × 1 KB cell | The Seq Scan stays, or p95 is above 150 ms ordered and not above it today: **reject, cost the covering index.** p95 is above 150 ms either way: **adopt, and cost the covering index**, because the order is then not enough. |
| **A3** | **m > 0.** The cost is bounded by the m = 0 walk. Cells: m = 500 spread; m = 1,500 spread; m = 1,500 all oldest (L2); m = all records of the type | No cell is more than 10 ms above the ordered m = 0 count of the same table, and none passes 150 ms today and fails ordered | A cell breaks the bound, or fails 150 ms only when ordered: **reject, cost the covering index.** A loss below 150 ms is recorded as a residual and does not change the verdict. |
| A4 | In a quiet window the ordered m = 0 count at N̂_e = 33,366 takes at most 80 ms (150 − 52 page − about 18 for probe and handler). 49 to 95 ms were measured in a busy window | ≤ 80 ms | Above 80 ms: **adopt, and cost the covering index.** |
| A5 | L3, L4 and L5 stay within their bounds | As in section 4 | Any of them over 150 ms ordered and under it today: **reject.** L4 shows a Sort over a **Seq Scan**: the cross-tenant note (section 8) comes back, so **Security rules before acceptance.** |
| A6 | The bench Seq Scan was not parallel. The script does not record workers | `Workers Launched` = 0 | With workers, today's times are too good under load. This strengthens adoption. |

A1 to A3 are the three cells the CTO named. A4 to A6 are mine: cheap to read off the same runs, and not required for the critique if the timebox is short.

## 6. Tests

| # | Test | What it catches |
|---|---|---|
| CI-1 | Text (`internal/records/store`): unbranched R; R + requester; two R arms; C3, W0 and CC arms. One `ORDER BY` per R arm, none elsewhere, none outside | An order on a driven arm or on the outer query |
| CI-2 | Text (`internal/authz`): `ListRead.Records` is true exactly for R-driven terms; `Records[i]` and a driver are never both set | D5 |
| CI-3 | Plan, shared world (620,000 rows, as today): D6's pin at 10,111 and 20,222 units | A changed plan |
| CI-4 | Plan, one-tenant world (420,000 rows, N̂_e = 20,000): D6's pin. **Mutant `r_count_unordered`** (the order stripped) must give a Seq Scan | Red before the change; the order being dropped later |
| CI-5 | Plan, branched policy (`under` or requester): the R arm meets D6's pin | A1 regressing |
| CI-6 | Existing agreement suite with R forced, and 1,000 / 1,001 / 1,002 matches through the handler | Rows or totals changing (A2-6, A3-4) |
| CI-7 | Existing five cross-tenant items of A3 under the ordered text | I1 |
| CI-8 | Generic-plan report (D7): logs, never fails | A2-R4 |

**API test scenarios:** no operation, status or body changes. `listRecords` 200 and 500 keep their tests from TICK-1104. The operation is a read: there is no invalid-write or replay case. Cross-tenant is CI-7.

## 7. A3-R2, rewritten (item 5)

| | Text |
|---|---|
| **Risk** | R's count reads outside the type's slice: a Seq Scan, or `records_org_unit`. The unordered count's Seq Scan depends on the **ratio** and the **share**, not on a row count. With one tenant and N̂_e = 33,366: a Seq Scan at ratio 24 and an index at 36 (empty rows); a Seq Scan at 7.5 and an index at 12 (1 KB rows). It also appears at a share of about 54 % and 91 %, and not at 10 % (20 tenants). A4 removes it in each of these cells that was measured ordered (two at N̂_e = 33,366, one at 20,000). |
| **Owner** | Postgres & Data Modeling Specialist |
| **Triggers** | 1. CI-3, CI-4 or CI-5 fails.<br/>2. TICK-1105, either A3-R2 cell (m = 0, m = 500): the count is not an index scan of `records_list_updated`, or handler p95 is above 150 ms, or the count is slower than CC in the same run.<br/>3. Production: record-list p95 above 100 ms while `authz_under_list_shape_total{shape="records"}` is rising (A3-R5's condition).<br/>The fix is then the covering index, costed in a further amendment. |
| **If A4 is rejected** | Trigger 1 becomes today's class pin. Trigger 2 decides. Added: before the first installation where one tenant holds about half of `records` or more and has a type with 1,000 < N̂_e ≤ 33,366 and |S| ≥ ⌈N̂_e/3.3⌉. |

## 8. Security: the Seq Scan timing note against AR-TICK-1040-01

- **Finding** (Security & Multi-Tenancy Reviewer, TICK-1104): while the count takes a Seq Scan, its time also covers **other tenants' rows** in `records`. So a list's time can carry a coarse signal of how much other tenants hold. No row, count or shape choice crosses the tenant wall; RLS and `$1` filter every row read.
- **AR-TICK-1040-01 stays as written.** Its three bits (K\*-side, N̂_e above α·U_cert, the size of S) do not depend on other tenants' rows. This signal is **not** one of them and is not accepted by that row.
- **Under A4:** removed in the cells that move to the index, whose range is `(tenant_id, entity_definition_id)`. **Not provably gone:** PostgreSQL may still pick Seq Scan + Sort where reading the whole table is cheaper than the type's slice (low ratio). The plan choice also uses table-wide statistics, as every query on `records` does.
- **Proposed residual A4-R1:** "Where R's count still takes a Seq Scan, its time also covers other tenants' rows. No row or total is affected." Proposed rating: Info. Owner: Security & Multi-Tenancy Reviewer. Trigger: a Seq Scan in CI-3 to CI-5, in TICK-1105, or in A5.
- **If A4 is rejected**, the signal stays in every Seq Scan cell, and Security decides whether it needs its own row in `docs/security/accepted-risks.md`.
- **Invariants:** I1 holds (no new read, RLS and `$1` unchanged). I5 holds: an order under the same cap cannot change `count(*)`. I8 does not apply (a fixed text, no setting). The others are not touched.

## 9. Options considered

| Option | Why it lost |
|---|---|
| **Reject; keep A3-1's count** | Its worst case grows with the whole table and breaks A3-5's bound. A2-4 hides only m ≤ 50 on a first page. The security note stays. |
| **Covering index** (`records_list_updated` with `INCLUDE (org_unit_id)`) | Probably the fastest count: an index-only scan when the visibility map is fresh. **I don't have the numbers.** It means building and swapping an index on `records`, 16 bytes more per entry, and a benefit that depends on vacuum. D6's pin would have to allow an Index Only Scan. Next step if A2, A3 or A4 asks for it. |
| **`SET LOCAL enable_seqscan = off` around the count** | Keeps the bitmap, so L1 does not lose. But it adds round trips to every R list, must be undone before the page, reaches the other `UNION ALL` arms, and distorts costs instead of changing what the query needs. |
| **Order by the list's own sort column** | Up to six texts and plans to pin for the same total. |
| **Order every arm, or the outer query** | Forces a Sort on driven arms and ends the early stop (D3). |
| **A driver, or a `MATERIALIZED` CTE, for the R count** | A3-1 rejected both. The unit-driven cost is what R removes. |
| **Extended statistics on `(tenant_id, entity_definition_id)`** | Might fix L4, but changes plans across the product. Not a count-arm change. Parked; revisit if L4 fails. |
| **Wait for TICK-1105 to decide** | It would measure the count twice, and after S21 the m = 0 cell measures no count at all. |

## 10. Risks and residuals

| # | Residual | Owner | Trigger |
|---|---|---|---|
| A4-R1 | Section 8: a Seq Scan count also times other tenants' rows | Security & Multi-Tenancy Reviewer | Section 8 |
| A4-R2 | L1 to L5: slower than today in some cells, bounded by one walk of the type's records | Solution Architect | A list that passed 150 ms before A4 fails it with the ordered count |
| A4-R3 | L6: cold-cache cost not measured | Postgres & Data Modeling Specialist | A3-R2 trigger 3, or p95 above 150 ms in the first hour after a database restart |
| A3-R2 | Rewritten in section 7 | Postgres & Data Modeling Specialist | Section 7 |
| A2-R4 | Reworded in D7 | Postgres & Data Modeling Specialist | Unchanged |

## 11. Sequencing

1. TICK-1104 (PR #891) merges as it is. A4 does not block it.
2. A4 is accepted, then implemented as one `core-dev` task against `main`: `acl.go`, `list.go`, `records_list.go`, the tests of section 6. Estimated under 30 lines outside tests.
3. TICK-1105 runs after that, with D10's cell.
4. **Conflict to plan for:** S21 rewrites `store.List` and the handler around it. PR #891 changes `CountStatement`, which S21's branch still has in its older form. Whoever merges second keeps D3's arm text. The Technical Lead orders them.

## 12. Open questions

1. **Postgres & Data Modeling Specialist:** is D6's single-index pin stable on your data at 10,111 and 20,222 units, and is L4's independence estimate right under RLS (does the policy's second `tenant_id` predicate count twice)?
2. **Security & Multi-Tenancy Reviewer:** is A4-R1 at Info, named in the amendment only, enough if A4 is adopted?
3. **Technical Architect:** A1 to A3, as in section 5.
