<!-- SP-1d report (TICK-1041), archived verbatim from the Technical Architect's comment 01a1fcc4-cf72 on 2026-11-21. Go harness files are renamed *.go.txt so they are not built. -->

## SP-1d verdict: fails at district size. The records-driven probe holds, the disjoint page helps by 1 to 2 ms, the disjoint count is unusable, and the probe's extra round trip shows only in wall-clock time

**Verdict (one line):** no combination passes. The best is the per-ancestor `unnest` probe P3 at K = 1,000, C2 / W0, the page with all branches disjoint and the count in today's form. It brings five of the six sizes within the 3.8 ms on server-side median. **District stays over budget**: +4.9 ms median / +9.4 ms Δp95 server-side, and +7.5 / +11.1 ms wall-clock.

### Hypothesis and pass criterion (Solution Architect, TICK-946 comment `01a1fca5`)
- **Hypothesis:** (1) a records-driven probe at K = 1,000 plus (2) the branches compiled without the four `AND NOT`s brings the district inside the headroom.
- **Pass:** the worst cell's per-pair median **and** its Δp95 are both ≤ 3.8 ms, across all six subtree sizes. (3) also asks for an idle-host pass on the winner, reporting wall-clock and server-side numbers.

### Setup
Same as TICK-1038:
- PostgreSQL 16.10 and the P3-01 settings, on `TICK-946/in-my-region` at `c84b1827`.
- The 800k seed with S31's `records_org_unit` index.
- A live-only closure: 49,321 rows, rebuilt in 205 to 234 ms.
- The six subtrees: area, district, thin, hot, region and root.

The harness is SP-1c's, plus `probe-sp1d.go` (attached).

**Renderings:**
- **P0:** SP-1c's probe, `IN (closure)`.
- **P2:** `closure CROSS JOIN LATERAL (records … LIMIT K+1)`.
- **P3:** `unnest(U) CROSS JOIN LATERAL (closure WHERE ancestor_id = u) CROSS JOIN LATERAL (records … LIMIT K+1)`.
- **Disjoint:** the `AND NOT`s are removed and the overlapping branches are de-duplicated:
  - the page uses `DISTINCT ON (sort_key, id)` over the merged stream, keeping the Merge Append and the early stop;
  - the count uses `DISTINCT id`.
- **M3r:** only the region branch is made disjoint.
- **M3a:** every branch is made disjoint.

### Known
1. **P3 is records-driven at every size.** It never reads more than 1,008 records and executes in ≤ 1.4 ms.
   - At root, execution falls from 5.74 ms to 0.64 ms, and the probe-only median Δ from +4.19 to +0.67 ms.
   - At region, execution falls from 3.56 ms to 0.93 ms.
   - **P2 does not hold.** At root it chooses a Seq Scan over `org_unit_closure`, which takes 10.7 ms because the depth-4 rows sit at the end of the heap.
   - P3 uses `ancestor_id = <one id>` per ancestor, which the planner prices at about 5 rows, so it keeps the PK scan.
   - P3's capped count equals P0's at every size, for K = 1,000 and 500.
   - It is O(K), not O(|S|): **no tree-size guard is needed for the probe.**
2. **Results are identical.** M3, M3r, M3a and the disjoint no-leaf page return the same page ids, in order, and the same capped count as SP-1c's M2. This holds for **40 of 40** agents in every set.
3. **The disjoint count is unusable.** De-duplication makes the count consume every branch. The region walk then runs inside the count, without an `ORDER BY` to stop it: at district, **496.6 ms, reading 173,723 records** (M3r and M3a alike). Today's count never runs the region branch, because the desk branch fills 1,001 first. **Disjointness is viable for the page only.** The count keeps today's `AND NOT` form.
4. **The disjoint page costs nothing on its own.** Without a region leaf it is −0.06 to −0.14 ms median against today's page at every size. Where it helps, it takes 1 to 2 ms off the region-leaf request (table below). It does not close the district gap: the walk itself still reads 949 records with a closure-PK SubPlan per row.
5. **Server-side `EXPLAIN` misses the probe's round trip.** At root, M3a is +0.36 ms server-side and +2.98 ms wall-clock. The gap is about one `SELECT 1` round trip on this host (floor p50 1.70 ms). It appears in every wall-clock cell, because the probe is a third statement. **Scoring A2 on server-side time alone would undercount every probed request by one round trip.**
6. **The P3-36 handler pass measured today.** For the first time in this series, the 8-client floor passed (p50 1.13 ms) at seed time, and the handler pass ran: **agent list, sort=-updated_at, p95 119.99 ms** (n = 2,224, `sp1d-handler-baseline-raw.md`). The 3.8 ms headroom is derived from that row's 146.2 ms in `docs/quality/acl-perf-800k.md:16`.

### Server-side paired pass
Planning plus execution, 300 pairs per cell, against today's list with no region leaf. Each cell is per-pair median Δ / Δp95, in ms.

| Set (units, tickets) | probe P0 only | probe P3 only | M2 = SP-1c | M3 (P3) | M3r (P3, region page disjoint) | **M3a (P3, page disjoint)** | M3 K=500 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| area (10, 220) | +0.5 / +0.3 | +0.5 / +0.8 | +2.3 / +3.9 | +1.6 / +2.5 | +1.3 / +2.3 | **+1.4 / +1.7** | +2.0 / +3.2 |
| district (101, 3,379) | +1.1 / +2.6 | +0.9 / +1.4 | +7.7 / +25.7 | +6.8 / +15.2 | +5.5 / +9.4 | **+4.9 / +9.4** | +7.6 / +15.3 |
| thin (101, 600) | +1.6 / +4.2 | +1.2 / +1.6 | +5.5 / +9.2 | +4.8 / +7.0 | +3.5 / +5.5 | **+3.0 / +4.2** | +5.6 / +17.1 |
| hot (101, 22,996) | +0.7 / +1.3 | +0.8 / +2.3 | +2.9 / +6.5 | +2.9 / +4.1 | +3.0 / +9.7 | **+3.0 / +5.1** | +3.0 / +4.4 |
| region (1,011, 37,930) | +2.1 / +2.9 | +0.9 / +4.8 | +4.1 / +9.4 | +2.9 / +7.4 | +1.9 / +2.9 | **+1.8 / +2.4** | +2.0 / +2.4 |
| root (10,111, 380,048) | +4.2 / +6.4 | +0.7 / +0.9 | +5.0 / +7.0 | +0.9 / +1.9 | +0.6 / +0.8 | **+0.4 / +0.6** | +0.7 / +1.1 |

### Wall-clock paired pass, winner vs. SP-1c
Same pairing, 300 pairs per cell, measured from the client (BEGIN, set_config, statements, COMMIT). The floor gate passed at the start: p50 1.70 ms, p95 5.43 ms.

| Set | M2 = SP-1c | **M3a** |
| --- | --- | --- |
| area | +5.4 / +8.9 | **+4.3 / +7.4** |
| district | +8.3 / +9.3 | **+7.5 / +11.1** |
| thin | +6.4 / +8.9 | **+5.3 / +6.7** |
| hot | +4.7 / +8.6 | **+4.5 / +8.8** |
| region | +7.0 / +11.2 | **+4.5 / +6.7** |
| root | +7.3 / +9.9 | **+3.0 / +4.6** |

### Host
**Not idle, and I cannot make it idle.**
- At seed time it was quiet: the 8-client floor was p50 1.13 ms, and the handler pass ran.
- During the timed passes, other tasks' integration clusters were running (`nava-it-pg-*` and `nava-iso-pg-*`), alongside the `nava` dev stack. None of these are mine.
- The wall-clock baseline p95 was 22 to 31 ms, against a server-side p95 of 6 to 13 ms.

So both bases are reported. Neither is budget-grade: every number is **unverified** for `docs/budgets.md`.

### Inferred, not measured
- **Folding the probe into the count statement** would remove the third round trip, which is +2 to +3 ms wall-clock at every size. One way is two `UNION ALL` arms, each gated by a one-time `(probe) <= K` or `> K` InitPlan, so the arm not chosen never executes. The cost would be planning both shapes.
- **The remaining district cost is the walk itself:** 949 records with a per-row PK SubPlan, about 4 to 5 ms server-side. The next lever is per-unit ordered access: a `(tenant_id, org_unit_id, updated_at DESC, id DESC)` index merged per descendant. It is a write-amplification trade, and untested.
- **The headroom itself may be the thing to remeasure.** If today's 119.99 ms handler p95 reproduces on a scored idle run, the headroom is about 30 ms, not 3.8, and every SP-1c and SP-1d cell fits. One run is not a budget. Rebasing it needs P3-36's scored rerun (QA).

### How to rerun
On `TICK-946/in-my-region`, put `probe-main.go` and `probe-sp1d.go` in `bench/acl-perf/region/probe/` (as `main.go` and `sp1d.go`) and `probe.sh` in `bench/acl-perf/region/`.
```sh
BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$' sh bench/acl-perf/run.sh   # seed + P3-36 handler pass if the floor gate passes
export SP1C_PG=<kept container> SP1C_OUT=$(pwd)/bin/sp1d-raw.md
sh bench/acl-perf/region/probe.sh setup      # region.sql, closure.sql, skewed districts, live-only closure
sh bench/acl-perf/region/probe.sh check      # id/count equality, 40 agents per set; probe P0/P2/P3 counts
sh bench/acl-perf/region/probe.sh explain2   # probe and disjoint plans, rows read
SP1C_PAIRS=300 SP1C_ONLY='(probe P0|probe P3|M2 =|M3 \(|M3r|M3a \(|M3 K=500|no leaf)' sh bench/acl-perf/region/probe.sh srv2
SP1C_PAIRS=300 SP1C_ONLY='(M3a \(|M2 =)' sh bench/acl-perf/region/probe.sh wall2
```

<details><summary>P3 and the disjoint page, as measured</summary>

```sql
-- P3: records-driven probe ($3 = K + 1)
SELECT count(*) FROM (SELECT 1 FROM unnest($2::uuid[]) acl_u(id)
  CROSS JOIN LATERAL (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou
                      WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = acl_u.id) c
  CROSS JOIN LATERAL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.org_unit_id IS NOT NULL
                      AND r.org_unit_id = c.descendant_id LIMIT $3) x
  LIMIT $3) p
-- With overlapping U (an ancestor and its descendant both in U) P3 can count a record twice:
-- it then switches to the walk earlier. It never changes which rows are visible.

-- Disjoint page: every branch without AND NOT, then
SELECT DISTINCT ON (u.sort_key, u.id) u.* FROM ( (branch 1 … LIMIT $8) UNION ALL … (branch 5 … LIMIT $8) ) u
ORDER BY u.sort_key DESC NULLS FIRST, u.id DESC LIMIT $8
-- The count stays today's AND NOT form (the disjoint count reads 173,723 rows at district).
```
</details>

**Attached:**
- `probe-main.go`, `probe-sp1d.go` and `probe.sh`: the harness;
- `sp1d-raw.md`: every table and plan;
- `sp1d-console.log`;
- `sp1d-handler-baseline-raw.md`: the P3-36 handler pass.

The bench container has been removed. Nothing has been committed.
