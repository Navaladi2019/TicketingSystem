# Region leaf via a closure-table subquery at 800k (SP-1b)

Scope: TICK-969, ADR-010 D7 and R1, follows SP-1 (TICK-932,
`docs/quality/acl-perf-region.md`). The question: on the P3-36 800k bench, can
the region leaf read an `org_unit_closure` table through a subquery that
depends only on the caller, and stay within the list budget's 3.8 ms p95
headroom (`docs/quality/acl-perf-800k.md:16`) at every |S| up to 10,000?

## Verdict

**None passes.** C1 (`IN (SELECT …closure…)`) is the fastest rendering. It
passes at the root (|S| = 10,111) and is marginal at the region and area
levels. It fails at district-size subtrees (|S| = 101): the planner
underestimates the matching records 8.5× and chooses nested loop + sort,
which reads 3,434 rows and adds about 31 to 37 ms p95. C2 (`ARRAY(SELECT …)`)
and B1 (`ANY((SELECT $S)::uuid[])`) hide S from the planner. It then picks a
bitmap scan over every record in S, which reads 380,048 rows and adds about
1 s at the root.

| Rendering | Verdict | Why |
| --- | --- | --- |
| **C1** `IN (SELECT descendant_id …)` | **fails at |S| = 101** (holds at 10,111; marginal at 10 and 1,011) | District: Δp95 +36.7, +33.3 (plain), +30.3 (R1); reads 3,434 rows. Root: Δp95 +1.6, +3.3, −4.4; reads 52. |
| **C2** `= ANY(ARRAY(SELECT …))` | **fails from |S| = 101** | Initplan: the planner cannot see S. Bitmap over S's records: 37,930 rows at region (+113 to +136 ms), 380,048 at root (+843 to +1,019 ms). |
| **B1** `= ANY((SELECT $S)::uuid[])` (control) | **fails from |S| = 101** | Same initplan blindness and the same bitmap plan as C2. It also binds the whole S. |
| SP-1's D7 as written at |S| = 1,000 (re-run) | **fails** (settles SP-1's marginal row) | Δp50 +14.5 to +15.7, Δp95 +21.5 to +28.6, in three paired passes on both indexes. |
| C1 with `records.org_unit_id` statistics target 1,000 | **fails** | The district estimate stays at 372 (actual 3,195). Same plan, same 3,434 rows. |
| Single-record check (Go `Allows` side) | **holds** | EXISTS on the closure PK, server side: p95 ≤ 0.32 ms at every size, hit or miss. |
| Closure build, full tree | **cheap** | 10,111 units → 49,321 rows in 260 to 347 ms (one statement). 8.5 MB with the PK. |

## Results

### 1 client, paired (the scored pass)

The host was not idle (see Setup). Measured one row at a time, the baseline
alone moved from 20 to 70 ms p50 between rows. So the 1-client pass
alternates a baseline request and a variant request for the same agent in one
loop, and alternates which goes first. Both sides see the same host state.
Each variant runs 15 s of warm-up, then 15 to 20 s. The variant and baseline
columns are full requests: `BEGIN`, `set_config`, count, page, `COMMIT`, as
`nava_app`, with `force_custom_plan`. Δ = variant − baseline from the same
pass. The plan and rows columns come from one `EXPLAIN (ANALYZE, BUFFERS)` of
the page statement for agent 1, taken in the 8-client pass. The region branch
is the fifth child of the `Merge Append`.

| Rendering | |S| (subtree) | Index | pairs | p50 | **p95** | Δp50 | **Δp95** | Region-branch plan (page) | Region branch: records read / removed | Closure rows read | Page plan / exec ms |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| C1 | 10 (area) | plain | 314 / 299 | 31.6 / 33.1 | **49.8 / 58.2** | +3.9 / +5.1 | **+6.7 / +2.7** | Index Scan `records_bench_org_unit` per unit, nested loop over the closure, top-N sort | 220 / 10 | 10 | 2.4 / 5.5 |
| C1 | 101 (district) | plain | 244 / 191 | 50.7 / 48.1 | **79.5 / 126.7** | +24.1 / +17.4 | **+36.7 / +33.3** | same (nested loop + top-N sort) | **3,434** / 202 | 101 | 2.2 / **21.4** |
| C1 | 1,011 (region) | plain | 313 / 321 | 31.0 / 30.2 | **49.8 / 51.5** | +3.3 / +3.1 | **+5.0 / +9.4** | Index Scan `records_list_updated`, ordered, with a closure PK probe per row | 329 / 33 | 329 probes | 5.9 / 8.5 |
| C1 | 10,111 (root) | plain | 368 / 268 | 26.8 / 35.7 | **38.6 / 56.2** | +1.3 / +1.1 | **+1.6 / +3.3** | same (ordered, probe per row) | 52 / 2 | 52 probes | 3.4 / 2.3 |
| C2 | 10 | plain | 183 | 40.2 | **64.7** | +3.4 | **+4.2** | InitPlan (closure) → Bitmap Heap Scan `records_bench_org_unit` → sort | 220 / 10 | 10 | 2.1 / 4.2 |
| C2 | 101 | plain | 207 | 38.5 | **59.7** | +11.5 | **+10.0** | same bitmap | 3,379 / 184 | 101 | 1.5 / 9.9 |
| C2 | 1,011 | plain | 101 | 109.2 | **182.9** | +84.2 | **+126.4** | same bitmap | **37,930** / 2,320 | 1,011 | 3.7 / **176.9** |
| C2 | 10,111 | plain | 20 | 748.4 | **888.9** | +716.1 | **+842.9** | same bitmap | **380,048** / 23,400 | 10,111 | 2.2 / **2,095** |
| B1 | 10 | plain | 203 | 36.3 | **51.9** | +3.0 | **+0.4** | InitPlan (S) → Bitmap Heap Scan → sort | 220 / 10 | none | 1.8 / 3.2 |
| B1 | 101 | plain | 198 | 41.4 | **71.2** | +12.3 | **+21.1** | same bitmap | 3,379 / 184 | none | 2.0 / 12.9 |
| B1 | 1,011 | plain | 81 | 140.0 | **212.1** | +106.6 | **+153.1** | same bitmap | **37,930** / 2,320 | none | 2.7 / **186.8** |
| B1 | 10,111 | plain | 14 | 999.8 | **1,594.9** | +972.5 | **+1,547.9** | same bitmap | **380,048** / 23,400 | none | 2.6 / **1,614** |
| D7 (SP-1) | 1,000 (first 1,000 of region 0) | plain | 267 / 304 | 41.5 / 38.0 | **72.1 / 61.2** | +14.6 / +14.5 | **+28.6 / +21.5** | Index Scan `records_list_updated`, ordered, filter | 329 / 298 | none | 4.3 / 3.0 |
| C1 | 10 | plain + R1 | 244 | 37.3 | **70.6** | +1.6 | **+5.9** | nested loop, Index Scan `records_bench_org_unit_updated`, top-N sort | 220 / 10 | 10 | 5.1 / 28.4 |
| C1 | 101 | plain + R1 | 241 | 51.5 | **75.8** | +24.4 | **+30.3** | nested loop, Index Scan `records_bench_org_unit`, top-N sort (R1 not chosen) | **3,434** / 202 | 101 | 4.6 / **68.3** |
| C1 | 1,011 | plain + R1 | 277 | 34.4 | **61.0** | +3.2 | **+1.2** | ordered `records_list_updated`, closure probe | 329 / 33 | 329 probes | 2.9 / 4.2 |
| C1 | 10,111 | plain + R1 | 295 | 32.1 | **52.9** | +2.1 | **−4.4** | ordered `records_list_updated`, closure probe | 52 / 2 | 52 probes | 3.5 / 2.2 |
| C2 | 10 / 101 / 1,011 / 10,111 | plain + R1 | 253 / 258 / 110 / 19 | 28.9 / 32.1 / 105.3 / 774.3 | **47.5 / 46.9 / 168.5 / 1,066.3** | +2.6 / +9.4 / +82.1 / +751.0 | **+5.2 / +10.5 / +135.6 / +1,019.5** | bitmap on `records_bench_org_unit_updated` | as plain | as plain | 4.1/6.3, 5.5/42.1, 2.5/131.2, 2.1/1,256 |
| B1 | 10 / 101 / 1,011 / 10,111 | plain + R1 | 264 / 257 / 108 / 19 | 27.5 / 34.2 / 109.9 / 802.6 | **48.8 / 43.8 / 153.5 / 978.0** | +1.8 / +11.5 / +86.5 / +781.6 | **+10.0 / +9.6 / +112.2 / +951.9** | bitmap on `records_bench_org_unit_updated` | as plain | none | 5.4/6.1, 6.6/68.2, 1.8/97.4, 7.5/1,776 |
| D7 (SP-1) | 1,000 | plain + R1 | 249 | 44.8 | **76.8** | +15.7 | **+22.6** | ordered `records_list_updated`, filter (R1 not chosen) | 329 / 298 | none | 3.7 / 3.2 |

Cells with two values are two separate paired passes, in run order.

- **How to read the noise.** The baseline p95 ranged from 26 to 93 ms
  between passes. A single-digit Δp95 is within that spread. Δp50 repeats to
  within about 1 to 2 ms. C1's Δp50 is +1.1 to +2.1 at the root, +3.1 to
  +3.3 at the region and +1.6 to +5.1 at the area. That is why region and
  area are called marginal: the p95 criterion cannot be scored to ±3.8 ms on
  this host. District is not marginal: +17 to +24 Δp50 and +30 to +37 Δp95 in
  all three passes.
- **The count statement never runs the region branch.** In every row it
  shows `never executed`, because the desk branch alone returns 1,001 rows.
  Its cost is planning only: 1.8 to 6.3 ms, the same order as the baseline's.

### 8 clients, 20 s per row (P3-36 load; not scored)

The full table is in the raw report. The ranking matches the 1-client pass.
Plain index, 8 clients:

| Rendering | |S| | p95 | Δp95 vs same-run baseline |
| --- | --- | --- | --- |
| C1 | 10 / 101 / 1,011 / 10,111 | 92.0 / 174.7 / 110.7 / 188.5 | +2.0 / +84.7 / −12.7 / +65.2 |
| C2 | 10 / 101 / 1,011 / 10,111 | 135.6 / 154.2 / 610.0 / 3,268.3 | +45.6 / +64.3 / +486.7 / +3,145.0 |
| B1 | 10 / 101 / 1,011 / 10,111 | 83.6 / 165.0 / 500.3 / 3,175.5 | −6.3 / +75.0 / +377.0 / +3,052.2 |
| D7 (SP-1) | 1,000 | 159.5 | +36.2 |

The transport-floor gate passed at the start of the area/district pass
(2.98 ms p50). It failed at the start of the region/root pass (3.85 ms), so
that pass is not scored. All the R1 8-client passes started at 4.6 to 4.7 ms
(not scored).

### Where C1 loses at district size

The plan of the page's region branch, C1, |S| = 101, plain index:

```
Limit (actual rows=7)
  -> Sort (top-N heapsort)                              actual time=17.364..17.375
       -> Nested Loop Anti Join (rows=375 est.)         actual time=0.089..14.123 rows=3195
            -> Nested Loop (rows=375 est.)              actual rows=3195
                 -> HashAggregate (rows=95 est.)        actual rows=101
                      -> Index Only Scan org_unit_closure_pkey (rows=95 est.)  actual rows=101
                 -> Index Scan records_bench_org_unit (rows=12 est.)  actual rows=32 loops=101
Planning Time: 2.290 ms   Execution Time: 19.976 ms
```

- **Known:**
  - The planner expects 375 matching records and finds 3,195 (8.5× low).
  - Most of the time (about 10 of 14 ms) is the anti join that applies the
    four earlier branches' `AND NOT` to every one of the 3,195 rows, before
    the sort.
  - For the same units, SP-1's literal list chose the ordered walk on
    `records_list_updated` and read 949 rows (SP-1, |S| = 100, Δp95 +0.8).
- **Inferred, not profiled:**
  - The semi-join estimate uses an average frequency per `org_unit_id`
    (n_distinct 3,254 over 9,000 leaves, with the seed's requester skew), not
    the subtree's real record count.
  - With an accurate estimate, the planner would likely choose the ordered
    walk, as it does at region and root.
- **Both shapes are expensive somewhere:**
  - Nested loop + sort reads every record in S.
  - The ordered walk reads about page / selectivity rows. For a 10-unit area
    (220 of 400,000 records) that is far over 1,000.
  - So no single plan shape meets "≤ about 1,000 rows" at every |S|. Only an
    accurate estimate per request picks the right shape each time.

### Single-record check (Go `Allows` side)

`SELECT EXISTS (SELECT 1 FROM org_unit_closure WHERE tenant_id = $1 AND
ancestor_id = ANY($2::uuid[]) AND descendant_id = $3)`. It runs as
`nava_app` (RLS on), as dynamic SQL so every call is planned, 2,000 calls per
case, timed inside the server (`exists.sql`):

| U ancestor | Record unit | p50 | **p95** | p99 (ms) |
| --- | --- | --- | --- | --- |
| area (10 units) | inside / outside | 0.071 / 0.075 | **0.149 / 0.163** | 0.238 / 0.287 |
| district (101) | inside / outside | 0.114 / 0.107 | **0.318 / 0.272** | 0.602 / 0.512 |
| region (1,011) | inside / outside | 0.082 / 0.097 | **0.203 / 0.281** | 0.413 / 0.593 |
| root (10,111) | inside | 0.069 | **0.173** | 0.379 |

The same check measured from Go, with `BEGIN`, `set_config`, `EXISTS` and
`COMMIT`, did not differ from the bare `BEGIN`/`set_config`/`COMMIT` floor at
any size. The floor was 10.5 ms p50 at that moment. The server-side numbers
above are therefore the ones that count.

### Closure build

| Run | INSERT … WITH RECURSIVE (ms) | ANALYZE (ms) | Rows |
| --- | --- | --- | --- |
| 1 | 259.6 | 67.8 | 49,321 |
| 2 | 347.1 | 65.6 | 49,321 |
| 3 | 300.8 | 71.4 | 49,321 |

10,111 units: 1 + 10×2 + 100×3 + 1,000×4 + 9,000×5 = 49,321 rows. The size
with the PK is 8,672 kB. The build is one statement, run as the superuser
under the large tenant.

## Implications for the design (for the Solution Architect)

1. **Drop C2 and B1.** Both hide S from the planner, so it plans for an
   unknown set and picks a bitmap scan over every record in S. This is the
   risk SP-1 named (about 375k rows), now measured: 380,048 rows read and
   +0.8 to +1.5 s p95 at the root. R1's index does not change it.
2. **C1 is the right shape and the only candidate.**
   - The subquery depends only on the caller, and the row test still reads
     only the row.
   - For large subtrees the planner walks `updated_at` order and probes the
     closure PK once per row: 52 rows at the root, 329 at the region.
   - Its weak point is mid-size subtrees, where the estimate is wrong (see
     above). R1 does not fix it either.
3. **SP-1's D7 as written fails at |S| = 1,000** (Δp95 +21.5 to +28.6,
   paired). The literal list is now out at every |S| ≥ 1,000.
4. **Statistics do not rescue C1. Checked here, after the main run.**
   - `ALTER TABLE records ALTER org_unit_id SET STATISTICS 1000` plus
     `ANALYZE` raised n_distinct from 3,254 to 3,700.
   - The district estimate stayed at 372 against 3,195 actual. The plan
     stayed nested loop + sort, reading 3,434 rows, with 22.7 ms execution.
   - Area, region and root plans were unchanged.
   - **Known:** this result. **Inferred:** the semi-join estimate prices
     each unit at the column's average non-MCV frequency (about 11 rows).
     This district's units hold about 32 each, so no column statistic can
     describe "the records under this subtree".
   - The remaining options change the design, so they are the Solution
     Architect's call. Each still needs a spike:
     - **(i)** Choose the leaf shape per request from a stored per-subtree
       record count: nested loop + sort when it is small, ordered walk when
       it is large.
     - **(ii)** Accept a bounded mid-size cost (district: about +17 to +24
       ms p50, +30 to +37 ms p95), which is outside the 3.8 ms headroom.
     - **(iii)** Restore budget headroom elsewhere first.
5. **The Go `Allows` side is free.**
   - The closure PK answers it in ≤ 0.32 ms p95, so D7's per-request
     expansion (H1, about 41 ms p95 at 10,111 units in SP-1) is not needed
     for single-record checks.
   - The closure build (≤ 0.35 s for the full tree) fits a group-sync write
     path.

## Known gaps

- **The host was not idle.** All along, the `nava` dev stack (15
  containers) was up. From 15:41 UTC another task's `nava-s18` stack (12
  containers) was also up, with host CPU at 100% at times.
  - Neither stack is on P3-36's allow-list. They are not this run's
    containers, so they were not stopped.
  - The P3-36 floor gate (`SELECT 1`, p50 ≤ 3 ms) passed at the start of 3
    of the 5 paired passes (2.28, 2.46, 2.91 ms). It failed at 2 (4.17,
    4.57 ms).
  - Absolute p95s are not scored against 150 ms. The verdicts use paired
    deltas.
  - The district failure (+30 to +37 ms Δp95 in every pass, plus 3,434 rows
    read) and the C2/B1 failures stand well above the noise. The area and
    region C1 rows do not, and an idle-host re-run would settle them. That
    would not change the overall verdict, because district fails.
- **The issue's literal B1 does not parse as meant.**
  `= ANY((SELECT $S::uuid[]))` is read as `= ANY(subquery)` and fails with
  `operator does not exist: uuid = uuid[]`. B1 was measured as SP-1's
  `= ANY((SELECT $S)::uuid[])`.
- **Sizes are whole subtrees** of the ancestor U = {area 0/0/0, district 0/0,
  region 0, root}: 10, 101, 1,011 and 10,111 units. SP-1 used the first k
  units. The re-run D7 row uses SP-1's exact 1,000-unit set.
- **Database work only, not through the handler**, as in SP-1. One U per
  size, and the region branch is in the worst (last) position.

## Setup

- **Code:** `main` `c276a374`. Bench only, no migration, no runtime code.
  - P3-36 seed (`TestRecordsACLPerf800k`, `BENCH_KEEP=1`): 800,000 records.
    The large tenant has 400,000 records, 5,000 users and 50 groups. Seeded
    in 5 min 37 s.
  - SP-1's `region.sql`: the 10,111-unit tree and `records.org_unit_id`
    (5.0% NULL), then `VACUUM FULL`, `ANALYZE` and `CHECKPOINT`.
  - `closure.sql`: `org_unit_closure (tenant_id, ancestor_id, descendant_id,
    depth)`, PK `(tenant_id, ancestor_id, descendant_id)`, with RLS like
    `org_units_select` and `SELECT` granted to `nava_app`. Live units only:
    the bench tree has no deleted units.
- **Statements:** SP-1's count and page (the handler's own, from P3-36), with
  the region leaf as the fifth `ReadBranches` branch. It is last, so it
  carries `AND NOT` of the four earlier branches. It runs as `nava_app` in the
  tenant transaction, with `QueryExecModeExec` and
  `plan_cache_mode = force_custom_plan` (checked: the `nava_app` role
  setting). The 200 P3-36 agents take turns. In C1 and C2, `$9` is U = {the
  ancestor}. In B1 and D7, `$9` is the whole S, bound from Go.
- **Indexes:** "plain" is `records (tenant_id, org_unit_id)`. "plain + R1"
  adds `(tenant_id, org_unit_id, updated_at DESC, id DESC)`.
- **Method:**
  - 8 clients × 20 s per row, then 1-client paired passes of 15 to 20 s per
    variant (see above).
  - Warm-up: one pass over the 200 agents, capped at 15 s.
  - Nearest-rank percentiles (the same `pct` as P3-36).
  - `ANALYZE`, `CHECKPOINT` and `pg_prewarm` of every relation before each
    index phase.
- **Machine:** the P3-36 reference host. Windows 11, i7-1260P, Docker on
  WSL2 (kernel 5.15.167.4), VM with 16 CPUs and 24.5 GiB.
  - `postgres:16.10` with P3-36's settings (`shared_buffers=2GB`,
    `effective_cache_size=6GB`).
  - Go client with pgx v5.11.0.
- **Date:** 2026-11-18, 14:55 to 16:50 UTC.

## Reproduce

```
# 1. Seed (about 25 min with region.sql) and keep the container: SP-1's step 1
BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$' sh bench/acl-perf/run.sh   # prints "kept: <name>"
# 2. Region tree, closure table, timed build
SP1_PG=<name> sh bench/acl-perf/region/closure.sh setup
# 3. 8-client passes (plans and rows)
for s in plain8a plain8b r18a r18b; do SP1_PG=<name> sh bench/acl-perf/region/closure.sh $s; done
# 4. Paired 1-client passes (scored): plain, then R1
SP1B_PAIRED=1 SP1B_PREP=0 SP1B_PHASE=plain SP1B_CLIENTS=1 SP1B_SECONDS=20 SP1B_ONLY='^(C1|D7) ' <closure.sh env> go run -tags perf ./bench/acl-perf/region/closure
#    (likewise with SP1B_ONLY='^(C2|B1) ', then SP1B_PHASE=r1)
# 5. Single-record check, server side
docker exec $SP1_PG psql -U pgadmin -d nava -c "CREATE TABLE bench_ou_ids AS SELECT pre, id FROM bench_ou; GRANT SELECT ON bench_ou_ids, perf_tenant TO nava_app;"
docker exec -i -e PGPASSWORD=test-pw-nava_app $SP1_PG psql -h 127.0.0.1 -U nava_app -d nava < bench/acl-perf/region/exists.sql
```

It needs Docker and Go and is not in CI. The raw output, with every plan, is
in `bin/acl-perf-region-closure-raw.md`. Remove the kept container with
`docker rm -f <name>`.
