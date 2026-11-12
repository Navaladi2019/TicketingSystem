---
title: "temporal-ui"
section: system
status: delivered
sources: [compose.yaml, docs/acceptance/guide/tools/tunnels.sh]
adrs: [ADR-001]
invariants: [I6]
tests: [TestCI6_OperatorLimitedToPinnedList, TestOperatorServicesNeedTheProfile]
---

The Temporal web UI, for operators. Off by default: it starts only with `docker compose --profile operator up -d`.

| | |
|---|---|
| Image | `temporalio/ui:2.40.1` |
| Network | `operator` only |
| Listener | `:8080` |
| Host port | none: `make operator-tunnel`, or `tunnels.sh up` (127.0.0.1:18233) |
| Talks to | `temporal:7233` with the `nava-operator` certificate, namespace `nava` |
| DB role | none |
| Introduced by | PLAN-001 T28 |

It publishes no port on purpose: a published port is reachable from platform containers through the host.

## Boundary it enforces

**I6.** The operator certificate may make only a pinned list of calls, so the UI cannot start or signal a workflow: `TestCI6_OperatorLimitedToPinnedList`.

[Back to the system map](index.md)
