# S1 regression evidence (TICK-43 kit), main @ 4392c26, 2026-10-28

Env for every compose command: `COMPOSE_PROJECT_NAME=nava-regr NAVA_CORE_PORT=28080 KEYCLOAK_PORT=28081 KEYCLOAK_PUBLIC_URL=http://localhost:28081 TEMPORAL_TEST_PORT=0`
(fixed free ports plus a matching pinned issuer, as the S4/S5 kits document, so the identity harnesses and hand-made tokens agree). Host also ran the unrelated `multica-*` containers.

## Fresh start
```
docker compose down -v && make dev-ca          -> wrote development certificates to ../deploy/compose/certs
docker compose build                           -> exit 0 in 390 s (not timed for crit 1)
docker compose up --wait   (started 17:01:02Z) -> exit 0 in 159 s; 12 long-running services Healthy,
                                                  db-roles / migrate / temporal-schema Exited (0)
```

## Criteria
| # | Result | Command -> excerpt |
|---|---|---|
| 1 healthy < 3 min | PASS (159 s) | `docker compose up --wait` -> `up exit 0 in 159s`, all `Healthy` incl. nava-onboard, nava-groupsync, nava-relay |
| 2 healthz lists deps; Keycloak down -> degraded, readyz 200 | PASS | baseline `wget -qO- :9090/healthz` -> `{"status":"ok","checks":{"keycloak":"ok","postgres":"ok","temporal":"ok"}}`, readyz `ok`. 8 s after `docker compose stop keycloak`: `{"status":"degraded","checks":{"keycloak":"fail",...}}`, readyz `HTTP/1.1 200 OK` 3/3; `nava_dependency_degraded{dependency="keycloak"} 1` |
| 3 Postgres down -> readyz non-200 < 5 s | PASS (~0.55 s) | poller inside nava-core every 0.2 s; `docker inspect` FinishedAt `17:05:07.349Z`; nava-core log `dependency state changed ... postgres ok->fail` at `17:05:07.899Z`; poller 14x 200 then 46x FAIL, no 200 after first FAIL; `wget` -> `server returned error: HTTP/1.1 503 Service Unavailable` |
| 4 pins | PASS | `make check-pins` -> `check-pins: ok` |
| 5 failing test turns CI red | COVERED-BY-CI | not re-run (no throwaway PR; CI rule). Kit's PRs #71/#78 stand |

Recovery: `docker compose start postgres` then `docker compose up --wait` (re-runs the one-shot db-roles/migrate/temporal-schema, all exit 0). ~10 s later healthz `ok` on all three, readyz `ok`. nava-core log `postgres fail->ok` 17:05:53Z.
Keycloak restart: `docker compose start keycloak` + `up --wait keycloak`; healthz back to `ok`.

Rough edges seen: right after `up --wait` returned on recovery, readyz still answered 503 for a few seconds (nava-core's check interval), i.e. `up --wait` exit is not "nava-core ready" after a postgres restart. pgbouncer logs `no such user: postgres` every 5 s (its own healthcheck), noisy but harmless.
