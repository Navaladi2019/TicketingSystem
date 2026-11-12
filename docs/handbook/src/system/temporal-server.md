---
title: "temporal-server"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/temporal/config, temporal-server/cmd/server/server_test.go]
adrs: [ADR-001, ADR-001-A1]
invariants: [I6]
tests: [TestCI6_PlatformNetworkCannotReachInternalPorts, TestInternodeRejectsFrontendClientCertificates, TestBootstrapRegistersNamespaceAndSearchAttribute]
---

The Temporal server, built as `nava/temporal-server` from `temporal-server/`. It runs with `--bootstrap`: it registers namespace `nava` and the `NavaTenantId` search attribute itself, with the `temporal-system` certificate that only it holds.

| | |
|---|---|
| Network | `temporal-backplane` only |
| Listeners | `:7233` frontend (client certificates required); internode and membership ports, reachable only on the backplane |
| Host port | none |
| DB role | `temporal`, owner of `temporal` and `temporal_visibility`, connecting directly |
| Runs after | `temporal-schema` completed |
| Introduced by | PLAN-001 T28 |

Healthy means bootstrapped: the health check dials the frontend and requires the namespace and the search attribute. Clients never reach it directly; they go through the [temporal gateway](temporal.md).

## Boundary it enforces

**I6, the durable path is entered only through the outbox.** Only the gateway's port 7233 is reachable from `internal`, and the internode port has its own CA. `TestCI6_PlatformNetworkCannotReachInternalPorts` and `TestInternodeRejectsFrontendClientCertificates` prove both.

[Back to the system map](index.md)
