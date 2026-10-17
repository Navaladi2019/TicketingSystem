# TODO tracker

Known gaps that are deliberately deferred. Each row has a matching
`TODO(TODO-NNN)` comment in the code, so `grep -rn "TODO(TODO-" .` finds every
site. When an item is fixed, remove its code comment and set its status to
`Fixed (YYYY-MM-DD, PR or commit)`. Statuses: `Open`, `Fixed`.

| ID | Status | Added | Area | Code | Problem | Fix |
| --- | --- | --- | --- | --- | --- | --- |
| TODO-001 | Open | 2026-10-17 | Security / observability | [`internal/server/server.go`](../../internal/server/server.go) (`NewHandler`) | `GET /metrics` is served on the public listener (`:8080`) with no authentication. Anyone who can reach the port can read route names, per-route traffic and latency, error rates, Go and service versions, process memory and CPU, and exemplar trace ids. | Serve `/metrics` (and later health checks) on a separate internal listener: add `NAVA_METRICS_ADDR` (default `:9090`), start a second `http.Server` in `run`, drain both on shutdown, remove `/metrics` from the public handler, and add a test that `:8080/metrics` returns 404. Rejected alternatives: blocking the path at the ingress (fragile, still reachable inside the network) and a bearer token or mTLS (secrets to manage for little gain). |
| TODO-002 | Open | 2026-10-17 | API errors | [`internal/platform/httpobs/recover.go`](../../internal/platform/httpobs/recover.go) (`serve`) | A recovered handler panic answers with a `text/plain` 500. The delivery plan fixes RFC 9457 `application/problem+json` with registered codes for every error response (T20), so this body would fail T20's contract test and give clients an unparseable error. | When T20 adds the shared problem writer, use it in `serve` for the 500, with a registered code such as `internal_error`, and keep the header reset and `X-Request-Id`. |
