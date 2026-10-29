# Acceptance kit: S6, API ground rules on the first endpoints

Story: TICK-48. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`), tasks T20 to T24 and their follow-ups. Design: ADR-001.
The Quality, Testing & CI Advisor verified this on 2026-10-29, on `main` at `05b103d`. The story branch `TICK-48/api-ground-rules` reached `main` in #266, so `main` contains all of S6. CI on `main` at `05b103d` is green ([run 40811416904](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40811416904)).

The stack was fresh: `docker compose down -v`, then `docker compose build && docker compose up --wait`. Every service was healthy, and `/readyz` returned `ok` on `nava-core` and `nava-onboard`. The final run, on a stack reset with `down -v` and brought up again (healthy in 72 s with images built), passed **28 of 28 checks**.

**Status: READY for the human.** Criteria 1, 2 and 4 pass on the stack. Criterion 3 passes in automated tests only, because no endpoint returns pages yet (see Known limits).

## What you can now do

- Read the API contract at `/api/v1/docs`. It lists every endpoint and links the OpenAPI 3.1 file.
- Every error, from any endpoint, has the same shape: `application/problem+json` with a `code` from one list (`api/errors.yaml`).
- An operator can safely resend an onboarding request with the same `Idempotency-Key`: they get the same answer and nothing is created twice. Reusing the key for a different request is refused.
- One tenant flooding the API is slowed down with `429`. Other tenants are not affected.

## Try it (about 10 minutes)

Needs Docker, Go 1.26 (for `make dev-ca`) and Python 3, from the repository root on `main`.

1. Start a fresh stack (about 5 minutes the first time, while the images build):
   ```sh
   make dev-ca
   docker compose down -v
   docker compose build && docker compose up --wait
   docker compose exec nava-core wget -qO- http://127.0.0.1:9090/readyz   # prints: ok
   ```
2. Open <http://127.0.0.1:18080/api/v1/docs> in a browser. You see a table with `GET /api/v1/me` and `POST /api/v1/operator/tenants`, and a link to `openapi.yaml`.
3. Run the walkthrough (about 1 minute):
   ```sh
   python -I docs/acceptance/s6-api-ground-rules/accept.py
   ```
   It seeds tenants A and B, then walks criteria 1, 2 and 4 and the cross-tenant checks. It ends with `result: PASS`. It prints every status and response body, and never a token.
4. Optional, see an error by hand: `curl -si http://127.0.0.1:18080/api/v1/me` shows `401`, `Content-Type: application/problem+json` and `"code":"invalid_token"`.
5. Clean up: `docker compose down -v`.

Seeded users, with the dev-only passwords from `deploy/compose/keycloak/import/realm-nava.json`: `alice@tenant-a.test` (tenant A, `...a1`), `bob@tenant-b.test` (tenant B, `...b1`) and the operator `operator@nava.test`. The script writes tenants A and B and their identity bindings as the bootstrap superuser, then runs one `nava-groupsync` cycle, which syncs Alice and Bob. Onboarding runs on `nava-onboard`, which publishes no port. The script calls it from a throwaway `curlimages/curl` container on the stack's `internal` network, passing the operator token through the environment.

There is no UI in this story. The evidence is the recorded requests and responses in [`s6-api-ground-rules/transcript.txt`](s6-api-ground-rules/transcript.txt), plus the docs page as served, [`s6-api-ground-rules/docs-page.html`](s6-api-ground-rules/docs-page.html). The script that produced them is [`s6-api-ground-rules/accept.py`](s6-api-ground-rules/accept.py).

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Every error is `application/problem+json` with a registered `code` | ✅ | Transcript, "Criterion 1": 9 different error codes from both services, each checked against `api/errors.yaml`. The 409 and 429 are checked under criteria 2 and 4 |
| 2 | Same key + same body returns the original response; same key + different body returns 409 | ✅ | Transcript, "Criterion 2" |
| 3 | A cursor from tenant A replayed by tenant B returns 400 | ✅ in tests only | `TestCrossTenantCursorIsInvalidCursor` (`internal/platform/cursor/request_test.go`, run again locally today: PASS) and the CI-8 cursor cases in `internal/tenancy/isolation/cursor_paths.go`, both green in CI on `main`. Not on the stack: see Known limits |
| 4 | Tenant A gets 429 while tenant B's calls succeed | ✅ | Transcript, "Criterion 4" |

### Criterion 1

Every error response was checked for four things: the `Content-Type` is `application/problem+json`; the `code` is in `api/errors.yaml`; the HTTP status equals the status registered for that code; and the body carries `status`, `title` and `instance`.

| Request | Response |
|---|---|
| `GET /api/v1/me` with no token, and with a garbage token | `401 invalid_token`, `WWW-Authenticate: Bearer` |
| `GET /api/v1/nope` | `404 not_found` |
| `DELETE /api/v1/me` | `405 method_not_allowed` |
| `POST /api/v1/operator/tenants` on `nava-core` (the tenant ingress) | `501 not_implemented` |
| Onboarding with Alice's tenant token | `403 forbidden` |
| Onboarding with `{"admin_email": 7}` | `400 validation_failed`, `errors: [{"pointer":"/body/admin_email","rule":"type"}]` |
| Onboarding with an IdP `tokenUrl` on `127.0.0.1` | `422 invalid_idp_config` |
| Same key, different body (criterion 2) | `409 idempotency_key_reuse` |
| Tenant A over its limit (criterion 4) | `429 rate_limited`, `Retry-After: 1` |

`GET /api/v1/docs` is `200 text/html` and lists both operations. `GET /api/v1/openapi.yaml` is `openapi: 3.1.0` and is byte-identical to `api/openapi.bundled.yaml` on `main`.

### Criterion 2

The operator onboards a new tenant C, with a fresh key and domain.

| Call | Response | Rows afterwards (tenants, bindings, email domains, invites) |
|---|---|---|
| Before | n/a | 2, 2, 0, 0 |
| First call | `201 {"state":"active","tenant_id":"bd32fa33-..."}` | 3, 3, 1, 1 |
| Same key, same body | `200` with a byte-identical body | 3, 3, 1, 1 |
| Same key, different `display_name` | `409 idempotency_key_reuse` (problem+json) | 3, 3, 1, 1 |
| Same key, original body again | `200` with the same body | 3, 3, 1, 1 |

### Criterion 4

The stack runs with the compose defaults, `NAVA_RATE_LIMIT_RPS=50` and `NAVA_RATE_LIMIT_BURST=100`. For 4 seconds, 48 workers called `GET /api/v1/me` as Alice, while Bob called it every 50 ms.

- Tenant A made 1,822 calls: 303 got `200`, and 1,519 got `429 rate_limited` with `Retry-After: 1`. 303 is what the limit allows: a burst of 100 plus 50 a second for 4.2 s. The 429 body names no tenant.
- Tenant B made 23 calls, 20 of them after tenant A's first 429. All 23 got `200` with tenant B.
- Three seconds later, Alice's next call got `200` again.

## Isolation check

| Tried | Result |
|---|---|
| The flood in reverse: 400 calls as Bob (169 got `200`, 231 got `429`), then one call as Alice | ✅ Alice gets `200` with tenant A. Each tenant has its own bucket |
| Bob sends tenant A's id in `X-Tenant-Id` | ✅ `200` with tenant B. The limiter and `/me` key on the token's tenant only |
| A tenant token on onboarding: on `nava-core` and on `nava-onboard` | ✅ `501` and `403`. A tenant cannot replay or create an onboarding |
| Tenant C, created by onboarding | ✅ Its id is neither A nor B. It is derived from the operator and the key |

## Not included

- **List endpoints.** None exist yet, so neither does an endpoint that returns a cursor. The first list endpoint brings the first real cursor (PLAN-001 decision D-P5).
- **Idempotency on tenant writes.** Only onboarding takes an `Idempotency-Key` today. The shared store (`idempotency_keys`, T22) is ready for the first tenant write endpoint.
- **A rendered API browser.** `/api/v1/docs` is a plain HTML table that links the contract, not Swagger UI or Redoc.

## Known limits

- **Criterion 3 cannot be tried by hand yet.** No endpoint returns pages. The behaviour is proven by `TestCrossTenantCursorIsInvalidCursor`, which uses a stand-in list handler built from the same helpers every list endpoint will use, and by the five CI-8 cursor cases: replayed by tenant B, by another user of the same tenant, with a changed filter, with one byte tampered, and after key rotation. As D-P5 planned, the first real list endpoint must repeat the cross-tenant test against itself. The reviewer of that endpoint should block it without that test.
- **A replay answers `200`, not the original `201`.** The body is byte-identical. This is the documented behaviour (`docs/operator/onboarding.md`), the same as in S4. If you want the original status code too, that is a small change; say so.
- **The rate limit is per `nava-core` replica**, in memory. Two replicas allow a tenant twice the rate. That is the T24 design; a shared store can replace it behind the `Limiter` interface.
- **There is no seed command yet.** The script writes tenants A and B as the bootstrap superuser, as S3's kit did. Onboarding writes them in production.
- Playwright is not installed on this host, so the docs page is recorded as served HTML, not as a screenshot. The page is static: a title, one paragraph and a two-row table.
