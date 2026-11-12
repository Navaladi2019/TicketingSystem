---
title: "I8: Contracts are the boundary; configuration, not code; AI is advisory"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, .golangci.yml, README.md, internal/identity/keycloak/contract_test.go]
adrs: [ADR-001, ADR-006]
invariants: [I8]
tests: [TestContract, TestCompileNumberingMode, TestAllocationTransactionIsTenantScoped]
packages: [internal/identity, internal/identity/keycloak]
status: delivered
---

**I8** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Tenant code, plugins and AI reach data only through the capability contract or the public API, with a scoped token. They never reach the database or Temporal directly. A tenant change never needs a schema change, with one exception: an operator-approved, measured and audited indexed attribute, run by the migrator outside the tenant's change set. Per-tenant numbering uses a counter table, not sequences. AI may change data only above an explicit confidence threshold, and the decision is recorded.

## Where it is enforced

Ports and adapters in the Go packages (the identity port is the worked example, `internal/identity`), a `depguard` import rule in `.golangci.yml`, and the numbering allocator.

## Proof

Annex B CI-11 is a static import and lint check, not a Go test: only the identity adapter package may import a Keycloak client. The contract suite `TestContract` runs the adapter against the port's contract. **No test on `main` covers the AI confidence threshold:** no AI code that writes data is on `main`, so that clause has nothing to prove yet.

Tests that prove it, by Go test name: `TestContract`, `TestCompileNumberingMode`, `TestAllocationTransactionIsTenantScoped`.

## Read next

Enforcement detail: [Identity boundary](../security/07-identity-boundary.md) and [Numbering](../database/numbering.md). All nine invariants: [the invariants page](invariants.md).
