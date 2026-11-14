# ADR-007 spike S4: CORS on 200/401/403/429 in three engines, and the D2 lookup p95

**Verdict: SPIKE: PASS.** TICK-809, 2026-11-14. ADR-007 D3 as written (CORS outermost) needs no order fix.

## What was run

Harness: `scratch/adr-007-s4/run.sh` (docker, go, node). Pieces:

- `server/main.go`: a scratch server with the **order** of `internal/server/api.go` (authn 401, client gate and origin check 403 inside `InTenantTx`, limiter 429, handler) and the D3 CORS handler, exactly the D3 headers. `-cors=outer` puts CORS outermost (the ADR); `-cors=inner` wraps only the handler (negative control).
- `pw/run.mjs`: Playwright 1.56.1, Chromium, Firefox and WebKit (build 26.0). A page on `http://127.0.0.1:9001` (a listed origin) or `:9002` (not listed) calls `fetch` with `Authorization`, `Content-Type`, `Idempotency-Key`, `If-Match`, `Nava-SDK` for GET, POST and PATCH, so each non-simple request has a preflight.
- `seed.sql`, `bench/main.go`: the D2 `embedding_client` table (ADR shape, `ENABLE`+`FORCE` RLS, `UNIQUE (tenant_id, client_id)`) at 1,000 tenants x 20 rows, `postgres:16.10`, PgBouncer 1.24.1 in transaction mode with `max_prepared_statements=100`, role `nava_app` (`NOBYPASSRLS`), lookup run through the real `tenancy.InTenantTx`.

**Deviation to know about:** authn and the limiter are stubs (fixed tokens `tok-ok`, `tok-limited`, `tok-off`; the limiter answers 429 with `Retry-After: 7` for one token), not the real Keycloak resolver and limiter. The spike tests the middleware order and the CORS headers, which do not depend on how a 401 or a 429 is decided. The D2 lookup and the origin check are real SQL under RLS. Bodies are RFC 9457-shaped (`status`, `code`), not the platform's full registry.

## Result 1: readable statuses

`-cors=outer`: **45/45 pass** (3 engines x 5 cases x 3 methods). Every case read the status and the `code` through `fetch`, and `Retry-After` on 429 (it is in `Expose-Headers`):

| Case | Status | Readable in Chromium, Firefox, WebKit |
|---|---|---|
| allowed origin, valid token | 200 (201 for POST) | yes |
| no or bad token | 401 | yes |
| token valid, `Origin` not in `allowed_origins` | 403 `origin_not_allowed` | yes |
| token valid, client row `disabled` | 403 `client_not_enabled` | yes |
| limited | 429, `Retry-After: 7` | yes |

Full table: `scratch/adr-007-s4/result-outer.md`.

Negative control `-cors=inner` (CORS wraps only the handler): **0/45 pass**; every call ends in an opaque `TypeError`, because the preflight carries no token and gets 401 without CORS headers. This is the failure the ADR order prevents: **CORS must stay outermost, before authn, so the preflight is answered with no token and 401/403/429 carry the headers.** `scratch/adr-007-s4/result-inner.md`.

Harness note: a page faked with Playwright request interception is classed as a public address space, and Chromium's Local Network Access then blocks the call to loopback with `Permission was denied ... unknown address space`. That is a harness artefact, not CORS; the pages are served by real loopback servers.

## Result 2: D2 lookup p95

The single indexed lookup (`SELECT allowed_origins, state FROM embedding_client WHERE tenant_id = $1 AND client_id = $2`) inside `InTenantTx`, through PgBouncer, as `nava_app`, 20,000 samples per run after a 2,000-sample warm-up, run from a container on the same docker network (no host port-forward). 16 CPUs, Docker Desktop on Windows, `postgres:16.10`.

| Clients | p50 | **p95** | p99 | max |
|---|---|---|---|---|
| 1 | 0.84 ms | **1.52 ms** | 2.06 ms | 3.6 ms |
| 8 | 2.22 ms | **3.12 ms** | 3.91 ms | 67 ms |

Command: `sh scratch/adr-007-s4/run.sh` (the bench alone: `bench -dsn <nava_app via PgBouncer> -n 20000 -c 8`). Raw: `scratch/adr-007-s4/bench-result.txt`; `run.sh` also captures the plan (`EXPLAIN (ANALYZE, BUFFERS)` as `nava_app`) to `explain.txt`.
Plan (`explain.txt`): index scan on `embedding_client_tenant_id_client_id_key`, 3 buffer hits, 0.14 ms execution. The rest is the `InTenantTx` BEGIN/set_config/COMMIT round trips.

A first run from the host through Docker Desktop's port forward read p95 39 ms (1 client) and 88 ms (8 clients). That measures the Windows port proxy, not the lookup; it is not the figure of record.

**For P8-08 (`docs/budgets.md`):** measure "D2 client-gate lookup inside `InTenantTx` through PgBouncer, 1,000 tenants x 20 rows". Measured **3.12 ms p95 at 8 clients** (1.52 ms at 1). Suggested limit **≤ 5 ms p95 at 8 clients**; P8-08 owns the final number. The lookup is one extra transaction per embedding request and no cache (D2), so it adds this to the request path.

## Consequences

- P8-07: implement CORS as the outermost handler, as D3 says. No order change. Keep a test that fails if a 401, 403 or 429 or a preflight lacks `Access-Control-Allow-Origin` and `Vary: Origin` (CI-E3).
- The 403 `origin_not_allowed` and `client_not_enabled` need the CORS headers too; the outer position gives that.
