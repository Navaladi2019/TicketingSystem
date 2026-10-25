# Acceptance kit: S1, local stack starts healthy and CI gates every PR

Story: TICK-43. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`). Design: ADR-001 and amendment A1.
Verified on `main` at `e682fe0` on 2026-10-25 by the Quality, Testing & CI Advisor.

## What you can now do

- Bring up the whole platform on one machine with one command: PostgreSQL, PgBouncer, Keycloak, Temporal, the Collector, nava-core and the worker.
- Ask the service whether it and its dependencies are well. It stays ready when only Keycloak or Temporal is down, and stops taking traffic within seconds when the database goes.
- Every pull request is checked against real PostgreSQL, Keycloak and Temporal, and a failing test turns it red.

## Try it (about 10 minutes, most of it waiting)

Needs Docker and Go 1.26, from a fresh clone of `main`.

1. Make the dev certificates, build the images and start the stack:
   ```sh
   make dev-ca
   docker compose build
   docker compose up --wait
   ```
   You should see every service `Healthy` in about 2–3 minutes. The build is a one-off and takes a few minutes more.

   If other copies of this stack already run on the machine, give yours its own name and ports first:
   ```sh
   export COMPOSE_PROJECT_NAME=nava-mine NAVA_CORE_PORT=0 KEYCLOAK_PORT=0
   ```
2. Check health. The health port 9090 is deliberately not published, so ask from inside the container:
   ```sh
   docker compose exec nava-core wget -qO- http://127.0.0.1:9090/healthz
   docker compose exec nava-core wget -qO- http://127.0.0.1:9090/readyz
   ```
   You should see `{"status":"ok","checks":{"keycloak":"ok","postgres":"ok","temporal":"ok"}}` and `ok`.
3. Stop Keycloak with `docker compose stop keycloak`. Within a few seconds, `/healthz` shows `"status":"degraded"` with `"keycloak":"fail"`, and `/readyz` still says `ok`. Start it again with `docker compose start keycloak`.
4. Stop PostgreSQL with `docker compose stop postgres`. `/readyz` now fails: `wget` prints `server returned error: HTTP/1.1 503`. Start it again with `docker compose start postgres`.
5. Optional: open a draft PR that adds a test calling `t.Fatal`, and watch the `unit` job go red. Close the PR afterwards.
6. Clean up with `docker compose down -v`.

No users or tenants are involved. This story has no sign-in.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Every container reports healthy within 3 minutes of `docker compose up` | ✅ | See below |
| 2 | `/healthz` lists PostgreSQL, Keycloak and Temporal; stopping Keycloak shows it degraded while `/readyz` stays 200 | ✅ | See below |
| 3 | Stopping PostgreSQL makes `/readyz` non-200 within 5 s | ✅ about 1.6 s | See below |
| 4 | `docs/pins.md` lists every image and Go module with the ADR-001 pins | ✅ | See below |
| 5 | A deliberately failing test turns CI red | ✅ | See below |

### Criterion 1 evidence

The plain `compose.yaml` was started with `down -v`, then `up --wait`, with the build not timed.

| Where | Elapsed |
|---|---|
| This verification, `e682fe0`, project `nava-t43` | **172 s** |
| TICK-166 (PR #90) | 144 s, 149 s |
| TICK-164 (PR #95) | 122 s, 128 s |
| GitHub Actions run [40209903952](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40209903952) | `stack` healthy in 37 s from `up -d` |

During this verification, 20 other containers were running on the host: a second `nava` stack and integration-test databases.

Keycloak is the slowest service. Its phases in the 172 s run:

| Phase | Time |
|---|---|
| Container starts after `up` (waits for db-roles) | 33 s |
| JVM start | 30 s |
| Database schema init | 61 s |
| Master realm and realm `nava` import | 29 s |

The CI gate `stack-test.sh healthy` fails the plain stack past 180 s (TICK-164).

### Criterion 2 evidence

Run on the fresh stack at `e682fe0`. Baseline:

```
{"status":"ok","checks":{"keycloak":"ok","postgres":"ok","temporal":"ok"}}
```

8 s after `docker compose stop keycloak`:

```
{"status":"degraded","checks":{"keycloak":"fail","postgres":"ok","temporal":"ok"}}
```

`/readyz` returned 200 on 3 of 3 samples, and `nava_dependency_degraded{dependency="keycloak"}` was 1. The same check also passed on `9f278dd` and `d1d998f`. CI runs it as `stack-test-dependencies`.

### Criterion 3 evidence

A poller inside nava-core requested `/readyz` every 0.2 s while PostgreSQL was stopped.

| Commit | Time from PostgreSQL exit to the first 503 |
|---|---|
| `9f278dd` | about 1.5 s |
| `d1d998f` | about 1.6 s |

`/readyz` returned 503 on every sample after that. Commits since `d1d998f` did not change the health code; they changed only the Keycloak realm import, Postgres durability settings and the new worker service.

### Criterion 4 evidence

- `make check-pins` reports `check-pins: ok` on `e682fe0`. It checks every direct require in all four `go.mod` files, every compose `image:` and every Dockerfile `FROM`.
- The pins were also checked by hand against ADR-001 §3 and amendment A1:
  - Keycloak 26.8.1 (floor 26.1.3)
  - `postgres:16.10` (floor 15)
  - PgBouncer v1.24.1-p1 (floor 1.21)
  - Temporal SDK v1.49.0 with API v1.63.5
  - `go.temporal.io/server` v1.29.7 with API v1.53.0
  - Go toolchain go1.26.10
- This PR adds the `nava/keycloak` row.

### Criterion 5 evidence

Each throwaway PR added a test that calls `t.Fatal`, and both were closed unmerged.

| PR | Commit | Run | Result |
|---|---|---|---|
| #71 | `d2d4b2e` | [40132272371](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40132272371) | `unit` and `integration` red on `TestRedCIProbeTICK43` |
| #78 | `d1d998f` | [40154212111](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40154212111) | `unit` and `integration` red |

## Isolation check

Not applicable. S1 creates no tenant data and has no sign-in, so there is nothing to try across tenants. Tenant isolation is first exercised by the stories that store tenant data. The CI isolation suite (`make isolation`, TICK-67) already runs inside `make test-integration`.

## Not included

- Signing in, tenants and the API itself are planned in the later PLAN-001 stories.
- The Temporal UI is behind `--profile operator` and `make operator-tunnel`. It is not part of the default stack.
- Production deployment and TLS on the published ports.

## Known limits

- **Startup stretches with host load.** Two runs on the shared dev host measured 280 s and 417 s while other agents' stacks were starting at the same time. Measured clean, startup is 122–172 s.
- **Project name `nava` clashes.** Plain `docker compose up` always uses project name `nava`, so two stacks on one host clash: container names collide, or one `down` removes the other's network. Set your own `COMPOSE_PROJECT_NAME`, with `NAVA_CORE_PORT=0 KEYCLOAK_PORT=0`, as in "Try it".
- **Test IdP realms are not imported at boot.** Only realm `nava` is imported at boot (TICK-166). The test IdP realms (`idp-tenant-a`, `idp-tenant-b`) are loaded by `make stack-idp-realms` and `make stack-test-identity`.
- **The dev PostgreSQL runs without durability.** It uses `fsync=off`, `synchronous_commit=off` and `full_page_writes=off` (TICK-166). A host crash can corrupt the dev volume; reset it with `docker compose down -v`.
- **The health port is not published.** `/healthz` and `/readyz` are not reachable from the host by design. Use `docker compose exec`, as above.
