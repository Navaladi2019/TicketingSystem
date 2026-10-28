# S2 regression evidence (TICK-44 kit), main @ 4392c26, 2026-10-28, stack nava-regr

`make isolation` (kit "Try it" step 1) was NOT run locally: it is a CI script (`scripts/ci/isolation.sh`, part of `make test-integration`), so criteria 1/3/4's report half and the probe PRs (#132, #143) are COVERED-BY-CI. Everything below is the kit's fresh-stack half, run on this stack.

Probe method: `docker run --rm --network nava-regr_internal postgres:16.10 sh probe.sh` (psql to `postgres:5432` direct and `pgbouncer:5432`). Tenants A (`…a1`) and B (`…b1`) seeded as `nava_onboard` (INSERT provisioning, UPDATE active under `SET LOCAL app.tenant_id`), 3 records each as `nava_app`.

## Criterion 1: no / empty / leftover tenant -> 0 rows: PASS (fresh-stack half)
```
role path            | no tenant | '' | prior tx set A | tenant A -> B rows | tenant A own
nava_app postgres    | 0 | 0 | 0 | 0 | 3
nava_app pgbouncer   | 0 | 0 | 0 | 0 | 3
nava_worker postgres | 0 | 0 | 0 | 0 | 3
nava_worker pgbouncer| 0 | 0 | 0 | 0 | 3
nava_app as B: SELECT id FROM tenants            -> 00000000-0000-4000-8000-0000000000b1 (only B)
nava_app as B: UPDATE records WHERE tenant_id=A  -> updated 0
nava_app as B: INSERT records (tenant_id=A)      -> ERROR: new row violates row-level security policy for table "records"
```
Report half (121 role x table lines, 3 ALLOWLISTED): COVERED-BY-CI.

## Criterion 2: no superuser / BYPASSRLS / ownership / membership: PASS
`psql -U pgadmin -d nava` (superuser over the socket):
```
nava_app, nava_worker, nava_relay, nava_purge, nava_onboard, nava_groupsync, nava_migrator,
pgbouncer_auth, keycloak, temporal : super f | bypassrls f | createrole f | createdb f | replication f | login t
nava_backup : replication t (by design), all else f ; pgadmin : t everywhere (bootstrap only)
pg_auth_members (non pg_ roles)  -> 0
owners of public relations       -> nava_migrator | 41   ; db nava owner -> nava_migrator
tables without RLS+FORCE         -> goose_db_version only (12 tenant tables all FORCE)
```
(Kit said 39 relations / 11 tenant tables; main now has 41 / 12: `idempotency_keys` and S4 tables added since. Not a failure.)

## Criterion 3: grants == db/grants.yaml; hand grant fails CI: PASS (stack half) / COVERED-BY-CI
`docker compose logs migrate` -> `"msg":"migrate done","migrations_applied":12,"grants_granted":134,"grants_revoked":0` (no gate problems; migrator would exit 1 otherwise). Hand-grant probe PR #143: COVERED-BY-CI (not re-run).

## Criterion 4: nava_relay cannot read tenant data; nava_backup no SQL session: PASS
```
relay SELECT records|tenants|principals -> ERROR: permission denied for table <t>
relay SELECT dedupe_key FROM outbox      -> ERROR: permission denied for table outbox
backup -> nava      -> FATAL: pg_hba.conf rejects connection for host ..., user "nava_backup", database "nava"
backup -> postgres  -> FATAL: pg_hba.conf rejects connection ... database "postgres"
backup replication=database -> FATAL: pg_hba.conf rejects connection ...
```

## Extra: table without FORCE RLS fails CI: COVERED-BY-CI (probe PR #132, not re-run)
