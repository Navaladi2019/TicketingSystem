# Observability notes: how telemetry flows, what protects it, what comes next

Recorded 2026-10-17 from design Q&A on T01. This explains the shape that
[nava-core-skeleton.md](nava-core-skeleton.md) defines and records options
for later tasks. Decisions still live in the
[decision log](../delivery/decision-log.md) and ADR-001; deferred work lives
in the [TODO tracker](../delivery/todo.md).

## 1. Where each signal goes

```
                        nava-core process
 HTTP request
   │
   ▼ otelhttp (OTel instrumentation)
   ├── span ──► TracerProvider ──► filteringExporter ──► OTLP exporter ──► push ──► Collector / Tempo / Jaeger
   │                                                   (only if an OTEL_EXPORTER_OTLP_*ENDPOINT is set)
   │
   ├── measurement ──► MeterProvider (+ View) ──► otelprom reader ──┐
   │   runtime.Start (go_* metrics) ─┘                              │
   │                                         process collector ─────┤
   │                                                                ▼
   │                                               prometheus.Registry ──► GET /metrics ◄── pull ── Prometheus
   │
   └── slog ──► ContextHandler (+trace_id, span_id, request_id) ──► JSON ──► stdout ──► container runtime / log agent

 OTel's own errors ──► telemetry.RouteErrors ──► slog (same stdout)
```

| Signal | Inside the app | Leaves the app as | Status |
|---|---|---|---|
| Traces | OTel API + SDK | OTLP/HTTP push | Exported only when an endpoint is set; spans always exist so logs have a `trace_id` |
| Metrics | OTel API + SDK | Prometheus text/OpenMetrics, scraped (pull) | OTel is the instrumentation layer; Prometheus is the format and storage (decision log, 2026-10-16) |
| Logs | `slog` only, no OTel | JSON lines on stdout | Collected by the platform. OTLP log export (`otelslog`) is deferred to T02 |

Correlation between them:

| From → to | Link |
|---|---|
| Log → trace | `trace_id`, `span_id` on every request log line |
| Metric → trace | Exemplars on duration buckets carry a `trace_id` (OpenMetrics; Prometheus needs `--enable-feature=exemplar-storage`) |
| Client → everything | `X-Request-Id` header = `request_id` in logs = `nava.request_id` on the span |

## 2. Security controls

The common threat: a hostile client getting its own data into, or control
over, our telemetry. The rules: the edge decides, never the client; only
bounded, non-client values leave the process; secrets are never echoed.

### 2.1 The edge is a trace trust boundary

| Control | Stops | Where |
|---|---|---|
| Every request starts a new root trace; an inbound `traceparent` is only linked (`WithPublicEndpointFn` → true) | Forged spans in our traces; chosen or colliding trace ids | `internal/platform/httpobs` |
| Sampling is decided by the server; an inbound sampled flag is ignored | A client forcing 100% tracing (cost DoS) | otelhttp + sampler |
| Propagator is W3C TraceContext only (B3, Jaeger, AWS, GCP ignored) | Other header formats sneaking in a parent | `internal/platform/telemetry` |
| Inbound baggage is dropped | `baggage: tenant_id=…,role=admin` reaching code that might trust it | propagator |
| `tracestate` is stripped from links | Arbitrary client vendor data being exported | `filter.go` |
| Request id is server-generated, never from a header, never derived from the trace id | Spoofed or colliding request ids; log injection | `requestid`, `httpobs` |

### 2.2 Spans carry no client values or PII

The export-time filter in `internal/platform/telemetry/filter.go` runs on
every span, event and link just before export. `filter.go` drops raw paths
and queries (`url.*`, `http.url`, `http.target`), client identity and headers
(`user_agent.original`, `client.address`, `network.peer.*`,
`http.request.header.*`), client-chosen values (`server.port`,
`http.request.method_original`, `network.protocol.version`), free text
(`db.statement`, `db.query.text`, `exception.message`) and the status
description (`filteredSpan.Status()`). `server.address` is fixed to `nava-core`
(`WithServerName`).

Backing tests: `TestServerSpanAttributeKeysAreGolden` pins the server span's
exact key set, so an otelhttp upgrade that adds a key fails CI until someone
decides on it. `edge_adversarial_test.go` writes crafted requests onto raw
TCP with a marker in every client-controlled slot and fails if it reaches a
span, link, metric or log.

### 2.3 Metrics are bounded

- The View keeps only method (standard, else `_OTHER`), status, route
  pattern (never the raw path), scheme, protocol name and `error.type`.
  Otherwise a client could create a series per request and exhaust
  Prometheus memory.
- Exemplars carry only trace and span id. Without the wrapping reservoir the
  SDK attached the View's dropped attributes as exemplar labels, putting
  client values back on `/metrics` and breaking the 128-character limit.
- Our own registry, never the Prometheus default one.
- Tenant and user ids are never metric labels (section 5).

### 2.4 Logs are bounded

- Access line: normalised `method`, `route` as the pattern or `unmatched`;
  no raw path, query, headers or client IP.
- `sloglint context: all` forces `*Context` calls, so every request line
  carries the correlation ids.

### 2.5 Config and secrets

- Config errors name the variable, never its value.
- OTLP endpoints are validated as http(s) URLs without echoing them (they can
  carry credentials).
- Exporter credentials (`OTEL_EXPORTER_OTLP_HEADERS`, certificates) are read
  only by the exporter; `.env` is gitignored, `.env.example` holds no
  secrets. In Kubernetes they come from Secrets.

### 2.6 HTTP server

- Read-header, read, write and idle timeouts (slowloris protection).
- Panic recovery: logged 500 with the handler's headers cleared, connection
  aborted if the response had started; the server keeps serving.
- Graceful shutdown drains in-flight requests on a signal or a serve
  failure, then flushes telemetry on its own deadline.

### 2.7 Known gaps

| Gap | Risk | Where tracked |
|---|---|---|
| `/metrics` has no auth of its own | Fixed for the public port (TODO-001): it is served only on the internal listener (`NAVA_ADMIN_ADDR`, `:9090`) and `:8080` answers 404. Anything that reaches `:9090` can read it, so that port must stay off the ingress | T02 deployment (network policy) |
| Panic value logged as-is | A panic message built from input can log client data | Not yet tracked |
| OTLP transport security comes from env | Plain `http://` or no auth if misconfigured | T02 deployment |
| `WithGroup` nests correlation ids | `x.trace_id` breaks log joins (section 6) | Comment in `logging.go` only |
| Deny-list for application spans | A future app span could add a sensitive key; the golden test covers the server span only | Review each new instrumentation |
| Plain-text 500 | Not RFC 9457 | TODO-002 |
| No `tenant_id` or rate limits yet | Arrive with T17 and Epic 07 | Plan |

### 2.8 Where error text may go

Error text is free-form, and it can contain client input (a parse error that
quotes the bad value, a driver error that quotes a row). Each store has one
rule:

| Store | May carry error text? | What enforces it |
|---|---|---|
| Spans | No, for the listed keys | `filter.go` drops `exception.message`, the status description and `db.statement` / `db.query.text`. An error shows as status `Error` and `error.type`; `exception.type` / `exception.stacktrace` are allowed if recorded (nothing calls `RecordError` today). It covers only the keys it lists, so new span keys are reviewed (section 2.7, "Deny-list for application spans") |
| Metrics | No, for the HTTP metrics | The View allow-list matches only `http.server.*` and keeps bounded attributes (section 2.3); `error.type` is the only error signal there. Runtime metrics have fixed attributes. Any new instrument needs its own View or a review |
| Logs, stdout JSON | Yes | No deny-list, now or later. Nothing inspects keys or values |
| Logs, OTLP via the Collector | Yes | Once T01e (TICK-101) lands: a deny-list on attribute **keys** only, values are not inspected. T01e is not in place yet |

Error text can sit in any log attribute value. Today that is `error`
(`slog.Any("error", err)`) and, from panic recovery, `panic` and `stack`.
It passes through both log paths. This is intended.

Each log line is joined to its trace by `trace_id`, `span_id` and
`request_id`, so the message stays reachable from the red trace without
being copied onto it.

Therefore the log backend's retention period and access control are the PII
control for error text. No deny-list cleans logs. Both must be
decided when a log backend is chosen. That is not part of T02, which only
uses the Collector's debug exporter. It is tracked as TODO-003 in the
[TODO tracker](../delivery/todo.md), triggered by a log backend being added
to the Collector pipeline.

`db.statement` and `db.query.text` stay stripped. When DB instrumentation
lands (Epic 05, records path), that task decides whether to allow the
bounded fields `db.operation.name`, `db.collection.name` and
`db.query.summary`.

Debugging an error: alert, then the red trace, then its status and
`error.type`, then its `trace_id`, then the log line with `error`, or
`panic` and `stack`. The span carries no stack today.

## 3. Prometheus

### 3.1 Pull or push

| | Pull (Prometheus scrapes) | Push (app sends OTLP) |
|---|---|---|
| Strengths | Liveness for free (`up == 0`); monitoring controls load; `curl /metrics` debugging; no app config; several scrapers possible | Short-lived jobs; works through NAT and firewalls (outbound only); one pipeline for traces, metrics, logs |
| Weaknesses | Needs reachability and discovery; misses jobs that end between scrapes | Silence is ambiguous; app needs endpoint, credentials, retry; a buggy app can flood the collector |

For nava-core: pull for the long-running service (current design), push
for short-lived jobs and for traces, and push from on-premise sites (D1)
if they are ever monitored centrally.

### 3.2 Several instances

Prometheus never scrapes through a load balancer: each scrape would hit a
random instance and counters would jump. It discovers every instance and
scrapes each directly; series are told apart by `instance` (and `pod`), and
queries aggregate, e.g. `sum(rate(http_server_request_duration_seconds_count[5m]))`.

- Kubernetes: Prometheus Operator `ServiceMonitor` selects the Service; the
  scrape list follows the ready pod IPs. A dead pod shows `up == 0`, then
  drops out.
- Docker Compose: use `dns_sd_configs` on the service name (Docker DNS
  returns every replica's IP). `static_configs` with the service name is the
  common mistake: it hits one random replica per scrape.
- Restarts reset counters to 0; `rate()` and `increase()` handle it.
- Scale-out: two identical Prometheus servers for HA; `remote_write` to
  Mimir or Thanos for long-term or global views.

### 3.3 Setups

- Local: `prom/prometheus` in Docker with target
  `host.docker.internal:9090` and `--enable-feature=exemplar-storage`.
- T02 compose stack: Prometheus + Grafana, plus Tempo or Jaeger and the
  Collector, so an exemplar click opens its trace.
- Production: kube-prometheus-stack with a `ServiceMonitor`.
- Whichever: scrape the internal listener (`NAVA_ADMIN_ADDR`, default
  `:9090`). `:8080/metrics` answers 404.

## 4. Authenticating push (OTLP)

| Method | How | Notes |
|---|---|---|
| TLS | `https://` endpoint, `OTEL_EXPORTER_OTLP_CERTIFICATE` | Verifies the receiver and encrypts; does not identify the sender |
| Header token | `OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer …`; Collector `bearertokenauth`, `basicauth` or `oidc` (Keycloak) | Most common; rotate; always with TLS |
| mTLS | `OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE` / `_CLIENT_KEY` | Strongest; identity bound to the cert; cert-manager or a service mesh |
| Network policy | Only nava namespaces reach 4317/4318 | Defence in depth, not authentication |

Recommended layout: app → Collector inside the cluster over mTLS (or mesh
mTLS) plus a NetworkPolicy; Collector → backend with one token or OIDC
credential held only by the Collector. Authentication proves who connected,
not what they claim: have the Collector overwrite identity attributes
(`service.name`, any tenant attribute) from the authenticated identity, in
line with ADR-001's rule that the tenant comes only from a verified source.

## 5. Tenant-specific numbers

Tenant ids are not metric labels: one series per label combination, so
1,000 tenants × 30 routes × 5 status classes × 15 buckets ≈ 2.25 million
series from one histogram, and `/metrics` is unauthenticated (it is kept
off the public port, not behind a login).

| Need | Source |
|---|---|
| Debug one tenant | Logs filtered by `tenant_id` (joins in T17) |
| Follow one tenant's requests | Traces with `tenant_id` on spans; counts are approximate under sampling |
| Usage, quotas, billing | Postgres rows, outbox events, rate-limiter counters: exact and auditable |
| SLOs by customer group | A bounded label such as `tenant_tier` (free/pro/enterprise) |
| A few key customers | An allow-list of named tenants, everyone else `tenant="other"` |

Check before T30a/T33: the plan's `outbox_relay_rows_scanned` "histogram
per tenant per poll" must not mean a `tenant_id` label.

## 6. Logging: the `WithGroup` caveat and options

`ContextHandler` delegates `WithGroup` to the JSON handler, so ids added in
`Handle` land inside the open group (`db.trace_id`) and break joins on
`trace_id`. Today the rule is documentation only: do not call `WithGroup` on
a request-scope logger; use `slog.Group` attributes.

Options, in order of preference:

1. T02: adopt the `otelslog` bridge for exported logs. Trace and span ids
   are record fields there, not attributes, so groups cannot move them.
2. T02: for stdout JSON, evaluate `github.com/veqryn/slog-context`, which
   places context attributes at the root regardless of groups (verify with a
   `WithGroup` test and `testing/slogtest` before adopting).
3. Own fix if neither fits: `ContextHandler` keeps groups itself and replays
   them in `Handle` around the record's attrs, writing correlation ids at the
   root (the pattern from the Go slog handler guide), tested with
   `testing/slogtest`.
4. Stopgap: a `forbidigo` lint rule banning `.WithGroup(`.

Also at T02: the Collector's `attributes`/`redaction` processors with the
same deny-list, as a second layer behind `filter.go`.

## 7. Adding an API gateway

Today a gateway would be treated like any internet client: its
`traceparent` is linked, not continued, so the gateway and nava-core produce
two traces with independent sampling (one half is often missing).

To get end-to-end traces, move the trust boundary to the gateway:

1. The gateway does what nava-core's edge does now: new root for internet
   traffic, client `traceparent` and baggage dropped, head sampling decided.
2. nava-core continues the context only for requests that come from the
   gateway, decided by authenticated peer, never by a header:
   `WithPublicEndpointFn(func(r *http.Request) bool { return !fromGateway(r) })`
   with `fromGateway` checking the mTLS peer certificate, or a separate
   listener. A NetworkPolicy must stop anything reaching nava-core around
   the gateway.
3. `ParentBased` sampling then follows the gateway's decision, so traces are
   whole or absent. Baggage stays dropped unless the gateway sanitises it;
   the tenant still comes only from the verified token.

| Sampling | How | Trade-off |
|---|---|---|
| Head at the gateway | Gateway samples; nava-core `parentbased_traceidratio` | Simple, consistent; decided before errors are known |
| Same ratio everywhere | `TraceIDRatioBased(x)` in every service | Decisions agree per trace id even without a parent flag |
| Tail in the Collector (deferred to T02) | Keep whole traces with errors, high latency, or a random % | Best traces; needs Collector memory and trace-aware routing |

This changes ADR-001's "no internal HTTP caller" assumption, so it needs a
decision-log entry or ADR amendment, the code change above, and adversarial
tests: a non-gateway request still gets a new root; a gateway request
continues; a client `traceparent` sent through the gateway never becomes
nava-core's parent.

## 8. Why some of this is our own code, not a library

Most of the telemetry stack *is* libraries: the OTel API and SDK, `otelhttp`,
the OTel runtime instrumentation, the OTel Prometheus exporter, the
Prometheus client and process collector, `httpsnoop` and the `logr` slog
bridge. What we wrote is policy and thin glue between them. The rule we
follow: standard library first, libraries for instrumentation and protocols,
our own code only where a library cannot know our policy or would break one
of the edge rules, and every piece we own has a test.

| Our code | Library alternative | Why ours (for now) | Revisit when |
|---|---|---|---|
| Span deny-list (`filteringExporter`, `filter.go`) | Collector `attributes` / `redaction` / `transform` processors; SDK span limits only cap counts and lengths | Libraries follow OTel semantic conventions, which *require* attributes such as `client.address` and `user_agent.original`; no library knows that our ADR forbids client values leaving the process. The Collector is the usual place, but there is no Collector until T02, and filtering in-process means the data never leaves at all (defence in depth) | T02: add the same list to the Collector as a second layer; keep this one |
| Metric View and `traceOnlyExemplars` | The SDK's own View and reservoir API (we configure them) | This *is* the library's extension point. The wrapper exists because the Prometheus exporter has no option to drop exemplar filtered-attributes, and without it client values reach `/metrics` | An exporter option to omit filtered attributes appears |
| `methodLabel` | otelhttp normalises methods internally | otelhttp's normaliser is in an internal package, so it cannot be imported; ours mirrors it so the access log and the metric agree | otelhttp exports it |
| `ContextHandler` (slog ids) | `otelslog` bridge (official); `github.com/veqryn/slog-context` (community) | About 20 lines, written before any log pipeline existed. `otelslog` exports over OTLP, which waits for the T02 Collector. Its one gap is the `WithGroup` caveat (section 6) | T02: adopt `otelslog` for export; try `slog-context` for stdout |
| `requestid` | Router middleware such as chi's `RequestID` | Common request-id middleware reuses an inbound `X-Request-Id` header. Our edge rule is that the id is always server-generated, never client-supplied | Not expected; the rule is deliberate |
| Panic recovery (`httpobs.serve`) | chi `middleware.Recoverer`, gorilla `handlers.RecoveryHandler` | We use the standard `ServeMux`, not a router framework. Recovery must sit inside `access` and otelhttp so a 500 still gets its access line, metrics and span status; it also clears the handler's headers and aborts the connection once the response is committed (including `101`). Typical recoverers write a 500 regardless | A router framework is adopted for T20; compare its recoverer against these tests |
| `config` (env reading) | `envconfig`, `viper`, `caarlos0/env` | About 150 lines. It reports every invalid variable at once and never echoes a value; general-purpose loaders typically quote the bad value in the error, which would leak a secret pasted into the wrong variable | Config grows (files, nesting, hot reload) |
| `serve()` / graceful shutdown | Standard library `http.Server.Shutdown` (used) | The standard pattern, plus panic-to-error in the serve goroutine; nothing to import | — |
| `RouteErrors` | `otel.SetErrorHandler`, `logr.FromSlogHandler` (used) | Two library calls; ours only wires them to the process logger | — |

Costs we accept by owning these: we maintain them, they need tests (each
has unit tests and the adversarial suite), and a library upgrade can change
the ground under them. The golden span-key test and the exemplar test exist
to catch exactly that.
