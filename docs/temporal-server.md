# Custom Temporal server (`temporal-server/`)

ADR-001 D22 and Annex A.2, plan task T27a. A separate Go module (R7) that builds
Temporal `go.temporal.io/server` v1.28.4 + `go.temporal.io/api` v1.50.1 with a
deny-by-default Authorizer and a certificate ClaimMapper. Worker command
inspection is T27b. Compose wiring (T28) is in `compose.yaml` and `deploy/compose/README.md`.

```mermaid
flowchart LR
  C[client cert<br/>installation CA] -->|mTLS handshake| F[frontend :7233]
  F --> CM[ClaimMapper<br/>CN to identity]
  CM --> AZ{Authorizer}
  AZ -->|identity, API and namespace allowed| H[handler]
  AZ -->|otherwise| D[PermissionDenied<br/>+ log + nava_authz_denied]
```

## Rules

- **Identity** comes only from the verified client certificate's CN:
  `nava-relay`, `nava-core`, `nava-worker`, `nava-operator`, `temporal-system`.
  Any other CN, no certificate, or a bearer token gives no identity and is denied.
- **API**: the per-identity lists in `temporal-server/authz/policy.go`.
  `temporal-system` (the server's system worker) is allowed everything.
- **Namespace**: an API whose request has a `namespace` field must target `nava`;
  an API without one (for example `GetSystemInfo`) must target none.
- `grpc.health.v1.Health/Check` is allowed for every caller, for probes.
- The full generated table is `temporal-server/authz/testdata/allowlist.golden.md`.

## Drift checks (run on every Temporal upgrade)

`authz/policy_test.go` reflects over the `WorkflowService`, `OperatorService`
and `AdminService` gRPC descriptors at the pinned versions and fails if:

- a descriptor method is neither allow-listed nor in `deniedByDefault`, or is in both;
- a listed method no longer exists;
- `policy.go` disagrees with the ADR rules (written as rules with the ADR's wildcards);
- any (identity, method, namespace) decision differs from those rules;
- the golden table is stale (`go test ./authz -run TestAllowListGolden -update`).

`gomod_test.go` fails on any `replace`, any dependency on a platform module, a
direct `go.temporal.io/sdk` dependency, or a moved pin.

## Observability

Each deny logs `Temporal API call denied` (warn) with `identity`, `api`,
`namespace`, `reason`, `cert_subject`, `cert_serial`, `cert_issuer`, and
increments `nava_authz_denied{identity,api,namespace,reason}`. Metric tags are
bounded (`none`/`other` for unknown values); the Prometheus reporter rewrites
`-` and `/` in tag values to `_`. The caller only sees a generic
`PermissionDenied`.

## Local use

```sh
make temporal-server-test        # unit, policy, go.mod and in-process end-to-end tests
make temporal-server-image       # builds nava/temporal-server:dev
make temporal-server-run         # dev certs + run the image on :7233 (in-memory SQLite)
```

The `nava` namespace can only be registered with the `temporal-system`
certificate; no other identity may call `RegisterNamespace`.

## Startup refusals (T28)

`checkConfig` refuses to start when: frontend `requireClientAuth` is false; the
frontend client CA is not a self-signed root with path length 0; the same CA is
trusted on frontend and internode (internode ports have no Authorizer, so it has
its own CA and certificate); `services.frontend.rpc.httpPort` is set;
`system.disableStreamingAuthorizer` is true (the dynamic config client also hides
that key, so a later file edit cannot enable it). The operator's admin
`GetWorkflowExecutionRawHistory[V2]` calls are denied: they skip the namespace
gate and `GetWorkflowExecutionHistory` covers the need.

`temporal-server --bootstrap` registers `nava` and the `NavaTenantId` Keyword
search attribute with the `temporal-system` certificate, so that certificate stays
in the server container. `temporal-server health` and `temporal-server proxy` are
the compose health probe and the 7233-only gateway.

## Operator tunnel to the Temporal UI (T28)

The UI (`docker compose --profile operator up -d`) wields the `nava-operator`
identity, so it publishes no port: a port published on the host is reachable from
platform containers (for example through `host.docker.internal` on Docker
Desktop), which would let them use that identity. Reach it with an explicit
tunnel that lives only while you run it:

```sh
make operator-tunnel        # 127.0.0.1:18233 -> temporal-ui:8080; Ctrl-C closes it
```

While the tunnel runs, that port is reachable from the host like any published
port, so run it only on a developer machine and close it when done. Login through
Keycloak OIDC is deferred until the UI must be reachable without a tunnel.
`stack-test.sh temporal_operator_reach` checks that no platform container reaches
the UI by name or through the host.
