# Finding 5 margin spike (TICK-450, P5-10b)

The Technical Architect asked for this in the S1 review (TICK-423). The pass rule: at the most tenants we plan per partition, the probe's best `(tenant_id, occurred_at)`-prefix path must cost at least 10× the R2-key path. If it fails, choose a stronger override, re-measure, and adopt no value that was not measured.

**Setup.** postgres:16.10 in Docker, 16 container CPUs, on a Windows 11 host (MINGW64, x86_64). Details are in `out/env.txt`. The schema is the real migration (`db/migrations/20261102233600_audit_log.sql`) applied on stub parent tables.
- 10,000 tenants. `occurred_at` is spread over 30 days at microsecond grain, so almost every value is distinct.
- 1 to 2 entries per `txid`, then `ANALYZE`.
- The probe is in the plpgsql-locals form, run as `nava_app` under RLS, both as a generic plan (what plpgsql caches) and as a custom plan (its first five calls).
- To get the prefix path's cost, the R2 key and the subject index are dropped inside a transaction that is rolled back.

Run it with `ND="-0.0001 1" ./run.sh 100000 1000000`.

| rows (rows per tenant) | override | plan the trigger gets | best prefix path | margin | verdict |
|---|---|---|---|---|---|
| 100,000 (10) | `-0.0001` (F1 as pinned) | `audit_log_default_subject`, 8.45 | `tenant_time` Index Scan, 8.45 | 1.0× | **FAIL** (tie) |
| 1,000,000 (100) | `-0.0001` | subject, 8.45 | `tenant_time`, 8.45 | 1.0× | **FAIL** (tie) |
| 100,000 (10) | `1` | subject, 8.45 | `tenant_time` Bitmap Heap Scan, 42.59 | 5.0× | **FAIL** |
| 1,000,000 (100) | `1` | subject, 8.45 | `tenant_time` Bitmap Heap Scan, 390.05 | 46× | **PASS** |

Generic and custom plans gave the same costs in every cell.

**Side effect checked.** The 3-day, one-tenant window read that the export and the reader use estimates 20 rows under both overrides, and returns 20. Range estimates come from the histogram, and the histogram is still built when `n_distinct` is overridden.

**Reading.**
- F1's `-0.0001` gives no margin at all once a partition holds 10,000 tenants. The planner estimates rows / 10,000 / (0.0001 × rows) ≈ 1 row for the prefix, which is the same as the R2 key.
- `n_distinct = 1` is the strongest value the column can take: equality on `occurred_at` filters nothing. Under it the prefix estimate is rows per tenant, so the margin scales with partition density. It passes at 100 rows per tenant per partition, fails at 10, and the crossover is about 20.
- No `occurred_at` statistics override can reach 10× in a sparse partition. A margin that holds at every density has to come from the probe form or from the TICK-428 plan test (P5-11). Neither is in this task.

**Adopted in P5-10b:** `n_distinct = 1`. It is measured, it is never worse than `-0.0001` in any cell, and it leaves range estimates unchanged. The sparse-partition residual goes back to the Technical Architect on TICK-450.
