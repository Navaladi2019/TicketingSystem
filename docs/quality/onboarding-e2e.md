# Onboarding end to end on the stack (T21b, TICK-234)

## Scope

The path from operator onboarding to a resolving tenant token, on the compose
stack: `nava-onboard` creates the tenant, the invited admin signs in through the
tenant's IdP (`nava-first-broker-login`), `nava-groupsync` syncs them, and
`nava-core` resolves `GET /api/v1/me`. Plus the two Security conditions from the
TICK-81 re-review and the kill/retry from its CI review.

Tests: `internal/identitystack/onboard_e2e_test.go` (build tag `identitystack`).

## Risks, ranked, and what covers each

| # | Risk | Test | Layer |
| - | ---- | ---- | ----- |
| 1 | An onboarded tenant cannot be used: the invite, the first broker login, the group sync and token resolution do not compose (each is only tested alone). | `TestOnboardedAdminSignsInAndMeShowsTenant`: 201, then a 200 replay; the binding row; brokered sign-in; the token `sub` is the invited user; the user is in `/t/<tenant_id>/` and an org member; one groupsync cycle, with the long-running `nava-groupsync` stopped for the test (its own cycle could hold the tenant's lock, so `--once` would skip it); `/me` gives the tenant and its IdP alias. | stack |
| 2 | A new admin's token resolves to another tenant. | Same test: `/me` is the new tenant with `X-Tenant-Id` or `?tenant_id=` naming another; the token carries exactly one organization. | stack |
| 3 | `nava-core` holds the onboarding secret (a `nava-core` compromise links an attacker IdP, TICK-81 S2). | `TestOnboardDeploymentSeparation`: separate containers; no `NAVA_ONBOARD_*` name, onboarding secret or `nava_onboard` URL in `nava-core`'s environment, with a control showing the same scan finds them in `nava-onboard`. | stack |
| 4 | The operator API is reachable through the tenant ingress. | Same test: `nava-onboard` publishes no port and is on `internal` only; `POST /api/v1/operator/tenants` with a valid operator token on `nava-core`'s port is 501 (or 404) and writes no tenant row. | stack |
| 5 | A crash between the Keycloak organization and the binding leaves duplicates or a half-bound tenant. | `TestOnboardKillBeforeBindThenRetryLiveStack`: a trigger fails the binding insert for this tenant only (the transaction never commits, as when the process dies there); 5xx, one org, `provisioning`, no binding or domain row; trigger dropped, the same key gives 201, one org, one binding naming it, one domain, `active`, one IdP. Real `nava-onboard`, `nava_onboard` PGStore and Keycloak. | stack |

## Fixtures

- `operator@nava.test` (realm export, fixed id `...f001`, on `NAVA_OPERATORS` in `compose.yaml`).
- Per run: a fresh `Idempotency-Key`, so a fresh tenant id; domain `onb-<8 hex>.test`;
  an IdP realm `onb-<8 hex>` with client `nava-broker` and the admin user. Removed after
  the test, with the tenant's organization, IdP and `nava` user. Tenant rows stay
  (there is no abandon operation yet, AR-036); every run's are distinct.
- Tokens are fetched with `Host: keycloak:8080`, so their issuer is the one the
  platform containers accept.

## Deliberately not covered, and why

- **The IdP's URLs.** Onboarding refuses non-https and private hosts, and the stack
  has no public IdP. The request names public address literals; the test then
  repoints only the three endpoint URLs at the test realm, as the master admin.
  Alias, organization link, first-login flow and the pinned `validateSignature` and
  `syncMode` stay as onboarding created them (the test checks the last two first).
  The URL validation itself is unit-tested in `internal/onboard`.
- **The invite email.** The stack sends none; the invite is the user in the tenant
  group, which the test checks.
- **Groupsync on its interval.** The test runs one `--once` cycle; the 5m loop is the
  T25 suite's.

## How to run

Locally, on a running stack (story acceptance or a human debugging, never as a
substitute for CI):

```
make stack-up
make stack-test-onboard
```

In CI: the `stack` job's `make stack-test-identity` step runs these tests until the
workflow has its own `make stack-test-onboard` step (TODO-012).
