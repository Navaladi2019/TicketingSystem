---
title: "I2: The tenant comes only from a platform-verified source"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/identitystack/ci5_isolation_test.go, internal/platform/cursor/request_test.go, internal/server/me_onboard_integration_test.go, internal/temporalstack/ci6_isolation_test.go]
adrs: [ADR-001]
invariants: [I2]
tests: [TestCI5, TestClaimsForIgnoresClientTenant, TestGetMeIgnoresSuppliedTenant, TestCI6_WorkerWrongTenantSignalDroppedWithAlertWorkflowKeepsRunning]
packages: [internal/identity, internal/identity/keycloak, internal/platform/cursor, internal/temporalstack]
status: delivered
---

**I2** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

The tenant of a request is never taken from anything the caller supplies. It comes from a validated token, resolved through the platform's identity binding table (decision D26), or from a row the platform itself wrote under row-level security (the outbox row, decision D22). Request bodies, payloads, script arguments, workflow inputs, signal bodies and search attributes are never a source. Temporal is a path like any other.

## Where it is enforced

The identity port and its Keycloak adapter (`internal/identity`, `internal/identity/keycloak`) resolve the token to a tenant; the API layer passes only that value on; the relay reads the tenant from the outbox row; the Temporal Authorizer checks workflow ids and commands (`internal/temporalstack`).

## Proof

Annex B CI-5 (a catalogue of rejected tokens: zero or two organizations, unknown issuer, wrong-realm key, ID token as access token, foreign IdP, duplicate binding) and CI-6 (Temporal).

Tests that prove it, by Go test name: `TestCI5`, `TestClaimsForIgnoresClientTenant`, `TestGetMeIgnoresSuppliedTenant`, `TestCI6_WorkerWrongTenantSignalDroppedWithAlertWorkflowKeepsRunning`.

## Read next

Enforcement detail: [Security](../security/index.md) and [Workflows](../workflows/index.md). All nine invariants: [the invariants page](invariants.md).
