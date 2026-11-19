---
type: adr-amendment
id: ADR-006-A1
title: "Amendment A1 to ADR-006: the D13 350 ms residual (R-a) is set by the side-index fan-out, not by the row ACL. It covers the 0.5–5% band with or without an ACL, and the 40% band with an ACL"
status: PROPOSED
date: 2026-11-20
issue: TICK-1001
rfc: none (amendment from a measurement; the decision request is the TICK-1001 description, raised by the Technical Lead after TICK-992 / P7-07b)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D13 (latency targets) and residual R-a"
source: "TICK-992 (P7-07b), PR #775: `docs/data/field-index-measured.md` § TICK-992, `tick992-lists.*`, `tick992-explain.md`; archived in `docs/adr/ADR-006/spikes/a1-d13-bands/`"
deciders: Solution Architect (author; verdict after critique); Technical Architect (critique pending); CTO (tie-break if contested); Technical Lead (requester; position (c) now, (a) trigger-gated)
---

# ADR-006 amendment A1: the 350 ms residual follows the fan-out, not the ACL

> **Status: PROPOSED** (2026-11-20). Awaiting the Technical Architect critique. The Solution Architect gives the verdict after it (and after a CTO tie-break if anything is contested).
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006 is.

## In plain words

On a tenant with 400,000 records, a list page that filters on a data field matching about 2% of the records takes 237–268 ms at p95 with 8 clients. ADR-006 allows 150 ms for that, and 350 ms only when the user also has a row ACL that hides records. The measurement shows the ACL is not the cause. The database reads every matching record (about 8,000) to sort them and return the newest 51. That costs the same with or without an ACL. So the line ADR-006 drew ("350 ms only with an ACL") is in the wrong place. This amendment moves it: the 350 ms limit applies to the 2% band whoever asks, and also to the 40% band when an ACL is present, where the matching-row count is the slow part. All other bands keep 150 ms. Nothing about isolation or correctness changes; this is a performance budget only.

## What was measured

TICK-992 scored all 40 D13 cells on the reference machine after F1–F3 (lists at `ea1b1785`, 2026-11-19). The owner's idle `nava` stack was still running (waived by the Technical Lead), so read each cell as up to about 1.3× inflated: the no-filter control read 86–99 ms against 73 ms on a fully idle host.

| Band | ACL | p95, 5 operators | Single-client server time | Cause (from `EXPLAIN (ANALYZE)`) |
|---|---|---|---|---|
| 2% | none | **236.6–267.6 ms** (miss of 150) | 84–97 ms | The page drives from the side index: ~8,000 ids, ~8,000 `records_pkey` lookups (32,000 buffer hits), then a top-N sort for 51 rows |
| 2% | narrowing | 223.1–317.6 ms (inside 350) | — | Same fan-out |
| 40% | narrowing | **176.1–327.6 ms** (miss of 150) | 42–102 ms | The capped count hash-semi-joins the 4,910 group-visible records against **all 160,258** side rows of the value (82 ms, hash spills to 2 batches). The page itself is 4–14 ms |
| 0.25% `in` | both | **160.9 / 152.8 ms** (miss of 150 by 2–7%) | 29–35 ms | No structural cause; inside the host's inflation |
| everything else (26 cells) | both | pass by more than 10% | — | — |

Even after taking off the full 1.3× host inflation, the 2% no-ACL band reads 182–206 ms. It is a real miss of 150 ms, and it is inside 350 ms. With the inflation removed, the worst 40%+ACL cell reads about 252 ms.

## Decisions

| # | Decision |
|---|---|
| **A1-1** | **D13 reads, from this amendment on:** "**Latency targets** (CTO ruling 2, as amended by A1): list p95 ≤ 150 ms for every band, except two that are accepted at **p95 ≤ 350 ms**: (i) **mid selectivity (0.5–5% of the type), with or without a narrowing row ACL**, and (ii) **broad selectivity (≥ 40%) with a narrowing row ACL**. D7's single `EXISTS` shape is kept. An adaptive shape chosen from a planner estimate is the planned follow-up (trigger in R-a)." The 150 ms target is unchanged for 40% without an ACL, 0.25% and 0.01%. |
| **A1-2** | **The reason for the line is the plan shape, not the actor.** A band gets 350 ms only where `EXPLAIN` shows a fan-out the D7 shape cannot avoid: the page reading every match to sort it (mid band), or the capped count joining every match before it can stop (broad band with ACL). The ACL was a proxy for this and the measurement shows it was the wrong one. A future band is added to the 350 ms list only with the same evidence (a plan showing a fan-out proportional to the match count), never on a latency number alone. |
| **A1-3** | **R-a is rewritten** (table below). Owner, triggers and the follow-up stay; the scope widens to both bands, and the count-only clause is dropped because the 40%+ACL page now lands inside 350 ms *including* its count (the count was 538–1,255 ms when R-a was accepted, and is 32–94 ms single-client after F1–F3). |
| **A1-4** | **The revisit trigger must be observable.** Today the only list latency signal is `otelhttp`'s per-route histogram, which cannot tell the 2% band from any other list. The Technical Lead creates a task for one histogram, `nava_records_list_data_filter_seconds`, recorded only for lists with at least one data filter, with exactly two labels: `row_acl` (`true`/`false`) and `count_capped` (`true` when the count reached the 1,001 cap). Both bands A1 re-budgets are the capped-count lists on a large type, so `count_capped="true"` is the signal to watch; the alert is its p95 > 350 ms over one hour. No tenant id, entity, field or value is a label (cardinality and I5); the slow request's tenant is found from the access log. Until it ships, the trigger is checked by hand from the access log at each S21 follow-up review. It is a follow-up, not a gate for A1's ACCEPTED. |
| **A1-5** | **The 0.25% `in` cells are not re-budgeted.** They miss by 2–7% with 29–35 ms of server time: a quarter of the budget, with no fan-out in the plan. A1-2 forbids moving them to 350 ms on a latency number. They are recorded here as **host-inflated misses** (÷1.3 gives 124 / 118 ms). The Technical Lead decides whether TICK-749 re-runs these two cells on a fully idle host; if a re-run on an idle host still misses 150 ms, that is a new finding for a new amendment, not covered by A1. |
| **A1-6** | **Option (a), the adaptive shape, stays the trigger-gated follow-up and its spike scope is fixed now.** When R-a triggers, a spike measures both halves: (1) the page: drive from the side index when the estimate is selective, otherwise walk `records_list_updated` and probe the side table by primary key (about 51 / 0.02 = 2,550 probes for a 2% page against ~8,000 heap fetches today); (2) the count with an ACL: probe the side table's primary key for each group-visible record (4,910 here) instead of hashing every side row of the value. Exit criterion: both bands ≤ 150 ms p95 on the reference machine, fully idle, with no change to RLS, the leakproof contract (CI-2) or the D7 `EXISTS` semantics. |

**Not changed:** D7 (one `EXISTS` on the typed side table), RLS, the row-ACL compilation, the 1,001 count cap, R-b, R8, R9, R11, R12. I1–I9 hold as in ADR-006: this changes a number in a budget, not a query, a grant or a boundary. No tenant data enters the new metric.

## R-a, as amended

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-a | Two D13 bands run at p95 ≤ 350 ms, not 150 ms, on large types: **0.5–5% with or without a narrowing ACL** (measured 223–318 ms with, 237–268 ms without), and **≥ 40% with a narrowing ACL** (measured 176–328 ms; the capped count's hash join is most of it). Both come from a fan-out proportional to the match count (A1-2). Performance only, no isolation effect | Technical Lead | `nava_records_list_data_filter_seconds{count_capped="true"}` p95 > 350 ms over 1 h for a real tenant (A1-4), or a tenant passes 400k records → run the A1-6 spike, then build the adaptive shape |

## Competitive Gate (ADR-008)

**Result:** PASS with accepted risk CP-FW-01, unchanged in kind, wider in scope. X3 (speed) keeps "list ≤ 150 ms" with the D13 exceptions now named as two bands. CP-FW-01 ("slow loading at high ticket volume") is the pain point this touches. A 2% filter on a 400k-record type at ≤ 350 ms p95 is still interactive, and the trigger to fix it is tied to a real tenant reaching that size. ADR-008's X3 and ADR-006 rows are updated in the same change.

## Rejected

| Option | Why |
|---|---|
| (b) A promoted column per field that needs the band | Single-digit ms (the P3-01 (d) ordered walk), but it means per-field DDL on `records`, breaks the schema-less goal for those fields, and is expensive to reverse once data lands. It stays available if the A1-6 spike fails. |
| Keep 150 ms and block S21 until (a) ships | (a) is unmeasured: the 2,550-probe figure is an estimate, and a broad filter could pick the wrong shape from a bad estimate (R12 already shows how estimates fail on rare values). Holding a story on an unspiked design trades certain delay for an uncertain fix, for a band that is 24–32% inside a limit we already accepted for the same plan shape. |
| Keep the ACL condition and re-run the 2% no-ACL cells on a fully idle host | Does not change the answer: with the whole 1.3× inflation removed, the band reads 182–206 ms, still a miss of 150. |
| Raise every band to 350 ms | Hides regressions in the selective bands, which run at 48–161 ms today. A1-2 keeps each exception tied to a plan shape. |
| Accept the 40%+ACL band as a count-only miss again (R-a as written) | The page is 4–14 ms; it is the count that costs, and the user waits for both. Budgeting the request, not the statement, is what D13 measures. |

## Evidence

- Raw lists, host and contention: `docs/adr/ADR-006/spikes/a1-d13-bands/tick992-lists-inserts.md`, `tick992-lists.host`, `tick992-nava-stats.txt` (copied from PR #775).
- Plans for the two re-budgeted bands: `docs/adr/ADR-006/spikes/a1-d13-bands/plans-excerpt.md` (num `lt`: the 40%+ACL count, 87.5 ms, `Hash Semi Join` over 160,258 side rows with `Batches: 2`; the 2% no-ACL page, 67.0 ms, 8,098 `records_pkey` lookups and a top-N heapsort).
- Full write-up: PR #775, `docs/data/field-index-measured.md` § TICK-992, and `tick992-explain.md` (all 40 cells).
- TICK-992 scoring rule (Technical Lead): a cell is final if it passes by more than 10%; otherwise provisional. A1 sets the budgets those cells are scored against; it does not re-score them.
