# ACL list and read performance at 800k (P3-36)

Scope: TICK-621, PLAN-003 P3-36 (`docs/delivery/plan-records-path.md`) and
the D25 escalation. The question: with the real compiled ACL predicate
(P3-31, `authz.CompileRead`) inside the real list SQL (P3-34,
`records/store.List`), is list p95 ≤ 150 ms and single-read p95 ≤ 50 ms at
800k records?

## Verdict

**Single read: met.** p95 41.0 to 44.6 ms across the three GET scenarios.

**List: missed, and it is not a count-only miss.** Every list on the
400k-record tenant misses: p95 is 660 ms to 5,044 ms against 150 ms. The page
statement alone, without the `total` count, has p95 400 to 3,300 ms under
load, and 430 to 1,740 ms in a single unloaded `EXPLAIN ANALYZE`. Both
small-tenant lists meet the target (p95 86 and 90 ms).

Per PLAN-003, this miss triggers the D25 escalation. It is reported on
TICK-336 for the Technical Lead. This task does not tune indexes or relax the
target.

## Method

- **Path: through the HTTP handlers.** `TestRecordsACLPerf800k`
  (`internal/server/records_perf_integration_test.go`, build tags
  `integration perf`) sends `GET /api/v1/records/ticket` and
  `GET /api/v1/records/ticket/{id}` to the real `server.NewHandler`: strict
  server, authn, entity-type store, ACL policy read, `CompileRead`,
  `planList`, `store.List`, shaping and JSON. Each latency is the
  whole handler round trip.
- **The SQL is captured, never written by hand.** A pgx `QueryTracer` on the
  handler's own pool records every statement the request ran, with its
  arguments and duration. For each scenario, the first request's `SELECT`s
  are re-run as `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)`, with the same text and
  arguments, as `nava_app` in the same tenant transaction.
- **Data: P3-01's shape** (TICK-339, `bench/records-list/02_seed.sql`) on the
  real migrated schema: the same RNG seed, distributions and tenant sizes.
  One tenant has 400,000 records and 399 tenants have 1,002 or 1,003 each, for
  800,000 in total. 15.0% are open, 50 groups per tenant, 3 groups per user,
  about 1 KB of `data`, and 15,815 user shares (2%). Every tenant's entity
  type is created with `POST /api/v1/entity-types`. The ACL is published with
  `acltest.Publish`. Records, principals, groups and shares are
  bulk-inserted. Every tenant's records belong to its one entity type, so
  each list runs over the whole tenant, as P3-01's did.
- **ACL** (the OR-combined shape of P3-01's query (d), in the real rule
  language):
  - `desk`: the agents group reads `assignment_group_id IN my_groups`.
  - `mine`: requester, assignee or shared.
  - `admins`: the admin group reads everything.
- **Actors:**
  - 200 large-tenant agents (P3-01's actors: agents group plus 3 assignment
    groups).
  - The large tenant's 8 heavy requesters (more than 1,000 records each).
  - The large tenant's admin.
  - One agent and the admin of each small tenant.
  - Page 20 is keyset page 20, the 951st row onwards, reached by following
    the handler's own cursors.
- **Load:** 8 concurrent clients, 20 s per scenario, after a warm-up of each
  request. Percentiles are nearest-rank, as in P3-01. All relations are
  `pg_prewarm`ed.

### Reproduce

```
make bench-acl-perf        # or: task bench:acl-perf
```

It needs Docker and Go and takes about 32 minutes on the host below. It is not
in CI. It starts `postgres:<pin>` with P3-01's server settings, runs the test
and writes the raw report to `bin/acl-perf-800k-raw.md`. The test fails when
any scenario misses its target. Env: `NAVA_PERF_SECONDS` (20),
`NAVA_PERF_CLIENTS` (8), `NAVA_PERF_OUT`, `BENCH_KEEP=1`.

Raw output for this run, with every statement, argument and plan:
`docs/quality/acl-perf-800k/run-2026-11-09.md`.

### Hardware and server

- Host: Windows 11, 12th Gen Intel Core i7-1260P, 39.7 GB RAM. Docker 29.9.2
  on WSL2 (kernel 5.15.167.4), with a VM of 16 CPUs and 19.4 GiB. This is the
  same host as P3-01.
- Load source: the Go test process on Windows, connected to PostgreSQL through
  Docker Desktop's port forward. There is no PgBouncer, unlike P3-01.
- `postgres:16.10`: `shared_buffers = 2GB`, `effective_cache_size = 6GB`,
  `max_wal_size = 8GB`, `track_io_timing = on`. Everything else is the
  default: `work_mem = 4MB`, `random_page_cost = 4`, `jit = on`,
  `max_parallel_workers_per_gather = 2`.
  - The raw file prints `max_parallel_workers_per_gather = 0` because it read
    the seeding session. That is fixed for later runs.
- `nava_app` runs with `plan_cache_mode = force_custom_plan` (db-roles) and
  the pool in `QueryExecModeExec`, so every plan is a custom plan.
- Noise: the idle `nava` and acceptance stacks were running on the same
  Docker VM. Each statement round trip costs about 3 to 5 ms at p50 under
  load: `begin` and `set_config` alone take that long. A request runs 10 (list)
  or 12 (GET) statements, so roughly 35 to 50 ms of each request is transport,
  not query. That makes the GET numbers pessimistic. It does not explain the
  list misses, which are seconds.

## Results

p50 / **p95** / p99 in ms, per request through the handler, 8 clients.

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Agent list by group, default sort, page 1 | large | 48 | 3,291.8 | **5,013.4** | 5,200.2 | 150 | **miss** |
| Agent list by group, default sort, page 20 | large | 48 | 3,274.5 | **5,044.4** | 6,180.0 | 150 | **miss** |
| Requester list (heavy requesters), page 1 | large | 56 | 2,981.7 | **3,364.1** | 3,577.6 | 150 | **miss** |
| Requester list (heavy requesters), page 20 | large | 57 | 2,985.7 | **3,282.5** | 3,374.1 | 150 | **miss** |
| Admin list, default sort, page 1 | large | 71 | 2,428.8 | **2,645.7** | 2,698.8 | 150 | **miss** |
| Admin list, default sort, page 20 | large | 63 | 2,687.7 | **3,318.0** | 3,806.9 | 150 | **miss** |
| Agent list, `filter[state]=new` | large | 264 | 590.4 | **747.4** | 823.8 | 150 | **miss** |
| Agent list, `filter[priority]=1` (data) | large | 51 | 3,314.6 | **3,787.3** | 4,545.6 | 150 | **miss** |
| Agent list, `sort=-updated_at` | large | 56 | 3,071.4 | **3,592.8** | 3,697.5 | 150 | **miss** |
| Agent list, `sort=assignee` | large | 70 | 2,431.3 | **2,729.5** | 3,008.8 | 150 | **miss** |
| Admin list, `filter[state][in]=new,in_progress`, `sort=-updated_at` | large | 295 | 545.0 | **660.2** | 702.5 | 150 | **miss** |
| Agent list, page 1 | small | 3,073 | 46.9 | **86.5** | 107.9 | 150 | met |
| Admin list, page 1 | small | 2,739 | 53.0 | **90.1** | 123.5 | 150 | met |
| Single GET, agent | large | 5,223 | 29.0 | **41.0** | 55.4 | 50 | met |
| Single GET, admin | large | 4,982 | 30.9 | **43.4** | 56.5 | 50 | met |
| Single GET, agent | small | 4,847 | 31.3 | **44.6** | 59.2 | 50 | met |

Every list returns `total`. The raw report splits each list into its two
heavy statements. Statement 8 is the `count(*)` for `total`, and statement 9
is the page:

| Large-tenant scenario | `count(*)` p95 | Page p95 | Single-run `EXPLAIN ANALYZE` page (of which JIT) |
| --- | --- | --- | --- |
| Agent, default sort, page 1 | 2,602 | 2,376 | 550 ms (377) |
| Agent, default sort, page 20 | 1,786 | 3,297 | 561 ms (381) |
| Requester, page 1 | 1,464 | 2,054 | 530 ms (405) |
| Admin, default sort, page 1 | 581 | 2,017 | 774 ms (23) |
| Admin, default sort, page 20 | 789 | 2,561 | 1,740 ms (30) |
| Agent, `filter[state]=new` | 302 | 400 | 134 ms (55) |

The page statement misses on its own, so a capped count cannot rescue it.
This is a full D25 trigger, not the count-only fallback.

## Why: the plans

The compiled ACL predicate itself is not the cost. The share `EXISTS` runs
once per statement, as a hashed subplan (`SubPlan 2`, `loops=1`, 13
buffers), and the small-tenant lists over the same predicate take 1 to 8 ms
in the database. The cost is in the shape of the list SQL around it.

Agent list, default sort, page 1, page statement (excerpt):

```
Limit (actual time=546.845..546.876 rows=51 loops=1)
  Buffers: shared hit=70049
  ->  Sort  Sort Key: r.created_at DESC NULLS LAST, r.id   Sort Method: top-N heapsort
        ->  Hash Left Join (groups g)
              ->  Nested Loop
                    ->  Index Scan using records_number_key on records r (rows=24666)
                          Index Cond: ((tenant_id = '8b89…') AND (entity_definition_id = 'd028…'))
                          Filter: (assignment_group_id = ANY (…) OR requester_id = … OR assignee_id = … OR (hashed SubPlan 2))
                          Rows Removed by Filter: 375334
JIT: … Total 377.234 ms
Execution Time: 550.007 ms
```

Admin list, default sort, page 20: `Seq Scan on records r … rows=399050`, then
a sort, at 1,740 ms.

1. **No list reads in index order. Each one reads the whole tenant and
   sorts.** The builder orders by `r.<col> <dir> NULLS LAST, r.id ASC`.
   P3-01's walk used `records_tenant_updated (tenant_id, updated_at DESC, id
   DESC)` with `ORDER BY updated_at DESC, id DESC`. A `DESC` index is
   `NULLS FIRST`, and the tiebreaker here runs `ASC`, so the index's order is
   never the requested one, even for `sort=-updated_at`. The planner instead
   walks `records_number_key (tenant_id, entity_definition_id, number)` to
   apply `entity_definition_id = $2`, reads 400k rows (70k buffers) and
   top-N sorts.
2. **The default sort is `created_at`, which no index covers.** P3-01
   measured and indexed `updated_at`. `planList`'s `defaultSortField` is
   `created_at`.
3. **The keyset predicate is not an index condition.** Page 20 renders
   `(col < $v OR col IS NULL OR (col = $v AND r.id > $id))`, not a row
   comparison `(col, id) < ($v, $id)`. Page 20 therefore costs as much as page
   1 or more: 1,740 ms for the admin.
4. **`total` is an exact, uncapped `count(*)`.** It reads every matching row:
   24,666 for an agent and 400,000 for the admin. PLAN-003's fallback for a
   count miss is a capped count. The builder has none.
5. **JIT compiles every large-tenant list statement.** Plans that read the
   whole tenant cost about 1.35M, far above `jit_above_cost` (100k). JIT then
   spends 200 to 430 ms per statement. That is two thirds of a single-run
   agent page. With JIT off, the remaining scan and sort would take 150 to
   190 ms per statement, two statements per request, and would still miss
   150 ms. That figure is **derived** (execution time minus JIT time), not
   measured with `jit = off`. JIT is a multiplier on cause 1, not a cause of
   its own.
6. **Concurrency multiplies the single-run cost.** 8 clients each reading
   400k rows twice per request is CPU-bound: 550 ms single-run becomes
   3 to 5 s at p95.

The JSONB filter (`filter[priority]=1`) cannot use an index under RLS, as
P3-01 found. It reads the whole tenant anyway, like the unfiltered list.

## Re-run (CTO's D25 decision)

The CTO decided D25 on TICK-336: no D25 work now, and the targets stay.
P3-36 runs again on this harness once TICK-639 and TICK-640 have merged. The
re-run covers every large-tenant row and both `data` selectivities, with the
targets unchanged. A miss goes back on TICK-336 with its plans.

For that re-run the harness has one more scenario, which is not in the
2026-11-09 numbers above:

- **Agent list, `filter[site]=s7`, the rare `data` value.** `site` is one of
  200 values, each 0.5% of rows, so it matches less than 1% of an agent's
  visible rows. It sits next to the existing `priority=1` row (5%). The seed
  derives `site` from `category`'s random draw, so every other column and
  the RNG stream are unchanged. The raw report's Data section prints the
  large tenant's `site = s7` count.

## What this does not cover

- Write paths (create, PATCH). The D-3-9 create p95 trigger is out of scope
  here.
- Through PgBouncer. Handler to PostgreSQL is direct, through Docker's port
  forward. The pooler adds queueing (P3-01) and would only make the lists
  slower.
- Filters on more than one data property, `in` filters over data, and sorts
  on `state` or `requester_id`. They share the plan shape above (whole tenant,
  then sort) and would not change the verdict.
- A CI perf job. The run takes about 32 minutes and holds about 4 GB of
  Docker memory, so it is a documented run (`make bench-acl-perf`), as P3-01
  was.

## Hand-offs

- **Technical Lead:** D25 escalation to the CTO (PLAN-003). List p95 misses
  150 ms on every large-tenant scenario, and the page statement misses on its
  own. The numbers and plans are in this document and in
  `docs/quality/acl-perf-800k/run-2026-11-09.md`.
- **Owner of P3-34 (`records/store.List`, `planList`):** causes 1 to 4 are
  drift between the list SQL and the query shape P3-01 measured and P3-09
  indexed. The CTO's D25 decision should settle the fix before anyone changes
  it. This task does not.
- **Principal Software Engineer:** reviewer of this measurement.
