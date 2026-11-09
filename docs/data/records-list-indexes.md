# Records list indexes (P3-34b)

Scope: TICK-640, the follow-up to P3-36's list-target miss (TICK-621,
PR #534). This replaces P3-09's three list indexes with a set led by
`(tenant_id, entity_definition_id)` and turns JIT off on the platform
database. PLAN-003 D25 rules out expression indexes and anything on `data`.

## Verdict

- **Agent and admin lists meet the target.** Every allowed sort, page 1 and
  page 20, is an ordered index scan with no `Sort`. Database time is
  1.2–23 ms. One exception is the agent's `sort=assignment_group_id` page 1,
  covered under the requester lists below.
- **Requester lists meet the target only once the SQL changes.** With
  today's OR-shaped ACL predicate, the lightest heavy requester takes
  38–166 ms: no `Sort`, but every page misses 50 ms. When the `mine` rule is
  written as a `UNION ALL` of index-ordered branches, it takes 2.0–4.0 ms
  over `records_list_requester` and `records_list_assignee`. That is SQL
  work for P3-34a / P3-31 (see Hand-offs). No index fixes it alone.
- **JIT is off.** No list statement compiles any more. It had added
  420–990 ms to each statement.
- **`count(*)` still misses.** It takes 0.75–1.5 s for agents and
  requesters, and 108–149 ms for the admin, with no JIT. That is P3-34a's
  capped-count path, not an index question.

## Index set

Six B-tree indexes. Each sort column ends in `DESC`, so a descending list
reads the index forward and an ascending list reads it backward.

| Index | Key | Serves |
| --- | --- | --- |
| `records_list_updated` | `(tenant_id, entity_definition_id, updated_at DESC, id DESC)` | the default sort `-updated_at`, `sort=updated_at` |
| `records_list_created` | `(tenant_id, entity_definition_id, created_at DESC, id DESC)` | `sort=[-]created_at` |
| `records_list_state` | `(tenant_id, entity_definition_id, state DESC, updated_at DESC, id DESC)` | `sort=[-]state`; `filter[state]=s` under the default sort |
| `records_list_group` | `(…, assignment_group_id DESC, updated_at DESC, id DESC)` | `filter[group]`; `sort=[-]group`; the `records_assignment_group_fkey` RI check |
| `records_list_requester` | `(…, requester_id DESC, updated_at DESC, id DESC)` | `filter[requester]`; the requester branch of the `mine` rule; `sort=[-]requester` |
| `records_list_assignee` | `(…, assignee_id DESC, updated_at DESC, id DESC)` | `filter[assignee]`; the assignee branch of the `mine` rule; `sort=[-]assignee` |

Dropped: `records_tenant_updated`, `records_tenant_requester` and
`records_tenant_group_open`. Only `store.List` read them. `Get`, the write,
cascade and outbox locks use `records_pkey`. The unique-value check and
numbering use `records_number_key`. The history reads `audit_log`.

```mermaid
erDiagram
    entity_definitions ||--o{ records : "tenant_id, entity_definition_id"
    groups ||--o{ records : "assignment_group_id (records_list_group)"
    principals ||--o{ records : "requester_id (records_list_requester)"
    principals ||--o{ records : "assignee_id (records_list_assignee)"
    records ||--o{ record_shares : "tenant_id, record_id"
    records {
        uuid tenant_id PK
        uuid id PK
        uuid entity_definition_id "every list binds it"
        text state "records_list_state"
        uuid requester_id
        uuid assignee_id
        uuid assignment_group_id
        timestamptz created_at "records_list_created"
        timestamptz updated_at "records_list_updated, tiebreak of the others"
        jsonb data "no index (D25)"
    }
```

### Changes to the shared contract (TICK-639 follows these)

1. **State and foreign-key sorts tie-break on `updated_at`.**
   - The order is `<col> DIR NULLS FIRST|LAST, updated_at DIR, id DIR`, and
     the keyset is `(r.<col>, r.updated_at, r.id) < ($v, $u, $id)`.
     Timestamp sorts keep `<col> DIR, id DIR`.
   - Why: the contract's `(<col>, id)` order is not the access-path index's
     order. With the contract's set, `sort=assignee` was a full-tenant `Sort`
     at 134–296 ms.
   - The alternative was a separate `(<col>, id)` index for each of the 4
     columns, which would make 10 indexes.
2. **`records_list_state` is shaped like an access path,**
   `(state, updated_at, id)` and not `(state, id)`. It also serves
   `filter[state]=new` under the default sort, in 1.2–7.4 ms.
3. **The access-path indexes lead with `<col> DESC`,** so the sort rule
   ("descending reads forward, ascending reads backward") holds for them
   too.
4. **The NULL segment** of a nullable sort column is its own branch:
   `r.<col> IS NULL AND (r.updated_at, r.id) < (…)`. The bench measured it
   as an index condition, with no `Sort`.

## Evidence

On P3-01's 800k shape (`scratch/tick-640/setup.go`), the same RNG seed,
distributions and tenant sizes as P3-36:

- 400,000 records in one tenant, 399 tenants of 1,002–1,003 records, 15%
  open, 50 groups.
- A heap of 1,042 MB and 15,815 shares.
- The real migrations and RLS, run as `nava_app` with
  `SET LOCAL ROLE` and `app.tenant_id` set.

The statements are the contract's shape (`scratch/tick-640/explain.sh`),
and every plan is `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)` on a warm cache
(`pg_prewarm`). Raw plans are in `scratch/tick-640/plans-*.md`; the `UNION ALL` plans are in `plans-union.md`.

- **Host:** Windows 11, i7-1260P, 39.7 GB RAM. Docker 29.9.2 on WSL2 with
  16 vCPU and 19.4 GiB, the same host as P3-01 and P3-36.
- **Server:** `postgres:16.10` with `shared_buffers = 2GB`,
  `effective_cache_size = 6GB`, `max_wal_size = 8GB`, and the defaults for
  everything else. Before, `jit = on`. After, the database has
  `jit = off`.

### Page statement, database time in ms (page 1 / page 20)

| List | Before: P3-09 set, JIT on | After: this set, JIT off |
| --- | --- | --- |
| agent, `-updated_at` | 10.7 / 10.6 | 8.6–22.9 / 8.6 |
| agent, `-created_at` | **1,424 / 1,058** (Sort; JIT 927 / 681) | 12.7 / 12.3 |
| admin, `-updated_at` | 1.5 / 2.6 | 2.6 / 2.2 |
| admin, `-created_at` | **1,212 / 1,680** (Seq Scan + Sort) | 3.2 / 3.1 |
| heaviest requester (5,568 records), `-updated_at` | 38.7 / 52.8 | 38.0 / 43.7 |
| lightest heavy requester (1,026), `-updated_at` | **239 / 124** | **105 / 114** |
| lightest heavy requester, `-created_at` | **873 / 804** (Sort) | **166 / 141** |
| lightest heavy requester, `mine` as `UNION ALL` | – | **4.0 / 2.8** (about 380 buffers, was 18,791) |
| heaviest requester, `mine` as `UNION ALL` | – | **2.0 / 2.0** |

The other sorts after the change, page 1 / page 20, with no `Sort` in any
plan:

| Sort | Agent | Admin | Heaviest req. | Lightest heavy req. |
| --- | --- | --- | --- | --- |
| `-state` | 11.1 / 9.5 | 1.2 / 1.7 | 34.2 / 49.6 | 138 / 128 |
| `-assignee` | 13.6 / 9.4 | 1.3 / 1.7 | 24.7 / 36.2 | 106 / 108 |
| `-requester` | 11.0 / 9.8 | 1.4 / 2.2 | 44.0 / 1.7 | 160 / 3.7 |
| `-assignment_group` | **176** / 2.0 | 1.2 / 1.7 | 28.0 / 26.2 | 87 / 133 |

Filters under the default sort (page 1, agent / admin):

| Filter | Agent | Admin |
| --- | --- | --- |
| `state = new` | 7.4 | 1.2 |
| `state IN (new, in_progress)` | 31.7 | 4.0 |
| `assignee = p` | 1.3 | 1.8 |
| `group = g` | 1.2 | 1.0 |
| `requester = p` | 5.9 | 1.2 |

An ordered walk under an OR-shaped ACL costs in proportion to how rare the
caller's rows are in sort order:

- An agent sees about 6% of the tenant, so 51 rows need about 900 index
  entries.
- The lightest heavy requester sees 0.28%, so 51 rows need about 18,000
  index entries and their heap rows. The `EXISTS` arm stops the planner from
  `BitmapOr`-ing the `requester_id` / `assignee_id` indexes.
- The agent's `sort=assignment_group_id` page 1 has the same cause: the
  first groups in sort order are not the agent's.

Only rewriting the SQL removes the walk.

### `count(*)` (agent / requesters / admin, ms)

| | Before (JIT on) | After (JIT off) |
| --- | --- | --- |
| agent | 888–1,022 (JIT 538–655) | 789–1,182 |
| requesters | 719–1,615 (JIT 419–848) | 753–1,471 |
| admin | 120–194 | 108–149 |

The count reads every row the caller can see, and with an OR predicate it
reads the whole tenant (400k heap rows). After the change, the planner reads
it through `records_list_created`. That index is uncorrelated with heap order
in this seed, which sets `created_at` at random. In production, rows are
inserted in time order. A 10,000 cap still reads about 166k rows for an agent
who sees 6% of the tenant. A capped count therefore helps the admin and the
requesters, the latter through the `UNION ALL` form, more than it helps the
agents.

### JIT

- **Before:** every list statement that read the whole tenant cost about
  1.3M, far over `jit_above_cost` (100k).
  - JIT added 420–990 ms per statement in this run, and 200–430 ms in
    P3-36.
  - A custom plan (`force_custom_plan`, `QueryExecModeExec`) is compiled
    again on every execution.
- **After:** page statements cost 137–700, so they are under the threshold
  either way.
  - The count still costs 1.3M, and JIT added 300–330 ms to it with JIT on
    (single runs).
- **P3-01 did not assume JIT was off.** Its report records `jit = on`, and
  its index walks stayed under the threshold. The relay (TICK-250) already
  shapes its claim to stay under `jit_above_cost`.
- **Where the setting lives:** the database, not the roles (`db/README.md`,
  JIT). `db/grants.yaml` is unchanged.

### Write amplification and storage

For write cost, the bench ran pgbench of one record `INSERT` in the large
tenant (`scratch/tick-640/insert.sql`): 8 clients, 60 s, `fsync` and
`synchronous_commit` on. Triggers and FKs were off, because the index set
changes neither, so the difference is index maintenance:

| | Indexes on `records` | p50 | p95 | tps | WAL per insert |
| --- | --- | --- | --- | --- | --- |
| P3-09 set (2 runs) | 5 | 8.0 / 8.0 ms | 21.5 / 11.0 ms | 846 / 915 | 2,215 / 2,193 B |
| This set (3 runs) | 8 | 7.5 / 8.4 / 8.0 ms | 29.4 / 23.8 / 19.6 ms | 762 / 643 / 836 | 2,880 / 3,022 / 2,832 B |

- **WAL:** +30% per insert (about +700 B).
- **Throughput:** 10–20% lower.
- **INSERT p95:** about 5–10 ms higher. Commit `fsync` on WSL2 dominates
  it, so the runs spread widely.
- **Updates:** every pipeline update moves `updated_at`, so it touches all
  6 list indexes. No update is HOT, as before (P3-09:
  `records_tenant_updated` already indexed `updated_at`).

Storage at 800k:

- The three dropped indexes took 148 MB. The six new ones take 402 MB:
  59 + 59 + 66 + 73 + 73 + 72 MB, so +254 MB, or about 330 B per record.
- `CREATE INDEX CONCURRENTLY` took 2.0–7.1 s per index over 800k rows.
- The DROP migration is catalogue-only.

### Foreign-key checks

- **Group:** with the new set, the RESTRICT check of
  `records_assignment_group_fkey` scans the tenant's range of
  `records_list_group`, with no heap reads, in 73 ms. Before, it scanned the
  whole tenant: 99 ms after E5 (`scratch/e5-groups-contract`), 502 ms under
  `SET NULL` (RFC-011).
- **Principal:** for a hard delete, the RI check now takes 91 ms. Before, it
  was a direct probe on `records_tenant_requester`.
  - Principals are tombstoned, and only offboarding hard-deletes them.
  - `assignee_id` had no index before.

## Not measured

- **Create p95 through the handler (D-3-9 trigger).**
  - The index set changes only the records `INSERT`. Measured above, it adds
    about 5–10 ms at p95 for a statement with p95 11–29 ms.
  - P3-36 has no create scenario, and this task did not build one. The full
    create p95 needs a handler-level run, for example a create scenario in
    `TestRecordsACLPerf800k`.
- **Through PgBouncer and under concurrent read load.** Every plan above is a
  single unloaded run.

## Reproduce

```
docker run -d --name tick640-pg --shm-size=3g -p 127.0.0.1:55640:5432 \
  -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench postgres:16.10 \
  -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB -c track_io_timing=on
# before: skip this task's migrations, seed, plans
go run scratch/tick-640/setup.go -url postgres://pgadmin:bench@127.0.0.1:55640/postgres \
  -seed -skip records_list_,drop_records_s9,platform_database_jit
COUNT=1 OUT=scratch/tick-640/plans-before.md sh scratch/tick-640/matrix.sh -updated_at -created_at
# after: apply them over the seeded rows, plans
go run scratch/tick-640/setup.go -url postgres://pgadmin:bench@127.0.0.1:55640/postgres
COUNT=1 OUT=scratch/tick-640/plans-after.md sh scratch/tick-640/matrix.sh -updated_at -created_at
TIE=updated_at OUT=scratch/tick-640/plans-after-attr.md sh scratch/tick-640/matrix.sh -state -assignee_id -requester_id -assignment_group_id
sh scratch/tick-640/union.sh <requester-uuid> [page-20 cursor]
SECS=60 sh scratch/tick-640/insert-bench.sh <label>
```

## Hand-offs

- **P3-34a (TICK-639), list SQL:**
  - Follow the order and keyset changes above.
  - To bring requester lists under 50 ms, emit the `mine` rule as a
    `UNION ALL` of three branches: requester, assignee not requester, and
    shared not either. Each branch has `ORDER BY <sort> LIMIT n` inside it,
    and the outer query merges them (`scratch/tick-640/union.sh`). That can
    sit in `store.List` or in `authz.CompileRead` (P3-31). The Technical
    Lead decides which.
  - The capped count is still needed.
- **Security & Multi-Tenancy Reviewer:** every new index leads with
  `tenant_id` (pinned in `TestP309RecordsIndexSet`). In every plan above,
  `tenant_id` is the leading index condition, and the RLS policy is checked
  once per statement (`One-Time Filter`).
