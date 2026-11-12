---
title: "nava-onboard"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-onboard/main.go, db/grants.yaml]
adrs: [ADR-001, ADR-001-E3]
packages: [cmd/nava-onboard, internal/onboard]
invariants: [I2]
tests: [TestOnboardIgnoresSuppliedTenantID, TestOnboardDeploymentSeparation, TestForgedIssuerTokenRejectedByOnboard]
---

The operator onboarding API, `POST /api/v1/operator/tenants`, in its own process. It is the only process that holds the onboarding service account's secret and the `nava_onboard` database URL.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:8081` operator API (`NAVA_ONBOARD_ADDR`), `:9090` admin |
| Host port | none: `docker compose exec`, or tunnel 18082 (API) and 19091 (admin) |
| DB roles | `nava_app` (token checks, registry reads) and `nava_onboard` (onboarding writes), both through PgBouncer |
| Talks to | keycloak (organisation and admin group, Admin API); pgbouncer; otel-collector |
| Introduced by | PLAN-001 T21b (TICK-234) |

Being unpublished means the tenant ingress (nava-core's port) cannot reach it.

## Boundary it enforces

**I2.** Only allow-listed operators from the pinned issuer may onboard, and a tenant id in the request is ignored: `TestOnboardIgnoresSuppliedTenantID`, `TestForgedIssuerTokenRejectedByOnboard`, and `TestOnboardDeploymentSeparation` for the separate process.

[Back to the system map](index.md)
