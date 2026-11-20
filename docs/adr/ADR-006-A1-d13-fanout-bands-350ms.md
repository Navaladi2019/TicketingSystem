---
type: adr-amendment
id: ADR-006-A1
title: "Amendment A1 to ADR-006: the D13 350 ms residual (R-a) follows the fan-out, not the row ACL. It covers the 2% point of the 0.5–5% band with or without an ACL, and the 40% band with an ACL"
status: ACCEPTED
date: 2026-11-20
issue: TICK-1001
rfc: none (amendment from a measurement; the decision request is the TICK-1001 description, raised by the Technical Lead after TICK-992 / P7-07b; critique thread under TICK-1001 comment 01a1f5fa-4a70-7571-a5db-b332867cfb39)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D13 (latency targets) and residual R-a"
source: "TICK-992 (P7-07b), PR #775: `docs/data/field-index-measured.md` § TICK-992, `tick992-lists.*`, `tick992-explain.md`; archived in `docs/adr/ADR-006/spikes/a1-d13-bands/`"
deciders: Solution Architect (author; verdict); Technical Architect (round-1 critique 2026-11-19, comment 01a1f5fa: one blocking item F1, two verify-by-spike items F2/F3, three non-blocking, three confirmed; "with items 1 and 2 done, I have no objection to ACCEPTED"); CTO (no tie-break needed: nothing contested); Technical Lead (requester; position (c) now, (a) trigger-gated)
---

# ADR-006 amendment A1: the 350 ms residual follows the fan-out, not the ACL

> **Status: ACCEPTED, rev 1** (2026-11-20). The Solution Architect gave the verdict after the Technical Architect's round-1 critique. Every finding was taken as proposed, so nothing was contested and no CTO tie-break was needed. The two "must" items were the conditions of the Technical Architect's approval. Item 1 (F1) uses the Technical Architect's own A1-4 text. Item 2 (F2) narrows band (i) to what was measured, one of the two wordings the critique accepted.
>
> **Rev 1 changelog (round-1 critique).**
> - **F1 (blocking):** A1-4 and the R-a trigger are replaced. The trigger now uses a share-above-350 ms alert plus a per-tenant slow-list log line. `count_capped` is removed: it missed the 2%+ACL band, whose count is 487, and included the 40% no-ACL band, which is budgeted at 150 ms.
> - **F2:** band (i) is accepted at 350 ms **only up to the measured 2% point**. Above 2% it is unmeasured until spike S1.
> - **F3:** A1-2 no longer says the ACL causes nothing. With an ACL, the 2% page fans out over the ACL-visible set. Band (i) with an ACL is accepted only up to the measured 6% visibility, until spike S2.
> - **F4:** 4,910 is where the count scan stops, not the size of the visible set (24,118). A1-6 gains the critique's scope additions.
> - **F5:** `work_mem = 4 MB` and about 160k side rows per value become conditions of band (ii).
> - **F6:** the ÷1.3 deflation is no longer used as evidence. 317.6 and 327.6 ms are provisional passes.
> - **F9:** ADR-008 X3 now names both bands.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006 is.

## In plain words

Take a tenant with 400,000 records. A list page that filters on a data field matching about 2% of them takes 237–268 ms at p95 with 8 clients. p95 is the time 95% of requests beat. ADR-006 allows 150 ms for that, and allows 350 ms only when the user also has a row ACL that hides records. The measurement shows that the ACL is not what makes the request slow. To return the newest 51 matches, the database reads every matching record (about 8,000) and sorts them. That costs the same with or without an ACL. A broad filter (40% of records) is slow for a user with an ACL for a related reason: counting the matches the user may see means loading all 160,000 matches of the value.

So this amendment moves the 350 ms line to those two measured cases: a 2% filter for any user, and a 40% filter for a user with an ACL. Every other band keeps 150 ms. Filters matching 2–40% have not been measured. Two spikes measure them before anyone claims a number for them. Nothing about isolation or correctness changes. This is a performance budget only, with an alert that tells us when to build the faster query.

## What was measured

TICK-992 scored all 40 D13 cells on the reference machine after F1–F3 (lists at `ea1b1785`, 2026-11-19). The fixture was an 800k seed with a 400k type, 8 clients and 20 s per cell, on PostgreSQL 16.10 with `work_mem = 4 MB`. The ACL agent's groups see 24,118 records, 6% of the type. The owner's idle `nava` stack was still running, under the Technical Lead's waiver. Host contention only adds latency, so each cell is an upper bound for the reference host. No deflation factor is applied anywhere in this amendment.

| Band | ACL | p95, 5 operators | Single-client server time | Cause (from `EXPLAIN (ANALYZE)`) |
|---|---|---|---|---|
| 2% | none | **236.6–267.6 ms** (miss of 150) | 84–97 ms | The page drives from the side index: 8,098 ids, 8,098 `records_pkey` lookups (32,000 buffer hits), then a top-N sort for 51 rows |
| 2% | narrowing (6% visible) | 223.1–317.6 ms (inside 350) | — | The page bitmap-scans the 24,118 ACL-visible records (37 of 49 ms) and hash-joins them to the 8,098 side rows. The count is 487, below the cap |
| 40% | narrowing (6% visible) | **176.1–327.6 ms** (miss of 150) | 42–102 ms | The capped count hash-semi-joins the group-visible records against **all 160,258** side rows of the value (82 ms; the hash spills to 2 batches at 4 MB). The scan stops at 4,910 records when it reaches the 1,001 cap. The page itself is 4–14 ms |
| 0.25% `in` | both | **160.9 / 152.8 ms** (miss of 150 by 2–7%) | 29–35 ms | No fan-out in the plan |
| everything else (26 cells) | both | pass by more than 10% | — | — |
| 5%, 10%, 20% | both | **not measured** | — | — |

## Decisions

| # | Decision |
|---|---|
| **A1-1** | **D13 reads, from this amendment on:** "**Latency targets** (CTO ruling 2, as amended by A1): list p95 ≤ 150 ms for every band, except two that are accepted at **p95 ≤ 350 ms**. (i) **Mid selectivity, 0.5–2% of the type, with or without a narrowing row ACL.** With an ACL, this is accepted for an agent whose groups see up to 6% of the type (the measured breadth). (ii) **Broad selectivity (≥ 40%) with a narrowing row ACL**, at `work_mem = 4 MB` and up to about 160k side rows per value (the measured conditions). Matches above 2% and below 40% are **unmeasured**. They keep 150 ms as the target, and they are scored as unmeasured, not as passing, until spike S1 (A1-7). D7's single `EXISTS` shape is kept. An adaptive shape chosen from a planner estimate is the planned follow-up (trigger in R-a)." The 150 ms target is unchanged for 40% without an ACL, 0.25% and 0.01%. The 0.5% lower edge is unchanged from ADR-006. Below 2%, the fan-out is smaller than at the measured point, so the measured cost bounds it. |
| **A1-2** | **What draws the line is the plan shape, not the actor.** A band gets 350 ms only where `EXPLAIN` shows a fan-out that the D7 shape cannot avoid. The fan-out must be proportional to a row set: either the **matches** (the 2% page without an ACL reads every match to sort it, and the 40% count with an ACL hashes every side row of the value) or the **ACL-visible set** (the 2% page with an ACL scans it). The old condition, "with an ACL", stood in for this and was the wrong proxy. A band is extended, in selectivity or in ACL breadth, only on the same evidence: a measured cell with its plan. A latency number alone, or an extrapolation, is never enough. |
| **A1-3** | **R-a is rewritten** (table below). The owner, the triggers and the follow-up stay. The scope covers both bands under their measured conditions. The count-only clause is dropped: the 40%+ACL page now lands inside 350 ms including its count. The count was 538–1,255 ms when R-a was accepted, and is 32–94 ms single-client after F1–F3. |
| **A1-4** | **The revisit trigger can be observed** (TA text, F1). One histogram, `nava_records_list_data_filter_seconds`, is recorded only for lists with at least one data filter. It has one label, `row_acl` (`true`/`false`), and no tenant, entity, field or value label. The alert fires when **more than 1% of data-filter list requests take longer than 350 ms over 1 h** (the ratio of the `le="0.35"` bucket). It uses a ratio rather than p95, so fast traffic cannot hide one tenant's slow band. Every data-filter list that takes longer than 350 ms also writes one structured log line with `tenant_id`, `entity_definition_id`, the filter operator, `row_acl`, the count value (≤ 1,001) and the duration. The line carries **no filter value or field name**; the field id is not needed to find the tenant. The revisit trigger is: the alert fires, **and** the log lines show one tenant whose own p95 is above 350 ms over 1 h. The Technical Lead owns this as a follow-up task. It does not gate A1. Until it ships, the access log is checked by hand at each S21 follow-up review. |
| **A1-5** | **The 0.25% `in` cells are not re-budgeted.** They miss by 2–7%, with 29–35 ms of server time and no fan-out in the plan. A1-2 does not allow moving them to 350 ms. They are recorded as **misses on a contended host**. The Technical Lead decides whether TICK-749 reruns these two cells on a fully idle host. If a rerun on an idle host still misses 150 ms, that is a new finding and needs a new decision. A1 does not cover it. |
| **A1-6** | **Option (a), the adaptive shape, stays the trigger-gated follow-up. Its spike scope is fixed now.** When R-a triggers, a spike measures: (1) the page: drive from the side index when the estimate is selective, otherwise walk `records_list_updated` and probe the side table by primary key (about 51 / 0.02 = 2,550 probes for a 2% page, if the value is independent of `updated_at`); (2) the count with an ACL: probe `rfv_*_pkey` `(tenant_id, record_id, field_id)` for each group-visible record, until the cap is reached, instead of hashing every side row of the value. The planner priced this probe and chose the hash, so **the compiler must emit a shape that forces the probe**, with no `enable_*` and no hints; (3) adversarial fixtures: a filter value anti-correlated with `updated_at` (the 2% matches are the oldest records), and a narrow ACL whose count does not reach the cap. Exit criterion: both bands ≤ 150 ms p95 on the reference machine, fully idle, in every fixture. RLS, the leakproof contract (CI-2) and the D7 `EXISTS` semantics must not change. If the walk cannot be bounded, (a) falls back to (b). |
| **A1-7** | **Two spikes settle the unmeasured edges** (TA S1, S2). Both use the TICK-992 harness and fixture: 800k seed, 400k type, 8 clients, 20 s per cell, `EXPLAIN (ANALYZE, BUFFERS)` per cell. The Technical Lead schedules them in TICK-1002 or TICK-749. They do not gate A1 or S21. **S1, band edge:** 5%, 10% and 20% for `num lt` and `text eq`, with and without an ACL. A 5% cell ≤ 350 ms extends band (i) up to that point under A1-2. A 10% or 20% cell over 150 ms is a new band and needs its own plan evidence under A1-2, through a new amendment. **S2, ACL breadth:** 2% with the agent's groups seeing 6%, 25% and 50% of the type. ≤ 350 ms at every breadth removes the 6% condition. A miss adds a visible-set-size trigger to R-a, or caps band (i)+ACL at the highest breadth that passes. |
| **A1-8** | **Scoring under TICK-992's rule** (TA F6). A cell that passes by 10% or less is provisional. So 2%+ACL `text eq` 317.6 ms (9.3% inside) and 40%+ACL `text in` 327.6 ms (6.4% inside) are **provisional passes** against 350. The rest of both bands passes by more than 10%. TICK-1002 rescores against the A1 lines now. No finding in the critique changes the 2% or 40% numbers. |

**Not changed:** D7 (one `EXISTS` on the typed side table), RLS, the row-ACL compilation, the 1,001 count cap, R-b, R8, R9, R11 and R12. I1–I9 hold as in ADR-006: this changes a number in a budget, not a query, a grant or a boundary. The new metric carries no tenant data. The slow-list log line carries the tenant id and no record or filter value. That is the same exposure as the access log.

## R-a, as amended

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-a | On large types, two D13 bands run at p95 ≤ 350 ms instead of 150 ms. **0.5–2% with or without a narrowing ACL**: measured 223–318 ms with an ACL (the agent sees 6% of the type) and 237–268 ms without one. **≥ 40% with a narrowing ACL**: measured 176–328 ms, at `work_mem = 4 MB` and about 160k side rows per value. Most of that is the capped count's hash join. Both come from a fan-out proportional to the matches or to the ACL-visible set (A1-2). 2–40% and broader ACLs are unmeasured until S1 and S2 (A1-7). Performance only, no isolation effect | Technical Lead | More than 1% of `nava_records_list_data_filter_seconds` requests are over 350 ms for 1 h, confirmed per tenant from the slow-list log (A1-4); or a tenant passes 400k records; or S1/S2 misses → run the A1-6 spike, then build the adaptive shape |

## Competitive Gate (ADR-008)

**Result:** PASS with accepted risk CP-FW-01. The kind of risk is unchanged; its scope is wider. X3 (speed) keeps "list ≤ 150 ms", and the D13 exceptions are now named as two bands. The pain point this touches is CP-FW-01, "slow loading at high ticket volume". A 2% filter on a 400k-record type at ≤ 350 ms p95 is still interactive. The trigger to fix it is tied to a real tenant reaching that size. ADR-008's X3 row and ADR-006 row are updated in the same change.

## Response — round 1

| TA item | Label | Resolution |
|---|---|---|
| F1: `count_capped` misses 2%+ACL (count 487) and includes 40% no-ACL | blocking | **Accepted as proposed.** A1-4 and the R-a trigger now use the Technical Architect's text. I added one constraint: the slow-list log line carries no filter value and no field name. |
| F2: the 5% edge is unmeasured; extrapolation gives about 590–670 ms | verify-by-spike | **Accepted. Band (i) is narrowed to 0.5–2%.** 2–40% is scored as unmeasured. S1 is A1-7, and A1-2 says how a passing S1 cell extends the band. |
| F3: with an ACL, the 2% page fans out over the visible set (24,118 rows = 6%) | verify-by-spike | **Accepted.** A1-2 is reworded. Band (i)+ACL is accepted up to 6% visibility. S2 is A1-7. |
| F4: 4,910 is where the scan stops; the probe must be forced; correlation risk | non-blocking | **Accepted.** Taken into A1-6 (2) and (3). |
| F5: band (ii) depends on `work_mem` and the side-row count | non-blocking | **Accepted.** Added to A1-1, R-a and the budget row. |
| F6: no rerun is needed to set the line; drop ÷1.3; 317.6 and 327.6 stay provisional | confirmed | **Accepted.** The deflation is removed from this record, from the explainer and from the rejected options. A1-8 sets the scoring. |
| F7, F8 | confirmed | No change. |
| F9: ADR-008 X3 misses column | non-blocking | **Accepted.** It now names both bands. |

## Rejected

| Option | Why |
|---|---|
| (b) A promoted column per field that needs the band | Single-digit ms (the P3-01 (d) ordered walk). But it needs per-field DDL on `records`, breaks the schema-less goal for those fields, and is expensive to reverse once data lands. It stays available if the A1-6 spike fails. |
| Keep 150 ms and block S21 until (a) ships | (a) is unmeasured. The 2,550-probe figure is an estimate that assumes no correlation with `updated_at` (F4), and a bad estimate could pick the wrong shape (R12 shows how estimates fail). Holding a story on an unspiked design trades a certain delay for an uncertain fix. The band is 24–32% inside a limit already accepted for the same plan shape. |
| Keep the ACL condition and rerun the 2% no-ACL cells on a fully idle host | Single-client server time is already 84–97 ms (56–65% of 150) before any concurrency, and the 8-client p95 is 2.4–2.8× that. Nothing measured suggests an idle host gets this band under 150 ms. A rerun can only lower the numbers, and it would not change where the line sits. |
| Keep band (i) at 0.5–5% as in ADR-006 | Only 2% was measured. A1-2's own rule predicts that cost grows with matches, and an extrapolation gives about 590–670 ms at 5% (F2). Claiming an edge we expect to fail is not a budget. |
| `count_capped` as the trigger label (rev 0) | It misses the 2%+ACL band and includes the 40% no-ACL band (F1). |
| Raise every band to 350 ms | It would hide regressions in the selective bands, which run at 48–161 ms today. A1-2 ties each exception to a plan shape. |
| Accept the 40%+ACL band as a count-only miss again (R-a as written) | The page takes 4–14 ms; the count is the cost, and the user waits for both. D13 budgets the request, not the statement. |

## Evidence

- Raw lists, host and contention: `docs/adr/ADR-006/spikes/a1-d13-bands/tick992-lists-inserts.md`, `tick992-lists.host`, `tick992-nava-stats.txt` (copied from PR #775).
- Plans for the band (ii) count and the band (i) no-ACL page: `docs/adr/ADR-006/spikes/a1-d13-bands/plans-excerpt.md` (num `lt`):
  - the 40%+ACL count takes 87.5 ms, a `Hash Semi Join` over 160,258 side rows with `Batches: 2`;
  - the 2% no-ACL page takes 67.0 ms, with 8,098 `records_pkey` lookups and a top-N heapsort.
- The 2%+ACL plans (count 487; page bitmap-scanning 24,118 visible records) are in `tick992-explain.md` lines 3219–3509 (PR #775). They were cited by the Technical Architect's critique.
- Full write-up: PR #775, `docs/data/field-index-measured.md` § TICK-992, and `tick992-explain.md` (all 40 cells).
- Technical Architect round-1 critique: TICK-1001 comment 01a1f5fa-4a70-7571-a5db-b332867cfb39.
