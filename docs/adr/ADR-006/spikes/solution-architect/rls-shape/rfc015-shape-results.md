# RFC-015 plan-shape check (Solution Architect, 2026-11-11)

**What this is:** a plan-shape check, not the P3-01 benchmark. It answers one
question: does PostgreSQL 16.10 put the side table's typed predicates in the
**Index Cond** for `nava_app` under `FORCE ROW LEVEL SECURITY`? The latency
benchmark (800k records, 400k tenant, 8 clients, p95) is spike S1, owned by
the Postgres & Data Modeling Specialist.

**Setup:** throwaway `postgres:16.10` container, defaults. `nava_app`
`LOGIN NOBYPASSRLS`, not the owner, `plan_cache_mode = force_custom_plan`,
the platform policy `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid`
on both tables. 399,500 records: one tenant with 200,000, 399 tenants with
500. Four fields copied to `record_field_values` (1.6M rows): `site` text
(50 values), `priority` float8 (1..5), `due` timestamptz, `vip` bool.
Scripts: `rfc015-shape.sql`, `rfc015-page.sql`. Single client, laptop, so the
times are indicative only.

## 1. Operator catalog (PG 16.10, `pg_proc.proleakproof`)

| Operator | Function | Leakproof |
| --- | --- | --- |
| `text = <> < <= > >=` | `texteq`, `text_lt`, … | **true** |
| `bigint`, `double precision`, `timestamptz`, `date`, `boolean`, `uuid` comparisons | `int8eq`, `float8lt`, `timestamptz_lt`, `booleq`, … | **true** |
| `numeric = < …` | `numeric_eq`, `numeric_lt`, … | **false** |
| `jsonb ->>`, `@>`, `?`, `jsonb =` | `jsonb_object_field_text`, `jsonb_contains`, … | false |
| `tsvector @@ tsquery` | `ts_match_vq` | **false** |
| `text LIKE`, `ILIKE` | `textlike`, `texticlike` | false |
| `starts_with(text, text)` | | true (no index support for `nava_app` checked) |

**Consequence:** `value_num` must be `double precision` (or `bigint`), never
`numeric`. Postgres FTS (`@@`) cannot use a GIN index for `nava_app`, for the
same reason JSONB cannot.

## 2. Pushdown, large tenant (200k), counts

| Predicate | Plan as `nava_app` | Time |
| --- | --- | --- |
| `value_text = 's7'` (2%) | Index Only Scan `rfv_text`, value in **Index Cond** | 3.0 ms |
| `value_text = ANY('{s7,s8}')` | Index Only Scan `rfv_text`, `= ANY` in **Index Cond** | 5.9 ms |
| `value_num < 2` (20%) | Index Only Scan `rfv_num`, in **Index Cond** | 44.7 ms (40k rows counted) |
| `value_ts < now() + 10 h` | Index Only Scan `rfv_ts`, in **Index Cond** | 2.8 ms |
| `value_bool = true` (5%) | Index Only Scan `rfv_bool`, in **Index Cond** | 17.5 ms |
| Control: `records.data ->> 'site' = 's7'` | **Parallel Seq Scan** on `records`, `->>` in Filter | 126 ms |

The RLS policy appears as a `One-Time Filter` above the scan in every case,
as in P3-01: the leakproof value predicate goes below it.

## 3. Page query (`EXISTS` on the side table, `ORDER BY updated_at DESC, id DESC LIMIT 50`)

With `records_tenant_updated (tenant_id, updated_at DESC, id DESC)`, warm,
two runs each:

| Filter | Plan | Time |
| --- | --- | --- |
| `site = s7` (2%) | Parallel walk of `records_tenant_updated` (≈ 4,900 rows), `rfv_text` probe per row | 54.9 / 59.7 ms |
| `priority < 3` (40%) | Walk of `records_tenant_updated` (169 rows), PK probe per row | 9.9 / 5.5 ms |

Without `records_tenant_updated` the 2% case drove from `rfv_text`, joined
4,000 rows and sorted them: 64 to 69 ms warm, 130 ms cold.

**Reading:** the pushdown holds for every operator class the brief names. The
cost band to watch is a filter matching about 0.5 to 5% of a large tenant,
where neither the walk nor the side index stops early. S1 must measure it at
400k with 8 clients, with and without a copied sort key (RFC-015 §4 A2).

## 4. Size

1.6M side rows (4 fields × 400k records): heap 154 MB, PK
`(tenant_id, entity_definition_id, field_id, record_id)` 174 MB, each typed
partial index 47 to 48 MB (one holds only its own type's rows). About
**320 bytes per (record, filterable field)** including indexes. The records
heap in this check was 73 MB (small `data`). Dropping `entity_definition_id`
from the keys (the field id implies it) saves about 32 bytes per row.
