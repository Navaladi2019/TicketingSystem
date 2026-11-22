# SP-1f (TICK-1092) report, archived from TICK-946 comment 01a20320 (Technical Architect, 2026-11-22)

## SP-1f (TICK-1092) spike report: PASS. Records-driven rendering brings `bench_conc` to ≤ 75.85 ms p95 at 9,099 and 10,111 units, α = 3.3 measured, A2-6 holds. The same-window control also passed, so most of TICK-1037's 161.57 was host noise

[@Solution Architect](mention://agent/021bfc50-c700-496e-a4f4-f30e60a9ba8d): this is the input for ADR-010-A3, as contracted. The contract numbers are below, followed by two findings A3 should take on board.

**Verdict: PASS on every contract line.**

| Contract line | Result |
| --- | --- |
| `bench_conc` ≤ 150 ms p95, both personas, 9,099 **and** 10,111 units | **Pass.** R: 67.66 / 64.40 ms (agents / count-only) at 9,099, and 36.43 / 75.85 ms at 10,111 |
| Every other TICK-1037 row within 5 ms of its TICK-1037 value | **Pass.** Every selector row is below its TICK-1037 p95. The largest is uniform m = 3,000 count-only at 64.86 against 86.83. See finding 1 for the same-window view, which is tighter. |
| Same ids and capped count as the reference (A2-6) | **Pass.** 0 mismatches across 201 callers (200 agents and the count-only caller) × 17 R sets, against the A2 control. The pipeline `TestUnderList*` tests with "R forced" added also pass against `Allows`. |
| Paired wall-clock behind the floor gate | **All 68 rows scored on attempt 1.** Floor p50 was 0.53–2.85 ms against the 3.0 ms gate, and no attempt was voided. No foreign container appeared. |
| α measured at the crossover | **α ≈ 3.3** (count-only, the binding persona; agents 3.5). That is above 2.2, so the selector picks R at 9,099 (N̂_e/\|S\| = 2.20) and at 10,111 (1.98). |

### Setup
- **Code:** story head `700259e9`, with TICK-1037's harness (PR #846) plus the throwaway SP-1f diff (attached, `git apply` on `700259e9`). It is not committed.
- **Data:** the 800k seed and the TICK-1037 region sets, plus `bench_conc` at root and the sweep sets (regions 0..k−1, m = 0). N̂_e: ticket 400,000 (K\* 3,613), `bench_rare` 500, `bench_conc` 20,000 (K\* 1,000).
- **Environment:** `postgres:16.10`, P3-01's settings. The Docker VM had 16 CPUs and 26 GB. Run alone, 10:27–11:57Z, `NAVA_PERF_SECONDS=60`, 1 client, paired with the plain agent request.
- **Order:** MC3 first (A2 unchanged on all 13 sets, which is the control), then the α sweep, then the selector at α = 8 (provisional) on all 13 sets.
- **Cleanup:** the bench database and container are removed. Only `multica-*` containers are up.

### The mechanism (the SQL A3 would adopt)
- **R leaf, page and count:** `(r.org_unit_id IS NOT NULL AND (r.org_unit_id IN (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $t AND ancestor_id = ANY($U))) IS TRUE)`, with no closure driver and no MATERIALIZED CTE.
  - `IS TRUE` blocks the sublink pull-up. Every R plan shows `(hashed SubPlan 1) IS TRUE`, building the 9,099-row closure set once in 2.8–3.2 ms.
- **Folded probe, one round trip:** `SELECT n, CASE WHEN n >= $cap THEN NULL ELSE (P3) END FROM (SELECT count(*) n FROM (<canonical-U closure> LIMIT $cap))`, with `$cap = ⌈N̂_e/α⌉`. R is chosen iff n reaches $cap.
  - **P3 shows `(never executed)` on every R path.**
  - The cap is N̂_e/α by construction, so the CTO's condition 3 holds: the count cannot fall back to the unit-driven shape at the sizes R exists for.

### Cells: `bench_conc`, handler p95 / p50 in ms, paired base p95 beside it
| \|S\| (N̂_e/\|S\|) | A2 control, agents | R, agents | A2 control, count-only | R, count-only |
| --- | --- | --- | --- | --- |
| **9,099** (2.20), m = 0 | 79.58 / 60.86 (base 26.06) | **67.66** / 45.13 (base 22.58) | 99.26 / 74.84 (base 26.20) | **64.40** / 51.49 (base 19.77) |
| **10,111** (1.98), m = 20,000 | 41.86 / 27.26 (base 26.91) | **36.43** / 27.17 (base 21.82) | 46.52 / 30.27 (base 28.70) | **75.85** / 43.61 (base 42.95, floor after 2.85) |

**Best R cell at 10,111 units: 36.43 ms (agents). The worst is 75.85 ms (count-only).**

### α sweep: `bench_conc`, m = 0. Net median = p50 − paired base p50, in ms
| \|S\| | N̂_e/\|S\| | A2 agents | R agents | A2 count-only | R count-only |
| --- | --- | --- | --- | --- | --- |
| 1,011 | 19.8 | 15.2 | 29.2 | 9.9 | 42.2 |
| 2,022 | 9.9 | 16.3 | 30.3 | 14.5 | 36.7 |
| 3,033 | 6.6 | 24.8 | 51.9 ¹ | 23.3 | 57.4 ¹ |
| 4,044 | 4.9 | 29.9 | 35.0 | 28.9 | 41.6 |
| 9,099 | 2.2 | 42.7 | 29.3 | 57.4 | 36.5 |

¹ A noisy pair: floor up to 2.29 ms and base p50 up to 29.6 ms. It is left out of the fit.

- **Known:** R is flat in \|S\|, at about 30 ms (agents) and 39 ms (count-only), because it reads the type's 20,000 records whatever the subtree. A2 grows linearly: 3.4 µs per unit for agents and 5.9 µs per unit for count-only.
- **Crossover:** the lines cross at \|S\| ≈ 6,000 (count-only) and ≈ 5,640 (agents). That is **α = 20,000 / 6,000 ≈ 3.3**, bracketed by the measured points at 4.9 (A2 cheaper) and 2.2 (R cheaper).
- **Inferred:** with α = 3.3 instead of the provisional 8, every row makes the same choice, because 3.3 lies in (2.2, 39.6). The `bench_conc` cap rises from 2,500 to 6,061 closure rows, which adds about 1–2 ms to the probe. That is not measured.

### Plan stability: `EXPLAIN (ANALYZE, BUFFERS)`, n_distinct(org_unit_id) analyzed (3,311) and forced 100
| Statement (R) | Access path at both n_distinct values | Execution time (analyzed / 100) |
| --- | --- | --- |
| Page, R arm | `Index Scan using records_list_updated`, filter `(hashed SubPlan 1) IS TRUE` | 9,099: 47.1 / 18.3 ms (count-only), 47.9 / 22.1 ms (agent) |
| Count, R arm | `Index Scan using records_org_unit`, Index Cond `(tenant_id, entity_definition_id)`, same filter | 9,099 count-only: 61.2 / 17.7 ms. For the agent, the arm is `never executed` (earlier branches fill the cap) |
| Folded probe | cap reached, P3 `never executed` | 9,099: 3.3 / 1.2 ms; 10,111 `bench_conc`: 2.0 / 1.2 ms |
| Folded probe, ticket at root (no R) | closure count to 10,111, then P3 | 8.5 / 7.0 ms, of which P3 is 2.9 |

The plan holds at both n_distinct values, and **no MATERIALIZED CTE is needed.** The analyzed and forced runs differ in time by up to 3.5×, with identical plans and every buffer a shared hit. That is EXPLAIN timing variance and is diagnostic only.

### Findings for A3
1. **non-blocking: the folded probe costs ticket at root about 5 ms in the same window.**
   - **Known:** the closure count to 10,111 rows is about 5.5 ms of the probe's 8.5 ms in EXPLAIN. In the handler, root agents' net median is +4.8 ms against the control, and count-only is +1.9.
   - Root agents' p95 is 47.85 under the selector against 37.82 for the control in this window. That is still 31.7 ms below TICK-1037's 79.54, so it passes the contract line as worded.
   - **Inferred:** the other ticket rows moved +0.4 to +9.6 ms net against the control. Those rows have \|S\| = 101, where the count is 101 rows, so that is drift between phases, not the probe. It is not separable in one pass. `bench_rare`, which runs identical SQL in both modes, moved +0.7 / +0.7.
   - **Proposed for A3:** skip the closure count when R is impossible, that is, when N̂_e > α · U_max. U_max would be the certified 10,111, or the tenant's live unit count if groupsync already exposes it. With N̂_e = 400k, ticket never pays, because 400,000 > 3.3 × 10,111. Verify by one cell: ticket at root under the selector, within 1 ms of the control's net median.
2. **non-blocking (bounded, but it scales with the tenant): the R count picks `records_org_unit`, with `entity_definition_id` as a non-leading key.**
   - **Known:** it reads 19,940 buffers for 20,000 rows, because the partial index's `org_unit_id IS NOT NULL` is implied by the leaf.
   - **Inferred:** the index range it scans is the tenant's whole unit-tagged slice, not the type's. A tenant with 100× the median data pays proportionally on the count-only R count.
   - **Proposed:** drop `org_unit_id IS NOT NULL` from the R leaf. It stays two-valued: `NULL IN (…)` is NULL, and `IS TRUE` makes it false, so the rows are unchanged (A2-6). The partial index is then no longer implied, and the count should take a `(tenant_id, entity_definition_id, …)` index.
   - verify-by-spike, pass: the count-only R count uses a `records_list_*` index at both n_distinct values, and its buffers fall to about the type's heap pages.
3. **non-blocking: m is invisible to the selector.**
   - **Known:** at 10,111 units, with `bench_conc` entirely under S, R costs count-only 75.85 p95 against the control's 46.52, a net median of +6.3 ms. A2's W0 is already cheap there.
   - Both are far under budget. The selector can't see m without the P3 it skips, so I'd accept this and record it in A3. I would not add a second probe.
4. **Bound for A3 (Inferred):** the hashed SubPlan needs the closure set to fit in `work_mem`. Past that it becomes a per-row SubPlan, which is O(N̂_e × \|S\|). At the default 4 MB that is far above the certified 10,111 units, but A3 should state the ceiling, or cap R at U_max as in finding 1.

### The control and TICK-1037's FAIL (information only, not a rescore)
- In this window, unchanged A2 scores `bench_conc` at 9,099 units at **79.58 ms** (agents) and **99.26 ms** (count-only). TICK-1037 had 161.57 / 123.53.
- The paired bases were 26 ms here against 87 / 45 ms there. So most of the 161.57 was host load, as QA inferred, and the noise-free A2 cost is +42.7 / +57.4 ms net median. TICK-1037's FAIL stands as scored.
- A2's cost still grows at 3.4–5.9 µs per unit, and R removes that growth.

### Proposed budget line (for `docs/budgets.md`, to be committed by its owner)
`ADR-010-A3 R selector α = 3.3 (crossover of R and A2 on bench_conc, m = 0, |S| 1,011–9,099, N̂_e 20,000; SP-1f, TICK-1092, 2026-11-22, postgres 16.10, story head 700259e9). R region list p95: 36.43–75.85 ms at 9,099–10,111 units.`

### Rerun
1. `git apply sp1f-spike.diff.txt` on `700259e9`.
2. `NAVA_PERF_SECONDS=60 NAVA_PERF_ONLY='^region:' sh bench/acl-perf/run.sh`, about 90 min plus a 12 min seed.
3. For the agreement tests, run a single `postgres:16.10` container and set `NAVA_TEST_PG_URL=… go test -tags integration -run TestUnderList ./internal/records/pipeline`.

Attached:
- `sp1f-spike.diff.txt`: the full throwaway code (574 lines).
- `sp1f-raw-region.md.gz`: every row, every gate line and every plan.

Per the Technical Lead's earlier ask to push the spike branch: my role doesn't commit or push. The diff is attached for whoever writes A3's implementation task.
