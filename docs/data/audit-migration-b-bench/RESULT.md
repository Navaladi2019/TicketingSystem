# P5-32 (TICK-442) migration B: lock-hold timing

`run.sh` starts `postgres:16.10` (the `docs/pins.md` pin) in Docker on the
dev host (Windows 11, Git Bash, Docker 29.9.2, container sees 16 CPUs;
`fsync`, `synchronous_commit` and `full_page_writes` off, as the integration
clusters run) and runs the opt-in `TestP532MigrationBTiming`
(`internal/db/dbtest/audit_migration_b_timing_test.go`). Per size, on the
real migrations: tenants with one principal each, 10 types per tenant with
versions 1 to 3 (half the tenants' v1 with no publisher), migration A's
backfill, then a version 4 with no entry for about 1% of the types (published
between A and the deploy). Then it times migration B alone, the whole
transaction from the six `LOCK TABLE`s to COMMIT, and reads its functions'
times from `pg_stat_user_functions` (`track_functions = pl`). Raw output:
`out/`.

## Numbers

| Tenants | S8 versions | Between A and B | Migration B (lock hold) | `audit_backfill_s8()` re-run | `audit_seed_chain_heads()` | of which UUIDv7 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 30,093 | 93 | 0.48 s | 175.5 ms | 202.0 ms | 21.3 ms (2,000 calls) |
| 10,000 | 300,972 | 972 | 4.29 s | 2,864.3 ms | 1,281.7 ms | 131.7 ms (20,000 calls) |

- Linear in both terms. The backfill re-run is about 9.5 µs per version
  already present (the `NOT EXISTS` probe on `audit_log_default_subject`), the
  same cost migration A's kit measured for its re-run (2.32 s at 300,000).
  The seeding is about 0.13 ms per tenant: two primary-key inserts and two
  UUIDv7s. The rest of B (locks, two `CREATE CONSTRAINT TRIGGER`s, the
  catalogue check) is about 0.1 s.
- After B, CI-A9 holds (each version has one entry), and 1,000 / 10,000 heads
  and seal rows exist.
- For reference, migration A's first run on the same data took 7.95 s and
  66.9 s (A's own kit: 5.62 s and 54.69 s on a quieter host).

## Write and storage cost

Per tenant, B writes one `audit_chain_head` row (heap plus its primary key)
and one `outbox` row (heap plus four indexes: the primary key, the dedupe key
and the two partial pending indexes), once. At 10,000 tenants:
`audit_chain_head` is 1,904 kB and `outbox` 4,424 kB, heap and indexes. These
are the rows onboarding writes for every new tenant (P5-33), so B adds no
index and no lasting write path.

## Lock cost

B holds ACCESS EXCLUSIVE on `tenants`, `entity_definitions`,
`entity_definition_versions`, `audit_log`, `audit_chain_head` and `outbox`
for the whole transaction: tenant resolution, S8, every audited write and
the relay wait about 0.13 ms per tenant plus 9.5 µs per S8 version plus
about 0.1 s (4.3 s at 10,000 tenants and 300,000 versions). `lock_timeout = 5s` bounds the wait to
take each lock. When this was measured, B retried a timeout itself, five
times, 1 to 4 s apart. Since TICK-606 the retry has one owner, the migrator
(TICK-579): a timeout fails B, its transaction rolls back and releases the
six, and the runner re-runs it, up to 10 attempts with a backoff doubling
from 1 s to 30 s (`TestP532LocksInOrderAndRetries`). `lock_timeout`
bounds each lock wait, so one attempt waits at most 6 x 5 s = 30 s: about
7.5 min worst case, about 3.4 min when one lock is contended.
The lock-hold times above are unchanged: they time B after it holds the
six.
