---
title: "I4: One write pipeline with optimistic concurrency"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/records/pipeline/pipeline_integration_test.go, internal/server/records_patch_test.go, internal/records/arch_test.go]
adrs: [ADR-001]
invariants: [I4]
tests: [TestEveryWriteBumpsVersionOnce, TestIfMatchVersions, TestRecordWritersOnlyInPipeline, TestPipelineImportRule]
packages: [internal/records/pipeline, internal/server]
status: delivered
---

**I4** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Every writer goes through the same steps: authorisation, defaults and computed values, JSON Schema validation, declarative checks, then commit with the audit entry in one transaction. Every update carries the version of the record it expects. A write against a stale version is a typed conflict, never last-writer-wins.

## Where it is enforced

The records pipeline (`internal/records/pipeline`), reached from the API handlers in `internal/server`. An architecture test keeps record writers inside the pipeline.

## Proof

Annex B CI-8 (a stale `If-Match` returns 412) and the architecture tests in `internal/records/arch_test.go`.

Tests that prove it, by Go test name: `TestEveryWriteBumpsVersionOnce`, `TestIfMatchVersions`, `TestRecordWritersOnlyInPipeline`, `TestPipelineImportRule`.

## Read next

Enforcement detail: [Tenant transaction](../security/03-transaction.md) and [Records](../database/records.md). All nine invariants: [the invariants page](invariants.md).
