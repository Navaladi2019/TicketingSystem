---
title: "I7: Every side effect is idempotent"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/records/pipeline/pipeline_integration_test.go, internal/records/pipeline/numbering_integration_test.go, internal/temporalstack/ci6_relay_isolation_test.go]
adrs: [ADR-001]
invariants: [I7]
tests: [TestIdempotencyKeyReplays, TestIdempotencyKeyMismatchConflicts, TestCI10_DuplicateDedupeKeyWritesOneRow, TestCI6_RelayRedeliveredSignalAppliedOnceAcrossContinueAsNew]
packages: [internal/platform/idempotency, internal/outbox, internal/relay]
status: delivered
---

**I7** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

A retry, a replay or a redelivery never acts twice. A repeated request with the same idempotency key returns the same response; the same key with a different body is a conflict. A redelivered workflow start or signal is applied once.

## Where it is enforced

The idempotency middleware and the pipeline (`internal/platform/idempotency`, `internal/records/pipeline`), the outbox dedupe key (`internal/outbox`) and the relay's reject-duplicate start (`internal/relay`).

## Proof

Annex B CI-8 (idempotency replay and mismatch) and CI-6 (redelivered start and signal).

Tests that prove it, by Go test name: `TestIdempotencyKeyReplays`, `TestIdempotencyKeyMismatchConflicts`, `TestCI10_DuplicateDedupeKeyWritesOneRow`, `TestCI6_RelayRedeliveredSignalAppliedOnceAcrossContinueAsNew`.

## Read next

Enforcement detail: [Workflows](../workflows/index.md) and [Security](../security/index.md). All nine invariants: [the invariants page](invariants.md).
