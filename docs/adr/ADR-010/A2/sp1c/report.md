<!-- SP-1c report (TICK-1038), archived verbatim from the Technical Architect's comment 01a1fca3-ab0f on 2026-11-21. Harness: probe-main.go.txt (renamed from probe-main.go so it is not built), probe.sh. Raw plans: sp1c-raw.md. Console: sp1c-console.log. -->

## SP-1c verdict: none passes. The probe mechanism holds the walk and the row cap, but district-size subtrees stay over the 3.8 ms headroom

**Verdict (one line):** no K and no rendering gives Δp95 ≤ 3.8 ms at every |S|. The best mechanism measured has a worst case of **+5.7 ms** per-pair median at the district (101 units), on server-side time. That mechanism is a probe with K = 1,000, then `= ANY(ARRAY(…))` (C2) at ≤ K and the `OFFSET 0` walk (W0) above K. About 40 % of the district cost is the four `AND NOT` anti-joins. The row criterion passes: every size reads ≤ 949 rows.

### Hypothesis and pass criterion (from TICK-9 `01a1ef96` and TICK-946 `01a1fc64`)
- **Hypothesis:** a capped probe (K + 1 rows) choosing C1 at ≤ K and the per-row `EXISTS … OFFSET 0` walk above K keeps the region leaf within the headroom at every size.
- **Pass:** at every |S| ∈ {10, 101, 1,011, 10,111}, plus skewed districts, **Δp95 ≤ 3.8 ms** on the 1-client paired pass, with the probe's cost included, and the region branch reading ≤ about 1,000 rows.

### Setup
- PostgreSQL 16.10 (`postgres:16.10`, P3-01 memory settings). Docker VM: 16 CPUs, 24.5 GiB, WSL2 5.15.
- **S31's migrated schema:** `TICK-946/in-my-region` at `c84b1827`, P3-36's 800k seed through the migrations, the migration's partial `records_org_unit` index and no bench indexes. On top of that: SP-1's `region.sql` tree (10,111 units) and SP-1b's `org_unit_closure`, rebuilt over **live units only** (`deleted_at IS NULL` on both sides of the recursion). The build gives 49,321 rows in 234 to 265 ms.
- Large tenant: 400,000 tickets. Requests run as `nava_app` under RLS with `plan_cache_mode=force_custom_plan`, the region branch fifth (worst position), count and page statements.
- **Skewed districts** (both with 101 units):
  - **hot:** 20,000 records moved from region 9 into one branch, giving 22,996 tickets, almost all in one unit;
  - **thin:** all but 600 of its records moved out, giving 600 tickets.
- Rendering of the ≤ K side: the probe guarantees at most K matching records, so a bitmap over S (C2) is bounded by K there. I added C2 because C1 did not take the plan the mechanism assumes (Known 3).

### Known
1. **`OFFSET 0` blocks the pull-up on PG 16.10.** W0's plan is `Index Scan records_list_updated > SubPlan > Index Only Scan org_unit_closure_pkey`, the ordered walk with one PK probe per row, at every size. Without `OFFSET 0` (W1), the `EXISTS` is pulled up into the same semi-join as C1, and the plans are identical.
2. **The walk must not run on small subtrees.** For the area (220 tickets), W0 reads **6,396** records, with a per-pair median of +30.6 ms and Δp95 of +71.5 ms. The ≤ K side is required.
3. **C1 does not control its plan; the planner's statistics do.** On this schema, C1 at district picks the ordered walk (949 records read), not SP-1b's nested loop + top-N sort (3,434 records). `pg_stats` for `records.org_unit_id` has n_distinct 3,156 here. So the C1 shape at |S| = 101 depends on a sampled statistic, and on the thin district it walks 913 rows to return 600. C2 and W0 each force one shape whatever the statistics say.
4. **The probe's cost is not bounded by K for large |S|.** At K = 1,000 the probe unique-ifies the whole closure set before it touches records:
   - region: a Merge Join reads 12,028 index entries;
   - root: an Aggregate over 10,111 closure rows, 7.0 ms;
   - root at K = 4,000: a Hash Join, 12.6 ms at region.

   At K ≤ 250 the probe is records-driven and costs ≤ 0.9 ms everywhere. Probe alone, as per-pair median: ≤ +1.2 ms for K ≤ 250 at every size, **+5.1 ms at root for K = 1,000**, and +2.4 ms at region.
5. **Anti-join share at district** (EXPLAIN, region branch ms, with vs. without the four `AND NOT`s): W0 7.09 vs. 4.29 (**39 %**), and C1 6.38 vs. 3.54 (45 %). On the area, C1 is 2.93 vs. 0.85.
6. **The row criterion passes** for M2 at K = 1,000: area 220, district 949, thin 600, hot 448, region 329 and root 52 records read.

### Results: server-side paired pass
Planning plus execution ms per request (probe + count + page), from `EXPLAIN (ANALYZE, TIMING OFF)` as `nava_app`. Baseline and variant alternate per agent, 300 pairs per cell. The two Δ columns are median per-pair Δ / Δp95.

| Set (units, tickets) | C1 | W0 | C2 | **M2 K=1000** (C2 ≤ K, W0 > K) | M2 K=500 | M K=250 (C1/W0) | M K=1000 (C1/W0) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| area (10, 220) | +3.2 / +4.7 | +30.6 / +71.5 | +1.6 / +2.5 | **+2.3 / +3.3** | +2.3 / +3.3 | +2.4 / +3.1 | +2.7 / +3.9 |
| district (101, 3,379) | +2.9 / +5.5 | +3.5 / +5.6 | +7.1 / +9.5 | **+5.7 / +8.5** | +5.7 / +9.5 | +5.9 / +10.4 | +7.0 / +15.5 |
| district thin (101, 600) | +8.6 / +28.5 | +8.3 / +26.0 | +2.9 / +4.3 | **+4.8 / +11.4** | +7.8 / +19.6 | +7.2 / +18.2 | +11.9 / +36.5 |
| district hot (101, 22,996) | +2.7 / +4.8 | +1.8 / +3.1 | +37.4 / +50.9 | **+2.7 / +4.3** | +2.7 / +4.6 | +3.3 / +8.2 | +2.9 / +4.6 |
| region (1,011, 37,930) | +1.4 / +2.9 | +1.6 / +2.8 | +77 / +123 | **+3.9 / +5.5** | +3.3 / +4.6 | +3.3 / +4.8 | +4.0 / +5.9 |
| root (10,111, 380,048) | +0.4 / +1.0 | +0.2 / +0.9 | +760 / +1,185 | **+4.4 / +6.7** | +1.7 / +2.4 | +1.1 / +1.5 | +5.0 / +6.8 |
| **worst case** | | | | **+5.7 median** (district) | +7.8 (thin) | +7.2 (thin) | +11.9 (thin) |

**The K that minimises the worst case is 1,000, with C2 below K.** This holds only if the probe's large-|S| cost is fixed (Known 4), because root and region pay +4.4 and +3.9 for the probe at K = 1,000. With the probe as specified, K = 500 gives the lower total at region and root. It then routes the thin district (600 records) to the walk at +7.8. Every combination fails at a 101-unit district. That holds on the median too, so it is not a p95 noise artefact.

### Caveats
- **The host was not idle, and the scored wall-clock pass is unusable.** The floor gate passed at the start (p50 2.79 ms) and failed at the end (p50 4.39 ms). The `nava` dev stack (12 containers) ran throughout. Other tasks' `nava-it-pg-*` and `nava-iso-pg-*` integration clusters started mid-run. None of these are mine, so I did not stop them.
  - In the 1-client wall-clock paired pass, the baseline p95 swung from 30 to 200 ms against about 7 ms of server execution. Its rows are in `sp1c-console.log`, not scored.
  - I scored the server-side pass instead. It excludes transport, which is the same for every variant, but CPU contention remains: baseline p95 is 6 to 13 ms.
  - Δp95 cells within ±3 ms of each other are not separable on this host. The per-pair medians are the robust column.
- Absolute p95s are not scored against 150 ms.

### Implications for ADR-010-A2 (Solution Architect's call)
- **Holds and can go into A2:**
  - the closure source, built over live units only;
  - the `OFFSET 0` walk above K;
  - C2 (not C1) at ≤ K, bounded by K through the probe;
  - the probe as the switch, which never changes which rows are visible;
  - the ≤ ~1,000-row cap on rows read.
- **Does not hold:** the 3.8 ms headroom at district size. Under your TICK-946 ruling this goes to option (iii). The lever measured here is branch disjointness: removing the four `AND NOT`s takes 39 to 45 % off the district region branch. Scaling the district's +4.5 to +5.7 ms median by that share gives roughly +3 to +3.5 ms, which is inside the headroom but with little margin. **Inferred, not measured end to end.**
- **Needs a fix before K = 1,000:** a probe rendering that stays records-driven at large |S|. Open, not tested here. It would also settle whether `MaxUnderSet` can be dropped, because the probe at K = 1,000 is currently O(|S|) over the closure.
- **No budget changes.** Every SP-1c number is **unverified** for `docs/budgets.md` until an idle-host rerun.

### How to rerun
On `TICK-946/in-my-region`, put the attached `probe-main.go` at `bench/acl-perf/region/probe/main.go` and `probe.sh` at `bench/acl-perf/region/probe.sh`.
```sh
BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$' sh bench/acl-perf/run.sh   # seed; exits 1 if the floor gate refuses the handler pass, the container is kept
SP1C_PG=<kept container> sh bench/acl-perf/region/probe.sh setup     # region.sql, closure.sql, skewed districts, live-only closure build
SP1C_PG=<kept container> sh bench/acl-perf/region/probe.sh explain   # plans, rows read, probe plans for K = 250/1000/4000
SP1C_PG=<kept container> SP1C_PAIRS=300 sh bench/acl-perf/region/probe.sh server   # scored server-side paired pass
SP1C_PG=<kept container> SP1C_KS=250,1000 sh bench/acl-perf/region/probe.sh paired # wall-clock 1-client paired pass (idle host only)
```

<details><summary>The renderings measured (from probe-main.go)</summary>

```sql
-- probe ($3 = K + 1)
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.org_unit_id IS NOT NULL
  AND r.org_unit_id IN (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou
                        WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = ANY($2::uuid[])) LIMIT $3) p
-- C1
r.org_unit_id IS NOT NULL AND r.org_unit_id IN (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = ANY($9::uuid[]))
-- C2
r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY(ARRAY(SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = ANY($9::uuid[])))
-- W0 (W1 = the same without OFFSET 0)
r.org_unit_id IS NOT NULL AND EXISTS (SELECT 1 FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1
  AND acl_ou.ancestor_id = ANY($9::uuid[]) AND acl_ou.descendant_id = r.org_unit_id OFFSET 0)
-- live-only closure build
WITH RECURSIVE c(tenant_id, ancestor_id, descendant_id, depth) AS (
  SELECT tenant_id, id, id, 0 FROM org_units WHERE tenant_id = $t AND deleted_at IS NULL
  UNION ALL
  SELECT c.tenant_id, c.ancestor_id, o.id, c.depth + 1 FROM c
  JOIN org_units o ON o.tenant_id = c.tenant_id AND o.parent_id = c.descendant_id AND o.deleted_at IS NULL)
```
The four base branches, the `AND NOT` chaining, the page and count statements are SP-1b's (`bench/acl-perf/region/closure/main.go` on `main`).
</details>

**Attached:**
- `probe-main.go` and `probe.sh`: the harness, throwaway;
- `sp1c-raw.md`: every table, plan and EXPLAIN tree;
- `sp1c-console.log`: the wall-clock and server-side passes, line by line.

The bench container has been removed. If the Solution Architect wants the harness and report in the repo with A2, the Technical Lead can commit them as `bench/acl-perf/region/probe/` and `docs/quality/acl-perf-region-probe.md`. Nothing has been committed.
