---
title: "nava-relay"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-relay/main.go, db/grants.yaml, docs/architecture/outbox-relay.md]
adrs: [ADR-001]
packages: [cmd/nava-relay, internal/relay]
invariants: [I6]
tests: [TestCI6_RelayCoreWorkerOutsideAllowListDenied, TestComposeRelayService]
---

The outbox relay: claims pending `outbox` rows of every tenant and starts or signals their workflows.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only |
| Host port | none (tunnel 19093) |
| DB role | `nava_relay` through PgBouncer; dispatch columns of `outbox` only |
| Talks to | temporal (start, signal, with the `nava-relay` certificate); pgbouncer; otel-collector |
| Introduced by | PLAN-001 T30b (TICK-85) |

`/readyz` follows both PostgreSQL and Temporal.

## Boundary it enforces

**I6.** Nothing starts or signals a workflow except the relay, from an outbox row. `TestCI6_RelayCoreWorkerOutsideAllowListDenied` proves each certificate is held to its own calls.

[Back to the system map](index.md)
