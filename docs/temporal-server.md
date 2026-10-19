# Custom Temporal server (`temporal-server/`)

ADR-001 D22 and Annex A.2, plan task T27a. A separate Go module (R7) that builds
Temporal `go.temporal.io/server` v1.28.4 + `go.temporal.io/api` v1.50.1 with a
deny-by-default Authorizer and a certificate ClaimMapper, plus worker command
inspection (T27b, ADR-001 erratum E1, tightened by T27c). Compose wiring, PostgreSQL config and the operator-network
restriction are T28.

```mermaid
flowchart LR
  C[client cert<br/>installation CA] -->|mTLS handshake| F[frontend :7233]
  F --> CM[ClaimMapper<br/>CN to identity]
  CM --> AZ{Authorizer}
  AZ -->|identity, API and namespace allowed| CI{nava-worker<br/>RespondWorkflowTaskCompleted?}
  CI -->|no| H[handler]
  CI -->|yes| INS{every command<br/>inside the task's tenant?}
  INS -->|yes| H
  INS -->|no| DC[PermissionDenied<br/>+ error log alert=true<br/>+ nava_authz_denied]
  AZ -->|otherwise| D[PermissionDenied<br/>+ warn log + nava_authz_denied]
```

## Rules

- **Identity** comes only from the verified client certificate's CN:
  `nava-relay`, `nava-core`, `nava-worker`, `nava-operator`, `temporal-system`.
  Any other CN, no certificate, or a bearer token gives no identity and is denied.
- **API**: the per-identity lists in `temporal-server/authz/policy.go`.
  `nava-worker` gets only the task-token forms of `RespondActivityTask*`; the
  `*ById` forms act on any workflow id and are denied (T27c; M1 has no async
  completion).
  `temporal-system` (the server's system worker) is allowed everything.
- **Namespace**: an API whose request has a `namespace` field must target `nava`;
  an API without one (for example `GetSystemInfo`) must target none.
- `grpc.health.v1.Health/Check` is allowed for every caller, for probes.
- The full generated table is `temporal-server/authz/testdata/allowlist.golden.md`.

## Command inspection (T27b, erratum E1; T27c)

For `RespondWorkflowTaskCompleted` from `nava-worker` the Authorizer reads the
request (`authz/commands.go`). The task's tenant is the `<tenant>` of the task
token's workflow id `t:<tenant>:…`. One bad command denies the whole response.

| Denied when | `reason` |
|---|---|
| the task token does not decode | `task_token_invalid` |
| its workflow id has no `t:<tenant>:` prefix | `workflow_id_no_tenant` |
| the request body is missing or not a `RespondWorkflowTaskCompletedRequest` | `request_malformed` |
| a command type is not on the E1 allow-list (Nexus, `Unspecified`, any type a later API adds) | `command_not_allowed` |
| child / signal-external / cancel-external targets a namespace other than `nava` (empty means the task's own, `nava`) | `command_namespace` |
| … or a workflow id with another tenant's prefix, or none | `command_cross_tenant` |
| schedule-activity without a `nava-tenant` header equal to the task's tenant | `activity_header` |
| child (same-tenant target) without a `nava-tenant` header equal to the task's tenant | `child_header` |
| continue-as-new without a `nava-tenant` header equal to the task's tenant | `continue_as_new_header` |
| continue-as-new with `cron_schedule` | `continue_as_new_cron` |
| child, continue-as-new or upsert sets `NavaTenantId` to anything but the task's tenant (unset is allowed) | `search_attribute_tenant` |

The `nava-tenant` header and `NavaTenantId` must be default-converter string
payloads (`json/plain`, a JSON string); anything else counts as missing.
`temporal-system` is not inspected. `TestCommandNamesCoverPinnedEnum` fails when
a Temporal upgrade adds a command type, until it is classified.

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

Each API deny logs `Temporal API call denied` (warn) with `identity`, `api`,
`namespace`, `reason`, `command` (`none`), `cert_subject`, `cert_serial`,
`cert_issuer`. Each command deny logs `Temporal workflow command denied`
(**error**, `alert=true`) with the same fields plus `workflow_id`,
`command_index`, `target_workflow_id` (when the command has one) and `detail`.
Both increment `nava_authz_denied{identity,api,namespace,reason,command}`.

Metric tags are bounded: `none`/`other` for unknown values, `command` is a
command type name or `unknown`. Workflow ids carry the tenant and are
unbounded, so they are in the log line only, never in a tag. The Prometheus
reporter rewrites `-` and `/` in tag values to `_`. The caller only sees a
generic `PermissionDenied`.

A denied workflow task retries until someone acts, so every command deny must
page. The alert rule, once an alerting backend exists (not in M1 compose):

```yaml
- alert: TemporalWorkflowCommandDenied
  expr: sum by (reason, command) (increase(nava_authz_denied{command!="none"}[5m])) > 0
  labels: {severity: page}
  annotations:
    summary: "nava-worker sent a cross-tenant or unlisted workflow command"
    description: "Find workflow_id and the certificate in the error log line 'Temporal workflow command denied'."
```

## Local use

```sh
make temporal-server-test        # unit, policy, go.mod and in-process end-to-end tests
make temporal-server-image       # builds nava/temporal-server:dev
make temporal-server-run         # dev certs + run the image on :7233 (in-memory SQLite)
```

The `nava` namespace can only be registered with the `temporal-system`
certificate; no other identity may call `RegisterNamespace`.
