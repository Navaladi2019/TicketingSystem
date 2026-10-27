# Review: T21c onboarding retry refuses a binding that names a different org (TICK-242, PR #157)

Reviewer: Quality, Testing & CI Advisor. Head `482f5f9`, base `TICK-46/operator-onboards-tenant`.

**Verdict: APPROVE.** No must-change findings.

## Acceptance criteria

| Criterion | Result | Evidence |
|---|---|---|
| Bind (issuer, org A), then Bind (issuer, org B) returns `ErrConflict`, tenant stays `provisioning`, binding unchanged | Pass | `TestT21BindRefusesARetryWithADifferentOrg` checks a different org and a different issuer. It asserts state, the binding row and the domain count. It passed in the CI integration job (run 40353754271). |
| A same-args retry still succeeds with no change | Pass | `TestT21BindRetryChangesNothing` and `TestT21BindIsIdempotentAndActivationIsLast` are unchanged and green. |
| AR-036 covers the stuck tenant | Pass | AR-036 row and `docs/operator/onboarding.md` updated |

## End-to-end path

- `PGStore.Bind` (`internal/onboard/store.go:133-141`) returns `ErrConflict` before any insert. `Service.Onboard` (`internal/onboard/onboard.go:177`) returns before the domain check, the invite and `Activate`, so the tenant cannot go `active`.
- `onboardError` (`internal/server/operator.go:196`) maps `ErrConflict` to 409 `onboarding_conflict`. It is covered by `operator_test.go:166`, and `onboard_test.go:311` covers a Bind conflict that does not activate.
- `Registry.Binding` (`internal/onboard/registry.go:45`) runs in `InTenantTx` with `WHERE tenant_id = $1`. As the comments now say, that clause scopes the read, not RLS. `pgx.ErrNoRows` maps to `found=false`. Any other scan error propagates with `found=false`.
- Concurrency: two retries of one tenant are serialized by the onboarding lock taken in `Begin`. Even without it, the loser of a racing first insert hits the unique key and gets `ErrConflict`, so no new race is introduced.

## Findings

1. **Missing test (should-change, non-blocking).** No test fails if `WHERE tenant_id = $1` is dropped from `Registry.Binding`. `binding_lookup` is `USING (true)` for `nava_app`, so without the clause, tenant B's first `Bind` would read tenant A's row and get `ErrConflict`. No onboarding after the first would ever succeed. The only two-tenant Bind test (`TestT21BindRefusesAnOrgBoundToAnotherTenant`) uses the same org for both tenants, so it expects `ErrConflict` either way. Suggested test: bind tenant A to `org-a`, then a first `Bind` of tenant B to `org-b` succeeds, and both binding rows hold their own org. The failure would be loud, not an isolation leak, so this does not block.
2. **Preference.** `docs/operator/onboarding.md:118` and the `Bind` doc comment at `internal/onboard/store.go:114` have one line well over the wrap width the rest of the file uses.
3. **Preference, pre-existing.** The comment at `internal/onboard/onboard.go:180` ("Bind skips rows that already exist, whoever owns them") no longer describes `Bind`. `Bind` now skips only when the stored pair matches, and refuses otherwise. A wording fix can ride with finding 1.

Tenant isolation is not reviewed in depth here (Security & Multi-Tenancy Reviewer). The change fails closed.

## CI

GitHub `ci` run 40353754271 on `482f5f9`: changes, lint, unit, temporal-server, integration, stack, stack-db and stack-internal all passed.

## Closure

QA confirmed all three findings closed on `4005b68`. The two follow-ups that came out of the review (the service-level assertion on a Bind conflict, and the one-binding-per-tenant assumption in `Registry.Binding`) were done in TICK-249.
