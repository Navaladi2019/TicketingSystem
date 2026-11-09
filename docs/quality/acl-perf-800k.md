# ACL list and read performance at 800k (P3-36)

Scope: TICK-621, PLAN-003 P3-36 (`docs/delivery/plan-records-path.md`) and
the D25 escalation. The question: with the real compiled ACL predicate
(P3-31, `authz.CompileRead`) inside the real list SQL (P3-34,
`records/store.List`), is list p95 ≤ 150 ms and single-read p95 ≤ 50 ms at
800k records?

## Re-run, after TICK-639 and TICK-640

**Run:** 2026-11-09, on `TICK-336/group-acl-masking` at `1aa5357e`. That
head has TICK-639 (list SQL in index order), TICK-640 (list indexes, `jit =
off`) and TICK-637 (`total` capped at 10,000). The later story-branch commit
`01fc85b4` adds only isolation tests. The harness, host and settings are the
same as the first run. Targets are unchanged.

**Raw reports:**
- `docs/quality/acl-perf-800k/rerun-2026-11-09.md`: every scenario, 8
  clients, 20 s each. Transport floor p50 2.65 / p95 4.85 ms.
- `docs/quality/acl-perf-800k/rerun-2026-11-09-repeat.md`: the GET and create
  rows again on a fresh database (`NAVA_PERF_ONLY='^(single GET|create)'`).
  Transport floor p50 1.65 / p95 2.76 ms.

### Verdict

**Not met.** Large-tenant lists, creates and the small-tenant single read
still miss. The page statement of a fixed-column list now meets the target on
its own (2 to 4 ms single-run). What remains has three causes: the capped
count, the `data` filters and the create path. Per the CTO's conditions,
these numbers go on TICK-336 and the task stops there.

### Reads

p95 in ms, per request, 8 clients. `count` is `total`'s
`SELECT count(*) FROM (… LIMIT 10001)`. `page` is the page statement. Both are
p95 under load, then single-run `EXPLAIN ANALYZE` in brackets.

| Scenario | Request p95 | Target | `count` p95 (single) | `page` p95 (single) | Class |
| --- | --- | --- | --- | --- | --- |
| Agent list by group, page 1 | **443.1** | 150 | 344.6 (244.1) | 36.2 (4.4) | count only |
| Agent list by group, page 20 | **367.0** | 150 | 294.3 (121.8) | 31.7 (2.5) | count only |
| Agent list, `sort=-updated_at` | **343.3** | 150 | 271.8 (103.4) | 29.7 (1.9) | count only |
| Agent list, `sort=assignee` | 290.2 | outside target | 237.2 (109.5) | 17.8 (2.1) | foreign-key sort, outside the target (option (b) on TICK-639); count only |
| Agent list, `filter[state]=new` | **473.8** | 150 | 210.9 (40.4) | 186.4 (27.5) | load: both statements |
| Admin list, `state in (new,in_progress)`, `sort=-updated_at` | **481.9** (p50 92.6) | 150 | 141.8 (34.5) | 145.7 (2.5) | load: tail only |
| Admin list, page 1 | 89.9 | 150 | 11.0 (5.5) | 24.3 (0.8) | met |
| Admin list, page 20 | 89.2 | 150 | 13.4 (6.1) | 23.1 (0.6) | met |
| Agent list, `filter[priority]=1` (data, 5%) | **840.5** | 150 | 704.2 (282.0) | 95.3 (26.5) | data filter |
| Agent list, `filter[site]=s7` (data, 0.5%) | **1,286.8** | 150 | 607.3 (274.9) | 692.9 (268.4) | data filter |
| Requester list, page 1 | **712.7** | 150 | 596.6 (634.5) | 92.1 (16.8) | known miss, TICK-644 |
| Requester list, page 20 | **698.9** | 150 | 566.0 (206.3) | 91.8 (7.3) | known miss, TICK-644 |
| Small tenant: agent list | 101.6 | 150 | 16.2 (1.6) | 27.3 (1.3) | met |
| Small tenant: admin list | 68.2 | 150 | 7.8 (0.7) | 19.2 (0.6) | met |

What the plans show:

- **The count is the agent-list miss.** The count walks
  `records_number_key (tenant_id, entity_definition_id, number)`, filters
  with the ACL predicate, and stops at 10,001 matches. An agent sees about
  24,700 rows, so the count reads about 161,000 rows (28,278 buffers): 100 to
  244 ms single-run. The page walks `records_list_updated` and stops after 51
  rows (736 removed): 2 to 4 ms. A 10,000 cap already reads too much at this
  shape. PLAN-003's count-only fallback is the cap itself, so this is the
  case D25 says does not escalate. The request still misses 150 ms, so the
  CTO decides.
- **`data` filters scan the tenant.** `priority=1` walks the whole tenant in
  the count (398,792 rows removed). `site=s7` also misses in the page: 155,922
  rows removed before 51 matches, 268 ms single-run. Neither filter is
  indexable under RLS (P3-01). This is a `data`-filter miss, so per the
  CTO's conditions it reopens D25.
- **`state=new` and the admin `state in` list are load misses.** Single-run,
  both statements take 2.5 to 40 ms. Under 8 clients the agent `state=new`
  page reaches p95 186 ms. The admin list's p50 is 92.6 ms with a 482 ms p95
  tail. Both are fixed-column lists over 150 ms at p95.
- **Foreign-key sorts are outside the target** (option (b) on TICK-639).
  `sort=assignee` is reported but not judged. Its miss is count-only anyway.
- **The count alone misses.** With `total_capped`, the result is bounded at
  10,000, but the work isn't. To find 10,001 matches the count walks about
  161,000 rows, because the ACL predicate rejects about 94% of what it reads.
  On the agent lists this is the only miss: request p95 290 to 443 ms, of
  which the count is p95 237 to 345 ms and the page p95 18 to 36 ms.
- **Requester lists (TICK-644, known miss).** Against 150 ms per request:
  p95 699 to 713 ms. Against TICK-644's 50 ms database time for the page
  statement: p95 92 ms under load (single-run 7 to 17 ms). The count costs
  more than the page: p95 566 to 597 ms. A heavy requester has about 5,600
  rows, below the cap, so the count reads the whole tenant (394,374 rows
  removed). TICK-644 leaves the count unchanged (its scope: "the total count
  is unchanged"), so the `UNION ALL` rewrite fixes the page only. Even with
  TICK-644 the request misses on the count, the same count-only miss as the
  agent lists.

### Single read

| Scenario | p95, re-run | p95, repeat | Target |
| --- | --- | --- | --- |
| GET, agent | **122.4** | 47.6 | 50 |
| GET, admin | **99.7** | 48.8 | 50 |
| GET, small-tenant agent | **60.7** | **55.3** | 50 |

A GET is 13 statements. Each of them takes ≤ 0.3 ms in the database
(`records_pkey` and primary-key lookups). The request is round trips: p50
32 to 37 ms is about 13 times the floor. In the re-run the floor's p99 was
12 ms and every GET statement's p95 was 6 to 15 ms, so the host was noisy.
In the repeat, agent and admin meet and the small-tenant GET misses by
5 ms. The single-read result depends on the host's transport, not on the
query.

### Create (D-3-9 trigger, 50 ms)

| Scenario | Request p95, re-run / repeat | Counter wait p95 | Counter hold p50 / p95 | Hold: transport (est.) / database / handler, p50 |
| --- | --- | --- | --- | --- |
| Agent, 1 client | **83.4** / **98.6** | 2.8 / 3.2 | 21.7 / 36.9; 29.2 / 42.4 | 29.1 / 0.0 / 1.2; 18.1 / 9.5 / 1.2 |
| Agent, 8 clients | **470.6** / **647.8** | 413.4 / 580.8 | 22.7 / 30.2; 30.6 / 41.6 | 29.1 / 0.0 / 1.1; 18.1 / 11.1 / 1.2 |
| Admin, 8 clients | **1,017.4** / **858.4** | 727.6 / 771.7 | 29.1 / 87.5; 35.5 / 70.5 | 29.1 / 0.0 / 1.2; 18.1 / 15.8 / 1.3 |
| Small-tenant agent, 8 clients | **424.6** / **370.9** | 13.0 / 12.8 | 61.5 / 182.5; 78.7 / 154.9 | 29.1 / 30.2 / 1.8; 18.1 / 57.7 / 2.3 |

Where two values share a cell, the first is from the re-run and the second
from the repeat.

- **One create alone misses 50 ms.** It runs 29 statements in two
  transactions, the idempotency claim and then the write. With 1 client and
  no queue (wait p95 about 3 ms), p95 is 83 to 99 ms. The database part of
  the counter hold is 0 to 10 ms. The rest is round trips: 29 × the floor's
  p50 is 48 to 77 ms on this host before any database work.
- **On one tenant, creates queue on the counter.** With 8 clients, the
  `record_counters` upsert waits p95 413 to 772 ms for the row. Each holder
  keeps the row for 11 statements, through INSERT, write key, audit, outbox
  and COMMIT: p50 23 to 36 ms. Seven waiters × that hold is the wait. That is
  D-3-9's design ("the lock window is short") meeting 11 round trips.
- **The small-tenant creates don't queue** (wait p95 13 ms). Their hold has
  more database time, 30 to 58 ms at p50, from COMMIT: the 8 clients write
  to different tenants and share the WAL flush (`fsync = on`). The COMMIT
  p99 was 696 ms in the re-run.

A create miss reopens D-3-9 (the CTO's conditions).

## Verdict (first run)

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
`NAVA_PERF_CLIENTS` (8), `NAVA_PERF_OUT`, `NAVA_PERF_ONLY` (a regular
expression over scenario names, for example `^create`), `BENCH_KEEP=1`.

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

For that re-run the harness has scenarios that are not in the 2026-11-09
numbers above:

- **Agent list, `filter[site]=s7`, the rare `data` value.** `site` is one of
  200 values, each 0.5% of rows, so it matches less than 1% of an agent's
  visible rows. It sits next to the existing `priority=1` row (5%). The seed
  derives `site` from `category`'s random draw, so every other column and
  the RNG stream are unchanged. The raw report's Data section prints the
  large tenant's `site = s7` count.
- **Create, end to end, against the D-3-9 trigger (p95 ≤ 50 ms)**, added at
  the Technical Lead's request. TICK-640 (PR #539) adds six indexes and
  measured only the INSERT statement: its p95 went from 11–22 ms to
  20–29 ms. These scenarios measure the whole `POST /api/v1/records/ticket`
  instead: authn, idempotency, validation, the ACL write check, numbering,
  the INSERT, the audit row and COMMIT. They run as an agent (into one of
  its own groups) and as the admin on the large tenant, and as a
  small-tenant agent. The body is P3-01's record shape, about 1 KB of
  `data`. Every request carries a fresh `Idempotency-Key`. The creates run
  after every read scenario, so the rows they add never reach a read. The
  raw report lists each create's statements and per-statement timings; its
  writes are not EXPLAINed, because `EXPLAIN ANALYZE` would run them again.
  8 clients on one tenant all take that tenant's `record_counters` row
  (D-3-9), so the large-tenant create p95 includes that lock wait.
- **Create, agent (1 client)**, next to the 8-client row. With one client
  no other create queues on the counter row, so this row is the create path
  alone. D-3-9's trigger states no concurrency; the two rows together show
  whether a miss comes from the path or from the counter queue.
- **Counter hold**, a table in the raw report for every create scenario. Each
  traced statement carries its start offset in the request. For each create,
  the window runs from the end of the `record_counters` upsert (the row is
  locked) to the end of the `COMMIT` that releases it. The table prints:
  - the upsert itself (the wait for the row plus its execution);
  - the hold, and how many statements run inside it;
  - the hold split into transport (statements in the window times the
    transport floor's p50, an estimate), database (the rest of their
    statement time) and handler time (between statements).

- **Transport floor.** Before the scenarios, 8 clients run `SELECT 1` 200
  times each on the handler's pool, and the report prints its p50/p95/p99.
  A create runs 29 statements and a list 10, so a request's floor is about
  its statement count times that p50. The per-statement tables are read
  against it.

A create-only smoke on 2026-11-09 (`NAVA_PERF_ONLY='^create'`, 5 s per
scenario, story branch before TICK-639/TICK-640, host shared with other
work) checked the scenario, not the target: every request returned 201.
Two things from it are for the re-run to confirm or clear, not results:

- The `record_counters` upsert waited p50 409–565 ms and p95 about 3 s on
  the large tenant. 8 creates on one tenant and type queue for one counter
  row, held from the upsert to COMMIT (D-3-9).
- Every round trip, `SELECT current_setting(...)` included, took about 13 ms
  at p50, against 3.4 ms in the 2026-11-09 list run. At 29 statements that
  is about 400 ms per create before any database work. The transport floor
  above exists to separate this from the database.

Known miss in the re-run: requester lists until **TICK-644** (P3-34c, the
requester list as a `UNION ALL` of index-ordered branches) lands. TICK-640
left them at 38–166 ms of database time. The re-run reports them against
their unchanged target with a link to TICK-644, not as a new finding.

## What this does not cover

- PATCH. Create is measured from the re-run on (above); PATCH is not.
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
