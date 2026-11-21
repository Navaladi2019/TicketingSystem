---
type: adr-amendment
id: ADR-006-A2
title: "Amendment A2 to ADR-006: the capped count runs under its own 200 ms time budget, and `total` may be unknown (`total_state`)"
status: PROPOSED
date: 2026-11-21
issue: TICK-1022
rfc: none (amendment from spikes; decision request in TICK-1022 comment 01a1fae7-c4a2-7997-b383-d420fedc2224; rev 0/1 critique thread 01a1faed-e8a8-7a44-acd8-c0f660981740; CTO ruling (C2) in comment 01a1fc67-fc69-76ca-ab83-b0cdcc17bf9e)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D7 (what `total` is) and the list response contract; does not change [[ADR-006-A1-d13-fanout-bands-350ms]] budgets"
source: "TICK-1022 runs 1-5 (PR #796) and the A2-11 spikes (PR #812, `64a6160e`, `docs/data/tick-1022-a2-spikes.md`); archived in `docs/adr/ADR-006/spikes/a2-bounded-probe/`"
deciders: Solution Architect (author; verdict); CTO (ruling (C2) with five conditions, 2026-11-21); Technical Architect (rev 2 critique pending); Security & Multi-Tenancy Reviewer (rev 2 isolation critique pending, CTO condition 3)
---

# ADR-006 amendment A2: the count gets a time budget

> **Status: PROPOSED, rev 2** (2026-11-21). Rev 2 replaces rev 0/1's statement-shape design, per the CTO's ruling (C2). Pending: the Technical Architect's critique, and the Security & Multi-Tenancy Reviewer's critique of A2-6 (CTO condition 3). If Security finds an isolation gap, that outranks the ruling and the decision goes back to the CTO.
>
> **Rev 2 changelog.**
> - **Removed:** rev 1's pre-count, the visible-set guard, the 3-field cap, `ListQuery.CountWhere`, the forced probe and R13. D7's side-table readers go back to the single `EXISTS`, and the D7 static test is unchanged.
> - **Added:** a count budget (A2-2, A2-3), the `total_state` contract (A2-1), the first-page shortcut (A2-4), the isolation argument (A2-6) and a new merge gate (A2-8).
>
> **Why rev 1 failed** (A2-11, PR #812). S-A2-skew passed; S-A2-anti and S-A2-multi failed. Two plans are possible for a broad value under a narrowing ACL. The hash join is right when the value sits on records the caller can't see. The per-record probe is right when it sits on the caller's own records. What decides between them is how the value correlates with the visible set, and nothing can know that before doing the work. In the 6% anti-correlated cell the probe added 341-557 ms; no guard limit `K` fixed it, and R13's step measured 230-470 ms. A bounded cost is the only fix that holds for every band and every plan (CTO).
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1-I9, as ADR-006 and A1 are.

## In plain words

A list request returns one page of records plus `total`, the number of matching records the user may read, counted up to 1,000. The page is fast in every case we measured. The count is not. When the user's row ACL hides most records and the filter matches a large share of the type, PostgreSQL can count in a way that reads every matching side row in the tenant, including rows the user can't see. How long that takes depends on vacuum state, on other tenants' edits, and on how the value is spread across records. We tried four ways to make PostgreSQL always pick the fast plan. Each one moved the slowness to a different case.

So we stop trying to choose the plan, and limit how long the count may run instead. The count gets 200 ms. If it finishes, the response says `total_state: exact` or `capped`, as today. If not, the page is still returned, `total` is `null` and `total_state` is `unknown`. The UI then shows "50+" (the matches it already has) rather than a number. A list in this band can no longer take 0.6-1 s because of its count.

## Decisions

| # | Decision |
|---|---|
| **A2-1** | **Contract (CTO condition 1).** `RecordList` gains a required field, `total_state: exact \| capped \| unknown`. `total` stays required but becomes `integer \| null`: an integer for `exact` (0-1,000) and `capped` (1,000), and **`null` for `unknown`, never 0**. A Go zero value can't be mistaken for a real total. `total_capped` stays, true exactly when `total_state = capped`, and is deprecated in favour of `total_state`. The OpenAPI sources (`api/paths/records.yaml`, the bundle), the generated server and SDK types, and the UI ship in the same story. The UI shows `exact` as the number, `capped` as "1,000+", and `unknown` as "N+", where N is the number of matches already listed (page × limit), or "many" when that isn't known. A client must never render `unknown` as 0. |
| **A2-2** | **The budget (CTO condition 2): `CountBudget = 200 ms`**, a named constant in `internal/records/store`, not a tenant setting. Sizing for band (ii) (≥ 40% with a narrowing ACL, limit 350 ms p95): **350 − 94 − 12 − 35 = 209, rounded down to 200 ms.** The 94 ms is the non-count part of the request at p95. It is the list handler's p95 on the idle reference machine, 72.5-73.5 ms (P3-36, quoted in `docs/data/field-index-measured.md`), plus the band (ii) page statement's worst server time, 20.6 ms (run 5; TICK-992 measured 4-14 ms). The 12 ms covers four extra round trips (`SAVEPOINT`, `SET LOCAL`, `RELEASE` or `ROLLBACK TO`, restore) at P3-36's 3 ms transport gate. The 35 ms is the 10% provisional-pass margin of A1-8. Adding p95s overstates the p95 of the sum, so this is conservative. A2-8 (2) measures the 94 ms directly; if it is higher, the formula is rerun, with a floor of 100 ms. If the floor is reached, the decision goes back to the CTO. |
| **A2-3** | **Mechanism (CTO condition 4).** After the page statement, the handler opens `SAVEPOINT list_count` and sets `SET LOCAL statement_timeout` to the time left on one deadline: `CountBudget`, capped by any remaining `DataFilterTimeout`. It then runs the existing capped count statement or statements. With ACL branches, every branch statement shares that one deadline, so the whole count phase is bounded, not each statement. On success it runs `RELEASE` and puts back the earlier `statement_timeout`, which `boundDataFilter` already does. On SQLSTATE `57014` with the request context still live, it runs `ROLLBACK TO SAVEPOINT list_count`, which also undoes the `SET LOCAL`. The handler then answers with the page, `total: null` and `total_state: unknown`. A 57014 caused by request cancellation (`ctx.Err() != nil`) stays an error, as in `timedOut`. A page-statement timeout still returns `filter_too_expensive` as today. The count budget never applies to the page. The savepoint is required because the transaction must stay usable after a cancelled statement. |
| **A2-4** | **First-page shortcut.** If the page statement is on the first page (no keyset) and returns at most `limit` rows (it fetches `limit + 1`), every match is on the page, so `total = len(items)` and `total_state = exact`, with no count statement. This is exact by construction. It removes the count for every short result, including the "no match" and "value only on hidden records" answers, which are now always `exact` (A2-6). |
| **A2-5** | **D7 reads, from this amendment on:** "… `total` is the existing capped count (1,001) of that same statement, run under the count budget (A2). It is `unknown` when that budget runs out, and it is taken from the page when the first page is short. There is no separate count, facet or progress query on a side table. A static test enforces this." The single `EXISTS` shape, the row-ACL compilation, the 1,001 cap and the D7 static test are unchanged. |
| **A2-6** | **Isolation (CTO condition 3, for the Security & Multi-Tenancy Reviewer).** `unknown` is a function of how long the count takes, and that time partly depends on rows the caller can't read. The hash join reads every side row of the value in the tenant. Pooled statistics and the shared visibility map (other tenants' edits) can change the plan. The claim: **this bit reveals no more than the timing channel the response already has, and less in two respects.** (a) It is a threshold of a quantity the caller already observes: the count's duration is part of the request's latency today, and repeated requests average away network noise, so the bit only lowers the attacker's cost of a reading they could already make. (b) The budget caps the timing channel. Above 200 ms, every count looks the same, where today it leaks continuously up to about 1 s. (c) Thanks to A2-4, a value that exists only on hidden records always gives `exact` 0 with no count, so the classic probe ("is there a hidden record with value X?") gets a deterministic answer that depends on no hidden row. `unknown` can only appear when the caller has more than `limit` matches they can read. (d) Load also flips the bit, so it is noisy, as latency is. There is no new statement, no ACL-free read and no new label. If Security finds that (a) does not hold, for instance that the bit is materially cleaner than latency for a cross-tenant observer, that finding outranks the ruling (CTO). |
| **A2-7** | **Observability (CTO condition 4).** `observeIndexedList` gains exactly one label, `total_state`, so its labels are `acl` × `selectivity_band` × `total_state`. No tenant, field, value or count goes in a label, log or span attribute. An alert, owned by the Technical Lead next to A1-4's, fires when more than 5% of data-filter lists in 1 h return `unknown`. That is the signal to look at (b) or the page half. |
| **A2-8** | **Merge gate for the C2 compiler PR**, on top of review and green CI. (1) A2 is ACCEPTED. (2) **Budget check:** on the kit's fresh stack, the non-count p95 of the band (ii) cell (page plus handler, with the count skipped by a test hook) is ≤ 94 ms, or A2-2's formula is rerun with the measured number. (3) **Failure path:** an injected count delay (a test-only hook that adds `pg_sleep` to the count) gives HTTP 200 with the full page, `total: null` and `total_state: unknown`, never a 5xx. The same transaction then runs a further statement successfully. `statement_timeout` is back to its earlier value. Request-context cancellation is still an error. A branched count shares one deadline. (4) **Shortcut:** a short first page runs zero count statements (statement hook), and a full first page runs one. (5) **Carried over from PR #812** (CTO condition 5): no count statement on a refused request (400 masked, 422 backfilling, 400 invalid value, 400 invalid or masked cursor); tenant B's 50k side rows, including rows forged under A's field id, never change tenant A's `total` or items; an ACL-denied broad value gives the same status, items, `total`, `total_state` and body keys as a value that exists nowhere; and the metric-label assertion of A2-7. (6) **Contract:** OpenAPI lint plus a round-trip test showing `unknown` serializes as `"total": null`, and a UI test for "1,000+" and "N+". (7) **TICK-1022's acceptance:** the S21 kit's ≥ 40% ACL cell is ≤ 350 ms p95 on **two** fresh stacks in a row, with `total_state` in `{capped, unknown}`. The kit records the share of `unknown`. |
| **A2-9** | **Budgets and bands.** D13 as amended by A1 doesn't change. A2 makes band (ii) hold by construction, not by plan choice. A 200 ms count budget doesn't help the 150 ms bands, which keep their existing scores. The page half is not covered. The 2% page (TICK-992), and the anti-correlated page walk measured on PR #812 (a6 page 96-335 ms server time, a50 2.2-3.4 s), stay on the R-a track (CTO). |

**Not changed:** RLS, the row-ACL compilation, the masks (D6, R9/CI-2b), the 1,001 cap, the page statement, D13 numbers, R-a, R-b, R8, R9, R11, R12. I1-I4 and I6-I9 aren't touched. There is no grant, no role, no write path and no workflow, and the `SET LOCAL statement_timeout` mechanism already exists (`boundDataFilter`).

## Invariant check

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. No new statement. The count is today's D7 statement under RLS. |
| I2 tenant source | Unchanged. |
| I3 roles | Unchanged. `SET LOCAL statement_timeout` is a user-settable GUC; there is no grant change. |
| I5 one authorization layer | Rows unchanged. `total` is exact, capped or explicitly unknown, never wrong. The timing argument is in A2-6, pending Security. |
| I7 idempotency | Read path only. A retry may see a different `total_state`, which is by design. |
| I8 configuration over code | `CountBudget` is a platform constant, not a tenant knob. |
| I4, I6, I9 | Not touched. |

## New residual

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R13 (rev 2) | Under load or on cold side tables, a list with more than `limit` readable matches may return `total_state: unknown`. The page is complete and `total` is null. | Technical Lead | More than 5% of data-filter lists `unknown` over 1 h (A2-7); or a consumer needs an exact total past the budget (then an async count endpoint is a separate decision) |

Rev 1's R13 (the pre-count timing bit) is withdrawn with the pre-count. It was never accepted.

## Rejected options

| Option | Why |
|---|---|
| Rev 0/1: bounded pre-count + forced probe + visible-set guard | It failed A2-11: anti-correlated +341-557 ms, multi-field +118-252 ms, no working `K`, and R13 at 230-470 ms (PR #812). |
| Visible-sample reshape (Postgres & Data Modeling Specialist; dissent on record) | With a biased sample the walk is unbounded: about 0.4 s at 6% visibility and about 3 s at 50%. That brings back the failure A1-6 already settled (CTO). |
| (C1) D13 exception for band (ii) | The miss depends on vacuum state and shows up at steady state after edits (run 2). It never closes by itself, so it would be a permanent hole, not an exception (CTO). |
| (b) Promoted column per field | Stays on the R-a track. Per-field DDL is too heavy for S21. |
| `total` absent instead of `null` for unknown | An absent field decodes to 0 in Go and most typed SDKs. `null` plus a required `total_state` makes the client handle it. |
| Async or deferred count endpoint | It's a new API surface with polling and a new read path on side tables, which D7 forbids. It is a separate decision if a consumer needs it. |
| ANALYZE or VACUUM at the flip, `random_page_cost`, `work_mem`, probe in the page | As in rev 1 (runs 1-4). |

## Competitive Gate (ADR-008)

**Result:** PASS. X3 (speed) improves: a band (ii) list can no longer exceed 350 ms because of its count. CP-FW-01 ("slow loading at high ticket volume") is reduced. The cost is a less exact `total` under load, which is shown honestly as "N+". No new vendor, cost or component.
