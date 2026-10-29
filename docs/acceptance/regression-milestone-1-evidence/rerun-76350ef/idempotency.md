# Idempotency-Key scoping (Security request), main @ 76350ef, 2026-10-28, stack nava-regr2

## Precondition found: no tenant route on main uses the idempotency middleware
- `api/openapi.yaml` declares `Idempotency-Key` on one operation only: `POST /api/v1/operator/tenants` (operator-only, nava-onboard). It has its own idempotency (tenant id = UUIDv5(operator, key), request hash on `tenants`), not `idempotency_keys`.
- `internal/platform/idempotency.Middleware` (T22/T22a, `idempotency_keys`, PK `(tenant_id, principal_id, key)`) is constructed only in the CI-8 isolation suite (`internal/tenancy/isolation/idempotency_paths.go`). No `cmd/` or `internal/server` code calls `idempotency.New`. The only tenant operation is `GET /api/v1/me`.
- Live on the product ingress: Alice `POST /api/v1/me` with `Idempotency-Key` -> `405`; `GET /api/v1/me` with `Idempotency-Key` twice -> `200`, `200`, no `Idempotent-Replayed`, 0 `idempotency_keys` rows. Alice/Bob cannot reach the operator route (`403`).

So "same key by Alice then Bob" cannot be sent to a product endpoint today. **Spec drift / gap (route to Solution Architect + Security):** T22's middleware ships but guards nothing until the first tenant write route mounts it; whoever adds that route must wire `authn -> ratelimit -> idempotency`.

## What was run instead (live, real components, harness route)
A probe build of nava-core (`go build -overlay`, nothing written to the repo) mounted `POST /idem-probe` = real `auth.Authenticator` (the same Keycloak resolver and `nava_app` pool as nava-core) -> real `idempotency.Middleware` on the live `nava_app` pool through PgBouncer -> a counter echo handler (`run` increments per execution, echoes the caller's tenant and principal). Container `nava-regr2-idemprobe` on the stack's networks, env copied from nava-core, `127.0.0.1:28090`. Real tokens for alice, bob and admin@tenant-a. Key `regr2-idem-220344`; body X `{"subject":"printer on fire"}`, body Y `{"subject":"printer fixed"}`.

```
(a) alice   K + X (first)                          -> 201 replayed=None {"principal_id":"e070bb96-…","run":1,"tenant_id":"…a1",...}
(a) bob     K + X (other tenant, same key+body)    -> 201 replayed=None {"principal_id":"aff7692f-…","run":2,"tenant_id":"…b1",...}
(b) admin-a K + X (same tenant A, other principal) -> 201 replayed=None {"principal_id":"28e393ce-…","run":3,"tenant_id":"…a1",...}
(c) alice   K + X again                            -> 201 replayed=true  run 1 (byte-identical to alice's first)
(c) alice   K + Y                                  -> 409 urn:nava:problem:idempotency_key_reuse
(c) bob     K + X again                            -> 201 replayed=true  run 2 (bob's own)
(c) bob     K + Y                                  -> 409 idempotency_key_reuse
(c) admin-a K + X again                            -> 201 replayed=true  run 3 (admin-a's own)
    alice   no Idempotency-Key                     -> 400 validation_failed, pointer /header/Idempotency-Key rule required
    no token, K + X                                -> 401 invalid_token (auth before idempotency)
```
| Case | Result |
|---|---|
| (a) same key + body, Alice (A) then Bob (B): Bob processed as his own, never Alice's response | PASS: new run 2, tenant B, not replayed, body never names Alice or tenant A |
| (b) same key, two principals of tenant A (Alice, admin-a): independent | PASS: new run 3, admin-a's principal |
| (c) same principal + key + body -> replay; other body -> 409 | PASS for all three principals |

### idempotency_keys rows (superuser)
```
tenant_id | principal_id | key | hash16 | state | status | response_body | lease_until>created_at | expires_at-created_at
…a1 | e070bb96-…(alice)   | regr2-idem-220344 | 1c7c310c1b735d6b | completed | 201 | {…"run":1,"tenant_id":"…a1"} | t | 1 day
…b1 | aff7692f-…(bob)     | regr2-idem-220344 | 1c7c310c1b735d6b | completed | 201 | {…"run":2,"tenant_id":"…b1"} | t | 1 day
…a1 | 28e393ce-…(admin-a) | regr2-idem-220344 | 1c7c310c1b735d6b | completed | 201 | {…"run":3,"tenant_id":"…a1"} | t | 1 day
```
Three rows, one per `(tenant_id, principal_id, key)`; the request hash is identical (it covers method, path and body, not the caller), so the separation comes from the primary key. As nava_app: tenant A sees 2 rows (alice, admin-a), tenant B sees 1 (bob), no tenant sees 0.

Harness artifact, not a product finding: the probe route sat outside nava-core's `httpobs` middleware, so its problems had `instance "urn:nava:request:"` (no request id). On a real route `httpobs` wraps everything (see S6).
Not covered here (CI-8 covers them): concurrent in-flight duplicate (409 `idempotency_in_flight`), lease takeover, 5xx not stored.
