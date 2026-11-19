# Region access at 800k: `org_unit_id = ANY(S)` (SP-1)

Scope: TICK-932, PLAN-011 P11-01, ADR-010 D7 and R1. The question: on the
P3-36 800k bench, does the support-agent incident list (`sort=-updated_at`,
first page) stay at or under the 150 ms p95 list budget
(`docs/quality/acl-perf-800k.md:16`, 146.2 ms today, so 3.8 ms of headroom)
when the read predicate gains D7's leaf
`org_unit_id IS NOT NULL AND org_unit_id = ANY($S)`, for |S| = 10, 100, 1,000
and 10,000?

## Verdict

**R1: FAIL at |S| = 10,000 with both indexes.** The region leaf adds about
136 ms p50 and 158 ms p95 per list request, even with one client. Projected
onto the 146.2 ms handler p95, that is about 300 ms, twice the budget. R1's
fallback index does not help, because the time is not spent scanning rows.

| |S| | Verdict | Why |
| --- | --- | --- |
| 10, 100 | **holds** | The added cost is within noise (≤ 3 ms p95, one client). The region branch reads ≤ 950 index entries. |
| 1,000 | **not settled; marginal** | +7.7 ms p95 with one client, against 3.8 ms of headroom. This host was contended, so the difference cannot be scored. An idle-host re-run decides it. |
| 10,000 | **fails** | +158 ms p95 with one client. Planning `= ANY(<10,000 uuids>)` costs 15–36 ms per statement, and the list runs two statements. Binding the 10,000-element array twice adds more. Execution stays under 3 ms. |

Under R1, this result triggers the closure-table fallback (ADR-010 option D).
S31 waits for a design decision. That decision belongs to the Solution
Architect. The measurement below suggests two cheaper candidates to check by
spike first: see "Implications".

## Results

### 8 clients, 20 s per row (the P3-36 load)

ms per request. A request is the list's database work: `BEGIN`,
`set_config('app.tenant_id')`, count, page, `COMMIT`.

| |S| | Index | n | p50 | **p95** | p99 | Region-branch plan (page) | Region branch: rows read / removed | Page exec / plan (ms, single EXPLAIN) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| none (baseline, 4 branches) | plain | 1411 | 82.5 | **306.7** | 436.6 | n/a | n/a | 3.7 / 1.6 |
| 10 | plain | 1443 | 96.4 | **219.7** | 369.7 | Index Scan `records_bench_org_unit` + top-N sort | 220 / 10 | 3.3 / 3.4 |
| 100 | plain | 1497 | 95.0 | **193.7** | 276.4 | Index Scan `records_list_updated`, ordered, filter | 949 / 943 | 5.2 / 3.2 |
| 1,000 | plain | 1285 | 112.7 | **228.3** | 301.2 | Index Scan `records_list_updated`, ordered, filter | 329 / 298 | 2.7 / 6.8 |
| 10,000 | plain | 374 | 419.4 | **615.6** | 674.7 | Index Scan `records_list_updated`, ordered, filter | 52 / 2 | 2.9 / **36.0** |
| 10 | plain + R1 | 2397 | 62.8 | **99.4** | 125.2 | Index Scan `records_bench_org_unit_updated` + top-N sort | 220 / 10 | 5.7 / 3.5 |
| 100 | plain + R1 | 2116 | 70.4 | **114.8** | 212.6 | Index Scan `records_list_updated` (R1 not chosen) | 949 / 943 | 6.0 / 4.2 |
| 1,000 | plain + R1 | 1807 | 83.4 | **136.5** | 173.5 | Index Scan `records_list_updated` (R1 not chosen) | 329 / 298 | 3.3 / 16.6 |
| 10,000 | plain + R1 | 402 | 393.8 | **516.1** | 579.3 | Index Scan `records_list_updated` (R1 not chosen) | 52 / 2 | 5.5 / **35.9** |

**Index per branch: yes, in every row.** Each of the five branches has its
own index scan under one `Merge Append`. No row fell back to a sequential
scan or to one sort over the whole tenant. The count statement never runs the
region branch: the desk branch alone already returns its 1,001 rows.

**This table is contended and is not scored.** The host's transport floor
(`BEGIN`, `set_config`, `COMMIT`) was 9.0 ms p50 and 30.0 ms p95 here. The
P3-36 gate allows 3 ms p50. The baseline p95 of 306.7 ms is therefore twice
the 146.2 ms of the scored P3-36 run. The "plain + R1" rows ran later, in a
quieter window, which is why even |S| = 10 looks faster there. Only the
|S| = 10,000 rows stand far above the noise.

### 1 client, 10 s per row (cost per request, no queueing)

| |S| | Index | n | p50 | **p95** | Δ p50 / Δ p95 vs baseline |
| --- | --- | --- | --- | --- | --- |
| none (baseline) | plain | 324 | 29.8 | **47.5** | n/a |
| 10 | plain | 336 | 27.2 | **50.2** | −2.6 / +2.7 |
| 100 | plain | 318 | 29.7 | **48.3** | −0.1 / +0.8 |
| 1,000 | plain | 250 | 38.7 | **55.2** | +8.9 / +7.7 |
| 10,000 | plain | 61 | 166.0 | **205.1** | **+136.2 / +157.6** |
| 10 | plain + R1 | 343 | 28.1 | **42.8** | −1.7 / −4.7 |
| 100 | plain + R1 | 304 | 31.3 | **47.5** | +1.5 / 0.0 |
| 1,000 | plain + R1 | 233 | 41.4 | **64.2** | +11.7 / +16.6 |
| 10,000 | plain + R1 | 60 | 161.4 | **235.1** | **+131.6 / +187.6** |

Transport floor in this pass: 5.8 ms p50, 10.1 ms p95.

### Subtree expansion (D7's recursive query alone)

ms per request: `BEGIN`, `set_config`, the recursive query over `org_units`, `COMMIT`.

| Subtree root | Units | 8 clients p50 / p95 | 1 client p50 / p95 |
| --- | --- | --- | --- |
| area | 10 | 12.0 / 23.3 | 10.9 / 20.4 |
| district | 101 | 16.7 / 37.1 | 16.2 / 31.7 |
| region | 1,011 | 22.2 / 41.6 | 15.9 / 28.9 |
| root | 10,111 | 79.6 / 102.4 | 33.5 / 50.8 |

After subtracting the floor (5.8 / 10.1 ms with one client), the expansion
costs about 28 ms p50 and 41 ms p95 at about 10,000 units. RFC-022 §7
assumed at most 5 ms p95 (hypothesis H1). **H1 fails above about 100 units**,
before any list runs. (Known: these numbers. Inferred: the cost is the
recursive CTE's per-level index lookups. It was not profiled.)

### Where the |S| = 10,000 time goes

Each statement was planned alone with `EXPLAIN (SUMMARY)`, on a minimal
query: one branch, `ORDER BY updated_at DESC, id DESC LIMIT 51`.

| Leaf rendering | Planning time |
| --- | --- |
| `org_unit_id = ANY('<10,000 uuids>'::uuid[])` (D7 as written) | 15.1, 18.4, 19.5, 24.1 ms (4 runs) |
| same, 1,000 uuids | 1.8, 2.0 ms |
| same, 10,000 uuids, column statistics off (`SET STATISTICS 0`) | 17.3 ms |
| `org_unit_id = ANY((SELECT $S)::uuid[])` (initplan, values hidden from the planner) | 0.24 ms |
| `org_unit_id IN (SELECT unnest($S))` | 0.41 ms |

- **Known:**
  - Planning time grows with |S|, roughly linearly (about 10× from 1,000 to 10,000). Inside the full five-branch page and count statements it is 36 ms each.
  - Column statistics do not change it, so the cost is not the MCV selectivity loop.
  - `plan_cache_mode = force_custom_plan` (db-roles) replans every request.
- **Inferred, not profiled:**
  - The rest of the +136 ms p50 is the bind and transfer of the 10,000-element `uuid[]` in each of the two statements, plus executor setup of the array.

## Implications for the design (for the Solution Architect)

1. **D7's 10,000-id cap is above what the list budget allows.** With D7 as
   written, the most |S| that measurably fits is between 100 and 1,000. 1,000
   is marginal against 3.8 ms of headroom.
2. **R1's index `(tenant_id, org_unit_id, updated_at DESC, id DESC)` is not
   the remedy.** The planner chooses it only at |S| = 10, where the plain
   index is already fine. Execution is under 6 ms in every row.
3. **Run a short spike before the closure table.** Two renderings cut
   planning to under 0.5 ms:
   - `ANY((SELECT $S)::uuid[])`
   - `IN (SELECT unnest($S))`

   Both keep "the plan resolves the caller, the row test sees only the row"
   (no second table in the predicate). Both are unmeasured end to end:
   - they still bind 10,000 ids twice;
   - the initplan form hides the values from the planner, which can change
     the plan choice. For example, it may pick the bitmap path at
     |S| = 10,000, which would read about 375k rows.

   A closure-table join avoids binding S entirely, but it still pays for the
   expansion (about 41 ms p95 at about 10,000 units).
4. **Expansion H1 (≤ 5 ms p95) fails.** If S stays in the design, cache the
   subtree per (tenant, org-tree version) or store the closure.

## Setup

- **Code:** harness at `main` `c276a374`.
  - Seed: the unchanged P3-36 harness (`TestRecordsACLPerf800k`), run with `BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$'`.
  - That run's own handler measurement was refused by its floor gate: `FLOOR … p50=8.18 ms > 3 ms gate; no numbers taken`. This report therefore has no same-day handler p95.
- **Data:** P3-36's shape, 800,000 records. The large tenant has 400,000 records, 5,000 users and 50 groups. No runtime code was changed.
- **Region data:** bench database only, `bench/acl-perf/region/region.sql`. It is not a migration.
  - The large tenant's tree is 1 root → 10 regions → 10 districts each → 10 areas each → 9 branches each, 10,111 units in all. Subtree sizes: area 10, district 101, region 1,011, root 10,111.
  - Each user's primary unit is a branch (leaf), chosen by md5 hash. 5% of users have no unit.
  - `records.org_unit_id` is the requester's primary unit (ADR-010 D8). 19,952 of the 400,000 records (5.0%) are NULL. The requester skew (r3²) makes units uneven: n_distinct is 3,254 over 9,000 leaves.
  - After the `UPDATE`, `VACUUM FULL records` rewrote the table, then `VACUUM (ANALYZE)` and `CHECKPOINT` ran, and every relation was `pg_prewarm`ed again before each index phase. The baseline was measured after this rewrite, on the same table.
- **S_k:** the first k units, in pre-order, of the subtree under one node.
  - |S| = 10: area 0/0/0, the whole subtree, 220 records.
  - |S| = 100: district 0/0, first 100 of its 101 units, 3,243 records.
  - |S| = 1,000: region 0, first 1,000 of its 1,011 units, 37,768 records.
  - |S| = 10,000: root, first 10,000 of its 10,111 units, 375,367 records (94%).
- **SQL:** the count and page statements are the handler's own, as captured in `docs/quality/acl-perf-800k/final-2026-11-09-full.md` ("agent list, sort=-updated_at").
  - The bench adds `pf_regional_view` (`all[is_regional_mgr, in_my_region]`, with `is_regional_mgr` folded to `true`) as a fifth `authz.ReadBranches` branch.
  - That branch is last, so it carries `AND NOT` of the four earlier branches, including `NOT EXISTS (record_shares …)`. That is the worst position.
  - The list runs as `nava_app` in the tenant transaction, with `QueryExecModeExec` and `plan_cache_mode = force_custom_plan`. The 200 P3-36 agents take turns.
- **Indexes:**
  - "plain" is `records (tenant_id, org_unit_id)`, D7's index.
  - "plain + R1" adds `(tenant_id, org_unit_id, updated_at DESC, id DESC)`.
- **Method:** P3-36's. 8 clients, 20 s per row, a warm-up of each request (one pass over the 200 agents), one run per row, nearest-rank percentiles (the same `pct` as `records_perf_integration_test.go:1271`). A second pass used 1 client, 10 s per row.
- **Machine:** the P3-36 reference host. Windows 11, i7-1260P, Docker 29.9.2 on WSL2 (kernel 5.15.167.4), VM with 16 CPUs and 24.5 GiB.
  - `postgres:16.10` with P3-36's settings: `shared_buffers=2GB`, `effective_cache_size=6GB`, `max_wal_size=8GB`, `track_io_timing=on`.
  - **Not idle:** another task's 12-container `nava-s18` stack, plus `buildx_buildkit_nava-ci0` and `multica-*`, were up during the whole run. That is outside P3-36's allow-list.
- **Date:** 2026-11-15, 16:40–17:40 UTC.

## Known gaps

- **The host was contended**, so no row is scored against 150 ms on its
  absolute value. The verdict uses the delta against a baseline measured in
  the same database and on the same host. At |S| = 10,000 that delta (+136 ms
  p50 and +158 ms p95 with one client) is far larger than the noise.
- **Database work only, not through the handler.** The handler does not have
  the leaf yet (S31), so these numbers leave out the handler's three small
  lookups and its Go time. Those are the same with or without the leaf.
- **One S per size, all the 200 agents' worst branch position.**

## Reproduce

```
sh bench/acl-perf/region/run.sh          # about 20 minutes: seed, region.sql, the 8-client pass
SP1_PG=<kept container> SP1_CLIENTS=1 SP1_SECONDS=10 sh bench/acl-perf/region/run.sh   # the 1-client pass
```

It needs Docker and Go. It is not in CI. The raw reports, with every plan,
go to `bin/acl-perf-region-raw.md`. The container is kept. Remove it with
`docker rm -f <name>`.
