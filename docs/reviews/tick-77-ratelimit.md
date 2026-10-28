# Review: T24 per-tenant rate limit (TICK-77, PR #136)

- Reviewer: Quality, Testing & CI Advisor
- Head reviewed: `242a807`
- CI: green at `242a807` ([run 40344888106](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40344888106)). All 8 jobs pass. The earlier red run (`08f1492`, `check-pins`) was fixed by keeping `google.golang.org/protobuf` as `// indirect`.
- Local: `go test -race -count=3 ./internal/platform/ratelimit ./internal/server ./internal/platform/config` passes.
- Specs: TICK-77 contract and acceptance criteria, the TICK-94 item 14 rulings (validated tenant only, evict idle limiters, document per-replica limits), the TICK-93 metric rule, ADR-001 D6, PLAN-001 T24.
- Out of scope: the tenant isolation review, which belongs to the Security & Multi-Tenancy Reviewer. This review checks only that the key is the resolved tenant.

## Verdict

Approve. No finding blocks the merge. Findings 1 to 3 are optional follow-ups.

## Acceptance criteria

| Criterion | Evidence | Result |
|---|---|---|
| Tenant A flooded gets 429 while tenant B's p95 stays within 2x baseline | `TestFloodOfOneTenantDoesNotSlowAnother`, `TestFloodedTenantGets429WithRetryAfterAndLog` | pass. The bound is weaker than 2x; see finding 1 |
| An idle limiter is evicted; the map stays bounded under many tenants | `TestMemoryEvictsIdleBuckets`, `TestMemoryMapStaysBoundedUnderManyTenants`, `TestActiveBucketIsNotEvicted` | pass |
| A forged tenant header with a token for tenant B is limited as B | `TestRateLimitKeyIsTheResolvedTenantNotAHeader` (full `NewHandler` stack, through `/api/v1/me`) | pass |
| Keyed only on the validated tenant | `ratelimit.go:167-172` reads only `identity.FromContext`. `api.go:68`: auth is outermost, so the limiter only sees resolved principals. `auth.go:70-96` refuses every request without a bearer token. A missing principal gets 500, never "unlimited" (`TestNoPrincipalIs500NotUnlimited`). | pass |
| No metric carries a tenant attribute | The diff adds no meter, counter or attribute. `tenant_id` appears only in the throttle log line (`ratelimit.go:179-180`). | pass |
| Limits are per replica, and this is documented | Package doc `ratelimit.go:8-10`, `config.go` `RateLimit` comment, `.env.example` | pass |
| 429 `rate_limited` with `Retry-After` | `ratelimit.go:177-183`. `rate_limited` is registered in `api/errors.yaml`, and `429` is declared on `getMe`. | pass |

Checked and correct:
- A refused request cancels its reservation, so it costs no token (`ratelimit.go:115-117`).
- A refused request still updates `last`, so a tenant that keeps hitting the limit is never evicted, and eviction never refills its bucket (`TestActiveBucketIsNotEvicted`).
- The config rejects `0`, negative and non-numeric values, with the variable named in the error.

## Findings

### 1. Missing test (non-blocking): the flood test asserts "under 25ms", not "within 2x baseline"

`ratelimit_test.go:265`: `bound := max(2*base, 25*time.Millisecond)`. A baseline request through the middleware takes microseconds, so in practice the bound is always the 25ms floor. The test catches gross head-of-line blocking, for example tenant B waiting behind a long-held lock. It does not catch tenant B getting 10x slower while staying under 25ms. The floor is a reasonable fix for Windows timer jitter, but the test name and the acceptance-criterion row claim more than it checks.

Suggested fix: keep the floor, and rename the bound in the test comment to what it is ("p95 under flood stays below max(2x baseline, 25ms)"). If a tighter bound matters later, add a contention benchmark on `Memory.Allow` (`b.RunParallel` with one hot tenant and one cold one) and compare `ns/op`. That gives a stable number without wall-clock jitter.

### 2. Breaks under many active tenants (non-blocking, accepted for PLAN-001): one global lock, and an O(n) sweep while holding it

`ratelimit.go:100-104, 132-141`: every request from every tenant takes one `sync.Mutex`. Once per `IdleTTL/2` (5 minutes by default), the sweep walks every live bucket while holding that lock. At today's tenant count this costs nothing. With tens of thousands of tenants active within the TTL, the request that triggers the sweep stalls every tenant's requests for the length of the walk. This is the cross-tenant slowdown the flood criterion exists to prevent, triggered by tenant count instead of by load.

Suggested fix when it matters: shard the map by tenant hash, or sweep a bounded number of entries per call. The `Limiter` interface already allows that without touching the middleware. No action now.

### 3. Missing test (minor): no full-stack assertion on `Retry-After` and the problem body

`internal/server/ratelimit_test.go` checks the 429 status through `NewHandler`, but not the `Retry-After` header or the `rate_limited` body. The package test covers both on the bare middleware. A full-stack check would catch a later change in middleware order, for example the response validator rewriting or stripping the 429. Add two assertions to `TestRateLimitKeyIsTheResolvedTenantNotAHeader`.

## Note for the spec owner (no code change)

RFC-001 (`docs/adr/ADR-001/rfc/RFC-001-platform-blueprint.md:121`) says "Token bucket per tenant and per principal. Over the limit returns 429 with `RateLimit-*`." The accepted ADR-001 D6 and the TICK-77 contract say per tenant only, with `Retry-After`. The code follows the ADR and the contract, which is correct. If per-principal limits or `RateLimit-*` headers are still wanted, they need their own task.
