# TICK-640 (P3-34b) bench

Records list index set and JIT, on P3-01's 800k shape. Report and numbers:
[`docs/data/records-list-indexes.md`](../../docs/data/records-list-indexes.md).

| File | What |
| --- | --- |
| `setup.go` | Roles plus the real migrations (`-skip` leaves some out); `-seed` loads P3-01's 800k shape as P3-36 does |
| `explain.sh` | `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)` of one contract-shaped list page (and count) as `nava_app` in the large tenant |
| `matrix.sh` | `explain.sh` over agent, heaviest and lightest heavy requester, and admin, page 1 and 20; one summary line per plan |
| `union.sh` | The `mine` rule as a `UNION ALL` of index-ordered branches (the requester-list fix handed to P3-34a) |
| `insert.sql`, `insert-bench.sh` | pgbench of one record INSERT: index write amplification (p50/p95/p99, WAL per insert) |
| `plans-before.md` | Before: P3-09's set, JIT on, `-updated_at` and `-created_at`, with counts |
| `plans-after.md`, `plans-after-attr.md` | After: this task's migrations applied over the seeded rows, JIT off; every sort |
| `plans-union.md` | The `UNION ALL` requester plans, page 1 and 20, after |

Needs Docker and Go. The steps are under "Reproduce" in the report. Not in CI.
