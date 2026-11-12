---
title: "I6: The durable path is deterministic, versioned and entered only through the outbox"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/temporalstack/demo_outbox_test.go, internal/temporalstack/ci6_relay_isolation_test.go, internal/temporalstack/ci7_isolation_test.go]
adrs: [ADR-001]
invariants: [I6]
tests: [TestDemoOutboxTwiceWithOneKeyIsOneWorkflowWithTwoSignalsInOrder, TestCI6_RelayRedeliveredStartAfterCompletionStartsNoSecondRun, TestCI7_ProducerSchemaRejectsRecordDataFields]
packages: [internal/outbox, internal/relay, internal/workflow]
status: delivered
---

**I6** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Workflow code is deterministic: all I/O, clocks, randomness and tenant code run in activities. A running execution keeps the version it started on. Nothing starts or signals a workflow except the relay, dispatching an outbox row that was committed in the same transaction as the change that caused it.

## Where it is enforced

The transactional outbox (`internal/outbox`), the relay (`internal/relay`), the workflow interpreter (`internal/workflow`) and the Temporal Authorizer, which refuses every other caller (`internal/temporalstack`).

## Proof

Annex B CI-6 (relay start and signal behaviour, certificate and command checks) and CI-7 (payloads carry references only).

Tests that prove it, by Go test name: `TestDemoOutboxTwiceWithOneKeyIsOneWorkflowWithTwoSignalsInOrder`, `TestCI6_RelayRedeliveredStartAfterCompletionStartsNoSecondRun`, `TestCI7_ProducerSchemaRejectsRecordDataFields`.

## Read next

Enforcement detail: [Outbox, relay and worker](../workflows/outbox-relay-worker.md). All nine invariants: [the invariants page](invariants.md).
