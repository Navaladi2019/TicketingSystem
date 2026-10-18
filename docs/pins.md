# Pinned versions

T05 owns this file and fills in the rest of the D-P3 rows (images, Keycloak,
Temporal, Go). Each row: what is pinned, the exact version, why, and the
ruling or task that set it.

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `github.com/pressly/goose/v3` | v3.29.0 | Migration runner. Session advisory lock (`lock.NewPostgresSessionLocker`) set explicitly. `golang-migrate` (dirty flag) and `jackc/tern` (`text/template` over SQL) are forbidden; no hand-rolled runner. Its go.mod raises, by MVS: `google.golang.org/grpc` v1.83.2 to v1.84.0 (own row below), `golang.org/x/net` v0.59.0 to v0.61.0, `golang.org/x/sys` v0.48.0 to v0.49.0, `golang.org/x/sync` v0.23.0 to v0.24.0, `golang.org/x/text` v0.42.0 to v0.43.0, `golang.org/x/tools` v0.49.0 to v0.51.0, `github.com/jackc/puddle/v2` v2.2.2 to v2.2.3 and `google.golang.org/genproto/googleapis/{api,rpc}` to `v0.0.0-20261005182115-fad411399dd8`. | TICK-94 item 8, T03 |
| `go.yaml.in/yaml/v3` | v3.0.5 | The one strict YAML loader (`internal/platform/strictyaml`) for `db/grants.yaml` and `api/errors.yaml`. Not the archived `gopkg.in/yaml.v3`. | TICK-94 item 9, T03 |
| `github.com/jackc/pgx/v5` | v5.11.0 | PostgreSQL driver for the jobs and the core. The version goose v3.29.0 is built against. | T03, T04 |
| `google.golang.org/grpc` | v1.84.0 | gRPC for the Temporal SDK client. Not chosen directly: goose v3.29.0 requires it, so MVS raises it. | T04 (v1.83.2), raised by goose in T03 |
