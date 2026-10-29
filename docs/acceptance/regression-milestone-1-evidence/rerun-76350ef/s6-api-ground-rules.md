# S6 regression re-run (TICK-48, guide `docs/acceptance/guide/s6.html`), main @ 76350ef (PR #266 in), 2026-10-28, stack nava-regr2

Kit = `guide/s6.html` S6.1-S6.10 (replayed with Python `urllib`, and again by Newman, see `cross-story.md`), plus the per-tenant 429 case that #266 added. All against `GET /api/v1/me` on nava-core `:28080` with real Keycloak tokens. Rate-limit config: compose passes no `NAVA_RATE_LIMIT_*`, so the defaults apply (50 rps, burst 100, per tenant, per replica).

| # | Guide criterion | Result | Evidence |
|---|---|---|---|
| S6.1 | `/api/v1/openapi.yaml` public | PASS | `200 application/yaml`, `openapi: 3.1.0` |
| S6.2 | `/api/v1/docs` public, lists both operations | PASS | `200 text/html`, lists `/api/v1/me` and `/api/v1/operator/tenants` |
| S6.4 | unknown /api/v1 route, no token -> 401 | PASS | `401 application/problem+json invalid_token` |
| S6.5 | unknown /api/v1 route, token -> 404 | PASS | `404 not_found` |
| S6.6 | path outside /api/v1 -> 404 | PASS | `404 not_found` |
| S6.7 | wrong method, token -> 405 + Allow | PASS | `405 method_not_allowed`, `Allow: GET, HEAD` |
| S6.8 | wrong method, no token -> 401 | PASS | `401 invalid_token` |
| S6.9 | X-Request-Id; log line carries it; traceparent linked not continued | PASS | `200`, `X-Request-Id 320cdd7e…`; 1 nava-core log line with `request_id 320cdd7e…`, `inbound_trace_id 4bf92f35…` (the sent one), own `trace_id b6154df2…` |
| S6.10 | markup in path not echoed | PASS | `404` fixed text, body has no `script` |
| all errors | problem+json, `type=urn:nava:problem:<code>`, `instance=urn:nava:request:<X-Request-Id>` | PASS | checked on every 4xx above and on the 429 |
| 429 | tenant A throttled with problem+json + Retry-After while tenant B at the same moment gets 2xx | PASS | below |
| 429 | RateLimit-* headers | NOT PRESENT (matches code and contract; see note) | no `RateLimit*` / `X-RateLimit*` header on any 200 or 429 |

## 429, per tenant, at the same moment
400 concurrent `/me` calls (48 threads) for the flooding tenant, with 60 calls for the other tenant interleaved in the same queue:
```
tenant A (alice) flood: 400 requests in 0.46s -> {200: 121, 429: 279}; Retry-After values ['1']
tenant B (bob) interleaved, same moment: 60 requests -> {200: 60}, tenant ...b1
  tenant B requests sent inside tenant A's 429 window (0.198s..0.454s): {200: 43}
tenant B (bob) flood: 400 requests in 0.39s -> {200: 118, 429: 282}; Retry-After values ['1']
tenant A (alice) interleaved, same moment: 60 requests -> {200: 60}, tenant ...a1
  tenant A requests sent inside tenant B's 429 window (0.142s..0.391s): {200: 44}
```
One 429, verbatim:
```
429 Content-Type: application/problem+json  X-Request-Id: 42d36b06…  Retry-After: 1
{"type":"urn:nava:problem:rate_limited","title":"Rate limited","status":429,"code":"rate_limited",
 "detail":"Too many requests. Retry after the interval in Retry-After.","instance":"urn:nava:request:42d36b06…"}
```
- The bucket is the tenant's, not the user's: Alice and tenant A's admin interleaved (200 each) -> alice `{200: 54, 429: 146}`, admin-a `{200: 62, 429: 138}`, tenant A total `{200: 116, 429: 284}` (≈ burst 100 + refill).
- Recovery: 3 s later A gets `200`.
- Log: 282 lines `"msg":"request rate limited","tenant_id":"...a1","retry_after_s":1,...,"request_id":...`; only the flooding tenant appears. Metrics: `http_server_*{http_response_status_code="429",http_route="/api/v1/me"} 282`, no tenant label.
- Operator routes are not limited (by design in `internal/server/api.go`): 200 operator calls to nava-onboard, all answered (405 for GET), no 429.

## Notes (severity-labelled)
- **Spec drift (kit):** `guide/s6.html` still says "Per-tenant rate limits (429) ... are not on main" and "Idempotent replay is tested in S4". #266 put the limiter on main; the guide and its Postman folder have no 429 request. Owner of the guide (TICK-287) should add a 429 step.
- **Spec question (not a failure):** the task brief asked for "RateLimit headers as documented". Nothing documents them: `ratelimit.go` sets only `Retry-After`, and `openapi.yaml` declares 429 as a plain Problem response with no headers. If `RateLimit-Policy`/`RateLimit` (IETF draft) are wanted, that is a new requirement for the Solution Architect, not a bug.
- **Bug (low):** `HEAD /api/v1/me` with a valid token -> `405 method_not_allowed`, while every 405 advertises `Allow: GET, HEAD`. Repro: `curl -I -H "Authorization: Bearer <alice>" http://127.0.0.1:28080/api/v1/me` -> `HTTP/1.1 405`. Expected 200 headers-only, or `Allow: GET`. (`HEAD /api/v1/openapi.yaml` -> 200.)
- **Preference:** the "request rate limited" log line carries `tenant_id` twice (once from the call, once from the context handler).
