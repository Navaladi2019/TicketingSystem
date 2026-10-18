# deploy/compose/

Support files for the local stack defined in `/compose.yaml`.

```
docker compose up -d --build        # or: task stack:up
docker compose -f compose.yaml -f deploy/compose/compose.internal.yaml up -d   # no-egress overlay
```

| Service | Image (pinned) | Notes |
| --- | --- | --- |
| postgres | `postgres:16.10` | Superuser password is a file, used only by `postgres` and `db-roles` |
| db-roles | built from the repo (`nava-db roles`) | One-shot, idempotent, bootstrap superuser: converges the roles, memberships, databases and PgBouncer lookup in `db/grants.yaml` |
| migrate | built from the repo (`nava-db migrate`) | One-shot, as `nava_migrator`, direct to Postgres: goose migrations, then the grant manifest |
| pgbouncer | `edoburu/pgbouncer:v1.24.1-p1` | Transaction mode, `max_prepared_statements=100`, fronts `nava`; logs in only roles with `pgbouncer_login` |
| keycloak | `quay.io/keycloak/keycloak:26.8.1` | `start-dev`, own database |
| temporal | `temporalio/auto-setup:1.28.4` | `default` namespace, SQL visibility on Postgres |
| otel-collector | built from `contrib:0.139.0` | Internal network only; tail sampling; debug exporter |
| nava-core | built from the repo | Exports OTLP traces to the Collector |

Host ports (bound to 127.0.0.1): nava-core `18080`, Keycloak `18081`.

**Adding a role or database.** List it in `db/grants.yaml` (see `db/README.md`);
db-roles creates it on the next `up`. A LOGIN role also needs a
`DBROLES_PASSWORD_<ROLE>` variable on the db-roles service.

**Tests.** `sh deploy/compose/test/stack-test.sh <healthy|pgbouncer|temporal|trace|redact|idempotent|dbroles|internal>`
(`task stack:test`). Start the stack with `OTEL_TAIL_SAMPLE_PERCENT=100` for the
`trace` test: an ordinary 404 is otherwise sampled out. CI runs them in
`.github/workflows/compose-stack.yml`.

Later tasks mount here: T08 `db/pg_hba.conf`, T28 the dev CA.

**CI workflow (parked).** `compose-stack.workflow.yml` here is the T03 version of
`.github/workflows/compose-stack.yml`; the agent token cannot write workflow
files, so a maintainer moves it into place.
