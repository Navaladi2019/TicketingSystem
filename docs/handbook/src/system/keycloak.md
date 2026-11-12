---
title: "keycloak"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/keycloak/Dockerfile, deploy/compose/README.md]
adrs: [ADR-001]
invariants: [I2]
tests: [TestComposeKeycloakIssuerPinned, TestEveryTokenValidatorTrustsPinnedIssuer]
---

The identity provider. Realm `nava` is imported at first boot from `deploy/compose/keycloak/import`.

| | |
|---|---|
| Image | `nava/keycloak` (built from `deploy/compose/keycloak/Dockerfile`, `start --optimized`) |
| Networks | `default`, `internal` |
| Listeners | `:8080` HTTP (public and back channel), `:9000` management (health; unpublished) |
| Host port | `127.0.0.1:18081` (`KEYCLOAK_PORT`) |
| DB role | `keycloak`, owner of database `keycloak`, connecting directly to `postgres:5432` |
| Introduced by | PLAN-001 T02 (TICK-51) |

## What talks to it

- Browsers and API clients log in on the published port.
- **nava-core**: discovery and JWKS on `keycloak:8080`, and its health check on `:9000` and the token endpoint.
- **nava-onboard**: creates the tenant organisation and its admin group (Admin API).
- **nava-groupsync**: reads users, groups and memberships (Admin API).

## Boundary it enforces

**I2, the tenant comes only from a platform-verified source.** The issuer is pinned (`KC_HOSTNAME`), so a token's `iss` never follows the request's `Host` header, and every validator trusts exactly that one value. `TestComposeKeycloakIssuerPinned` and `TestEveryTokenValidatorTrustsPinnedIssuer` prove it.

[Back to the system map](index.md)
