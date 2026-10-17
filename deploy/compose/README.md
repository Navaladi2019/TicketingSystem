# deploy/compose/

Support files for the local stack defined in `/compose.yaml`.

```
docker compose up -d --build        # or: task stack:up
docker compose -f compose.yaml -f deploy/compose/compose.internal.yaml up -d   # no-egress overlay
```

| Service | Image (pinned) | Notes |
| --- | --- | --- |
| postgres | `postgres:16.10` | Superuser password is a file, used only by `postgres` and `db-init` |
| db-init | `postgres:16.10` | One-shot, idempotent; creates roles `keycloak`, `temporal`, `pgbouncer_auth` and databases `keycloak`, `temporal`, `temporal_visibility`, `nava`. T03 grows it into `db-roles` |
| pgbouncer | `edoburu/pgbouncer:v1.24.1-p1` | Transaction mode, `max_prepared_statements=100`, fronts `nava` |
| keycloak | `quay.io/keycloak/keycloak:26.8.1` | `start-dev`, own database |
| temporal | `temporalio/auto-setup:1.28.4` | `default` namespace, SQL visibility on Postgres |
| otel-collector | built from `contrib:0.139.0` | Internal network only; tail sampling; debug exporter |
| nava-core | built from the repo | Exports OTLP traces to the Collector |

Host ports (bound to 127.0.0.1): nava-core `18080`, Keycloak `18081`.

**Extending the init script.** Add `ensure_role` / `ensure_database` calls to
`db-init/init.sh`; both are safe to re-run.

**Tests.** `sh deploy/compose/test/stack-test.sh <healthy|pgbouncer|temporal|trace|redact|idempotent|internal>`
(`task stack:test`). Start the stack with `OTEL_TAIL_SAMPLE_PERCENT=100` for the
`trace` test: an ordinary 404 is otherwise sampled out. CI runs them in
`.github/workflows/compose-stack.yml`.

Later tasks mount here: T08 `db/pg_hba.conf`, T28 the dev CA.
