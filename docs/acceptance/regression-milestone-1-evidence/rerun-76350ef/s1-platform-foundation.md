# S1 regression re-run (TICK-43 kit), main @ 76350ef, 2026-10-28

Env for every compose command: `COMPOSE_PROJECT_NAME=nava-regr2 NAVA_CORE_PORT=28080 KEYCLOAK_PORT=28081 KEYCLOAK_PUBLIC_URL=http://localhost:28081 TEMPORAL_TEST_PORT=0`. The unrelated `multica-*` containers ran on the host throughout and were not touched.

## Fresh start
```
docker compose down -v ; make dev-ca        -> dev CA already present (kept)
docker compose build                        -> exit 0 in 362 s
docker compose up --wait  (21:41:16Z)       -> exit 0 in 80 s; 12 long-running services healthy,
                                               db-roles / migrate / temporal-schema Exited (0)
migrate log: "migrate done","migrations_applied":13,"grants_granted":135,"grants_revoked":0
```
13 migrations = the 12 of the previous run plus `20261028175712_idempotency_lease.sql`; 135 grants = 134 plus nava_app `UPDATE (lease_until)`.

## Criteria
| # | Result | Evidence |
|---|---|---|
| 1 healthy < 3 min | PASS (80 s) | `up --wait` exit 0 in 80 s, every service `healthy` |
| 2 healthz lists deps; Keycloak down -> degraded, readyz 200 | PASS | baseline `{"status":"ok","checks":{"keycloak":"ok","postgres":"ok","temporal":"ok"}}`, readyz `ok`. 8 s after `docker compose stop keycloak`: readyz `HTTP/1.1 200 OK` 3/3; healthz `{"status":"degraded","checks":{"keycloak":"fail",...}}`; `nava_dependency_degraded{dependency="keycloak"} 1` |
| 3 Postgres down -> readyz non-200 < 5 s | PASS (~2.3 s) | poller in nava-core every 0.2 s: 28x 200 then 32x FAIL, no 200 after the first FAIL. postgres `FinishedAt 21:43:34.877Z`; nava-core `postgres ok->fail` at `21:43:37.190Z`; `wget` -> `HTTP/1.1 503 Service Unavailable` |
| 4 pins | PASS | `make check-pins` -> `check-pins: ok` |
| 5 failing test turns CI red | COVERED-BY-CI | not re-run (CI rule); kit PRs #71/#78 stand |

Recovery: `docker compose start postgres` + `up --wait` (one-shots re-ran, exit 0, `migrations_applied":0`), healthz `ok` x3, readyz `ok`.

## Observation (Breaks under: Postgres restart), not a criterion
After the criterion 3 drill, Keycloak kept dead pooled JDBC connections. The next 6 Keycloak requests that needed the DB answered `400 {"error":"unauthorized_client","error_description":"Unexpected error when authenticating client"}` (log: `GenericJDBCException ... FATAL: terminating connection due to administrator command`), one per stale connection, then recovered. nava-core healthz kept reporting `keycloak: ok` during this. Impact here: the first S3 `accept.py` run aborted on its admin token (see s3 file). Repro: S1 crit 3 drill, then `curl -d grant_type=password -d client_id=admin-cli ... /realms/master/protocol/openid-connect/token` x8 -> `400 200 200 ...`. Suggest a Keycloak pool validation setting (`KC_DB_POOL_*` / agroal validation) or a kit note to retry.
