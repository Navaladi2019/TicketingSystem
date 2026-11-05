# P5-15 (TICK-433) migration A backfill: timing

`run.sh` starts `postgres:16.10` (the `docs/pins.md` pin) in Docker on the
dev host (Windows 11, Git Bash, Docker 29.9.2, container sees 16 CPUs), applies
the real `20261102233600_audit_log.sql` and `20261103120000_audit_backfill_s8.sql`
on stub parents owned by `nava_migrator` (RLS forced), loads synthetic S8
history (10 types per tenant, versions 1 to 3, half the tenants' v1 with no
publisher), then times `SELECT audit_backfill_s8()` as `nava_migrator`.
Raw output: `out/`.

## Final numbers

| Tenants | Versions | First run (migration A) | Re-run, all present (migration B) | audit_log_default size |
| ---: | ---: | ---: | ---: | ---: |
| 1,000 | 30,000 | 5.62 s | 0.14 s | 17 MB |
| 10,000 | 300,000 | 54.69 s | 2.32 s | 172 MB |

Linear, about 0.18 ms per backfilled version. The dedupe probe per tenant
(30 versions) is an index scan on `audit_log_default_subject`: 0.5 to 0.8 ms
(`plan.sql` output in `out/`).

## What changed on the way

| Step | 30,000 versions | 300,000 versions | Cause |
| --- | ---: | ---: | --- |
| First cut | 10.01 s | 291.73 s | Superlinear: the cached generic plan seq-scanned `audit_log_default` (stats say empty while the run fills it): 28 ms per tenant at 90k rows. |
| `SET enable_seqscan = off` on the function | — | — | 90k: 56.98 s to 27.78 s, now linear. |
| SHA-1 rounds split into four loops (no per-round IF) | 5.62 s | 54.69 s | `audit_backfill_id` 283 µs to 129 µs per id. |

Per tenant (30 rows) after both: 9.2 ms under `EXPLAIN ANALYZE`: id hashing
about 3.9 ms, the three FK checks 1.8 ms, the R10 trigger 0.4 ms, the rest
heap and six index inserts.

## Lock cost

The temporary policies take ACCESS EXCLUSIVE on `tenants`,
`entity_definition_versions` and `audit_log` until COMMIT, so tenant
resolution and S8 reads wait for the whole first run: about 5.6 s per
30,000 existing versions. The re-run inside migration B (P5-32) is the
cheap case: 0.14 s per 30,000 versions already present.
