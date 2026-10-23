---
type: rfc
id: RFC-003
title: ADR-001-A1 follow-up F2, system-workflow reach via service certificates (erratum E2 + amendment A2)
status: DRAFT (round 0, for Technical Architect critique)
date: 2026-10-23
issue: TICK-129
source: TICK-122 findings comment 01a16770 (Security & Multi-Tenancy Reviewer), PR #36 (skipped tests)
amends: "[[ADR-001-A1-temporal-server-1-29-7-workload-allow-list]] D9, D6-K; [[ADR-001-platform-blueprint]] Annex A.2"
author: Solution Architect
---

# RFC-003: close the F2 findings (E2 + A2)

> **Invariants source.** The project description is empty, so it defines no invariants. As in A1, I checked against [[ADR-001-platform-blueprint]] §1 **I1–I9** (nine, not eight). If the project is meant to carry its own list of eight, the invariant checks below are incomplete against it.

## 1. Recommendation

Publish one record, **ADR-001-A2**, with two labelled parts:

- **Erratum E2** (A1's wording already requires it; the D9 table just failed to list the rows): **F2-3**, plus one item I found while framing, **F2-5** (below).
- **Amendment A2** (new rules): **F2-1**, **F2-2**, **F2-3b**, **F2-4**, and the `nava-core` JWT-tenant rule **R-CORE**.

Five rules close everything:

| Rule | Closes | Part |
|---|---|---|
| **D9d** token prefix for `nava-worker` task-token APIs | F2-3 | E2 |
| **D9e** workflow-id prefix for `nava-worker` `GetWorkflowExecutionHistory` | F2-5 | E2 |
| **D9f** workflow-id prefix for every `nava-core` API | F2-2, F2-4 | A2 |
| **D9b+** polls must be unversioned (F2-1 option: the **D9b rule**, not the D6-K row) | F2-1 | A2 |
| **D6-K** `frontend.enableTokenNamespaceEnforcement` must be `true`; guard gains a must-be-true mode | F2-3b | A2 |
| **R-CORE** `nava-core` builds every workflow id from the JWT tenant | at-scale requirement | A2 |

None of this is a tenant-to-tenant gap. Every finding needs a stolen service certificate. Urgency is normal; it does not block PR #36.

## 2. Erratum or amendment, item by item

The test I applied: does the **accepted A1 text**, read as written, already require the behaviour? If yes, the code and the D9 table are short of the text, so it is an erratum. If the text is silent or explicitly excludes the case, it is a new rule, so an amendment.

The governing A1 sentence is: *"A request that names a workflow type, a task queue or a workflow id is allowed only if **every** such field is allow-listed."* The limiting A1 sentence is: *"D9 doesn't change … `nava-core` / `nava-operator`."*

| ID | Verdict | Why |
|---|---|---|
| **F2-1** versioning fields on polls | **Amendment** | `DeploymentOptions` and `WorkerVersionCapabilities` are not a workflow type, a task queue or a workflow id. D9b inspects only `TaskQueue`. Nothing in A1 covers them. |
| **F2-2** `nava-core` calls on `temporal-sys-*` ids | **Amendment** | A1 explicitly says D9 does not change `nava-core`. Bringing `nava-core` under the prefix rule reverses an explicit A1 exclusion, which only an amendment may do. |
| **F2-3** forged activity / workflow-task tokens | **Erratum** | A task token *names a workflow id* (`token.v1.Task.workflow_id`), and `nava-worker` is inside D9's scope. The governing sentence already requires the prefix; E1 already applies it to `RespondWorkflowTaskCompleted`. The D9 table and the code just omit the other token APIs. |
| **F2-3b** token namespace vs request namespace | **Amendment** | A new D6-K row, and a new guard mode (must-be-true). D6-K is a closed table of named keys; adding one is a new rule. |
| **F2-4** `nava-core` `UpdateWorkflowExecution` on `temporal-sys-worker-deployment:*` | **Amendment** | Same rule and same reason as F2-2; F2-4 is one instance of it. |
| **F2-5** (new) `nava-worker` `GetWorkflowExecutionHistory` on any id | **Erratum** | Found while framing: `GetWorkflowExecutionHistory` is on `nava-worker`'s allow-list (`authz/policy.go:59`), names a workflow id, and has no `workloadChecks` entry, so a worker cert can read the history of `temporal-sys-*` workflows in `nava`. Same governing sentence as F2-3. Severity is Security's call; I'd expect Low (needs a worker cert, reads system-workflow history, no tenant rows). **Not yet tested; Security to confirm.** |

## 3. The rules (proposed A1 text)

### D9d (erratum E2): task-token prefix for `nava-worker`

For `nava-worker` on `RespondWorkflowTaskFailed`, `RespondActivityTaskCompleted`, `RespondActivityTaskFailed`, `RespondActivityTaskCanceled` and `RecordActivityTaskHeartbeat`: the Authorizer decodes `TaskToken` as `token.v1.Task` and **denies, with an alert,** if it does not decode or its `WorkflowId` lacks the `t:<tenant>:` prefix. This is E1's first bullet, extended from `RespondWorkflowTaskCompleted` to every task-token API on the allow-list.

`RespondQueryTaskCompleted` is out of scope: its token is `token.v1.QueryTask` (namespace, task queue, task id) and names no workflow id. The `*ById` activity APIs stay denied by default (A1, T27c).

### D9e (erratum E2): history reads for `nava-worker`

`nava-worker` `GetWorkflowExecutionHistory`: `Execution.WorkflowId` must carry the `t:<tenant>:` prefix. The SDK only fetches history for workflows it is executing, which are all `t:`-prefixed by D9a, so no legitimate call is affected.

### D9f (amendment A2): workflow-id prefix for `nava-core`

Replace A1's *"D9 doesn't change … `nava-core`"* with: for `nava-core` on `RequestCancelWorkflowExecution`, `UpdateWorkflowExecution`, `QueryWorkflow`, `DescribeWorkflowExecution` and `GetWorkflowExecutionHistory`, the request's workflow id (`WorkflowExecution.WorkflowId` / `Execution.WorkflowId`) must carry the `t:<tenant>:` prefix. Missing or empty is denied. This closes F2-4 because `temporal-sys-worker-deployment:<name>` has no `t:` prefix.

`nava-operator` stays outside D9: inspecting system workflows is its job, and it is restricted to the operator network (T28).

The consolidated statement for Annex A.2, after E2 + A2: **every workflow id that an allow-listed `nava-relay`, `nava-core` or `nava-worker` API names, including the one inside a `token.v1.Task` task token, must carry the `t:<tenant>:` prefix.** This is Security's shape (1), scoped by identity.

### D9b+ (amendment A2): polls must be unversioned (F2-1)

For `nava-worker` `PollWorkflowTaskQueue` and `PollActivityTaskQueue`, in addition to D9b: `DeploymentOptions.WorkerVersioningMode` must be `UNSPECIFIED` or `UNVERSIONED` (absent counts as unspecified), and `WorkerVersionCapabilities.UseVersioning` must be `false`. Anything else is denied with an alert.

Verified at source: matching registers a deployment version (and so starts `temporal-sys-worker-deployment*` workflows) only when `worker_versioning.DeploymentFromCapabilities` returns non-nil, which happens only for mode `VERSIONED` or `UseVersioning && DeploymentSeriesName != "" && BuildId != ""` (`server@v1.29.7 common/worker_versioning/worker_versioning.go:159-181`, called from `service/matching/physical_task_queue_manager.go:372`). Our SDK v1.49.0 always sends `WorkerVersionCapabilities{BuildId, UseVersioning: false}` and sends `DeploymentOptions` only when a deployment version is configured (`sdk@v1.49.0 internal/worker.go:586`, `internal_task_pollers.go:1411`). So the rule denies nothing our workers send today.

**Drift guard (makes the rule allow-list shaped):** a CI-C test holds a golden list of the field names of `PollWorkflowTaskQueueRequest` and `PollActivityTaskQueueRequest` (from the proto descriptor). Any `go.temporal.io/api` bump that adds a field fails CI until someone classifies the field in a policy PR. This answers the obvious objection that D9b+ is a deny-list on two named fields.

### D6-K row (amendment A2): token-namespace enforcement (F2-3b)

| Key | Go symbol (CI-E) | Refuse start if | Shipped |
|---|---|---|---|
| `frontend.enableTokenNamespaceEnforcement` | `dynamicconfig.EnableTokenNamespaceEnforcement` | not `true` (missing also refuses) | `true` |

Verified: key and default `true` at `server@v1.29.7 common/dynamicconfig/constants.go:789`, a global setting. `checkSecurityConfig` (`temporal-server/cmd/server/check.go:85`) only implements must-be-false today; `guardedKey` gains a `want bool` field. I require the explicit value, like the other `requireExplicit` rows, so that an upstream default change cannot silently turn it off.

### R-CORE (amendment A2, stated rule): `nava-core` derives the tenant of every workflow id from the JWT

> **R-CORE.** `nava-core` builds every workflow id it sends to Temporal as `t:<tenant>:<target_key>`, where `<tenant>` is the tenant resolved by D26 from the request's validated access token (through `tenant_identity_binding`), and `<target_key>` is read from `workflow_ref` under RLS in that tenant's context. It never takes a full workflow id, or the tenant segment of one, from the request path, query string, body, headers other than the bearer token, or any stored column. A record that is not visible under RLS yields the same "not found" as a missing one (I5) and no Temporal call is made.

Why it must be a rule: D9f's prefix check proves a `nava-core` call names *a* tenant's workflow, never *which* tenant, because the `nava-core` certificate carries no tenant. The only thing between tenant A's JWT and tenant B's workflow is R-CORE. It is I2 applied to the `nava-core` → Temporal path, mirroring the relay's existing rule (Annex A.2: *"builds every workflow id as `t:<outbox.tenant_id>:<target_key>`, and never reads a full id from a column"*).

Enforcement: the Authorizer cannot check it, so it is enforced by (a) a single helper in `nava-core` that is the only constructor of Temporal workflow ids (lint/review: no other code formats `t:`), and (b) an acceptance test: tenant A's JWT on a path naming tenant B's record makes no Temporal call and returns the same 404 as a missing record. Security reviews the first `nava-core` change that calls these APIs against R-CORE (Security's own offer on TICK-122). Today no `nava-core` product code calls them, so this binds future work.

## 4. F2-1: the option and the reason

**Chosen: the D9b rule (D9b+). Rejected: the D6-K row `system.enableDeploymentVersions` must be `false`.**

1. **The Authorizer is the wall; config is defence in depth.** A1 already rejected "close S1 by config only". D9b+ stops the abusing identity at request time, per request, with an alert naming the certificate. The D6-K row only fixes a server setting.
2. **D6-K checks only at startup (RS2).** `enableDeploymentVersions` is a *namespace-scoped* dynamic setting. Any future runtime write path to dynamic config reopens F2-1 silently. D9b+ is compiled into the binary.
3. **It keeps the door for a deliberate feature.** If we ever adopt worker versioning, that is one policy PR to D9b+ with Security review, the same path as every other allow-list change (RS5). Flipping a feature flag server-wide is a blunter instrument with RS4 risk (internal workflows that assume it).
4. **The usual weakness of the rule option is closed** by the descriptor drift test above.

Cost: one more check in `checkWorkflowTaskPoll` / `checkActivityTaskPoll`, two red→green tests from PR #36, one drift test. No config change.

## 5. Other options considered

| Option | Why not |
|---|---|
| D6-K `enableDeploymentVersions = false` instead of D9b+ (Security's alternative) | §4. |
| Both D9b+ and the D6-K row | The CTO asked for one. Defence in depth here buys little: D9b+ is request-time and drift-guarded. Revisit if a second identity ever gets poll rights. |
| Authorizer compares the token's `NamespaceId` with `nava`'s id (instead of the F2-3b D6-K row) | The Authorizer has no namespace registry and maps nothing by id today; adding one is new state in the auth path. The frontend already does exactly this check; we only need to guarantee it stays on. |
| Deny-list `temporal-sys-*` ids for `nava-core` | Rejected in A1 for the same reason: it allows everything Temporal adds until someone notices. The prefix rule is an allow-list. |
| Give `nava-core` a per-tenant certificate so the Authorizer can tell tenants apart | One certificate per tenant per pod: rotation and issuance cost grows with tenants, and the Authorizer gains tenant state. R-CORE plus RLS gets the same guarantee in code we already test. Revisit if `nava-core` ever runs tenant-supplied code. |
| Leave F2-2/F2-4 as accepted residuals until `nava-core` calls these APIs | The fix is a few lines in a code path with no callers yet; leaving it means the first caller lands without the wall. |

## 6. Invariants touched

| Invariant | Effect |
|---|---|
| I2 | **Strengthened.** R-CORE states I2 for the `nava-core` → Temporal path; D9d/D9e/D9f stop service certificates reaching non-tenant workflows. |
| I6 | Unchanged. No new path starts or signals a workflow; D9b+ removes one server-side start path (worker-deployment workflows). |
| I5 | R-CORE reuses I5's "not visible = not found". |
| D22 | Enforced more completely. |
| I1, I3, I4, I7, I8, I9 | Not touched. |

## 7. Acceptance (for the `core-dev` task the Technical Lead plans)

- **CI-I extension:** every PR #36 test skipped with F2-1, F2-2, F2-3, F2-3b or F2-4 is un-skipped and passes. Each one was shown red on `main` @ `84d9bf0` by Security.
- **New tests:** F2-5 red → green (worker cert, `GetWorkflowExecutionHistory` on `temporal-sys-scheduler:x`); D9e allow path (`t:` id); D9d allow path for each token API; D9b+ allow path for the exact poll our SDK v1.49.0 sends.
- **CI-E:** the new D6-K row asserted by symbol, including "missing refuses" and "false refuses".
- **CI-C:** poll-request field golden; `allowlist.golden.md` regenerated with D9b+, D9d, D9e, D9f.
- **CI-D / CI-G:** the stack smoke still passes (proves D9b+/D9e deny nothing legitimate).
- **R-CORE:** a tracked acceptance test for the first `nava-core` change that calls Temporal; not part of this task, because there is no caller yet. The Technical Lead records it as a dependency of that future task.

## 8. Residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS6 | R-CORE is enforced by code and tests, not by the Authorizer | Security & Multi-Tenancy Reviewer | First `nava-core` Temporal call; any `nava-core` path that runs tenant-supplied code |
| RS2 (unchanged, widened) | The new D6-K row is startup-only, like the rest | Security & Multi-Tenancy Reviewer | Any runtime write path to dynamic config |

## 9. Assumptions not verified (for the Technical Architect)

1. **A1** No other request path at 1.29.7 registers a deployment version for a `nava-worker` cert. I found three `RegisterTaskQueueWorker` call sites: matching's poll path and the two deployment clients, which run in the system worker. `RecordWorkerHeartbeat` and `ShutdownWorker` carry deployment fields; I believe they do not register, but have not traced it. **verify-by-spike candidate.**
2. **A2** All five task-token APIs in D9d carry a `token.v1.Task` (not a different token type) at 1.29.7. E1's decoder uses that type; I have not checked each frontend handler.
3. **A3** F2-5 is real as described (no `workloadChecks` entry for worker history). Read from `authz/authorizer.go:143-152` and `policy.go:59`; not tested.
4. **A4** Turning off nothing (we keep `enableDeploymentVersions` at its default) means CI-G needs no change.

## 10. Questions for reviewers

- **Technical Architect:** A1 and A2 above; and whether the poll-request descriptor golden belongs in CI-C or its own CI step.
- **Security:** confirm F2-5 and its severity; confirm the D9b+ field rule matches what you exploited in PR #36.
- **Technical Lead:** one `core-dev` task for D9b+/D9d/D9e/D9f + guard mode, or split the guard (`cmd/server`) from the Authorizer (`authz`).

**Numbering assumed:** RFC-003 (highest in repo is RFC-002), erratum E2 (highest is E1 in ADR-001), amendment A2 (highest is A1).
