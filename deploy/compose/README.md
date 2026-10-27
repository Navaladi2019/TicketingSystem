# deploy/compose/

Support files for the local stack defined in `/compose.yaml`.

```
make dev-ca                         # once: dev CA and certificates in deploy/compose/certs/
docker compose up -d --build        # or: task stack:up
docker compose -f compose.yaml -f deploy/compose/compose.internal.yaml up -d   # no-egress overlay
```

| Service | Image (pinned) | Notes |
| --- | --- | --- |
| postgres | `postgres:16.10` | Superuser password is a file, used only by `postgres` and `db-roles` |
| db-roles | built from the repo (`nava-db roles`) | One-shot, idempotent, bootstrap superuser: converges the roles, memberships, databases and PgBouncer lookup in `db/grants.yaml` |
| migrate | built from the repo (`nava-db migrate`) | One-shot, as `nava_migrator`, direct to Postgres: goose migrations, then the grant manifest |
| pgbouncer | `edoburu/pgbouncer:v1.24.1-p1` | Transaction mode, `max_prepared_statements=100`, fronts `nava`; logs in only roles with `pgbouncer_login` |
| keycloak | `nava/keycloak` (built on `quay.io/keycloak/keycloak:26.8.1`) | `start --optimized`, own database |
| temporal-schema | `nava/temporal-server:dev` | One-shot, idempotent: Temporal schema in `temporal` and `temporal_visibility` |
| temporal-server | `nava/temporal-server:dev` (T27a) | Frontend and internode mTLS, namespace `nava`, `NavaTenantId`; holds the `temporal-system` certificate; `temporal-backplane` network only |
| temporal | `nava/temporal-server:dev` | Gateway: forwards TCP 7233 only, no certificate. The only route from the platform network to the server |
| temporal-ui | `temporalio/ui:2.40.1` | Profile `operator` only, on the `operator` network with the `nava-operator` certificate; `http://127.0.0.1:18233` |
| otel-collector | built from `contrib:0.139.0` | Internal network only; tail sampling; debug exporter |
| nava-core | built from the repo | Exports OTLP traces to the Collector |

**Temporal mTLS (T28).** `make dev-ca` writes, to `deploy/compose/certs/`
(git-ignored): the installation CA (`ca.pem`, path length 0, signs the frontend
certificate and the client certificates), a separate internode CA and
certificate, and `nava-relay`, `nava-core`, `nava-worker`, `nava-operator` and
`temporal-system`. Each is mounted only into its own service. Internode and
membership ports (7234, 7235, 7239, 6933-6939) are on `temporal-backplane`, which
no platform process joins; the gateway forwards 7233 only. `FORCE=1 make dev-ca`
reissues. Cron on a started workflow cannot be seen by the Authorizer (T27b).

Host ports (bound to 127.0.0.1): nava-core `18080`, Keycloak `18081` (`NAVA_CORE_PORT`, `KEYCLOAK_PORT`). A second stack on the same host: `make ci-isolated ID=...` (`docs/quality/ci-pipeline.md`, "Parallel runs").

**Adding a role or database.** List it in `db/grants.yaml` (see `db/README.md`);
db-roles creates it on the next `up`. A LOGIN role also needs a
`DBROLES_PASSWORD_<ROLE>` variable on the db-roles service.

**S7 demo: `make demo-outbox`** (`task demo:outbox`). With the stack up
(`make stack-up`), it writes a start row and two signal rows for a record of the
seeded tenant `tenant-a` (`00000000-0000-4000-8000-0000000000a1`) through
`outbox.Enqueue` in `InTenantTx`, as `nava_app`, so `outbox_payload_check` and
row-level security apply as in the product (`cmd/demo-outbox`, run from a
throwaway container on the `internal` network by `scripts/dev/demo-outbox.sh`;
the database port is never published). It needs only what `.env.example` and
`make dev-ca` provide. The relay starts one workflow `t:<tenant-a id>:nava.smoke.relay:<start row id>`
and signals it twice with `demo.note`, in commit order. The command prints the
workflow id. To see it: `make operator-tunnel`, open `http://127.0.0.1:18233`,
namespace `nava`, and open that workflow; its history holds the two signals. No
worker runs this workflow type in the stack, so it stays Running.
`DEDUPE_KEY=<key>` names the rows (default `demo-outbox`) and picks the record: run
it twice with the same key and the second run writes nothing and prints the same
ids, still one workflow; a new key makes a new workflow. `make stack-test-demo_outbox`
runs exactly that and checks the history (CI: the `stack` job).

**Git Bash on Windows.** MSYS rewrites a leading-slash container path in a
`docker` argument (`--entrypoint /demo-outbox` becomes `C:/Program Files/Git/demo-outbox`).
`make demo-outbox`, `make operator-tunnel`, `make temporal-server-run` and
`deploy/compose/test/stack-test.sh` set
`MSYS_NO_PATHCONV=1` themselves, so no `export` is needed. A `docker run`, `create`
or `exec` you add with a container path must do the same.

**Tests.** `sh deploy/compose/test/stack-test.sh <healthy|pgbouncer|temporal|trace|redact|idempotent|dbroles|internal|temporal_handshake|temporal_isolation|temporal_confinement|temporal_custody|temporal_health>`
(`task stack:test`). Start the stack with `OTEL_TAIL_SAMPLE_PERCENT=100` for the
`trace` test: an ordinary 404 is otherwise sampled out. CI runs them in
the `stack` job in `.github/workflows/ci.yml`.

The SDK test (every identity dials, a worker runs a workflow) needs the stack
with the test overlay, which publishes the gateway on `127.0.0.1:17233`:
`docker compose -f compose.yaml -f deploy/compose/compose.test.yaml up -d`, then
`go test -tags temporalstack -count=1 ./internal/temporalstack`.

**Keycloak realm (T14).** `keycloak/import/` holds the realm `nava` (Organizations,
ADR-001 A.1 mappers and broker flow, operator role, seeded tenants `tenant-a` and
`tenant-b`, group subtree `/t/<tenant_id>/ou/...`). Keycloak imports it on first
start (`--import-realm`; existing realms are never overwritten, so a change needs a
fresh `keycloak` database). The two test-only IdP realms the tenant IdPs broker to,
`idp-tenant-a` and `idp-tenant-b`, are in `keycloak/test-realms/` and are not
imported at first boot (it keeps `docker compose up` fast, TICK-166): the identity
test creates them through the admin API when it starts, deleting and creating
them fresh each run. On a plain `docker compose up`, `make stack-idp-realms`
(set `KEYCLOAK_PORT` if it is not 18081) creates them with the same code, for a
manual step that needs the tenant IdPs. The IdP URLs assume the published port
18081 (`KEYCLOAK_PORT` default). All secrets are dev-only. Test:
`make stack-test-identity` (`go test -tags identitystack ./internal/identitystack`).
`TestRealmFile` in the same package checks the file without a container.

Realm decisions to know when you build on it:

- **Client scopes.** The realm file replaces Keycloak's built-in client scopes with
  `basic`, `roles`, `profile`, `email`, `organization`, `nava-idp` and
  `nava-operator-audience` (Keycloak adds back only `offline_access`). Dropped
  built-ins: `acr`, `web-origins`, `service_account`, `address`, `phone`,
  `microprofile-jwt`, `role_list`, `saml_organization`. `defaultDefaultClientScopes`
  is `basic`, `roles`, `profile`, `email`, so a client created later through the
  admin API (T15 onboarding) still gets `sub`.
- **Organization ids are fixed.** `tenant-a` and `tenant-b` carry the same UUID as
  their group tree, `00000000-0000-4000-8000-0000000000a1` and `...b1`; the import
  honours the `id` (checked on 26.8.1). T15 seeds can use that id for
  `tenant_identity_binding(issuer, org_id)` without a lookup.
- **Operator audience.** `nava-operator-cli` uses the direct-grant flow
  `nava-operator-direct-grant`, which denies users without the `nava-operator` role.
  The audience mapper alone adds `nava-operator-api` to any token, so the operator
  resolver (T15) must also require the realm role, not only the audience.
- **Platform service account** holds `manage-realm`, so its secret is the key to
  every tenant: outside compose supply it from the environment or a secret store,
  never the `dev-only-platform-secret` literal.

Later tasks mount here: T08 `db/pg_hba.conf`.
