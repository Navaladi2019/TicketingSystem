---
type: adr
id: ADR-006
title: Filterable tenant fields — typed per-type side tables kept by trigger, late-flag backfill behind a snapshot barrier
status: PROPOSED
date: 2026-11-12
issue: TICK-714
rfc: RFC-015 (rev 1, docs/adr/ADR-006/rfc/RFC-015-filterable-searchable-fields.md)
supersedes: 2026-11-09 D25 deferral of data-filter indexing; PLAN-003 cut of range filters on data fields
---

# ADR-006: Filterable tenant fields

## Decision

A tenant field marked `x-nava-filterable` is copied, by a trigger on `records` in the same transaction, into one of four typed side tables. Its filters run as leakproof B-tree comparisons that PostgreSQL pushes below RLS for `nava_app`. `records.data` stays the only source of truth. A field flagged after data exists is backfilled by an outbox-started Temporal workflow behind a snapshot barrier. No DDL runs per tenant or per field. **Free-text search is not decided here.** It moves to RFC-016 (CTO ruling 4). `x-nava-searchable` is reserved: publish refuses it until RFC-016 is accepted.

| # | Decision |
|---|---|
| **D1** | `records.data` is the source of truth. The side tables are derived and rebuildable, and are never read to build a payload. |
| **D2** | Registry `record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version, updated_at)`. `value_type ∈ {text, num, ts, bool}`, `state ∈ {backfilling, ready, removing}`. Unique key `(tenant_id, entity_definition_id, field_key)`. |
| **D3** | **Four side tables, one per value type** (CTO ruling 1): `rfv_text`, `rfv_num`, `rfv_ts`, `rfv_bool`, each `(tenant_id, field_id, record_id, ord smallint, value <type> NOT NULL)`. `text` uses `COLLATE "C"` and `num` is `double precision`, never `numeric`, which is not leakproof. PK `(tenant_id, record_id, field_id, ord)`. Index `(tenant_id, field_id, value, record_id)`. NO ACTION FK to `records (tenant_id, id)` and a **required** NO ACTION FK `(tenant_id, field_id) → record_index_fields (tenant_id, id)`. `CREATE STATISTICS (mcv) ON tenant_id, field_id` on each table, plus `(tenant_id, field_id, value)` on text and bool; bool uses statistics target 10,000 (R12). Every table is `ENABLE` + `FORCE` RLS and listed `exempt` in `db/audited-tables.yaml` ("derived from records.data, rebuildable"). |
| **D4** | Maintenance runs in a plpgsql trigger: `AFTER INSERT OR UPDATE OF data ON records FOR EACH ROW`, plus a `BEFORE DELETE` purge. It is `SECURITY INVOKER` with `SET search_path = pg_catalog, public`, `SET TimeZone = 'UTC'` and `SET DateStyle = 'ISO, YMD'`. For each field of the type in `{backfilling, ready}` whose `data -> key` changed, it runs DELETE then INSERT. The trigger is chosen because it covers every write path. **It is not the race fix; D9 is.** |
| **D5** | Coercion is one `STABLE` function, `record_field_coerce(v jsonb, value_type)`, shared by the trigger and the backfill, with the same `SET` list as D4. `text` requires a JSON string. `num` requires a finite JSON number (no NaN or Infinity, no numeric strings). `bool` requires a JSON boolean. `ts` requires RFC 3339 with an explicit offset, or `YYYY-MM-DD` for `format: date`, stored as 00:00 UTC. A value that does not coerce writes no row and increments a counter. It never fails the write. |
| **D6** | `filter[f]` on a data field is checked in this order:<br>1. masked → 400 `masked_field`<br>2. not flagged → 400 `field_not_filterable`<br>3. `backfilling` → 422 `field_not_ready`, with `index_progress` as a whole percentage read from the registry and no counts<br>4. `ready` → side-table predicate<br>Operators: `eq`, `in` (≤ 100 values); `lt`, `lte`, `gt` and `gte` for `num` and `ts`. **Sort on data fields stays out.** The legacy JSONB scan and `store.DataFilterTimeout` are removed once the seed types are migrated. |
| **D7** | **One compiled query.** A side table is read only as a correlated `EXISTS (… v.tenant_id = r.tenant_id AND v.record_id = r.id …)` inside the single list statement that carries the compiled row ACL. `total` is the existing capped count (1,001) of that same statement. There is no separate count, facet or progress query on a side table. A static test enforces this. |
| **D8** | Flagging is configuration. `"x-nava-filterable": true` on a **top-level** property only (422 `filterable_field_not_top_level` otherwise) is classified `Compatible`. Publishing writes the registry row and an outbox row in one transaction. A type change of a flagged field → 422 `filterable_field_type_change`. Flagging during `removing` → 409 `field_index_busy`. |
| **D9** | **Snapshot barrier after every state flip** (`backfilling`, `ready`, `removing`). An activity takes `pg_current_snapshot()` once, keeps `pg_snapshot_xip` in its heartbeat details, and polls `pg_xact_status` once per attempt. The workflow waits on a timer between polls (1 s, backing off to 10 s); nothing sleeps inside SQL. After 15 minutes it raises `field_index_barrier_slow` and keeps waiting. No extra grant is needed. |
| **D10** | **Backfill** `FieldIndexBackfill(tenant_id, field_id)` is started by the relay from the outbox row. Its inputs come only from that row.<br>1. The barrier runs first.<br>2. Then batches of 1,000, each one `InTenantTx`: (a) `SELECT state … FOR SHARE` on the registry row, stopping unless `backfilling`; (b) keyset `SELECT id, data FROM records … ORDER BY id LIMIT 1000 FOR SHARE`; (c) for each record, DELETE + INSERT of the coerced values; (d) progress %.<br>3. Activities return only the cursor and the progress.<br>4. **The pause between batches starts at 500 ms and adapts** to keep live-write p95 within +25% (CTO ruling 5).<br>5. One backfill runs per tenant at a time.<br>6. At the end: flip to `ready`, then the barrier. |
| **D11** | **Removal** `FieldIndexRemove`:<br>1. Flip to `removing` (filters answer 400 at once).<br>2. Barrier.<br>3. Batched deletes.<br>4. `NOT EXISTS` on the field's side rows, looping back to step 3 if any remain.<br>5. Registry delete, which the D3 FK also guards. |
| **D12** | **Quota** (CTO ruling 3): 5 filterable fields per type by default, and up to 10 with operator approval under D25 governance after a measured cost check (an audited `tenant_limits` change, no DDL). At most 50 per tenant. Over quota → 422 `index_quota_exceeded`. |
| **D13** | **Latency targets** (CTO ruling 2): list p95 ≤ 150 ms for every band except **mid selectivity (0.5–5%) with a narrowing row ACL, which is accepted at p95 ≤ 350 ms**. D7's single `EXISTS` shape is kept. An adaptive shape chosen from a planner estimate is the planned follow-up (trigger in Risks). |
| **D14** | **Grants:**<br>- `nava_app` and `nava_worker`: `SELECT, INSERT, DELETE` on the four side tables, with no `UPDATE`.<br>- Registry: both roles get `SELECT`; `nava_app` gets `INSERT` and `UPDATE (state, progress, requested_version, updated_at)`; `nava_worker` gets `UPDATE (state, progress, updated_at)` and `DELETE`.<br>- One policy per command and per role, never `FOR ALL`, all in the isolation registry.<br>- Every role that writes `records` is in the registry's SELECT policy. |

## Context it rests on

- Under forced RLS, PG 16.10 uses an index only for leakproof comparisons. `->>`, `@>`, `@@` and `numeric` comparisons are not leakproof; `text`, `float8`, `timestamptz` and `bool` comparisons are (RFC-015 §1).
- Today a data filter is a seq scan: `filter[priority]=1` p95 841 ms and `filter[site]=s7` 1,287 ms (TICK-621), held only by the 2 s timeout.
- ADR-001 I8: no tenant-motivated DDL.

## Evidence

**PR #587** (`docs/data/field-index-spike.md`), archived under `docs/adr/ADR-006/spikes/postgres/s1-s3/`. Setup: 800k records, one 400k tenant, 8 clients, custom plans, as `nava_app` under forced RLS.

| Check | Result |
|---|---|
| Pushdown below RLS | `text =`, `= ANY`, `float8 <`, `timestamptz` range and `bool =` are all Index Cond; `numeric =` stays a Filter |
| Estimates, 2% filter | Single table 469 vs 8,024 (17×). **Per-type tables + MCV: 8,000 vs 8,024** (text), 7,976 vs 7,872 (num), 8,085 vs 8,042 (bool). 40% range + ACL: 1,754 → 4.7 ms |
| List p95 ms, ACL / no ACL | 40%: 33–83 / 31–103. 0.25%: 36–86 / 55–96. 0.01%: 19–48 / 21–51. **2%: 221–327 / 99–224** (D13) |
| Insert tps, 5 fields | 338 vs baselines 779 / 533, i.e. 43–63% (D12) |
| Required registry FK | 331 tps with the FK vs 318 without (alternating A/B): no measurable cost |
| Storage | 212 bytes per record-field: 1,700 MB for 8.0M rows |
| Backfill, 400k | Diff 0 alone and under load. 3.0 min alone, 4.6 min under load. Live p95 +31% at a 200 ms pause (D10 now starts at 500 ms). WAL 1.24 MB per 1,000 rows |
| Races | TA R1 (insert across the flip): missing 1 without the barrier, 0 with it. R3 (write across removal): orphan without, or the FK refuses the delete; 0 with the barrier. Re-run independently by the Technical Architect, round 2 (`spikes/technical-architect/r1-r3/`) |

## Consequences

- Data filters on large tenants go from a 0.8–1.3 s scan to tens of ms, except the D13 band, which drops to 221–327 ms.
- Range filters on numbers and dates return.
- Each flagged field costs about 212 bytes per record. Inserts with 5 flagged fields run at about half the unindexed rate (338 tps measured).
- A late flag on 400k records takes about 5 minutes, plus the barrier wait. Its WAL is about 1.24 MB per 1,000 records (`FOR SHARE` header writes), which checkpoints and replicas pay for.
- D25's per-tenant `CREATE INDEX CONCURRENTLY` path is not needed for data filters. I8 holds.

## Rejected options

| Option | Why it lost |
|---|---|
| B: slot columns `str_1..N` on `records` | About 20 B-trees on `records`, a cost every tenant pays whether it flags anything or not. A late flag would rewrite audited rows. |
| C: promote priority, category and site to fixed columns | Only helps fields we name. Moves values out of audited rows. Kept as a fallback only. |
| D: external store (OpenSearch, Elasticsearch, MongoDB, Cassandra) for filters | Isolation and ACL move out of RLS (I1, I5), and saves show up seconds late. Cassandra cannot run ad-hoc filters. MongoDB adds nothing and is SSPL. |
| One shared typed side table (RFC-015 rev 1 D3) | Range estimates are 10× off (about 90% NULL per column), giving side-first plans of 1.3–1.8 s. |
| Store-layer write instead of the trigger | Misses write paths that bypass the pipeline. It needs the same barrier anyway. |
| Fallback JSONB scan while the field is `backfilling` | Latency would depend on index state and tenant size, and the non-leakproof scan would stay in production. |
| UNION-compiled ACL as the only shape | Fixes 2% (82–142 ms) but breaks 40% (341–392 ms). |
| Term table for search (RFC-015 D12) | Multi-term 528–1,161 ms, prefix 1,290–1,371 ms, 3.5 GB, 90 tps. Redesign in RFC-016. |

## Risks and accepted residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-a | The D13 band runs at p95 ≤ 350 ms, not 150 ms. Counts with ACL at 40% take 538–1,255 ms (count-only miss, capped at 1,001) | Technical Lead | Telemetry p95 > 350 ms on that band for a real tenant, or a tenant passes 400k records → build the adaptive shape |
| R-b | Inserts with 5 flagged fields run at 43–63% of baseline (floor 40%) | Postgres specialist | Sustained insert rate passes 50% of measured capacity for any tenant |
| R8 | **Low, accepted.** Runtime roles hold `SELECT, INSERT, DELETE` on the side tables, so a bug can only mis-filter inside its own tenant | Security reviewer | A side row becomes a payload or count source |
| R9 | D6 and D7 rely on masks being row-independent (amendment A2, `internal/authz/mask.go:19`) | Security reviewer | `authz.Mask` gains a row-dependent subject (CI-2b fails) |
| R11 | A long transaction at the flip delays `ready` | Technical Lead | `field_index_barrier_slow` alert |
| **R12** | **Accepted.** Rare values of low-cardinality fields depend on the MCV list. Bool needs statistics target 10,000: 5–11 ms of planning and an 11.8 s `ANALYZE`. Rare combinations can fall out of the list as tenants multiply | Postgres specialist | A filter on a rare value plans a type-wide walk (> 1 s) in telemetry |
| R5 | A non-coercing value is not filterable; it is counted, not raised | Technical Lead | Counter > 0 for a tenant → admin report |
| R3 | float8 decimals beyond 15 significant digits | Solution Architect | A field needs exact decimals beyond 2^53 → `bigint` scaled column |

## Recorded dissent

None. The Technical Architect (round 2) and the Security & Multi-Tenancy Reviewer returned APPROVE. The CTO ruled on S1–S3 with no dissent raised.

## API test scenarios

| Operation | Scenarios |
|---|---|
| `GET /records/{entity}` with `filter[f]` | `ready` → 200 with the correct rows for each operator (`eq`, `in`, `lt`, `lte`, `gt`, `gte`); `backfilling` → 422 `field_not_ready` with a percentage and no count; unflagged → 400 `field_not_filterable`; masked (also while `backfilling`) → 400 `masked_field`; `in` > 100 values or a value of the wrong type → 400 `validation_failed`; a value only in a row-ACL-denied record gives the same hits and `total` as a non-existent value; cross-tenant: tenant A never matches B's rows |
| Publish entity definition (flag added or removed) | Adds the registry row and outbox row once; an idempotent replay writes once; invalid publish (nested property, type change of a flagged field, over quota) writes nothing and returns 422 `filterable_field_not_top_level` / `filterable_field_type_change` / `index_quota_exceeded`; flag during `removing` → 409 `field_index_busy`; `x-nava-searchable` → 422 until RFC-016; another tenant's definition → 404 |

CI pins (from RFC-015 §11):
- CI-1: pushdown `EXPLAIN` per operator.
- CI-2: leakproof catalog.
- CI-2b: masks stay row-independent.
- CI-3: cross-tenant, RLS and FK.
- CI-4: trigger vs. recompute, for every writer role.
- CI-5: R1/R3 and replay races.
- CI-5b: grants.
- CI-5c: no record values in Temporal history.
- CI-6c: row-ACL oracle.
- CI-7: API status codes.
- CI-8: audit bytes unchanged.
- CI-9: coercion across sessions.
