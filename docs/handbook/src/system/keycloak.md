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
- **nava-groupsync**: reads users, groups and memberships (Admin API), and the org-unit groups under `/t/<tenant_id>/ou/`.

## The `nava_primary_org_unit` attribute (S31)

The `nava` realm user profile declares `nava_primary_org_unit`, a user's
primary org unit when they are in more than one. It is **admin-only**, for
view and edit, so users cannot set it themselves. Group sync reads it to pick
the user's primary unit (D-11-6; see [nava-groupsync](nava-groupsync.md#org-units-s31)).
Filling it from a tenant's own identity provider through a mapper is
deferred to a follow-up issue. Since S31 the realm fixture also seeds tenant
A's `EMEA`, `EMEA/Germany` and `APAC` units, the `/requesters` group and five
users for the S31 kit.

## Boundary it enforces

**I2, the tenant comes only from a platform-verified source.** The issuer is pinned (`KC_HOSTNAME`), so a token's `iss` never follows the request's `Host` header, and every validator trusts exactly that one value. `TestComposeKeycloakIssuerPinned` and `TestEveryTokenValidatorTrustsPinnedIssuer` prove it.

[Back to the system map](index.md)
