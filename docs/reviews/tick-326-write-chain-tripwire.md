# Review: TICK-326 write-chain tripwire (PR #279)

**Verdict: APPROVE.** Nothing blocks the merge. Two findings are for the first task that adds a tenant write route.

Read first: the TICK-326 ask (R2 from `docs/acceptance/regression-milestone-1.md`), `internal/platform/idempotency/idempotency.go`, `internal/server/api.go` (`newAPI`) and `internal/server/operator.go` (`routeAuth`).

## What was checked

- **Enumeration** comes from the embedded `api/openapi.yaml` through `apihttp.LoadSpec`, the same file `newAPI` mounts. It is not a hand-written list. A spec with zero writes fails the test (`len(all) == 0`), so a broken parser cannot pass on nothing. The tenant/operator split uses the real `operatorPrefix` (exported through `export_test.go`), so it cannot drift from `routeAuth`.
- **The probes match the middleware.** `idempotency.Handler` acts on every unsafe method, DELETE and PUT included. It checks the key before it reads the body and counts `rejected` only there, so P3's metric delta of 1 is that middleware's own signature. The validator can emit the same `/header/Idempotency-Key` error, but it cannot move that counter. The malformed body in P3 also catches a validator or handler mounted outside idempotency.
- **Order.** P1 catches anything outside authn: the limiter or the counter moves, or the answer is not 401. P2 catches idempotency outside ratelimit (400 instead of 429) and checks that the limiter got the tenant authn resolved. The six fixtures use the real `auth`, `ratelimit` and `idempotency` constructors, not fakes. Each one fails for a different probe.
- **Red on the real handler.** `POST /api/v1/me` fails P3 only, which matches R2 exactly.
- **Ran locally** (`go test ./internal/server/ -run 'WriteChain|TenantWriteRoutes' -count=1`): all pass. It logs `contract writes: [POST /api/v1/operator/tenants]; tenant writes checked: []`. CI `unit` is green on the PR.

## Findings

### 1. Breaks under nava-core wiring that differs from the test wiring (must be in the contract for the first write route)
`internal/server/write_chain_test.go:58` (`newTripwireHandler`). The tripwire probes a handler the test builds itself. The stated contract says to mount idempotency "with the same option `cmd/nava-core` uses". If idempotency arrives as an optional `server.With...` option, the test can pass it while `cmd/nava-core` forgets it, and production ships writes with no idempotency while the tripwire stays green. That is R2 again, one level up.
**Fix (for the owning engineer of the first write route):** make it impossible to construct the handler without idempotency. Either `NewHandler` takes the idempotency store as a required argument, or it returns an error when the contract has a tenant write route and no store is configured. Then the test cannot diverge from production. A weaker alternative is to build the handler through the same constructor `cmd/nava-core` calls. Add this sentence to the "Contract for later work" in `docs/security/isolation-suite.md` when that task lands.

### 2. Missing test: idempotency scoped per tenant on the real handler (later, with the first write route)
The probes prove the middleware runs and in which order. They do not prove that tenant A's key and tenant B's key with the same value stay separate on the real route. `idempotency_test.go` covers that for the middleware itself. Once a route mounts it, add a testcontainers case: same key, two tenants, two different responses, no replay. It is an isolation property, so route it to the Security & Multi-Tenancy Reviewer when that task is reviewed. It does not belong in this PR.

### 3. Preference: exact `WWW-Authenticate` match
`write_chain_test.go:205` requires `WWW-Authenticate` to equal `Bearer`. If auth later adds RFC 6750 parameters (`Bearer error="invalid_token"`), every tenant write route goes red even though the chain is correct. `strings.HasPrefix(..., "Bearer")` would keep the signal without the brittleness.

### Residual risk (accepted as stated in the PR)
A write route mounted on the mux outside the spec is not enumerated. Today every `/api/v1` route goes through the generated router, and an unmatched path is 404/405 behind authentication, so the risk is theoretical.

## How to run
`task test` locally, or the `unit` job of the GitHub `ci` workflow.
