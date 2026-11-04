<!-- Archived verbatim from Final re-check of F1/F2 at 38 partitions (Technical Architect), TICK-502 comment 01a1a00e-cb7b, 2026-11-03T06:18:06Z. -->

**RFC-010 rev 1, final re-check (Technical Architect): F1 and F2 only.**

In short: both §6 rows match the CTO ruling, and the flat condition works at 38 partitions. I **reject** the inferred 150–172 µs: measured, the import costs the same at 38 partitions as at 2. The re-run also exposed a new risk: absolute µs ceilings on this laptop can move by about 2× within an hour. I propose one text change for it below; it is not blocking.

**Setup:** a fresh `postgres:16.10` container (same flags as S2), with schema from `7b5e388` built by `TestP511CommitCheckPlanSparse`. Then the S2 `setup.sql`, plus 36 monthly partitions of 200 rows each, analysed. That makes 38 partitions in total. The young partition is rebuilt before every run. The scripts are the ones in my round-1 comment; the partition builder is below.

## The re-check

| Item | Verdict | Evidence | Known / Inferred |
|---|---|---|---|
| F1 row: ≤ 300 µs per call, flat (max/min < 1.5) across N = 200 / 1,000 / 5,000, at 38 partitions, replacing the 42.2 µs row | **confirmed sound** (text matches the ruling; the flat condition discriminates) | With the clause, 38 partitions, 3 reps: 443 / 575 / 452, 366 / 362 / 417 and 412 / 402 / 425 µs. Max/min per rep is **1.30 / 1.15 / 1.06**, so it passes flat. As shipped, at 38 partitions: 256 → 253 → 519 µs, max/min **2.05**, so it fails flat. The condition separates the two arms at the budget shape | Known |
| F2 row: ≤ 175 µs per probe, 10k import, fresh session, median of 3, at 38 partitions | **confirmed sound** (text matches the ruling) | See the import table below | Known |
| Rev 1's inferred 150–172 µs per probe at 38 partitions | **rejected**: the 38-partition cost is about the same as the 2-partition cost, not 30 µs more | Interleaved in the same host state, 3 reps. With the clause, the median COMMIT is **2,939 ms at 38 partitions and 2,943 ms at 2**; per rep, 38 against 2 is 1.08 / 1.00 / 0.80. F3's +30 µs was single-call planning, and it does not carry into a large import. Scaled to the round-1 host state (120–142 µs at 2 partitions), that gives **about 96–153 µs at 38, centred near 130**. Not measured in that host state: the next item explains why | Known (38/2 ratio); Inferred (the scaled µs value) |
| **New:** absolute ceilings measured on a dev laptop | **non-blocking**, but S3 could fire spuriously | Same laptop, same fixture, about 1 h after round 1, and everything is 1.8–2.3× slower. Leaf planning went 0.021 → 0.038 ms, parent planning 0.125 → 0.249 ms, the clause arm per call at 2 partitions 190–205 → 351–402 µs, and the 10k import 1.2–1.4 s → 2.8–3.0 s. In this host state both rows "fail": 361–575 µs against 300, and 294 µs against 175. Yet the mechanism behaves the same. The clause-to-shipped ratio on the import stayed put: round 1 **10.0 / 12.9 / 12.7**; today **13.1 / 12.4 / 12.1** at 2 partitions and **9.5 / 9.5 / 10.0** at 38 | Known (numbers); Inferred (cause: host power or thermal state, or other load. Host CPU was at 23 % during the run, containers idle) |

**10k import, fresh session, COMMIT time (today's host state):**

| Shape | With the clause | As shipped |
|---|---|---|
| 2 partitions | 2,957 / 2,943 / 2,816 ms (median 294 µs per probe) | 225 / 237 / 233 ms |
| 38 partitions | 3,196 / 2,939 / 2,261 ms (median 294 µs per probe) | 335 / 309 / 225 ms |

## Proposed text change

This does not change the ruling's ceilings; it only makes the S3 trigger falsifiable. The Solution Architect can fold it into ADR-004-E3 / §6 without another round. Add to the §6 notes:

```
- **Where the rows are measured.** Both rows are measured in the CI integration job (pinned `postgres:16.10` image, the same runner class every time), not on a developer laptop. The baselines are re-taken there before the Postgres specialist commits the rows; the S2 laptop numbers are provenance, not the baseline. On one laptop, the same fixture varied 1.8–2.3× within an hour (TICK-502 TA re-check).
- **Same-run reference.** Each run also measures the shipped function (no clause) on the same fixture in the same job. An over-ceiling result fires the S3 trigger only if the same-run ratio is also out of range: import clause/shipped > 15 (measured 9.5–13.1 across two host states), or per call at N = 200 clause/shipped > 5 (measured 1.7–4.1). An over-ceiling result with an in-range ratio is a runner problem: re-run it, and do not fire S3.
```

<details><summary>38-partition builder (on top of the S2 setup.sql / young.sql)</summary>

```sql
\set ON_ERROR_STOP 1
DO $$
DECLARE m int; lo timestamptz; n text;
BEGIN
  FOR m IN 1..36 LOOP
    lo := date_trunc('month', now()) - make_interval(months => m);
    n := 'audit_log_s2_m' || m;
    EXECUTE format('CREATE TABLE %I PARTITION OF audit_log FOR VALUES FROM (%L) TO (%L)', n, lo, lo + interval '1 month');
    EXECUTE format('ALTER TABLE %I ALTER COLUMN occurred_at SET (n_distinct = 1)', n);
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', n);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', n);
  END LOOP;
END $$;
SET session_replication_role = replica;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
 SELECT ('0190b000-0000-7000-8000-' || lpad(to_hex(1 + g % 50), 12, '0'))::uuid, gen_random_uuid(),
        date_trunc('month', now()) - make_interval(months => m) + (g * interval '3 hours'), (1000000 + m*1000 + g)::text::xid8,
        'record','update','record', gen_random_uuid(), 'workflow','workflow'
 FROM generate_series(1,36) m, generate_series(1,200) g;
SET session_replication_role = DEFAULT;
ANALYZE audit_log;   -- 38 partitions; the monthly leaves are 4 pages / 200 rows
-- 2-partition shape: ALTER TABLE audit_log DETACH PARTITION audit_log_s2_m<k> for k = 1..36; re-ATTACH with the same bounds.
```

The per-call runs use the S2 `budget_<arm>.pgbench` at `-c 1 -t 200|1000|5000`, after `pg_stat_reset()` and a fresh `young.sql`. The import runs use the S2 10k script, one fresh psql session per run, alternating 2 and 38 partitions within each rep.
</details>

VERDICT: **ACCEPT** on F1 and F2. Both rev 1 rows match the CTO ruling, the flat condition separates the arms at 38 partitions (1.06–1.30 with the clause, 2.05 as shipped), and the import row is within budget at 38 partitions: measured equal to 2 partitions, about 130 µs scaled, which **rejects** the inferred 150–172 µs. One **non-blocking** text change goes into the ADR: measure in CI, with a same-run reference ratio before S3 can fire.
