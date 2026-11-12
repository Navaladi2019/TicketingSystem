---
type: rfc
id: RFC-015
title: Filterable and searchable tenant fields — typed index table, late-flag backfill, free-text search, external store
status: DRAFT (rev 0)
date: 2026-11-11
issue: TICK-714
amends: PLAN-003 (sort and range on data fields), 2026-11-09 D25 decision (data-filter deferral)
author: Solution Architect
---

# RFC-015: Filterable and searchable tenant fields (rev 0)

> **Invariants source.** The Multica project description is empty, so it holds no invariants. I loaded them from `docs/adr/ADR-001-platform-blueprint.md` §1 (I1–I9) on `main`, plus the S10 field-mask rules (`api/paths/records.yaml`, PLAN-003 P3-33/P3-37) and the ADR-004 audit manifest. The checks in §9 are made against that text.
> **Numbering.** The highest RFC in the repo is RFC-014 and the highest ADR is ADR-005. This is RFC-015, and its ADR is assumed to be **ADR-006**.
> **Evidence status.** §1 rows marked *SA shape check* come from a single-client plan-shape check I ran (`docs/adr/ADR-006/spikes/solution-architect/rls-shape/`). They prove plan **shape**, not the 150 ms target. Spike S1, owned by the Postgres & Data Modeling Specialist, gives the P3-01 numbers. Any decision below that depends on S1 says so.

## 0. Summary

1. **Adopt option A, the human's proposal, with four changes.** `records.data` stays the source of truth. A typed side table `record_field_values` holds one row per (record, filterable field, value), with B-tree indexes led by `tenant_id`. **An `AFTER` row trigger on `records` keeps it in step in the same transaction.** The changes: numbers are `double precision`, not `numeric`, because `numeric` comparisons are **not leakproof** in PG 16.10 (catalog checked); the field is named by a registry id, not by text; a type change of a filterable field is refused at publish; and the trigger writes only the fields whose value changed.
2. **The pushdown holds at plan level.** As `nava_app` under forced RLS, `text =`, `= ANY`, `float8 <`, `timestamptz <` and `bool =` on the side table are all **Index Cond** below the RLS One-Time Filter. The same `data ->> 'site'` predicate on `records` is a sequential scan (SA shape check). The p95 at 400k with 8 clients is still owed by S1, and it gates the ADR.
3. **Free-text search: a term side table in Postgres, not `tsvector @@`.** `@@` is not leakproof either, so a GIN index on `tsvector` is unusable for `nava_app` in the same way the JSONB GIN was. `record_search_terms` stores the lexemes of each `searchable` field as B-tree rows (`text =` and prefix range are leakproof). Masked fields are excluded per caller by `field_id`, so S10 holds by construction. There is no relevance ranking and no fuzzy matching. OpenSearch is the upgrade path, and it is a **business decision for the human** (§7).
4. **Late flag:** the field moves `backfilling → ready`, and `removing` when cleared. A Temporal workflow, started through the outbox, backfills in keyset batches of 1,000 under `FOR SHARE`, so a concurrent write can never be overwritten by a stale value. Until the field is `ready`, `filter[field]` answers **422 `field_not_ready`** with progress. **No DDL per tenant or per field. I8 holds, and D25's indexed-attribute path is no longer needed for data filters.**
5. **Rejected:** B (slot columns: about 20 indexes on `records`, and a backfill that rewrites audited rows), C alone (only helps the named fields, and a stopgap we will not need if S1 passes; it is the fallback if S1 fails), D for filtering (moves isolation and authorization out of the database, against I1 and I5). Cassandra does not fit at all. MongoDB buys nothing over A, and its licence is SSPL.
6. **What the human must accept:** a filterable field costs about 320 bytes per record in Postgres (about 2.6 GB at 800k records with 10 fields each), plus write cost that S1 measures against the 560 inserts/s baseline. Search has no relevance ranking or fuzzy matching until an OpenSearch decision is made.

## 1. Context, measured

| Fact | Source |
| --- | --- |
| `records.data jsonb NOT NULL`, no index on it | `db/migrations/20261101190000_records_pipeline.sql:52` |
| `->>` and `@>` not leakproof. GIN control: 9.4 ms as superuser, 292 ms seq scan as `nava_app` | `docs/data/records-list-bench.md` "JSONB indexes under RLS" |
| `filter[priority]=1` p95 841 ms, `filter[site]=s7` p95 1,287 ms (target 150) | TICK-621, P3-36 re-run |
| Today: `total` capped at 1,000, data filter under a 2 s timeout → 422 `filter_too_expensive` | `docs/delivery/decisions/2026-11-09-d25-list-total-cap-and-data-filter-timeout.md`, `internal/records/store/list.go:180` |
| Insert baseline with the P3-01 index set: 560 tps (−18% against PK only, about 4.5% per B-tree) | `records-list-bench.md` "Write throughput" |
| PG 16.10 leakproof: `text`, `int8`, `float8`, `timestamptz`, `date`, `bool`, `uuid` comparisons. **Not** leakproof: `numeric` comparisons, `@@`, `LIKE`, `ILIKE`, every JSONB operator | SA shape check §1 (`pg_proc.proleakproof`) |
| Side table predicates are Index Cond as `nava_app`: text `=` 3.0 ms, `= ANY` 5.9 ms, float8 `<` 44.7 ms (40k rows), timestamptz `<` 2.8 ms, bool `=` 17.5 ms (counts, 200k tenant) | SA shape check §2 |
| Page of 50 via `EXISTS`: 2% filter 55–60 ms, 40% filter 5–10 ms (single client, 200k tenant, warm) | SA shape check §3 |
| Size: about 320 bytes per (record, field) including PK and one typed index | SA shape check §4 |
| Masks are per field, per caller (`acl_versions.masks: [{field, read, write}]`); a masked field in `filter` or `sort` is already 400 `masked_field` | `db/migrations/20261109100000_acl_versions_shape_check.sql:205`, `api/paths/records.yaml` |
| PLAN-003 cut sort and range on `data` fields: "need expression indexes: D25 work" | `docs/delivery/plan-records-path.md:52` |
| Every tenant table must be in `db/audited-tables.yaml`, as audited or exempt with a reason (CI-A3) | `db/audited-tables.yaml` header |
| A type change is already `Breaking` in the classifier; an unknown keyword change is `Breaking` too | `docs/architecture/metadata-engine.md:66` |
| `x-nava-unique` checks scan the JSONB under an advisory lock (no index, D25) | `docs/architecture/record-write-pipeline.md:145-155` |

## 2. Decisions

| # | Decision | Why / what it prevents |
| --- | --- | --- |
| **D1** | **Option A.** `records.data` stays the only source of truth. `record_field_values` is a derived copy, rebuildable from `records.data` at any time, and is never read to build a record payload. | One truth: a side-table bug can mis-filter, but it can never corrupt or reveal a record's content. |
| **D2** | **Field registry.** New tenant table `record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, kind, state, progress, requested_version, updated_at)`. `kind ∈ {filter, search}`, `value_type ∈ {text, num, ts, bool}`, `state ∈ {backfilling, ready, removing}`. Unique `(tenant_id, entity_definition_id, field_key, kind)`. Side rows carry its `id` as `field_id`. | A 16-byte id instead of a text key in every row and index entry. The state lives in a row the trigger reads in-statement, not in a cache. |
| **D3** | **Side table** `record_field_values (tenant_id, field_id, record_id, ord smallint, value_text text COLLATE "C", value_num double precision, value_ts timestamptz, value_bool boolean)`. PK `(tenant_id, record_id, field_id, ord)`. One partial index per type: `(tenant_id, field_id, value_<t>, record_id) WHERE value_<t> IS NOT NULL`. FK `(tenant_id, record_id) → records` with NO ACTION. `ENABLE` + `FORCE` RLS with the canonical policy for `nava_app` and `nava_worker`. Listed under `exempt` in `db/audited-tables.yaml`: "derived from records.data, rebuildable, never a payload source". Exact layout: Postgres specialist. | Every predicate is a leakproof B-tree comparison, so it is pushed below RLS (§1). `ord` makes multi-valued fields (multi-select arrays) one row per element. `COLLATE "C"` makes prefix ranges correct and cheap. |
| **D4** | **Numbers are `double precision`.** Not `numeric` (not leakproof in PG 16.10). Integers are exact to 2^53. Decimals compare as the float8 of both sides, parsed the same way, so `=` is stable. | `numeric` would silently turn every number filter back into a scan for `nava_app`. CI-2 pins it. |
| **D5** | **Maintenance by trigger**, not by the store layer: `record_field_values_sync()`, `AFTER INSERT OR UPDATE OF data ON records FOR EACH ROW`, plus a `BEFORE DELETE` row trigger that removes the record's side rows (an `AFTER DELETE` would run after the NO ACTION FK check, which fires first by trigger-name order, and fail). Both `SECURITY INVOKER`, plpgsql. It reads the type's `record_index_fields` rows in `{backfilling, ready}` in-statement. On update it compares `OLD.data -> key` with `NEW.data -> key` per field and touches only changed fields. A value that does not coerce (`pg_input_is_valid`) writes no row and increments a counter; it never fails the record write. | §3 sets out the trigger vs. store-layer comparison. In short: the in-statement state read closes the backfill race without cache coordination, and every write path is covered. |
| **D6** | **Filter contract.** `filter[f]` on a `data` field: field `ready` → side table via `EXISTS`; `backfilling` → **422 `field_not_ready`** with `index_progress`; not flagged → **400 `field_not_filterable`**; masked → 400 `masked_field` (unchanged, checked first). Operators: `eq`, `in` (≤ 100 values), and **range `lt`, `lte`, `gt`, `gte` for `num` and `ts`**. Range on `text` is out. The legacy JSONB scan and `store.DataFilterTimeout` are deleted once every type is migrated (§6.4). | The answer is fast or a clear error, never a 2 s gamble that depends on tenant size. Range is cheap once the pushdown holds (§1), so the PLAN-003 cut is reversed for range. |
| **D7** | **Sort on a data field stays out** in this ADR. It returns in an amendment if S1 variant A2 (sort key copied into the side table) meets the target at acceptable write cost. | Sorting by a side-table value needs the record's `updated_at` copied into the side table. Then every record update rewrites every side row. That cost is unmeasured. |
| **D8** | **Configuration.** Two schema keywords per property: `"x-nava-filterable": true` and `"x-nava-searchable": true`. Adding or removing either is classified **`Compatible`**. Publishing a version that adds one inserts the registry row as `backfilling` and an outbox row in the same transaction; removing one sets `removing`. | Configuration, not code (I8). The flag lives where the admin already edits the field. The index lifecycle is asynchronous and never blocks publish. |
| **D9** | **A type change of a flagged field is refused at publish** with 422 `filterable_field_type_change`. The admin clears the flag, publishes, waits for removal, changes the type, then flags again. | No mixed-type side rows, no in-place reindex race. It is already a `Breaking` change, so this adds one rule, not a new path. |
| **D10** | **Backfill** is the Temporal workflow `FieldIndexBackfill(tenant_id, field_id)`, started by the relay from the D8 outbox row (I6). One activity per batch: `SELECT id, data FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND id > $cursor ORDER BY id LIMIT 1000 FOR SHARE`, then upsert the batch with `ON CONFLICT (tenant_id, record_id, field_id, ord) DO UPDATE`, and write `progress` and the cursor, in one transaction. The workflow keeps the cursor. It waits between batches (a workflow timer, starting at 200 ms, adjusted to keep live write p95 within budget). One backfill per tenant at a time. When the scan ends, it sets `ready`. | Resumable (cursor), idempotent (upsert), throttled (timer), deterministic (all I/O in activities). `FOR SHARE` serializes with a concurrent `UPDATE`, so a live trigger write and a backfill write to the same record cannot interleave (§5). |
| **D11** | **Removal:** `removing` stops the trigger for that field at once; `filter[f]` answers 400 at once. `FieldIndexRemove` deletes side rows in batches of 5,000, then deletes the registry row. Flagging a field in `removing` answers 409 `field_index_busy`. | Clearing is instant for users and cheap for the database. |
| **D12** | **Search** (§4): `record_search_terms (tenant_id, field_id, term text COLLATE "C", record_id)`, PK `(tenant_id, field_id, term, record_id)`, RLS as D3, exempt from audit as D3. Same trigger, same backfill workflow. Terms are `to_tsvector('simple', value)` lexemes, at most 256 distinct terms per field per record. Query `q` is split the same way; each term is an `EXISTS` with `t.field_id = ANY($readable_searchable)` and `t.term = $term` (or `t.term >= $p AND t.term < $p_next` for a trailing `*` prefix, minimum 3 characters). Terms are ANDed, at most 8. Results are in the list's normal order. | Leakproof B-tree only. The caller's mask set is a plain id list in the query, so a masked field cannot match, count or highlight. |
| **D13** | **Highlights** are computed in Go from the record payload **after** the S10 shaping layer has removed masked fields. | A highlight can only quote what the caller can already read. |
| **D14** | **Quotas (D25 governance, no DDL):** per entity type at most **10 filterable** and **5 searchable** fields; per tenant at most **50 filterable** and **20 searchable**. Publish over quota → 422 `index_quota_exceeded`. Above quota: operator approval recorded as an audited configuration change (`tenant_limits`), the D25 approval path. The numbers are provisional until S1 gives the cost per field. | Write cost and storage grow linearly with flagged fields. A quota is the only bound that a tenant admin cannot route around. |
| **D15** | **Option C is the fallback, not part of the plan.** If S1 misses the target, the ADR revisits promoting named fields to fixed columns (platform DDL, D-3-2 mapping). | It does not need doing if A passes, and it does not generalise to tenant fields. |
| **D16** | **No external store for filtering.** OpenSearch for search only is a deferred option with a named trigger (§7), decided by the human. | Filtering and ACL stay in one database under RLS (I1, I5). |

## 3. Option A in detail: trigger vs. store-layer write (D5)

| Criterion | Trigger (chosen) | Store layer, in the pipeline batch |
| --- | --- | --- |
| Covers every write path | Yes, including `nava_worker` writes, bulk import and repair scripts | Only paths that remember. I4 says there is one pipeline, but a migration that rewrites `data` would bypass it |
| Backfill race (§5) | Reads the field state in-statement, so a write that commits after the `backfilling` flip always copies | Reads the field set from the definition cache. A node with a stale cache writes no row after the flip, and a backfill batch that already passed that record never revisits it. Needs a "wait out every cache" barrier |
| Round trips | None added | None added if appended to the batch |
| Type coercion | plpgsql with `pg_input_is_valid` (PG 16); must never raise | Go with the compiled schema; easier to express |
| Testability | Integration tests against real PG (`testing-ci`); hidden from Go unit tests | Go unit tests plus integration |
| Determinism and replays | No Temporal involvement; an idempotent replay writes nothing, so the trigger does not fire | Same |
| Audit chain | Writes only an exempt table. `records.data` bytes and `audit_log` rows are unchanged (CI-9) | Same |
| Write amplification | Same rows either way; the per-field diff skips unchanged fields | Same |

The race row decides it. The cost of the trigger is opacity: the rule is that the trigger is plain, short, pinned by CI-4 and CI-5, and owns nothing but the copy.

```mermaid
sequenceDiagram
    participant P as Pipeline (nava_app)
    participant R as records
    participant T as trigger record_field_values_sync
    participant F as record_index_fields
    participant V as record_field_values
    P->>R: UPDATE records SET data, version
    R->>T: AFTER UPDATE OF data, row
    T->>F: fields of this type in backfilling or ready
    T->>T: diff OLD.data and NEW.data per field, coerce
    T->>V: delete and insert changed fields only
    P->>P: audit row, outbox row, COMMIT
```

### Write cost (unmeasured, S1 measures)

Each filterable field adds one heap row and two index entries (PK + typed index) on insert, and on an update that changes it. P3-01 measured about 4.5% insert throughput per B-tree. A rough extrapolation for 5 filterable fields (5 heap rows, 10 index entries, one registry read) is a 25–40% insert loss against 560 tps. **This is an estimate, not a number.** S1 measures 0, 5 and 10 fields. The acceptance budget I propose: **insert tps with 5 filterable fields ≥ 70% of the S1 baseline**, and the update path no worse than that. If S1 misses it, the quota in D14 drops before anything else changes.

## 4. Free-text search

| Way | Under RLS for `nava_app` | Verdict |
| --- | --- | --- |
| `tsvector` column + GIN on `records` | `@@` (`ts_match_vq`) is **not leakproof**, so the GIN is never used: the same failure as the JSONB GIN | Rejected (catalog fact) |
| Mark a wrapper of `ts_match_vq` `LEAKPROOF` | Needs a superuser and changes what RLS guarantees, the same call the Security reviewer refused for `jsonb_contains` | Rejected |
| **Term side table** `record_search_terms` (D12) | `text =` and `text <`/`>=` are leakproof: Index Cond below RLS | **Chosen.** S2 must prove latency and write cost |
| OpenSearch | No RLS at all: isolation and masks move into adapter code | Deferred, human decision (§7) |

What the term table gives: whole-word and prefix match, AND of up to 8 terms, per-field scoping, mask-safe counts and highlights. What it does not give: relevance ranking, fuzzy or typo tolerance, stemming (the `simple` configuration is language-neutral; a per-tenant language configuration is an amendment), phrase search.

**Masked fields (S10).** The handler computes `readable_searchable = searchable fields of the type − fields masked for this caller` from the compiled ACL, before the query. A masked field's terms are never in the predicate, so it cannot affect which records match, the count or the highlight. Row-level ACL is the same compiled predicate as the list (I5), so one authorization layer still decides visibility.

**Volume.** A short text field yields about 3–20 terms; a description up to the 256 cap. At 400k records with 3 searchable fields averaging 40 terms, that is about 48M rows: several GB. That is why D14's search quota is half the filter quota and why S2 measures it. Long text fields may be flagged, but the cap truncates their terms (documented as a residual).

## 5. The late flag at 100k+ records

```mermaid
stateDiagram-v2
    [*] --> backfilling: publish adds x-nava-filterable
    backfilling --> ready: backfill scan ends
    backfilling --> removing: publish removes the flag
    ready --> removing: publish removes the flag
    removing --> [*]: rows deleted, registry row deleted
```

**Ordering that makes it race-free:**

1. Publish commits the registry row as `backfilling` and the outbox row in one transaction. From that commit on, every statement that writes a record of the type runs the trigger, which reads `backfilling` and copies the field.
2. The relay starts `FieldIndexBackfill` from the outbox row. Every batch starts after step 1 committed.
3. A batch locks its 1,000 records `FOR SHARE`. A live update that started before the flip still holds its row lock; the batch waits for it, then reads the committed value. A live update after the batch has its trigger copy the new value (the flip is visible to it). A live update during the batch waits for the batch's commit, then its trigger overwrites.
4. The batch upserts. If a live trigger has already written the row, the upsert writes the same value (it read the latest committed data under the lock). Replaying a batch is a no-op upsert.
5. A record inserted after the flip is copied by its own trigger. A record inserted before is in the keyset scan (ids are UUIDv7 and the scan is by `id`, so a late insert with a smaller id is still covered by the trigger, because it committed after the flip).

**Duration (unverified).** At 1,000 rows per batch, about 20–40 ms per batch and a 200 ms pause, 400k records take about 1.5–2 minutes per field. S3 measures it with live writes running.

**Why 422 `field_not_ready` and not a fallback scan.** A fallback makes latency depend on an invisible index state and on tenant size, keeps the non-leakproof JSONB scan in production, and keeps `DataFilterTimeout` alive. A 422 with `index_progress` lets the UI say "Indexing, 43%". The window is minutes, and only after an admin action.

**DDL.** None per field or per tenant. One platform migration creates `record_index_fields`, `record_field_values`, `record_search_terms`, their indexes, policies and the trigger. **I8 holds**, and D25's per-tenant `CREATE INDEX CONCURRENTLY` path is not needed for data filters.

## 6. Options considered

### 6.1 Option A — typed side table + trigger (chosen, see D1–D14)

Lost nothing to the others on isolation (RLS on every table), and it is the only option that needs no per-field DDL **and** no external system. Its costs: storage (about 320 bytes per record per field) and write throughput (S1).

### 6.2 Option B — pre-allocated slot columns (`str_1..N`, `num_1..N`, `ts_1..N`)

Lost because:
- Each slot needs an index to help. Pre-creating them for, say, 10 + 5 + 5 slots puts about 20 B-trees on `records`. At about 4.5% insert throughput per B-tree (P3-01) that roughly halves write throughput for every tenant, whether it flags anything or not. Indexing per tenant on demand is D25 DDL per flag, against the spirit of I8.
- A late flag must write the slot column on existing rows: an `UPDATE` of an **audited** table. Either it goes through the pipeline (100k audit rows, `version` bumps and outbox events for a configuration change) or it bypasses the audit chain. Both are wrong.
- It gives sort for free, which A does not. That alone does not outweigh the above.

### 6.3 Option C — promote common fields to fixed columns

Lost as the plan: it only helps named fields (`priority`, `category`, `site`), which are tenant vocabulary, not platform vocabulary; moving existing values out of `data` rewrites audited rows; and A covers the same filters. Kept as **the fallback if S1 fails** (D15).

### 6.4 Migration from today

There is no tenant data in production that depends on `filter[<data field>]` beyond the bench and acceptance kits (to confirm, open question Q3). The PLAN ships the schema and trigger first, then flags the fields the seed templates (`incident`, `request`, `change`) filter on today, then switches the list handler to D6 and deletes the JSONB filter path.

### 6.5 Option D — an external store fed from the outbox

| Store | Fits ad-hoc tenant filters? | Isolation without RLS | Licence | Verdict |
| --- | --- | --- | --- | --- |
| **Cassandra** | **No.** Queries must follow the partition key; secondary indexes and `ALLOW FILTERING` scan. Each new filter would need a new query table, which is DDL per field | Keyspace or partition key per tenant | Apache 2.0 | **Does not fit.** |
| **MongoDB** | Yes, with one index per field per collection (`createIndex` is per-field DDL by another name, 64-index limit per collection) | Forced `tenant_id` in every query, in code | **SSPL** | Buys nothing over A, adds a second source of truth and a licence question. Rejected |
| **Elasticsearch** | Yes, plus relevance search | Forced tenant filter or alias per tenant, in code | Elastic License 2.0 / SSPL / AGPLv3 (to verify with legal) | Search only, deferred |
| **OpenSearch** | Yes, plus relevance search | As above; document-level security plugin is optional | Apache 2.0 | **The search upgrade path**, deferred, human decision |

Why D loses for filtering:
- **Isolation moves out of the database (I1).** Every query must carry a forced tenant filter added by our adapter. One bug in that adapter leaks across tenants; under RLS the same bug returns nothing.
- **Authorization is implemented twice (I5).** The compiled ACL and field masks would have to be translated into the store's query language and kept in agreement with the SQL compiler. A re-check of candidate ids in Postgres fixes leakage but breaks paging and counts (a page of 50 shrinks to 31 after the re-check; `total` is wrong).
- **Read-your-writes lag.** The outbox relay plus the index refresh is seconds. A user who saves a ticket and returns to a filtered list does not see it. The UI would need a "just saved" overlay or a per-record re-check, which is more code in the core path.
- **A second system to run:** cluster, snapshots, upgrades, reindex on mapping change, on-prem packaging.

For **search only**, the trade is different: relevance and fuzzy matching are things Postgres B-trees cannot do. Search results could still be re-checked in Postgres (ids from OpenSearch, page built under RLS and the compiled ACL), and short pages are more tolerable in search than in filters. That is why §7 asks the human about it, rather than rejecting it.

## 7. Business tradeoffs for the human (listed separately)

| # | Question | Recommendation | What it costs |
| --- | --- | --- | --- |
| H1 | Accept Postgres-only search (word and prefix match, no relevance ranking, no fuzzy, no stemming) for now? | **Yes.** Revisit when a story needs relevance or typo tolerance, or when S2 shows term-table cost above budget | Users get "contains these words", ordered by last update |
| H2 | If and when relevance search is needed: OpenSearch (Apache 2.0) or Elasticsearch (licence to check with legal)? | **OpenSearch**, search only, results re-checked in Postgres | A new component to run: at least 3 nodes for HA, JVM sizing, snapshots, on-prem packaging. **I don't have cost data**; the ops estimate would come with the amendment |
| H3 | Accept the storage cost of option A? | Yes, bounded by the D14 quotas | About 320 bytes per record per filterable field (about 2.6 GB at 800k records × 10 fields), more for search terms (S2 gives the number) |
| H4 | Accept that isolation for filtering stays in Postgres RLS, i.e. no external store for filters? | Yes | None beyond A |

None of A–C weakens tenant isolation. Option D does, and is not recommended for filtering; H2 would bring it back for search only, which is why it is a human decision.

## 8. Risks and accepted residuals

| # | Risk | Owner | Revisit trigger |
| --- | --- | --- | --- |
| R1 | Mid-selectivity filters (about 0.5–5% of a 400k tenant) walk thousands of rows; may miss 150 ms at 8 clients | Postgres specialist (S1) | S1 p95 > 150 ms on that band → test variant A2 (copied sort key) or tighten D14 |
| R2 | Write throughput loss per filterable field | Postgres specialist (S1) | Insert tps with 5 fields < 70% of baseline → lower D14 quotas |
| R3 | float8 equality on decimals beyond 15 significant digits | Solution Architect | A tenant field with `multipleOf` finer than 1e-9 or values beyond 2^53 → `bigint` scaled column |
| R4 | Term table size for long text fields; truncation at 256 terms per field per record | Postgres specialist (S2) | S2 size > 3× records heap at the quota → lower the cap or the quota |
| R5 | A value that fails coercion (older schema version) is not filterable; counted, not raised | Technical Lead (metric and alert) | Counter > 0 for a tenant → admin report |
| R6 | Search has no ranking or fuzzy matching | Human (H1) | A story asks for relevance |
| R7 | Trigger logic hidden from Go unit tests | Technical Lead | Any trigger change must come with CI-4 and CI-5 changes |
| R8 | Runtime roles may write `record_field_values` directly, inside their own tenant, and could store a wrong value | Security reviewer | Reviewer asks for column-level or trigger-only write → `REVOKE` and a dedicated trigger-owner path, without `SECURITY DEFINER` (I3) |

## 9. Invariant check (ADR-001 §1)

| Invariant | A (chosen) | B | C | D |
| --- | --- | --- | --- | --- |
| I1 tenant wall | Holds: RLS forced on 3 new tables, keys led by `tenant_id` | Holds | Holds | **Weakened**: no RLS in the store |
| I3 roles | Holds: trigger `SECURITY INVOKER`, no new role | Holds | Holds | New service credential |
| I4 one pipeline | Holds: the trigger is inside the pipeline's transaction | Backfill bypasses it or floods it | Backfill bypasses it or floods it | Holds |
| I5 one authz layer | Holds: list and search use the compiled ACL; masks as an id list | Holds | Holds | **Weakened**: second ACL implementation |
| I6 outbox and determinism | Holds: backfill and removal start from outbox rows; I/O in activities | — | — | Holds (fed from the outbox) |
| I7 idempotency | Holds: upsert batches, cursor resume | — | — | Indexing must be idempotent per event |
| I8 no tenant DDL | **Holds: zero DDL per field** | Per-tenant indexes or 20 global ones | Platform DDL per promoted field | Mapping changes per field |
| Audit chain (ADR-004) | Holds: side tables exempt with reason; audited bytes unchanged (CI-9) | **Broken or flooded** by backfill | Rewrite of audited rows | Holds |
| S10 masks | Holds: filter 400 `masked_field` first; search excludes masked ids; highlights from shaped payload | Holds | Holds | Must be re-implemented in the store |

## 10. Spikes that gate the ADR

| Spike | Owner | Question | Pass |
| --- | --- | --- | --- |
| **S1** | Postgres & Data Modeling Specialist | P3-01 setup (800k records, one 400k tenant, custom plans, 8 clients, 60 s, PgBouncer): list p95 for `eq` text, `in`, num `lt`, ts range, bool `eq` at 0.01%, 0.25%, 2% and 40% selectivity; single read p95; variant A2 (sort key copied). Insert and update tps with 0, 5 and 10 filterable fields | List p95 ≤ 150 ms, single read ≤ 50 ms, insert tps with 5 fields ≥ 70% of baseline |
| **S2** | Postgres & Data Modeling Specialist | Term table at 400k with 3 searchable fields: 1-, 2- and 3-term query p95, prefix query p95, size, insert tps | p95 ≤ 300 ms (proposed search target), size and tps reported |
| **S3** | Postgres & Data Modeling Specialist | Backfill of one field at 400k with 8 clients writing: duration, live write p95 during it, and a diff of the side table against a recomputation from `records.data` at the end | Diff = 0 rows; live write p95 within +25% |

## 11. Test and acceptance spec

| # | Test | Catches |
| --- | --- | --- |
| CI-1 | `EXPLAIN` as `nava_app` for every operator the filter compiler emits: the value predicate is in Index Cond of a `record_field_values` index, not in Filter | A change that drops the pushdown (a cast, `numeric`, a function on the column) |
| CI-2 | Every operator the filter and search compilers emit has `proleakproof = true` in `pg_proc` | A non-leakproof operator added later |
| CI-3 | Cross-tenant: tenant A's filter and search never match B's rows; `record_index_fields`, `record_field_values`, `record_search_terms` have FORCE RLS and `tenant_id`-led keys (CI-A3, isolation registry rows) | Tenant leakage |
| CI-4 | Property test: random inserts, updates and deletes; then the side tables equal a recomputation from `records.data` for every `ready` field | Trigger drift |
| CI-5 | Backfill race: concurrent updates during backfill, a batch replayed twice, a worker killed mid-batch; final diff = 0 | Missed, duplicate or stale rows |
| CI-6 | Masks: a term only in a masked field gives no hit, the same `total`, no highlight; filter on it gives 400 `masked_field` | S10 leak through search |
| CI-7 | API: `filter` on `ready` → 200, on `backfilling` → 422 `field_not_ready`, unflagged → 400 `field_not_filterable`; publish over quota → 422 `index_quota_exceeded`; type change of a flagged field → 422 `filterable_field_type_change`; flag during `removing` → 409 `field_index_busy`; invalid publish writes nothing; idempotent publish replay writes once; cross-tenant definition → 404 | Contract |
| CI-8 | A record write with flagged fields produces byte-identical `records.data` and `audit_log` rows to the same write with no flags; side tables are in the `exempt` list | Audit chain contamination |
| CI-9 | A row whose value does not coerce: write succeeds, no side row, counter +1 | A coercion error failing a record write |

## 12. Open questions

| # | Question | Who |
| --- | --- | --- |
| Q1 | Is p95 ≤ 300 ms the right target for search, given it is a new feature and not the list? | Technical Lead |
| Q2 | Should `nava_app` lose direct INSERT/UPDATE/DELETE on the side tables (R8)? Without `SECURITY DEFINER`, the trigger runs as the caller, so it needs those grants; the alternative is a dedicated owner path that I3 must allow | Security & Multi-Tenancy Reviewer |
| Q3 | Does any client or acceptance kit depend on filtering by an unflagged data field (the D6 breaking change)? | Technical Lead |
| Q4 | Can the `x-nava-unique` check (today a JSONB scan) use the side table when the field is also filterable? Out of scope here; proposed as a follow-up | Postgres specialist |
