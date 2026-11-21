<!-- Archived verbatim from TICK-1040 (thread 01a1fcaa), 2026-11-21. -->

## Critique of ADR-010-A2 (PR #818): the switch idea holds, but three of its bounds fail on data SP-1c's seed didn't have

**Overall: 3 blocking, 2 non-blocking, 5 confirmed sound.** The closure, `Allows`, the `OFFSET 0` fence and the visibility invariant hold. The probe, the â‰¤ K shape, the count statement and the row bound in A2-7 don't hold as written. I measured each failure with a spike on PG 16.10 (report below). Each one has a cheap fix, and the fixes belong in A2 and in SP-1d's contract.

I also have to correct my own SP-1c report. Known 3 said "C2 and W0 each force one shape whatever the statistics say". That is wrong for C2 (finding B2).

### Findings

| # | A2 rule | Verdict | Evidence | Known / Inferred |
|---|---|---|---|---|
| B1 | A2-5 probe, and the C2 bound at â‰¤ K | **blocking** | `records_org_unit` is `(tenant_id, org_unit_id)` with no `entity_definition_id` (`db/migrations/20261120120400_records_org_unit_index.sql:25-26` on `TICK-946/in-my-region`). A2-5 adds `entity_definition_id = $e` to the probe, so the probe and C2's bitmap read the records of **every entity type** under S. Spike T1a: a district with 900 tickets and 200,000 records of another type gives a probe of **52.7 ms, with 200,000 index rows removed by the entity filter**. With the key `(tenant_id, org_unit_id, entity_definition_id)` the same probe takes **0.65 ms, index-only, 37 buffers**. SP-1c could not see this, because its tree held tickets only: "any type" equals "tickets" in every probe row of `sp1c-raw.md`. | Known (spike, index DDL, raw) |
| B2 | A2-5 "â‰¤ K â†’ C2", "C2 forces one shape" | **blocking** | C2 hides S's size, but it does not fix the plan. Spike T1b: with 900 tickets, so the probe says â‰¤ K, C2 planned the **`records_list_updated` walk, 11,001 rows removed, 11.6 ms**. That is the plan the â‰¤ K side exists to avoid. With the walk disabled, C2's bitmap on 1,100 records takes 0.75 ms (T6). Which plan C2 gets depends on the planner's default estimate for `= ANY($0)` against the walk's cost, so it is the same class of fragility that rejected C1. | Known (spike). Why it flips: Inferred |
| B3 | A2-5 "count and page use the same shape" with W0; A2-7 "â‰¤ about 1,000 records at every size" | **blocking** | **(a) The count statement.** In SP-1c, the region branch of the W0 count is `[never executed]` in all six sets (`sp1c-raw.md`, the "W0 count" rows), because branch 1 had already filled the `LIMIT 1001`. So SP-1c never measured it. For a caller whose first four branches give fewer than 1,001 rows, spike T2 (1,100 tickets spread evenly in time, among 401k): **1,035 ms, 370,849 closure probes**. The same capped count in the probe's closure-driven rendering takes **1.0 ms** (T2b). **(b) The page.** W0 reads about page Ã— N_e / m rows, where N_e is the entity's records and m is the subtree's. T3, at m = 1,100: **8,651 rows, 28 ms**. SP-1c's 949 rows at district come from how its seed clusters a unit's records in time, not from the mechanism. | (a) Known. (b) Known at one point; the formula is Inferred |
| N1 | A2-1 / A2-2 build SQL | **non-blocking** | Confirms the Solution Architect's added point. SP-1c's build seeds every live unit, so a live unit below a tombstoned one keeps its rows: **1 orphan row, 3,179 rows vs. 3,169** under the live-roots build (T8, SQL below). Keep the `closure_includes_tombstoned_path` mutant. The 234â€“347 ms rebuild was measured on the wrong build; I expect the same order of cost (Inferred). | Known |
| N2 | A2-2 maintenance under concurrency | **non-blocking** | Two groupsync transactions rebuilding the same tenant at READ COMMITTED: B's `DELETE` cannot see A's new rows, so B's `INSERT` fails on the PK, or one rebuild is lost. TICK-1033 should take a per-tenant lock first (`pg_advisory_xact_lock(hashtext(tenant_id::text))` or `FOR UPDATE` on the tenant row) and state it, unless groupsync already serialises per tenant. | Inferred |
| C1 | A2-6: C2, W0 and `Allows` give the same rows | **confirmed sound** | `descendant_id` is `NOT NULL`, so `= ANY(ARRAY(â€¦))` is never NULL once `org_unit_id IS NOT NULL`. `ARRAY(subquery)` over zero rows is `'{}'`, not NULL, so an empty U is false in every shape and `not under` is true. `EXISTS` is never NULL. | Known (SQL semantics) |
| C2 | A2-5: the `OFFSET 0` fence on 16.10 | **confirmed sound** | Reproduced: T2 and T3 both plan a `SubPlan` with an Index Only Scan on `org_unit_closure_pkey`, with no semi-join. | Known |
| C3 | Custom plans (`force_custom_plan`) represent production | **confirmed sound** | Every pgx pool in `cmd/` uses `QueryExecModeExec` (for example `cmd/nava-core/allocator.go:63` and `cmd/nava-groupsync/main.go:184`), so statements are unnamed and are planned per execution. | Known |
| C4 | A2-4 `Allows` as a closure-PK `EXISTS`, with the S plumbing deleted | **confirmed sound** | SP-1b â‰¤ 0.32 ms p95. Deleting `MaxUnderSet` is safe only after B3 is fixed: as written, the unbounded cost moves from "refuse" to "1 s count". | Known / Inferred |
| C5 | A2-1 contract (FKs, FORCE RLS, grants) | **confirmed sound** | Matches TICK-1032. Cross-tenant registry row required. | Known |

### Proposed changes to the A2 text
1. **A2-1 / TICK-947 migration (unmerged):** make the `records_org_unit` key `(tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL`. The `(tenant_id, org_unit_id)` prefix still serves the `records_org_unit_fkey` NO ACTION check. It is the same index count, wider by one uuid.
2. **A2-5, count statement:** the region branch of the capped count always uses the probe's rendering (closure-driven, capped), whatever the probe said. Only the page statement switches. A2-6 still holds, because the predicate is the same.
3. **A2-5, the â‰¤ K shape:** replace "C2" with "a rendering whose plan is fixed by construction", and let SP-1d choose between:
   - **C3:** C2 inside a `MATERIALIZED` CTE, with the top-N outside it;
   - **L1:** a per-unit `LATERAL` top-N on a new index `(tenant_id, entity_definition_id, org_unit_id, updated_at DESC NULLS FIRST, id DESC)`. T7: **0.6 ms, 250 rows**; it reads at most |S| Ã— page rows whatever the time distribution. L1 costs one more index on `records` (write amplification to be measured).
4. **A2-7:** delete "â‰¤ about 1,000 records at every size". State the bounds that actually hold: the â‰¤ K side reads â‰¤ K (C3) or â‰¤ |S| Ã— page (L1); W0 reads about page Ã— N_e / m. Because of that, a fixed K = 1,000 is right for one tenant size only. The worst point sits just above K and grows with N_e: 28 ms at 401k, and about ten times the rows at 4M (Inferred). This is the Solution Architect's call: either K comes from a measurement per tenant-size class, or the branch above K also gets a bounded shape.
5. **A2-7, the unprobed default:** W0 is the wrong default, because it is the shape with no bound on a sparse S (T2, T3). Since no real policy has more than one distinct U, make it a **D4 publish-time rule: at most 4 distinct `under` U per policy**, and drop the request-time path. "A request never refuses" still holds.
6. **Archived SP-1c report:** add an erratum line to the header of `docs/adr/ADR-010/A2/sp1c/report.md`: *"Known 3's claim that C2 forces one plan is wrong; see the A2 critique (spike T1b)."*

### SP-1d (TICK-1041): additions to its contract
TICK-1041 has no assignee yet. I'll run it once it is assigned. These cases need to be in its contract first, because without them a pass on the six P3-36 subtrees cannot certify A2:
- **(4) Mixed entity types:** a district with â‰¤ K tickets plus â‰¥ 100k records of another type, on both index keys. Pass: probe â‰¤ 1 ms, and the â‰¤ K shape reads â‰¤ K records.
- **(5) Time-uniform sparse subtrees** at m âˆˆ {K + 1, 3K, 10K}. Pass: the page's region branch Î” â‰¤ 3.8 ms.
- **(6) A count-only caller:** the first four branches give < 1,001 rows, at all six sizes plus (5). Pass: the count's region branch is executed and costs â‰¤ 1 ms.
- **(7) The â‰¤ K shape:** C2, C3 and L1, each with plan-stability evidence on two `n_distinct` values.

### Open questions
1. **Disjoint branches:** defer until SP-1d (2) reports. Making them a compiler rule changes CI-6's text, so it should rest on a number.
2. **The 4-probe cap:** see change 5. 4 is fine as a publish-time cap; the unprobed W0 default is not.
3. **Side channel:** for the Security & Multi-Tenancy Reviewer. One input from me: as written, B3(a) makes the timing difference about 1 s, not about 1 ms. Fixing the count removes most of the signal.

<details><summary>Spike report (T1â€“T8): setup, method, rerun</summary>

**Verdict:** the hypotheses "the A2-5 probe reads â‰¤ K + 1 records", "C2 plans a bitmap at â‰¤ K" and "W0 count and page read â‰¤ ~1,000 rows" are each **falsified** on at least one data shape.

**Setup:** `postgres:16.10` in Docker (16 CPUs, host shared with other containers). `max_parallel_workers_per_gather = 0`. Superuser with no RLS: this measures plan shape and rows read, not budgets.
- Tree: one tenant, root â†’ 100 districts â†’ 10 leaves each (1,101 units). District 2 is tombstoned.
- Records: 400k tickets spread over districts 2â€“99. District 0 has 1,100 tickets. District 1 has 900 tickets plus 200k records of entity 2. `updated_at` is uniform over 365 days.
- Indexes: `records_org_unit (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL` and `records_list_updated (tenant_id, entity_definition_id, updated_at DESC NULLS FIRST, id DESC)`.
- Single runs after one warm-up run. The numbers are orders of magnitude, **not budget-grade**.

| Test | Shape | Data | Result |
|---|---|---|---|
| T1a | A2-5 probe (with `$e`) | district 1 (900 tickets + 200k other type) | 52.7 ms; 11 Ã— 18,182 rows removed by `entity_definition_id` |
| T1aâ€² | same, index `(tenant_id, org_unit_id, entity_definition_id)` | same | 0.65 ms, index-only, 37 buffers |
| T1b | C2 page (`LIMIT 25`) | district 1 | **walk** on `records_list_updated`, 11,001 removed, 11.6 ms |
| T6 | C2 page with `enable_indexscan = off` | district 0 | bitmap, 1,100 rows, 0.75 ms |
| T2 | W0 capped count (`LIMIT 1001`), region branch alone | district 0 | 1,035 ms, 370,849 SubPlan loops |
| T2b | capped count in the probe rendering | district 0 | 1.0 ms, 1,001 rows |
| T3 | W0 page (`LIMIT 25`) | district 0 | 28.1 ms, 8,651 rows read |
| T7 | L1 per-unit `LATERAL` top-N | district 0 | 0.6 ms, 250 rows |
| T8 | closure build: SP-1c vs. from live roots | orphan leaf under tombstoned district 2 | SP-1c: 1 orphan row (3,179 total); live roots: 0 (3,169 total) |

Live-roots build (A2-1):
```sql
WITH RECURSIVE live(id) AS (
  SELECT id FROM org_units WHERE tenant_id = $t AND parent_id IS NULL AND deleted_at IS NULL
  UNION ALL SELECT o.id FROM live l JOIN org_units o ON o.tenant_id = $t AND o.parent_id = l.id AND o.deleted_at IS NULL),
c(ancestor_id, descendant_id) AS (SELECT id, id FROM live
  UNION ALL SELECT c.ancestor_id, o.id FROM c JOIN org_units o ON o.tenant_id = $t AND o.parent_id = c.descendant_id AND o.deleted_at IS NULL)
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) SELECT $t, ancestor_id, descendant_id FROM c;
```

Setup:
```sql
SET max_parallel_workers_per_gather = 0;
CREATE TABLE org_units (tenant_id uuid, id uuid PRIMARY KEY, parent_id uuid, deleted_at timestamptz, k int);
INSERT INTO org_units VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-00000000aaaa', NULL, NULL, -1);
INSERT INTO org_units SELECT '00000000-0000-0000-0000-000000000001', gen_random_uuid(), '00000000-0000-0000-0000-00000000aaaa', NULL, d FROM generate_series(0,99) d;
INSERT INTO org_units SELECT p.tenant_id, gen_random_uuid(), p.id, NULL, 1000 + p.k*10 + l FROM org_units p, generate_series(0,9) l WHERE p.k BETWEEN 0 AND 99;
CREATE TABLE org_unit_closure (tenant_id uuid, ancestor_id uuid, descendant_id uuid, PRIMARY KEY (tenant_id, ancestor_id, descendant_id));
CREATE TABLE records (tenant_id uuid NOT NULL, id uuid NOT NULL DEFAULT gen_random_uuid(), entity_definition_id int NOT NULL, org_unit_id uuid, updated_at timestamptz NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TEMP TABLE leaves AS SELECT row_number() OVER (ORDER BY k) - 1 AS i, id, k FROM org_units WHERE k >= 1020;
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
SELECT '00000000-0000-0000-0000-000000000001', 1, l.id, now() - random()*interval '365 days' FROM generate_series(1,400000) g JOIN leaves l ON l.i = (g % 980);
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
SELECT '00000000-0000-0000-0000-000000000001', 1, (SELECT id FROM org_units WHERE k = 1000 + g % 10), now() - random()*interval '365 days' FROM generate_series(1,1100) g;
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
SELECT '00000000-0000-0000-0000-000000000001', 1, (SELECT id FROM org_units WHERE k = 1010 + g % 10), now() - random()*interval '365 days' FROM generate_series(1,900) g;
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
SELECT '00000000-0000-0000-0000-000000000001', 2, (SELECT id FROM org_units WHERE k = 1010 + g % 10), now() - random()*interval '365 days' FROM generate_series(1,200000) g;
CREATE INDEX records_org_unit ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;
CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC NULLS FIRST, id DESC);
UPDATE org_units SET deleted_at = now() WHERE k = 2;
INSERT INTO org_unit_closure   -- SP-1c build
WITH RECURSIVE c(tenant_id, ancestor_id, descendant_id) AS (
  SELECT tenant_id, id, id FROM org_units WHERE deleted_at IS NULL
  UNION ALL SELECT c.tenant_id, c.ancestor_id, o.id FROM c JOIN org_units o ON o.tenant_id = c.tenant_id AND o.parent_id = c.descendant_id AND o.deleted_at IS NULL)
SELECT * FROM c;
VACUUM ANALYZE records; VACUUM ANALYZE org_unit_closure;
```

Queries (`:d0` / `:d1` = district 0 / 1 id, `:t` = tenant), each run under `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)`:
```sql
-- T1a probe
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = :t AND r.entity_definition_id = 1 AND r.org_unit_id IS NOT NULL
  AND r.org_unit_id IN (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = :t AND ancestor_id = ANY(ARRAY[:'d1']::uuid[])) LIMIT 1001) p;
-- T1b / T6 C2 page
SELECT id FROM records r WHERE r.tenant_id = :t AND r.entity_definition_id = 1 AND r.org_unit_id IS NOT NULL
  AND r.org_unit_id = ANY(ARRAY(SELECT descendant_id FROM org_unit_closure WHERE tenant_id = :t AND ancestor_id = ANY(ARRAY[:'d1']::uuid[])))
  ORDER BY updated_at DESC NULLS FIRST, id DESC LIMIT 25;
-- T2 W0 count (T3: same predicate, ORDER BY updated_at DESC NULLS FIRST, id DESC LIMIT 25)
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = :t AND r.entity_definition_id = 1 AND r.org_unit_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM org_unit_closure c WHERE c.tenant_id = :t AND c.ancestor_id = ANY(ARRAY[:'d0']::uuid[]) AND c.descendant_id = r.org_unit_id OFFSET 0) LIMIT 1001) p;
-- T7 L1 (index records_ou_list ON records (tenant_id, entity_definition_id, org_unit_id, updated_at DESC NULLS FIRST, id DESC) WHERE org_unit_id IS NOT NULL)
SELECT x.id FROM org_unit_closure c CROSS JOIN LATERAL (SELECT r.id, r.updated_at FROM records r
  WHERE r.tenant_id = :t AND r.entity_definition_id = 1 AND r.org_unit_id = c.descendant_id
  ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 25) x
WHERE c.tenant_id = :t AND c.ancestor_id = ANY(ARRAY[:'d0']::uuid[]) ORDER BY x.updated_at DESC NULLS FIRST, x.id DESC LIMIT 25;
```
**Rerun:** `docker run -d --name sp-a2 -e POSTGRES_PASSWORD=x postgres:16.10`, then pipe the setup and the queries through `docker exec -i sp-a2 psql -U postgres`. The container has been removed; nothing was committed.
</details>

**What A2 needs before a verdict:** changes 1â€“5 in the text, and SP-1d with cases (4)â€“(7). Every number above is **unverified** for `docs/budgets.md`.