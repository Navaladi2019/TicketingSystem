---
type: audit
plan: PLAN-001
issue: TICK-93
date: 2026-10-16
owner: Technical Lead
---

# PLAN-001 spec audit: cross-cutting choices

Every task from T02 to T33 was checked against the tasks that consume its output, the trust boundaries it touches, and the telemetry contract decided on TICK-89 (OpenTelemetry, edge trace root, no tenant ids in metric attributes). Rule 1 of `review-standard-core.md`, applied to the plan itself.

**Result:** 16 findings across 20 tasks. #1–#9 are fixed in the task descriptions (section "Audit TICK-93"); #2 also opens TICK-94 item 7. #10–#16 are choices or dependencies nobody had stress-tested and open TICK-94 items 1–6 and 8–16. 15 tasks have no finding.

## Findings

| # | Task | Finding | Kind | Action |
|---|---|---|---|---|
| 1 | T30a TICK-74 | Relay histogram "per tenant" readable as a tenant label | metric shape | Fixed on TICK-89 |
| 2 | T30a TICK-74, T33 TICK-80 | With no tenant label, T33 can only assert the 200-row cap through a bucket at 200, and OTel's default buckets have none | consumer conflict | Descriptions fixed (explicit bucket view); TICK-94 #7 |
| 3 | T25 TICK-78 | `groupsync_lag_seconds` **per tenant**: the same defect as #1 | metric shape | Description fixed: histogram with no tenant attribute, laggard tenant in a log line |
| 4 | T27b TICK-84 | Specifies a new counter tagged with the workflow id (unbounded, contains the tenant). T27a already ships `nava_authz_denied` with bounded tags | metric shape, consumer conflict | Description fixed: reuse `nava_authz_denied`, workflow id in log and alert only |
| 5 | T17 TICK-72 | T01 deferred `tenant_id` on log lines to T17, but T17 does not carry it | dropped deferral | Description fixed |
| 6 | T17 TICK-72 | No forged-tenant case for inbound `baggage` (4a drops it at the edge, untested) | trust boundary | Description fixed: `CI-2` case added |
| 7 | T02 TICK-51 | The decision log puts the Collector and tail sampling in T02; T02 has neither | dropped deferral | Description fixed |
| 8 | T29 TICK-70 | The "traces through the outbox" deferral is triggered by T29, which does not mention it | dropped deferral | Description fixed: decide or re-defer before coding |
| 9 | T28 TICK-59 | T04's Temporal health check dials without a certificate; after T28 it would fail | consumer conflict | Description fixed |
| 10 | T04 TICK-55 (+ T01) | `/healthz`, `/readyz` and `/metrics` sit on the listener that 4a made the public trust boundary. Background binaries (T13, T25, T30b, T31) have no metrics endpoint at all | trust boundary | TICK-94 #1 (admin listener); T04, T13, T25, T30b, T31 point to it |
| 11 | T15 TICK-63, T18 TICK-73 | Outbound Keycloak calls have no shared client. `otelhttp` client spans would export user ids (`/users/{id}`) and emails from lookup queries (`?email=`, `?search=`) in `url.full` | trust boundary | TICK-94 #2; descriptions fixed with a no-email-in-spans test |
| 12 | T07 TICK-61 | No task owns the pgx pool or its `otelpgx` tracer; query parameters would carry tenant ids and record data | ownership gap | TICK-94 #4; T07 owns the pool, parameters never recorded |
| 13 | T30b TICK-85, T31 TICK-65, T32b TICK-86 | The Temporal OTel interceptor and metrics handler are unspecified; baggage in Temporal headers would land in unencrypted histories (R1); guard metrics invite tenant labels | trust boundary, metric shape | TICK-94 #5; descriptions fixed, CI-7 case added |
| 14 | T20 TICK-52 | A generated router that is not `net/http.ServeMux` leaves `r.Pattern` empty, so every API request would record no route | consumer conflict | TICK-94 #6; description fixed with a route test |
| 15 | T17 (+ T07, T30b, T31) | No rule on whether tenant or principal ids may be span attributes | open policy | TICK-94 #3 |
| 16 | T03, T07, T15, T18, T20, T24, T05 | Unnamed dependencies: migration runner, YAML, UUID, JWT/JWKS, Keycloak admin client, OpenAPI tooling, token bucket, lint plugin approach, `testcontainers-go` | unstress-tested dependency | TICK-94 #8–16; T05 adds `make check-pins` |

## Tasks with no finding

T06, T08, T09, T10, T11, T12, T14, T16, T19, T21, T22, T23, T26, T27a (checked against PR #3: the deny counter's tags are already bounded), T32a.
