# RFC-012 round 1 spikes (Postgres & Data Modeling Specialist)

Posted on TICK-549, comment 01a1a547-6361-79b7-84ec-76b6ac90973b. Round 2 re-check: comment 01a1a54d-76fd-7375-b9b5-2b6c00e5176e.
Host: Windows 11, Docker 29.9.2, PostgreSQL 16.10, PgBouncer 1.24.1. Base: `origin/TICK-541/audit-diff-reader-guard@fcd17e8`. Nothing committed; edits reverted.

## A1: the revoke breaks only the reader path
Method: removed `diff, diff_salt` from `nava_app.tables.audit_log.SELECT`; `nava-db migrate` revoked them; ran `scripts/ci/test-integration.sh` over every integration package, unsharded, `-timeout 25m`.
Result: 1103 passed, 4 skipped, 6 failed, plus 1 unit failure. All 7 are listed in ADR-004-A1 (moved or inverted). No `nava_app` query outside `internal/audit/read` failed; records pipeline, backfill, sealer, erase, relay and groupsync passed.

## B1: a FORCE-RLS policy scoped to other roles returns 0 rows, no error
Table with FORCE RLS, SELECT policy `TO` another role, reader granted column SELECT `(definition_id, schema)`: with `app.tenant_id` set, 0 rows and no error. After `ALTER POLICY … TO app, nava_audit_reader`: 1 row.

## A2: connection cap
Eight concurrent 2-second queries, transaction mode.

| Mechanism | Server conns | Waiting | Result |
|---|---|---|---|
| `[users] nava_audit_reader = pool_size=5` | 5 | 3 | 8/8 ok in 4 s; clients queue |
| `ALTER ROLE … CONNECTION LIMIT 5`, pool 20 | 5 | 3 | 8/8 ok in 4 s, but PgBouncer logs `FATAL too many connections for role` per extra login |

Chosen: `pool_size=5` plus `pgxpool MaxConns = 5`; no `CONNECTION LIMIT`.
