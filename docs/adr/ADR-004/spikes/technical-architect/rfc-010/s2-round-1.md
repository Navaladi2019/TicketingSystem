<!-- Archived verbatim from S2 and critique round 1 (Technical Architect), TICK-502 comment 01a19fdc-c359, 2026-11-03T05:23:27Z. -->

**RFC-010 critique round 1 (Technical Architect): open questions 1 and 3, and spike S2.**

In short: 1c removes the reuse fault, but **S2 fails the ≤ 100 µs budget by about 2×, and it makes the healthy import path 10–13× slower.** Most of the planning cost comes from planning through the partitioned parent. The RFC says it goes back to the triad with the number in that case, so here are the numbers.

**Setup:** `postgres:16.10` (Debian 16.10-1.pgdg13+1), Docker Desktop on Windows 11, 16 CPUs. Flags as TICK-428: `fsync=off synchronous_commit=off full_page_writes=off track_functions=pl`. Schema and `audit_commit_check` are verbatim from `7b5e388`.
- **Arms:** `ctrl` is the function as shipped. `fcp` is a copy made with `pg_get_functiondef` plus `ALTER FUNCTION … SET plan_cache_mode = force_custom_plan`. Each arm has its own `DEFERRABLE INITIALLY DEFERRED` constraint trigger on its own table.
- **Fixture:** the young-partition fixture from item 2 (b): 60 rows over 20 tenants, 2 pages, analysed, autovacuum off. It is rebuilt before every run.
- **Partition shapes:** "2 partitions" is the young partition plus DEFAULT, like the TICK-428 harness. "38 partitions" adds 36 monthly partitions of 200 rows each.
- **Not re-measured:** the specialist's linearity results and Q2 answers.

## S2 results

**Per-call self time.** Source is `pg_stat_user_functions` (self_time / calls). One pgbench connection, which behaves like one pooled backend. Single-subject transactions as `nava_worker`, the same script as `budget.pgbench`.

| Writes in the run | `ctrl` (as shipped) | `fcp` (with the clause) |
|---|---|---|
| 200 | 50.2 µs | 204.6 µs |
| 1,000 | 80.4 µs | 190.2 µs |
| 5,000, 2 partitions, 3 reps | 242.6 / 280.0 / 252.7 µs | **208.6 / 216.9 / 224.1 µs** |
| 5,000, 38 partitions, 3 reps | 279.7 / 274.5 / 298.3 µs | **249.7 / 248.4 / 251.0 µs** |

**Planning time of the probe alone.** `EXPLAIN (SUMMARY)` with literal values, median of 300 runs inside one transaction:

| Target | Median |
|---|---|
| through `audit_log`, 2 partitions, as `nava_worker` | 0.125 ms |
| same, `enable_bitmapscan = off` | 0.122 ms |
| same, as superuser (no RLS) | 0.087–0.116 ms |
| same, DEFAULT partition detached | 0.096 ms |
| through `audit_log`, 38 partitions | 0.154 ms |
| **leaf partition directly**, as superuser (no RLS; `nava_worker` has no direct grant on the leaf) | **0.021–0.022 ms** |

**10,000-subject import, fresh session, COMMIT time** (S1 item 2 shape), 3 reps:

| Function | COMMIT time |
|---|---|
| `ctrl` | 119.9 / 104.7 / 111.5 ms |
| `fcp` | **1,198 / 1,355 / 1,415 ms** |

## Findings

| # | Mechanism | Verdict | Evidence | Known / Inferred |
|---|---|---|---|---|
| F1 | 1c's per-call cost against `docs/budgets.md` ≤ 100 µs | **blocking** | `fcp` measures 190–225 µs per call at 2 partitions and 248–251 µs at 38. It stays flat from 200 to 5,000 writes, so this is the planning cost, not plan decay. The RFC's guess of "tens of µs" (§8) is wrong by about 10× | Known |
| F2 | 1c on the healthy path: a large import in a fresh session | **blocking**, the same budget seen from the import side | 10k COMMIT goes from 105–120 ms to 1.2–1.4 s, about 120–130 µs of planning per probe. The specialist's 4k numbers with the clause (202–394 ms) agree. Linear, but 10–13× the cost the fresh-session path pays today | Known |
| F3 | Where the planning cost comes from | **non-blocking**, points at the fix | Planning through the parent costs about 90–125 µs. Planning the leaf costs 21–22 µs. RLS (about 10–30 µs), `enable_bitmapscan` (no change) and the DEFAULT partition (about 30 µs, within noise) are minor. Cost also grows with partition count: 125 → 154 µs from 2 to 38 partitions | Known (timings); Inferred (that this is partitioned-parent planning overhead in PG 16, not source-traced) |
| F4 | The existing 42.2 µs budget row | **non-blocking**, but the row is misleading | In a pooled backend, `ctrl`'s per-call cost grows with the writes since the last ANALYZE: 50 → 80 → 234 µs at 200 / 1,000 / 5,000. This is the cached Seq Scan hurting single-subject traffic too, not only imports. The 42.2 µs was measured in a state where this does not show. With autovacuum on, autoanalyze would cap committed growth near 10 %. It cannot cap growth inside an import (the specialist's point) | Known (curve, autovacuum off); Inferred (production cap) |
| F5 | 1c's determinism (§4) | **confirmed sound**, on the specialist's evidence, not re-measured | My `fcp` curve stays flat across N in one backend, which agrees with it independently | Known |
| F6 | §4 row 3a, and the 10-page rule | **confirmed sound** | Specialist's Q2a. Also see Q1 below: an analysed empty partition is exactly the state that defeats the remaining plan-shape candidate | Known |

## Open question 1: is there a plan shape that beats a 2-page Seq Scan?

**Yes, but it does not hold in every state the probe can reach at COMMIT, so it is not a fix.** I confirm the RFC's rejection of plan-shape routes, with one more measured candidate.

The candidate is a function-level `SET random_page_cost = 1.0`. No scan type is disabled. Generic plans under `force_generic_plan`, after `VACUUM ANALYZE`:

| Partition state | `random_page_cost = 4` (today) | `= 1.1` | `= 1.0` |
|---|---|---|---|
| 60 rows, 2 pages | Seq Scan 3.35 | Index Scan, `subject` index, 2.37 | Index Scan, `subject` index, 2.17; `subject_id` and `occurred_at` in `Index Cond` |
| 20 rows, 1 page | Seq Scan 1.45 | Index Only Scan, R2 key, 1.27 | Index Only Scan, R2 key, 1.17; all 5 columns in `Index Cond` |
| **analysed empty, then +1 / +5 / +30 rows in the transaction** (the state at COMMIT) | Seq Scan | not run | **Seq Scan 1.34** |

The last row is the counter-example. An empty leaf gets `reltuples = 0` from any `ANALYZE audit_log`, which recurses into the leaves; that includes the standard `vacuumdb --analyze-in-stages` after an upgrade. The probe then meets the partition with at least one row on one page, and the cached plan is a Seq Scan again.

The margins are also tie-sized: 1.17 against 1.45. That is the S1 F1 regime. **Verdict: confirmed sound** (the RFC's §3 conclusion holds). Plan-shape routes only move the size at which the plan flips; they do not remove it.

## Open question 3: the budget form

**Keep a per-call self-time budget. Do not move to a p95 overhead ratio.** TICK-423 already chose self time over a p95 ratio because the ratio was too noisy, and nothing here changes that. Change how the budget is measured, and add an import row. Proposed rows for `docs/budgets.md`, for the Postgres specialist to commit **once the triad picks the mechanism**:

```
| Commit-check trigger (`audit_commit_check`, ADR-004 decision 2), single subject | `pg_stat_user_functions` self_time / calls, `track_functions = pl`; ONE connection (pooled-backend state), 5,000 single-subject txns as `nava_worker`, young partition analysed at 60 rows, autovacuum off on it; reported at N = 200, 1,000 and 5,000 and must be flat (max/min < 1.5) | ≤ <N> µs per call, at the production partition count | ctrl 50→234 µs (not flat); force_custom_plan 190–225 µs (2 partitions), 248–251 µs (38). 2026-11-03, PostgreSQL 16.10 (TICK-502 S2) | Technical Architect S2 (TICK-502) | Postgres & Data Modeling Specialist |
| Commit-check trigger, import | COMMIT time / N for one 10,000-subject import, fresh session, median of 3 | ≤ <M> µs per probe | ctrl 10.5–12.0 µs; force_custom_plan 120–142 µs. 2026-11-03, PostgreSQL 16.10 (TICK-502 S2) | Technical Architect S2 (TICK-502) | Postgres & Data Modeling Specialist |
```

`<N>` and `<M>` stay **unverified** until the CTO chooses between the options below. The "must be flat" condition is what the old 42.2 µs row lacked: it catches plan decay that a single N cannot show.

**Partition count:** `non-blocking`. Cost grows with partitions, and neither ADR-004 nor `docs/` states how many `audit_log` partitions are retained. The Solution Architect should state it, so that the budget is measured at that shape.

## The specialist's two text changes for item 3 (c)

- **Run under both `auto` and `force_generic_plan`: agree.** `auto` is the production default, and the specialist measured ratios of 13–20 there without the clause. Keep the warm-up at 10 calls, past the 5 custom-plan calls.
- **Median of 3 or more runs: agree, with one addition.** Each run uses a fresh fixture, and both arms run in the same CI job, so machine noise affects both equally. Pass if the clause arm's median is < 8 and the negative control's median is ≥ 8. The specialist's spread (clause arm at most 5.4, control at least 12.3) leaves enough margin for a median of 3.

## What the triad needs to decide (the RFC's §7 rule; a CTO tradeoff)

1. **Accept 1c and rebase the budget** to the measured numbers: about 250 µs per call (single subject) and about 150 µs per probe on imports. That adds about 0.15–0.2 ms to every audited write and about 1.3 s to a 10k COMMIT, and it stays linear. It is the simplest mechanism, and its determinism is proven.
2. **`verify-by-spike` S3, a custom plan on the leaf partition.** Planning the leaf costs 21–22 µs (measured without RLS, so add about 10–30 µs), so a probe that targets the current leaf could meet ≤ 100 µs. Inferred, not yet measured: a leaf probe also needs a grant on the leaf, which the I3 grant model must allow. The catch is the leaf name: it has to be resolved deterministically. Either Epic 30 fixes a naming rule derived from `occurred_at`, or the write path hands over the `tableoid`. Without that, this means `EXECUTE format('%I')` dynamic SQL, which the RFC rejected in 1d. **Pass:** per-call ≤ 100 µs, flat across N, at the production partition count. Item 2 (b′) and item 3 (c) green under both modes. The negative control red. **Fail:** any of those red, or the leaf name needs a catalog lookup per call. Timebox: half a day. Owner: whoever the Technical Lead names.

Routes 2 and 3 stay rejected; nothing above revives them.

<details><summary>How to rerun S2 (scripts)</summary>

Start `postgres:16.10` with `-c fsync=off -c synchronous_commit=off -c full_page_writes=off -c track_functions=pl`, apply the migrations at `7b5e388`, then:

```sql
-- setup.sql: control = shipped function; fcp = same body + the clause
DO $$ BEGIN EXECUTE replace(pg_get_functiondef('audit_commit_check'::regproc), 'public.audit_commit_check()', 'public.audit_commit_check_fcp()'); END $$;
ALTER FUNCTION audit_commit_check_fcp() SET plan_cache_mode = force_custom_plan;
CREATE TABLE s2_ctrl (tenant_id uuid NOT NULL, id uuid NOT NULL, val text, PRIMARY KEY (tenant_id, id));
CREATE TABLE s2_fcp  (LIKE s2_ctrl INCLUDING ALL);
GRANT SELECT, INSERT ON s2_ctrl, s2_fcp TO nava_worker;
CREATE CONSTRAINT TRIGGER s2_ctrl_cc AFTER INSERT ON s2_ctrl DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');
CREATE CONSTRAINT TRIGGER s2_fcp_cc  AFTER INSERT ON s2_fcp  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check_fcp('record', 'id');
INSERT INTO tenants (id, state) SELECT ('0190b000-0000-7000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active' FROM generate_series(1, 50) g ON CONFLICT DO NOTHING;
INSERT INTO tenants (id, state) SELECT ('0190e000-0000-7000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active' FROM generate_series(1, 4000) g ON CONFLICT DO NOTHING;

-- young.sql (run before every measurement): 60 rows / 20 tenants / 2 pages, analysed
TRUNCATE s2_ctrl, s2_fcp;
DROP TABLE IF EXISTS audit_log_p511_young;
CREATE TABLE audit_log_p511_young PARTITION OF audit_log FOR VALUES FROM (date_trunc('month', now())) TO (date_trunc('month', now()) + interval '1 month') WITH (autovacuum_enabled = false);
ALTER TABLE audit_log_p511_young ALTER COLUMN occurred_at SET (n_distinct = 1);
ALTER TABLE audit_log_p511_young ENABLE ROW LEVEL SECURITY; ALTER TABLE audit_log_p511_young FORCE ROW LEVEL SECURITY;
SET session_replication_role = replica;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
 SELECT ('0190b000-0000-7000-8000-' || lpad(to_hex(1 + g % 20), 12, '0'))::uuid, gen_random_uuid(), date_trunc('month', now()) + (now() - date_trunc('month', now())) * (g::float8/61), (3000000+g)::text::xid8, 'record','update','record', gen_random_uuid(), 'workflow','workflow' FROM generate_series(1,60) g;
SET session_replication_role = DEFAULT;
ANALYZE audit_log;
```

```
-- budget_<arm>.pgbench  (arm = ctrl | fcp); run: pg_stat_reset(); pgbench -n -c 1 -t N -f budget_<arm>.pgbench nava
--   then: SELECT calls, self_time*1000/calls FROM pg_stat_user_functions WHERE funcname LIKE 'audit_commit_check%' AND calls > 0;
SELECT gen_random_uuid() AS sid \gset
BEGIN;
SET LOCAL ROLE nava_worker;
SELECT set_config('app.tenant_id', '0190b000-0000-7000-8000-000000000004', true);
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source) VALUES ('0190b000-0000-7000-8000-000000000004', gen_random_uuid(), 'record', 'create', 'record', ':sid', 1, 'workflow', 'workflow');
INSERT INTO s2_<arm> VALUES ('0190b000-0000-7000-8000-000000000004', ':sid', 'v');
COMMIT;
```

```sql
-- planning time: median of 300 EXPLAIN (SUMMARY) with literal values
CREATE OR REPLACE FUNCTION s2_plantime(rel text, tenant uuid, iters int DEFAULT 300) RETURNS numeric LANGUAGE plpgsql AS $$
DECLARE j json; a numeric[] := '{}'; i int;
BEGIN
  FOR i IN 1..iters LOOP
    EXECUTE format('EXPLAIN (SUMMARY, FORMAT JSON) SELECT 1 FROM %s WHERE tenant_id = %L AND occurred_at = %L AND txid = %L AND subject_type = ''record'' AND subject_id = %L LIMIT 1',
                   rel, tenant, now(), pg_current_xact_id(), gen_random_uuid()) INTO j;
    a := a || (j->0->>'Planning Time')::numeric;
  END LOOP;
  RETURN (SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY x) FROM unnest(a) x);
END $$;
-- SET ROLE nava_worker; SELECT set_config('app.tenant_id','0190b000-0000-7000-8000-000000000004',false);
-- SELECT s2_plantime('audit_log', '0190b000-0000-7000-8000-000000000004');             -- through the parent
-- RESET ROLE; SELECT s2_plantime('audit_log_p511_young', '0190b000-0000-7000-8000-000000000004');  -- leaf

-- 10k import, fresh psql session per run, s2_<arm>:
SET ROLE nava_worker; BEGIN;
SELECT set_config('app.tenant_id', '0190e000-0000-7000-8000-000000000f01', true);
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
  SELECT '0190e000-0000-7000-8000-000000000f01', gen_random_uuid(), 'record','create','record', md5('x' || g)::uuid, 1, 'workflow','workflow' FROM generate_series(1, 10000) g;
INSERT INTO s2_<arm> SELECT '0190e000-0000-7000-8000-000000000f01', md5('x' || g)::uuid, 'v' FROM generate_series(1, 10000) g;
\timing on
COMMIT;

-- Q1 grid: shrink the young partition to K rows (0 / 20 / 60), VACUUM ANALYZE it, then:
SET plan_cache_mode = force_generic_plan; SET random_page_cost = 1.0;   -- also 1.1 and 4
PREPARE p(uuid, timestamptz, xid8, text, uuid) AS SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5 LIMIT 1;
EXPLAIN EXECUTE p('0190b000-0000-7000-8000-000000000001', now(), pg_current_xact_id(), 'record', gen_random_uuid());
-- analysed-empty case: DELETE all rows, VACUUM ANALYZE, then in one transaction insert 1/5/30 audit rows and EXPLAIN as above.
```

The 38-partition shape adds 36 monthly partitions (200 rows each, analysed) below the current month. The 2-partition shape detaches them.
</details>

VERDICT: **REVISE.** 1c is sound on determinism but **blocking** on the ≤ 100 µs budget: 190–251 µs per call, and 10k imports go from 0.11 s to 1.3 s. Both plan-shape routes are confirmed dead, and I agree with both of the specialist's item 3 (c) text changes. Choosing between **(1) accept 1c with a rebased budget** and **(2) spike S3 on a leaf-targeted probe** is a CTO tradeoff under RFC §7.
