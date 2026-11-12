---
title: "otel-collector"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/otel-collector/config.yaml]
packages: [internal/platform/telemetry]
tests: [TestCollectorDenyListMatchesFilter, TestLogDenyListMatchesCollector]
---

The OpenTelemetry Collector. Every `nava-*` service sends traces and logs to it (`OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`).

| | |
|---|---|
| Image | `nava/otel-collector` (built from `deploy/compose/otel-collector`) |
| Network | `internal` |
| Listeners | `:4318` OTLP/HTTP, `:4317` OTLP/gRPC, `:13133` health |
| Host port | none |
| Exports to | the debug exporter, and `openobserve:5080` (OTLP/HTTP, only reachable with the operator profile) |
| DB role | none |
| Introduced by | PLAN-001 T02 (TICK-51) |

Traces are tail-sampled (`OTEL_TAIL_SAMPLE_PERCENT`, default 10%); errors and slow requests are always kept.

## Boundary it enforces

No ADR-001 section 1 invariant is owned here. The boundary is the second layer of the telemetry filter: the Collector deletes denied attributes (URL paths and queries, client addresses, user agents, headers) from spans and logs, behind the in-process allow-list in `internal/platform/telemetry`. `TestCollectorDenyListMatchesFilter` and `TestLogDenyListMatchesCollector` keep the two lists equal.

[Back to the system map](index.md)
