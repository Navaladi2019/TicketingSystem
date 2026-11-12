---
title: "temporal (gateway)"
section: system
status: delivered
sources: [compose.yaml]
adrs: [ADR-001]
invariants: [I6]
tests: [TestCI6_UnlistedCertificateDeniedByAuthorizer, TestCI6_RelayCoreWorkerOutsideAllowListDenied]
---

A TCP proxy (`temporal-server proxy -listen :7233 -to temporal-server:7233`) from the same image. It forwards port 7233 and nothing else, holds no certificate, and does not end TLS: each client's mutual TLS reaches the server end to end.

| | |
|---|---|
| Networks | `internal`, `operator`, `temporal-backplane` |
| Listener | `:7233` |
| Host port | none |
| DB role | none |
| Clients | `nava-relay` (start, signal), `nava-worker` (poll `nava-tq`), `nava-core` (readiness probe only), `temporal-ui` |
| Introduced by | PLAN-001 T02 (TICK-51) as the Temporal service; a gateway since PLAN-001 T28 |

## Boundary it enforces

**I6.** Because the gateway forwards only 7233, a platform process can speak only the frontend API, and only with its own certificate. The server's authorizer then allows each certificate only its listed calls: `TestCI6_UnlistedCertificateDeniedByAuthorizer` and `TestCI6_RelayCoreWorkerOutsideAllowListDenied`.

[Back to the system map](index.md)
