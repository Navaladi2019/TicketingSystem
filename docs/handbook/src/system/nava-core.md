---
title: "nava-core"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-core/main.go, cmd/nava-core/deps.go, internal/platform/config/config.go, db/grants.yaml]
adrs: [ADR-001, ADR-004-A1]
packages: [cmd/nava-core, internal/server]
invariants: [I2]
tests: [TestGetMeIgnoresSuppliedTenant, TestComposeAuditReaderDSNOnNavaCoreOnly, TestComposePublishesNoAdminPort]
---

The core API service: every tenant-facing `/api/v1` route. It is the only platform service with a published port.

| | |
|---|---|
| Networks | `default`, `internal` |
| Listeners | `:8080` public (`NAVA_HTTP_ADDR`), `:9090` admin with `/metrics`, `/healthz`, `/readyz` (`NAVA_ADMIN_ADDR`) |
| Host port | `127.0.0.1:18080` (`NAVA_CORE_PORT`); the admin port is never published (tunnel 19090) |
| DB roles | `nava_app` on pool `nava`; `nava_app` on pool `nava_alloc` (record-number allocator); `nava_audit_reader` on `nava` (the only service holding it). All through PgBouncer |
| Talks to | pgbouncer; keycloak (discovery, JWKS, health); temporal (readiness probe with the `nava-core` certificate); otel-collector |
| Introduced by | PLAN-001 T02 (TICK-51) |

`/readyz` follows PostgreSQL. It never runs migrations and answers the operator onboarding routes `501`: those live in [nava-onboard](nava-onboard.md).

## Boundary it enforces

**I2.** The tenant comes from the validated token through `tenant_identity_binding`, never from the client. `TestGetMeIgnoresSuppliedTenant` proves a supplied tenant is ignored.

[Back to the system map](index.md)
