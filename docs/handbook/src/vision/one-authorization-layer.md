---
title: "I5: One authorization layer, inside the query"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/tenancy/isolation/i5_agreement_isolation_test.go, internal/records/pipeline/acl_integration_test.go, internal/server/records_integration_test.go, internal/server/records_list_test.go]
adrs: [ADR-001]
invariants: [I5]
tests: [TestI5Agreement, TestCompileReadAgreesWithAllows, TestRecordFourNotFoundCasesAreByteIdentical, TestMaskedFieldInFilterIsRefusedBeforeSQL]
packages: [internal/authz, internal/records/pipeline]
status: delivered
---

**I5** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Access rules are compiled into SQL, and the single-record check in Go gives the same answer as the SQL. Field masks apply on write and on read. A record the caller cannot read, a record of another tenant and a record that does not exist all produce the same response.

## Where it is enforced

The ACL compiler and checker (`internal/authz`) used by the records pipeline and the list queries (`internal/records/pipeline`, `internal/server`).

## Proof

Annex B CI-8 (byte-identical 404s, a masked-field filter returns 400) and the I5 agreement suite in `internal/tenancy/isolation`.

Tests that prove it, by Go test name: `TestI5Agreement`, `TestCompileReadAgreesWithAllows`, `TestRecordFourNotFoundCasesAreByteIdentical`, `TestMaskedFieldInFilterIsRefusedBeforeSQL`.

## Read next

Enforcement detail: [Security](../security/index.md). All nine invariants: [the invariants page](invariants.md).
