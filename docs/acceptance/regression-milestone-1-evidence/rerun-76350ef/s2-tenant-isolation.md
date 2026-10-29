# S2 regression re-run (TICK-44 kit), main @ 76350ef, 2026-10-28, stack nava-regr2

`make isolation` (kit step 1) NOT run: CI script. The report half of criteria 1/3/4 and probe PRs #132/#143 are COVERED-BY-CI. Fresh-stack half below: `docker run --rm --network nava-regr2_internal postgres:16.10 sh probe.sh` (psql to `postgres:5432` and `pgbouncer:5432`); tenants A (`…a1`) / B (`…b1`) seeded as `nava_onboard`, 3 records each as `nava_app`.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | no / empty / leftover tenant -> 0 rows | PASS (stack half); report COVERED-BY-CI | table below |
| 2 | no superuser / BYPASSRLS / ownership / membership | PASS | catalogue below |
| 3 | grants == db/grants.yaml; hand grant fails CI | PASS (stack half) / COVERED-BY-CI | migrate `grants_granted":135,"grants_revoked":0` (gate would exit 1 on drift); PR #143 not re-run |
| 4 | nava_relay cannot read tenant data; nava_backup no SQL session | PASS | below |
| extra | table without FORCE RLS fails CI | COVERED-BY-CI | PR #132 not re-run |

```
role path            | no tenant | '' | prior tx set A | tenant A -> B rows | tenant A own
nava_app postgres    | 0 | 0 | 0 | 0 | 3
nava_app pgbouncer   | 0 | 0 | 0 | 0 | 3
nava_worker postgres | 0 | 0 | 0 | 0 | 3
nava_worker pgbouncer| 0 | 0 | 0 | 0 | 3
nava_app as B: SELECT id FROM tenants           -> 00000000-0000-4000-8000-0000000000b1 only
nava_app as B: UPDATE records WHERE tenant_id=A -> UPDATE 0
nava_app as B: INSERT records (tenant_id=A)     -> ERROR: new row violates row-level security policy for table "records"
```
Catalogue (`pgadmin` over the socket):
```
keycloak, nava_app, nava_groupsync, nava_migrator, nava_onboard, nava_purge, nava_relay, nava_worker, pgbouncer_auth, temporal:
   super f | bypassrls f | createrole f | createdb f | replication f | login t
nava_backup: replication t (by design), all else f ; pgadmin: t everywhere (bootstrap only)
pg_auth_members (non pg_ roles) -> 0 ; owners of public relations -> nava_migrator | 41 ; db owner -> nava_migrator
tables without RLS+FORCE -> goose_db_version only ; RLS+FORCE tables -> 12
```
Criterion 4:
```
relay SELECT records|tenants|principals -> ERROR: permission denied for table <t>
relay SELECT dedupe_key FROM outbox      -> ERROR: permission denied for table outbox
backup -> nava | postgres | replication=database -> FATAL: pg_hba.conf rejects connection for host ..., user "nava_backup"
```
The lease migration added a column, not a table: still 41 relations / 12 tenant tables (kit text says 39 / 11, known drift).
