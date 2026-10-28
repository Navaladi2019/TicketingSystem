# TICK-287 acceptance run: Postman collection over the whole stack

- Collection: `docs/acceptance/guide/postman/TicketingSystem.postman_collection.json` (Newman 6, environment `local.postman_environment.json`)
- Stack: fresh (`docker compose down -v`, `up --wait`, healthy in 117 s), images built from `main` at `4392c26` (`main` is now `741623e`, which changes one test file only); `tools/seed.sh`, `tools/tunnels.sh up`, `make demo-outbox DEDUPE_KEY=accept` twice. `.env`: `NAVA_GROUPSYNC_INTERVAL=15s`, `COMPOSE_PROFILES=operator`.
- Started 2026-10-28T17:40:47Z; duration 341 s

| | executed | failed |
|---|---|---|
| requests | 95 | 0 |
| testScripts | 83 | 0 |
| prerequestScripts | 23 | 0 |
| assertions | 244 | 0 |

**Result: PASS** (0 failures)

| Request | Sends (polls repeat) | Last status | Assertions passed | Failed |
|---|---|---|---|---|
| S1.1 Liveness: GET /healthz lists every dependency | 1 | 200 | 3 | 0 |
| S1.2 Readiness: GET /readyz | 1 | 200 | 2 | 0 |
| S1.3 Metrics: GET /metrics | 1 | 200 | 3 | 0 |
| S1.4 Negative: /healthz is not on the public port | 1 | 404 | 3 | 0 |
| S1.5 Negative: /metrics is not on the public port | 1 | 404 | 3 | 0 |
| S1.6 Keycloak realm nava is up with the pinned issuer | 1 | 200 | 2 | 0 |
| S2.1 Sign in as Alice (tenant A) | 1 | 200 | 3 | 0 |
| S2.2 Sign in as Bob (tenant B) | 1 | 200 | 3 | 0 |
| S2.3 Alice: GET /me is tenant A | 1 | 200 | 4 | 0 |
| S2.4 Bob: GET /me is tenant B, none of Alice's groups | 1 | 200 | 6 | 0 |
| S2.5 Negative: Alice names tenant B in headers, query and body | 1 | 200 | 4 | 0 |
| S2.6 Negative: Bob names tenant A | 1 | 200 | 4 | 0 |
| S2.7 Negative: Alice asks for a tenant B record by id | 1 | 404 | 4 | 0 |
| S3.1 Sign in as Alice | 1 | 200 | 4 | 0 |
| S3.2 Alice: GET /me | 1 | 200 | 5 | 0 |
| S3.3 Sign in as tenant A's admin | 1 | 200 | 3 | 0 |
| S3.4 Tenant A admin: GET /me | 1 | 200 | 4 | 0 |
| S3.5 Negative: no token | 1 | 401 | 4 | 0 |
| S3.6 Negative: garbage token | 1 | 401 | 3 | 0 |
| S3.7 Negative: tampered token (payload changed) | 1 | 401 | 4 | 0 |
| S3.8 Negative: Alice's ID token used as an access token | 1 | 401 | 3 | 0 |
| S3.9 Get a token from another issuer (Keycloak master realm) | 1 | 200 | 2 | 0 |
| S3.10 Negative: wrong issuer | 1 | 401 | 3 | 0 |
| S3.11 Sign in as the operator | 1 | 200 | 4 | 0 |
| S3.12 Negative: operator token on a tenant endpoint | 1 | 401 | 3 | 0 |
| S3.13 Setup: create Dave, a user in no organization | 2 | 201 | 2 | 0 |
| S3.14 Sign in as Dave | 1 | 200 | 4 | 0 |
| S3.15 Negative: a token with no organization | 1 | 401 | 3 | 0 |
| S3.16 Setup: create Carol | 2 | 201 | 2 | 0 |
| S3.17 Setup: look up Carol's id | 2 | 200 | 2 | 0 |
| S3.18 Setup: add Carol to tenant A's organization | 2 | 201 | 2 | 0 |
| S3.19 Setup: add Carol to tenant B's organization | 2 | 201 | 2 | 0 |
| S3.20 Sign in as Carol asking for every organization | 1 | 200 | 4 | 0 |
| S3.21 Negative: a token naming two organizations | 1 | 401 | 4 | 0 |
| S4.1 Sign in as the operator | 1 | 200 | 4 | 0 |
| S4.2 Sign in as tenant A's admin | 1 | 200 | 3 | 0 |
| S4.3 Negative: onboarding with no token | 1 | 401 | 3 | 0 |
| S4.4 Negative: a tenant admin's token | 1 | 403 | 3 | 0 |
| S4.5 Negative: the operator route on the tenant ingress (nava-core) | 1 | 501 | 3 | 0 |
| S4.6 Negative: no Idempotency-Key | 1 | 400 | 4 | 0 |
| S4.7 Negative: IdP URL on a private address | 1 | 422 | 3 | 0 |
| S4.8 Onboard a new tenant | 1 | 201 | 2 | 0 |
| S4.9 Replay: same key, same body | 1 | 200 | 2 | 0 |
| S4.10 Negative: same key, different body | 1 | 409 | 3 | 0 |
| S4.11 Negative: another tenant claims the same domain | 1 | 409 | 4 | 0 |
| S4.12 Negative: claim seeded tenant A's domain | 1 | 409 | 4 | 0 |
| S4.13 Negative: a tenant id in the body and a header is ignored | 1 | 201 | 3 | 0 |
| S4.14 Negative: the onboarding service serves operator routes only | 1 | 404 | 3 | 0 |
| S5.1 Setup: look up Alice and tenant A's service-desk group | 3 | 200 | 6 | 0 |
| S5.2 Sign in as Alice | 1 | 200 | 3 | 0 |
| S5.3 Sign in as Bob (the control) | 1 | 200 | 3 | 0 |
| S5.4 Alice: GET /me shows /service-desk | 1 | 200 | 4 | 0 |
| S5.5 Admin removes Alice from service-desk | 2 | 204 | 2 | 0 |
| S5.6 Poll: the same token's /me loses /service-desk | 2 | 200 | 5 | 0 |
| S5.7 Control: Bob still has his /service-desk | 1 | 200 | 4 | 0 |
| S5.8 Restore: add Alice back to service-desk | 2 | 204 | 2 | 0 |
| S5.9 Sign in as Alice again (fresh token) | 1 | 200 | 3 | 0 |
| S5.10 Poll: /me shows /service-desk again | 2 | 200 | 5 | 0 |
| S5.11 Admin disables Alice | 2 | 204 | 2 | 0 |
| S5.12 Negative: Alice cannot sign in again | 1 | 400 | 2 | 0 |
| S5.13 Poll: the same unexpired token gets 401 tenant_unresolved | 3 | 401 | 5 | 0 |
| S5.14 Control: Bob is unaffected | 2 | 200 | 6 | 0 |
| S5.15 Restore: re-enable Alice | 2 | 204 | 2 | 0 |
| S5.16 Sign in as Alice (re-enabled) | 1 | 200 | 3 | 0 |
| S5.17 Poll: Alice resolves again | 2 | 200 | 5 | 0 |
| S6.1 GET /api/v1/openapi.yaml (public) | 1 | 200 | 2 | 0 |
| S6.2 GET /api/v1/docs (public) | 1 | 200 | 2 | 0 |
| S6.3 Sign in as Alice | 1 | 200 | 3 | 0 |
| S6.4 Negative: unknown /api/v1 route with no token | 1 | 401 | 3 | 0 |
| S6.5 Negative: unknown /api/v1 route with a token | 1 | 404 | 3 | 0 |
| S6.6 Negative: unknown path outside /api/v1 | 1 | 404 | 3 | 0 |
| S6.7 Negative: wrong method on /api/v1/me | 1 | 405 | 4 | 0 |
| S6.8 Negative: wrong method with no token | 1 | 401 | 3 | 0 |
| S6.9 Every response carries X-Request-Id; an inbound traceparent is accepted | 1 | 200 | 5 | 0 |
| S6.10 Negative: request-specific input never appears in a problem | 1 | 404 | 4 | 0 |
| S7.1 Temporal: the demo's workflows, one execution each | 1 | 200 | 4 | 0 |
| S7.2 Temporal: its history holds the start and two signals in commit order | 1 | 200 | 3 | 0 |
| S3.22 Negative: expired token | 1 | 401 | 4 | 0 |

Tokens and response bodies are left out; the raw Newman JSON holds bearer tokens and is not published.

## TICK-317: S6 folder re-run on `main` after PR #266 (per-tenant 429)

- `main` at `2cc0f21` (PR #266 merged). Fresh stack: `docker compose down -v`, `make dev-ca`, images built from that commit, `up --wait`, `tools/seed.sh`. `.env` from `.env.example` (default limit: `NAVA_RATE_LIMIT_RPS=50`, `NAVA_RATE_LIMIT_BURST=100`).
- `newman run … --folder "S6 — API ground rules"`, 2026-10-28, 5.3 s. Run three times; all three passed.

| | executed | failed |
|---|---|---|
| requests (S6.12 sends a 320-request burst) | 333 | 0 |
| assertions | 49 | 0 |

**Result: PASS**

| Request | Last status | Assertions passed | Failed |
|---|---|---|---|
| S6.1–S6.10 (unchanged) | as above | 37 | 0 |
| S6.11 Sign in as Bob (tenant B) | 200 | 3 | 0 |
| S6.12 Tenant A over its limit gets 429 while tenant B succeeds | 200 (Bob) | 11 | 0 |
| S6.13 Tenant A is served again after Retry-After | 200 | 3 | 0 |

S6.12 burst counts per run (status: count): tenant A `{200: 118, 429: 182}`, `{200: 121, 429: 179}`, `{200: 117, 429: 183}`; tenant B `{200: 20}` every time.
