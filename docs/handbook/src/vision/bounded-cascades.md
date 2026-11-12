---
title: "I9: Cascades are bounded"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/records/pipeline/cascade_integration_test.go, internal/outbox/outbox_integration_test.go, internal/server/records_integration_test.go]
adrs: [ADR-001, ADR-001-A5]
invariants: [I9]
tests: [TestCI10_DepthBeyondLimitRejectedAndAuditedOnTheRunner, TestCI10_DepthBeyondLimitWritesNothing, TestCI10_SiblingAndSelfRetriggeringWritesAreNotCycles, TestCreateRecordIgnoresCausationFromTheRequest]
packages: [internal/records/pipeline, internal/outbox]
status: delivered
---

**I9** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Every write and every outbox row carries a causation record: the id of the origin event and a depth. The pipeline rejects a write beyond a hard, platform-wide depth limit, and rejects a cycle (the same origin seen again on the same record), with a typed error and an audit entry. The limit is platform configuration, never tenant configuration.

## Where it is enforced

The write pipeline (`internal/records/pipeline`) and the outbox (`internal/outbox`), which records `causation_id` and `causation_depth`.

## Proof

Annex B CI-10 (a cascade beyond the limit, or a cycle, is rejected with a typed error and audited). The cycle-suppression rule for workflow starts is amendment ADR-001-A5.

Tests that prove it, by Go test name: `TestCI10_DepthBeyondLimitRejectedAndAuditedOnTheRunner`, `TestCI10_DepthBeyondLimitWritesNothing`, `TestCI10_SiblingAndSelfRetriggeringWritesAreNotCycles`, `TestCreateRecordIgnoresCausationFromTheRequest`.

## Read next

Enforcement detail: [Cascades](../system/core-cascades.md), where I9 is enforced. Also [Interpreter](../workflows/nava-interpreter.md) and [Workflow state](../database/workflow-state.md). All nine invariants: [the invariants page](invariants.md).
