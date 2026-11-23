# Technical Architect critique of ADR-010-A3 rev 0 (TICK-946 comment 01a2032d, 2026-11-22)

## Technical Architect critique of ADR-010-A3 (PR #856 @ `9b29a69`): VERDICT: CHANGES 1

Shape R, the folded probe, α and the agreement argument hold. One blocking finding: R has no upper bound on |S|. A tenant above the certified size gets R with a non-hashed SubPlan, and that runs without limit on the shared database. The fix is one extra bound in the same probe statement. It costs about 1.7 ms at `bench_conc` 9,099 and needs no new spike.

I ran a 15-minute check on `postgres:16.10` (defaults: `work_mem` 4MB, `hash_mem_multiplier` 2). The setup was synthetic: a 10,111-unit tenant A with 33,400 records of one concentrated type, and a 1,011,100-unit tenant B with 2M records. Raw lines and the script are below.

### Findings

| # | Mechanism | Verdict | Evidence | Known / Inferred |
|---|---|---|---|---|
| 1 | R above the hash ceiling (A3-5, A3-R4) | **blocking** | See below | Known: plan flip. Inferred: run time |
| 2 | Q1: U_cert as a constant | **confirmed sound** | Only step 3 (the skip rule) uses it, and there it changes cost only. `groupsync_org_units_live_max` is a gauge, not a per-request source. A live count would need a new table or cache. With finding 1's fix, the folded count already is the per-leaf live bound wherever R can run | Known |
| 3 | Q2: R winning over C3 when the cap is reached | **confirmed sound; reword the question** | C3 is A2's choice when m ≤ K\*. That is `bench_conc` at m = 0, the case R exists for (64.40 against 99.26 ms). A dense type gets W0 under A2, not C3, so the real tradeoff is R against W0. That is A3-6's +6.3 ms. Choosing wrong near the crossover costs little by construction, because both shapes cost about the same there. So α's precision (A3-R1) is not material | Known (SP-1f) |
| 4 | `IS TRUE` blocks the sublink pull-up and gives a hashed SubPlan | **confirmed sound** | Page and count both show `Filter: ((hashed SubPlan 1) IS TRUE)`, building 10,111 closure rows once. Count: 778 buffers, 7.5 ms | Known |
| 5 | Folded probe: P3 never runs when the cap is reached | **confirmed sound** | P3 is an uncorrelated InitPlan inside the CASE ELSE (diff `FoldedProbeStatement`). InitPlans run on first reference, and SP-1f's plans show `(never executed)`. Also, n ≥ ⌈N̂_e/α⌉ ⟺ N̂_e ≤ α·\|S\| for integer \|S\|, so the cap condition is exact | Known |
| 6 | Generic plans | **non-blocking** | Under `force_generic_plan`, the R count switches to `Seq Scan on records`: 2,033,400 rows removed, 274.7 ms, against 7.4 ms with a custom plan. Production is not exposed today. The nava-core pool sets `QueryExecModeExec` (`cmd/nava-core/identity.go:41`, `allocator.go:63`), and unnamed statements always get custom plans. **Proposed:** add one sentence to A2-R4 (extended): "R's count assumes custom plans (`QueryExecModeExec`); a change to cached statements must re-run the plan test under `force_generic_plan`." | Known (plan); Inferred (exposure) |
| 7 | Finding 2: dropping `IS NOT NULL` | **confirmed sound for rows; plan not reproduced** | On my data, both leaf texts pick `records_list_updated` (778 buffers), with the real index `(tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL`. So I could not reproduce SP-1f's `records_org_unit` choice. The plan test in A3's Consequences is still the arbiter, and A3-R2 stays as written | Known (my data); SP-1f data not rerun |
| 8 | Mutant `r_cap_floor` | **non-blocking (test spec)** | Using ⌊⌋ instead of ⌈⌉ changes no row and no plan, so the agreement suite cannot kill it. Only the selector test kills it, and only with N̂_e/α not an integer. **Proposed text:** "killed by the selector test at n = ⌈N̂_e/α⌉ − 1 with N̂_e = 20,000 (cap 6,061)" | Known (arithmetic) |
| 9 | A3-5's probe cost "1.2 to 3.3 ms" | **non-blocking (nit)** | That range is for `bench_conc`. The worst case where R is possible is a cap near U_cert, which is about 5.5 ms (SP-1f, closure count to 10,111 in EXPLAIN). State the worst case | Known (SP-1f) |
| 10 | A3-R5, a stale-low N̂_e | **non-blocking** | R reads the type's actual size. A bulk import that happens before the hourly refresh (TICK-1071) leaves the estimate fresh but low, so "age above 3 h" never fires. Bounded by the refresh period. **Proposed trigger addition:** "or `authz_under_list_shape_total{shape="records"}` p95 above 100 ms" | Inferred |

### Finding 1 (blocking): R has no ceiling on |S|, and past the hash limit it is O(N̂_e × |S|)

**Known (PG 16.10, custom plans, R page on tenant B):**

| U (actual \|S\|) | Planner rows for SubPlan 1 | Plan |
|---|---|---|
| 20 regions (~20,220) | 4,851 | `hashed SubPlan 1` |
| 600 regions (~606,600) | 161,171 | `hashed SubPlan 1` |
| 1,000 regions (~1,011,000) | 254,827 | `(SubPlan 1) IS TRUE` over `Materialize` |
| root (1,011,100) | 1,001,347 | `(SubPlan 1) IS TRUE` over `Materialize`, total cost 34,050,684,535 |

- **The hash decision follows the planner's row estimate, not the actual size.** It flips between 161k and 255k estimated rows. That is consistent with the 8 MB `hash_mem` limit in `subplan_is_hashable` (`src/backend/optimizer/plan/subselect.c`, REL_16_STABLE). The formula is Inferred; the flip is Known.
- **Without the hash, every record that misses rescans the materialized set.** At N̂_e ≤ 33,400 and \|S\| ≈ 1M, that is up to about 3×10¹⁰ comparisons on one list (Inferred). A2 on the same leaf is O(\|S\|), seconds at worst.
- **Nothing stops it.** The selector picks R here: N̂_e ≤ α·U_cert passes step 3, and n reaches a cap ≤ 10,122. A2-7 lets such tenants exist, because groupsync never refuses a sync. I found no `statement_timeout` in the repo's Go, SQL or YAML at `TICK-946/in-my-region` `141dc96`.
- **A3-R4 records this as a residual whose trigger is the alert.** But an alert prompts a re-decision; it does not stop a query. Meanwhile one tenant's list saturates CPU on the Postgres every tenant shares. That is a regression against A2 at the same size, which is why I'm calling it blocking rather than residual.

**Proposed fix (A3-2 step 4 and A3-5).** Bound R by the size the CI plan test pins. U_hash = 20,222 = 2·U_cert, a computed constant under A3-3.

```sql
SELECT s.n,
       CASE WHEN s.n >= $cap AND s.n < $uhash THEN NULL ELSE (<P3 at K*>) END
FROM (SELECT count(*) AS n FROM (SELECT 1 FROM <canonical-U closure> LIMIT $uhash) z) s
```

- **R iff ⌈N̂_e/α⌉ ≤ n < U_hash.** Otherwise P3 decides exactly as in A2, so a tenant above U_hash keeps A2's linear cost.
- **Cost:** on the R path the count runs to min(\|S\|, U_hash) instead of the cap. At about 0.55 µs per closure row (SP-1f: 10,111 rows ≈ 5.5 ms), `bench_conc` at 9,099 counts 3,038 more rows, about +1.7 ms. The worst case, \|S\| just under 20,222, is about +11 ms, where A2 would cost about +120 ms. These numbers are Inferred; TICK-1037's rerun measures the 9,099 cell anyway.
- **The ticket type never pays,** because step 3 still skips it.
- **Tests to add:** a selector test with n = U_hash − 1 → R and n = U_hash → P3's shape; a mutant `r_no_upper_bound` killed by it; and the existing plan pin at 20,222 now covers the whole range where R can run.
- **A3-R4 text then becomes:** "Above U_hash, R is never chosen. Residual: the planner's estimate at \|S\| < U_hash exceeding `hash_mem`. The plan test at 20,222 guards it."

No further spike is needed: the fix only narrows where R runs, and A2-6 holds because rows are unchanged.

### What holds

A3-1 (leaf text, two-valued, `IN` duplicate-safe), A3-2 steps 1 to 3, A3-3 (α and U_cert computed), A3-4, A3-6, A3-8 and A3-9 are sound as written. The md and the html explainer are both in the PR. A3-7 is the Security & Multi-Tenancy Reviewer's to judge. For them: finding 1 is an availability effect on the shared database, not a data signal.

**For revision:** finding 1 (blocking) plus the text for findings 6, 8, 9 and 10. One revision round. If finding 1 is contested after it, it goes to the CTO.

<details><summary>Spike script and rerun (throwaway, not committed)</summary>

`docker run -d --name spike-a3-pg -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16.10`, then `psql < setup.sql`:

```sql
CREATE TABLE org_unit_closure (tenant_id uuid NOT NULL, ancestor_id uuid NOT NULL, descendant_id uuid NOT NULL, PRIMARY KEY (tenant_id, ancestor_id, descendant_id));
CREATE TABLE records (id bigserial, tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, org_unit_id uuid, updated_at timestamptz NOT NULL);
CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);
CREATE INDEX records_org_unit ON records (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL;
CREATE FUNCTION u(t text, i int) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$ SELECT md5(t||i)::uuid $$;
INSERT INTO org_unit_closure SELECT 'aaaaaaaa-0000-0000-0000-000000000000', u('a',0), u('a',i) FROM generate_series(0,10110) i;
INSERT INTO org_unit_closure SELECT 'aaaaaaaa-0000-0000-0000-000000000000', u('a',i), u('a',i) FROM generate_series(1,10110) i;
INSERT INTO org_unit_closure SELECT 'bbbbbbbb-0000-0000-0000-000000000000', u('b',0), u('b',i) FROM generate_series(0,1011099) i;
INSERT INTO org_unit_closure SELECT 'bbbbbbbb-0000-0000-0000-000000000000', u('b',i), u('b',i) FROM generate_series(1,1011099) i;
INSERT INTO org_unit_closure SELECT 'bbbbbbbb-0000-0000-0000-000000000000', u('br',i%1000), u('b',i) FROM generate_series(1,1011099) i;
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
 SELECT 'aaaaaaaa-0000-0000-0000-000000000000','cccccccc-0000-0000-0000-000000000000', CASE WHEN i%50=0 THEN NULL ELSE u('x',i%5000) END, now()-i*interval '1 s' FROM generate_series(1,33400) i;
INSERT INTO records (tenant_id, entity_definition_id, org_unit_id, updated_at)
 SELECT 'bbbbbbbb-0000-0000-0000-000000000000','dddddddd-0000-0000-0000-000000000000', u('b',i%1011099), now()-i*interval '1 s' FROM generate_series(1,2000000) i;
ANALYZE;
```

Probe: the A3-1 leaf as page (`ORDER BY updated_at DESC, id DESC LIMIT 51`) and count (`LIMIT 1001`), with `EXPLAIN (ANALYZE, BUFFERS)` under `force_custom_plan` and `force_generic_plan`. Then plain `EXPLAIN` of the page on tenant B with U = the first k `u('br',i)` regions, k ∈ {20, 150, 200, 220, 300, 600, 1000}, and U = `u('b',0)`.

Raw lines (selected):
```
A custom count:  Index Scan using records_list_updated ... Filter: ((hashed SubPlan 1) IS TRUE)  Rows Removed by Filter: 33400  Buffers: shared hit=778  Execution Time: 7.382 ms
A generic count: Seq Scan on records r ... Filter: (((hashed SubPlan 1) IS TRUE) AND (tenant_id = $1) AND (entity_definition_id = $3))  Rows Removed by Filter: 2033400  Execution Time: 274.662 ms
B 600 regions:   Filter: ((hashed SubPlan 1) IS TRUE) ... Index Only Scan using org_unit_closure_pkey (rows=161171)
B 1000 regions:  Filter: ((SubPlan 1) IS TRUE) -> Materialize (rows=254827)
B root:          Parallel Index Scan using records_list_updated (cost=0.43..34050569785.55) Filter: ((SubPlan 1) IS TRUE) -> Materialize (rows=1001347) -> Seq Scan on org_unit_closure
```
Container removed after the run.
</details>
