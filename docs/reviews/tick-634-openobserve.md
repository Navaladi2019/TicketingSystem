## TICK-634 review: replace Jaeger with OpenObserve (PR #529)

Reviewer: Quality, Testing & CI Advisor. Commit reviewed: `4a865c69`.

**Verdict: REQUEST CHANGES.** One bug breaks the CI `stack` job, and one bug makes the README log queries fail. With the password fixed, every live acceptance check passed.

## CI (GitHub Actions run 42556175303, label `ci:full`)

| Job | Result |
| --- | --- |
| lint, unit, integration, coverage, temporal-server, commit-check-budget, changes, story-branches | pass |
| stack | **fail** in all three phases (`stack`, `stack-db`, `stack-internal`): `FAIL: not healthy before the deadline: openobserve`, container `Restarting (1)` |

## Findings (most severe first)

### 1. Bug: OpenObserve crashes at startup, so the CI `stack` job fails

`compose.yaml` and `.env.example` set the default `ZO_ROOT_USER_PASSWORD` to `dev-only-openobserve`. OpenObserve v1.1.2 refuses it and exits 1:

```
ZO_ROOT_USER_PASSWORD is too weak: Password must be 8-128 characters and contain at least one lowercase letter, one uppercase letter, one digit, and one special character.
Error: backend job init failed: channel closed
```

Reproduced with `docker run` of the pinned image and the PR's env. With the password `Dev-only-OpenObserve-1`, the container runs and `curl /healthz` inside it returns 200. That also confirms the `-debug` image ships curl, so the healthcheck works.

**Fix:** change the dev default in all three places, and keep them in step:
- `compose.yaml` `ZO_ROOT_USER_PASSWORD` default
- `compose.yaml` `OPENOBSERVE_BASIC_AUTH` default: `ZGV2QG5hdmEudGVzdDpEZXYtb25seS1PcGVuT2JzZXJ2ZS0x` = base64(`dev@nava.test:Dev-only-OpenObserve-1`)
- `.env.example`, the three `OPENOBSERVE_*` lines

The CI `stack` job going green after this change is the test that proves the fix.

### 2. Bug: the README log queries fail

In `deploy/compose/README.md`, "View and filter logs", two of the three example queries return `unknown field`. OpenObserve flattens attribute names, and the log record has no `level` field.

| README says | Live result | Use instead (verified) |
| --- | --- | --- |
| `level = 'ERROR'` | `unknown field 'level'` | `severity = 'ERROR'` |
| `"nava.request_id" = '<id>'` | `unknown field '"nava'` | `request_id = '<id>'` (logs). In the Traces view the span field is `nava_request_id` |
| `trace_id = '<id>'` | works | unchanged |
| `match_all('timeout')` | works on logs | unchanged; note that it does not work on the traces stream |

Opening the trace from a log needs no copy and paste: expand the log line and click **View Trace** (screenshots 08 and 09). Please replace the manual-copy instruction with that.

### 3. Spec drift: the comments say the CI stack has no OpenObserve, but it does

`make/stack.mk:63` exports `COMPOSE_PROFILES ?= operator`, so the CI stack starts the `operator` profile. Jaeger ran there before, and OpenObserve runs there now, which is why finding 1 broke CI. The "the CI stack never starts it" and "Without the `operator` profile (the CI stack) there is no OpenObserve" lines in `compose.yaml`, `deploy/compose/otel-collector/config.yaml` and `deploy/compose/README.md` are wrong. Say instead that OpenObserve is optional and the Collector tolerates it being absent.

I verified that tolerance live: with `openobserve` stopped, 3 requests to `/api/v1/me` returned 200, and `otel-collector` and `nava-core` stayed healthy. The Collector logged 25 export errors and dropped those batches.

### 4. Missing test: no stack case checks that telemetry reaches OpenObserve

Nothing under `deploy/compose/test/` queries the backend. The stack would pass even if the exporter, its auth header or the `logs` pipeline were wrong. It would also pass if a denied attribute reached OpenObserve.

**Suggested case (area `observability`):** call `/api/v1/me?probe=<random>` with `X-Forwarded-For`. Then poll `POST /api/default/_search?type=logs` for the response's `request_id`, and `type=traces` for `nava_request_id`, and assert that:
- each query returns at least 1 hit;
- the log hit has a `trace_id`;
- the traces schema (`GET /api/default/streams/default/schema?type=traces`) has none of `url_query`, `client_address`, `exception_message`, and the probe value appears nowhere.

This is not a merge blocker, but it is the regression net this change lacks. I can write it as a follow-up if the Technical Lead wants.

### 5. For the Security & Multi-Tenancy Reviewer (not a finding of this review)

Spans in OpenObserve carry `nava_tenant_id`; logs carry no tenant id. This comes from existing span attributes, not from this PR. It is noted here for the isolation review the issue asks for.

## Live acceptance, operator profile

Setup: isolated project `nava-tick634` (API on 28080, Keycloak on 28081, Prometheus on 29091; OpenObserve on `127.0.0.1:5080` as specified). `OTEL_TAIL_SAMPLE_PERCENT=100`. `OPENOBSERVE_PASSWORD` and `OPENOBSERVE_BASIC_AUTH` were overridden to the strong password from finding 1. Two tenants were seeded with `docs/acceptance/s3-tenant-signin/accept.py` (`result: PASS`).

| Criterion | Result | Evidence |
| --- | --- | --- |
| Every service healthy, no `jaeger` container, UI login | ✅ with the finding 1 override; ❌ with the PR defaults | `docker compose ps`: 15 services healthy and 3 one-shot jobs exited 0 |
| `nava-core` trace after `GET /api/v1/me`, with `http.route` and `nava.request_id` | ✅ | Span `GET /api/v1/me`: `http_route=/api/v1/me`, `nava_request_id` set (07, 09) |
| nava-core logs in the Logs view | ✅ | 01 |
| Filter by level | ✅ `severity='ERROR'` | 02 |
| Filter by `trace_id` (only that request's lines) | ✅ 2 lines for the forced 500 | 03 |
| Filter by request id | ✅ `request_id=...` | 04 |
| Free text | ✅ `match_all('authentication')` | 05 |
| Time range | ✅ Past 5 minutes: 1 event; Past 1 hour: all | 06 vs 01 |
| Open the trace from a log line | ✅ View Trace in the expanded log line | 08, 09 |
| No `url.query`, `client.address`, `exception.message` | ✅ | Requests with `?secretq=leakme…`, `X-Forwarded-For: 203.0.113.77`, and a forced 500 (pgbouncer stopped). Exception events carry only `exception.type`. Across all 256 spans and 102 log records: no `url_query`, `client_address` or `exception_message` field, and no probe value |
| Collector healthy with OpenObserve absent | ✅ | Finding 3 |

Screenshots are in `docs/reviews/tick-634-openobserve/`.
