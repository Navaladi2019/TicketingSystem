# Pinned versions

T05 owns this file and fills in the rest of the D-P3 rows (images, Keycloak,
Temporal, Go). Each row: what is pinned, the exact version, why, and the
ruling or task that set it.

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `github.com/pressly/goose/v3` | v3.29.0 | Migration runner. Session advisory lock (`lock.NewPostgresSessionLocker`) set explicitly. `golang-migrate` (dirty flag) and `jackc/tern` (`text/template` over SQL) are forbidden; no hand-rolled runner. Its go.mod raises `google.golang.org/grpc` to v1.84.0, `golang.org/x/net` to v0.61.0 and `golang.org/x/sys` to v0.49.0 (MVS). | TICK-94 item 8, T03 |
| `go.yaml.in/yaml/v3` | v3.0.5 | The one strict YAML loader (`internal/platform/yamlstrict`) for `db/grants.yaml` and `api/errors.yaml`. Not the archived `gopkg.in/yaml.v3`. | TICK-94 item 9, T03 |
| `github.com/jackc/pgx/v5` | v5.11.0 | PostgreSQL driver for the jobs and the core. The version goose v3.29.0 is built against. | T03, T04 |
