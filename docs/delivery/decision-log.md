# Decision log

Delivery decisions that change a task's contract, newest first.

## 2026-10-16: T01 telemetry on OpenTelemetry; the edge is a trace trust boundary
- **Decided by:** nava (human), on TICK-89. Shaped by the Technical Lead's review.
- **Decision:** nava-core uses the OTel API for traces and metrics. Prometheus stays only as the `/metrics` scrape format, through the OTel Prometheus exporter. The public listener uses `otelhttp.WithPublicEndpoint()`, so an inbound `traceparent` is linked and never continued. The edge propagator is TraceContext only, which drops inbound baggage. Sampling is `parentbased_traceidratio`. The request id stays server-generated.
- **Supersedes:** T01's "inbound `traceparent` is honoured" and the metric names `http_requests_total` and `http_request_duration_seconds`.
- **Dissent:** the Technical Lead preferred a separate T01c, merged after T01, to keep the PLAN-001 root unblocked. The human chose the same PR. T02 and T20 wait on it.
- **Deferred:**
  - OTLP log export (`otelslog`). Trigger: the T02 Collector exists.
  - Collector tail sampling. Trigger: T02.
  - Trace continuity across the outbox (the producer stores the trace context on the outbox row, and the relay starts the workflow as a linked span). Trigger: T29 and T30a start.
