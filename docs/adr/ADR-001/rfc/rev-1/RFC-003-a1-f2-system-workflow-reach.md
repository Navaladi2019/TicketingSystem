---
type: rfc
id: RFC-003
title: ADR-001-A1 follow-up F2, system-workflow reach via service certificates (erratum E2 + amendment A2)
status: REVISED (round 1 revision; round 2 re-checks blocking items only)
date: 2026-10-23
issue: TICK-129
source: TICK-122 findings comment 01a16770 (Security & Multi-Tenancy Reviewer), PR #36 (skipped tests, incl. commit 92d0863); round-1 critiques 01a16780 (Technical Architect), 01a167d3 (Security)
amends: "[[ADR-001-A1-temporal-server-1-29-7-workload-allow-list]] D9, D6-K; [[ADR-001-platform-blueprint]] Annex A.2"
author: Solution Architect
---

# RFC-003: close the F2 findings (E2 + A2)

## Changelog (round 1 → revision)

| # | Change | Raised by | Where |
|---|---|---|---|
| 1 | **F2-6** added to E2 as an erratum: new rule **D9g** (`ShutdownWorker`, `RecordWorkerHeartbeat`), plus a **D9d query-token clause** (`RespondQueryTaskCompleted`). The §3 consolidated statement now covers every **task queue** as well as every workflow id. | Technical Architect (blocking), Security (blocking) | §1, §2, §3 |
| 2 | **R-CORE page-token clause**: `nava-core` never forwards a client-supplied Temporal page token or other opaque token. Matching acceptance case added and listed under RS6. | Security (blocking) | §3 R-CORE, §7, §8 |
| 3 | **Drift golden widened** to the request message of every API allow-listed for `nava-relay`, `nava-core` and `nava-worker`, in CI-C next to `TestAllowListGolden`. | Technical Architect, Security | §3 D9b+, §7 |
| 4 | **§7 acceptance** points at the PR #36 tests at `92d0863`: `TestAdversarialF2WorkerHistoryTenantPrefix`, `TestAdversarialF2WorkerShutdownTaskQueue`, `TestAdversarialF2WorkerQueryTokenTaskQueue`; keeps the SDK-shaped `ShutdownWorker` allow-path control. | CTO ruling | §7 |
| 5 | Technical Architect #7 (heartbeat queue) and #8 (query token) are closed **by rule**, not residual (CTO ruling). One residual recorded: **RS7**, a pod can unload a sibling pod's SDK-shaped sticky queue. | CTO ruling | §8 |
| 6 | §9 assumptions A1–A3 resolved by the Technical Architect's spike and Security's tests; F2-5 confirmed (Low). | — | §2, §9 |

> **Invariants source.** The project description is empty, so it defines no invariants. As in A1, I checked against [[ADR-001-platform-blueprint]] §1 **I1–I9** (nine, not eight). If the project is meant to carry its own list of eight, the invariant checks below are incomplete against it.

## 1. Recommendation

Publish one record, **ADR-001-A2**, with two labelled parts:

- **Erratum E2** (A1's wording already requires it; the D9 table just failed to list the rows): **F2-3**, **F2-5** (found while framing, confirmed by Security) and **F2-6** (found by the Technical Architect in round 1, confirmed by Security).
- **Amendment A2** (new rules): **F2-1**, **F2-2**, **F2-3b**, **F2-4**, and the `nava-core` JWT-tenant rule **R-CORE**.

Seven rules close everything:

| Rule | Closes | Part |
|---|---|---|
| **D9d** token prefix for `nava-worker` task-token APIs, plus the query-token task-queue clause | F2-3, F2-6 (query token) | E2 |
| **D9e** workflow-id prefix for `nava-worker` `GetWorkflowExecutionHistory` | F2-5 | E2 |
| **D9g** task queues named by `nava-worker` `ShutdownWorker` and `RecordWorkerHeartbeat` | F2-6 | E2 |
| **D9f** workflow-id prefix for every `nava-core` API | F2-2, F2-4 | A2 |
| **D9b+** polls must be unversioned (F2-1 option: the **D9b rule**, not the D6-K row) | F2-1 | A2 |
| **D6-K** `frontend.enableTokenNamespaceEnforcement` must be `true`; guard gains a must-be-true mode | F2-3b | A2 |
| **R-CORE** `nava-core` builds every workflow id from the JWT tenant and never forwards a client-supplied opaque Temporal token | at-scale requirement; Security round-1 item 4 | A2 |

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
| **F2-5** (new) `nava-worker` `GetWorkflowExecutionHistory` on any id | **Erratum** | `GetWorkflowExecutionHistory` is on `nava-worker`'s allow-list (`authz/policy.go:59`), names a workflow id, and has no `workloadChecks` entry (`authz/authorizer.go:149`), so a worker cert can read the history of `temporal-sys-*` workflows in `nava`. Same governing sentence as F2-3. **Confirmed** by the Technical Architect's spike and Security's `TestAdversarialF2WorkerHistoryTenantPrefix`. **Severity: Low** (Security). |
| **F2-6** (new, round 1) `nava-worker` `ShutdownWorker` / `RecordWorkerHeartbeat` / `RespondQueryTaskCompleted` name unchecked task queues | **Erratum** | `ShutdownWorker.StickyTaskQueue` makes the frontend force-unload WORKFLOW partition 0 of the named queue (`server@v1.29.7 frontend/workflow_handler.go:2733`); for `"nava-tq"` that is the root partition every tenant's dispatch uses. `WorkerHeartbeat.TaskQueue` (both APIs) and `token.v1.QueryTask.TaskQueue` also name task queues and are unchecked. Each field *names a task queue*, so the governing sentence already requires it to be allow-listed. Same test that put F2-3 and F2-5 in E2. **Confirmed** by the Technical Architect's spike and Security's `TestAdversarialF2WorkerShutdownTaskQueue` / `TestAdversarialF2WorkerQueryTokenTaskQueue`. **Severity: Low**, availability only (Security). |

## 3. The rules (proposed A1 text)

### D9d (erratum E2): task-token prefix for `nava-worker`

For `nava-worker` on `RespondWorkflowTaskFailed`, `RespondActivityTaskCompleted`, `RespondActivityTaskFailed`, `RespondActivityTaskCanceled` and `RecordActivityTaskHeartbeat`: the Authorizer decodes `TaskToken` as `token.v1.Task` and **denies, with an alert,** if it does not decode or its `WorkflowId` lacks the `t:<tenant>:` prefix. This is E1's first bullet, extended from `RespondWorkflowTaskCompleted` to every task-token API on the allow-list.

All five APIs carry a `token.v1.Task` (`common/tasktoken/serializer.go:21`; confirmed by the Technical Architect).

**D9d, query-token clause (erratum E2, F2-6):** for `nava-worker` `RespondQueryTaskCompleted`, the Authorizer decodes `TaskToken` as `token.v1.QueryTask` and **denies, with an alert,** if it does not decode, or if its `TaskQueue` is neither a platform queue nor sticky-shaped (the name half of the D9b sticky rule, `stickyQueueName`). The query token names a task queue, not a workflow id; delivery is keyed by a server-generated UUID task id (`matching_engine.go:1028`), so the clause closes reach, and forging a result still needs a live UUID.

The `*ById` activity APIs stay denied by default (A1, T27c).

### D9e (erratum E2): history reads for `nava-worker`

`nava-worker` `GetWorkflowExecutionHistory`: `Execution.WorkflowId` must carry the `t:<tenant>:` prefix. The SDK only fetches history for workflows it is executing, which are all `t:`-prefixed by D9a, so no legitimate call is affected.

### D9g (erratum E2): `nava-worker` `ShutdownWorker` and `RecordWorkerHeartbeat` (F2-6)

> **D9g.** For `nava-worker` on `ShutdownWorker` and `RecordWorkerHeartbeat`: `StickyTaskQueue` must be empty or pass the name half of the D9b sticky rule (`stickyQueueName`, and not a platform queue). Every `WorkerHeartbeat.TaskQueue` must be a platform queue. Anything else is denied with an alert.

Text as drafted by the Technical Architect. Notes for the implementer:

- An absent heartbeat (nil on `ShutdownWorker`, empty list on `RecordWorkerHeartbeat`) is allowed. A present heartbeat with an empty `TaskQueue` is denied.
- **Legitimate traffic is unaffected.** sdk-go v1.49.0 sends `StickyTaskQueue` = its own `<host>:<uuid>` for the workflow worker and `""` for activity and session workers, and fills `WorkerHeartbeat.TaskQueue` from the worker's own task queue (`sdk@v1.49.0 internal/internal_worker.go:1760-1802`, `:2668-2678`).
- **Fields the server does not see yet.** sdk-go v1.49.0 is built against `go.temporal.io/api` v1.63.5 and also sets `ShutdownWorkerRequest.TaskQueue`, `TaskQueueTypes` and `WorkerInstanceKey`. Our server's api v1.53.0 has only fields 1–5 (`namespace`, `sticky_task_queue`, `identity`, `reason`, `worker_heartbeat`), so the Authorizer never sees the extra fields today. The widened drift golden (below) fails the api bump that adds them; that policy PR must classify `TaskQueue` as "platform queue" (session-worker queues included, if we ever enable sessions).
- **Accepted consequence (RS7):** `stickyQueueName` checks shape, not ownership. One `nava-worker` pod can still unload a sibling pod's SDK-shaped sticky queue. This is harmless: sticky dispatch falls back to the normal queue.

### D9f (amendment A2): workflow-id prefix for `nava-core`

Replace A1's *"D9 doesn't change … `nava-core`"* with: for `nava-core` on `RequestCancelWorkflowExecution`, `UpdateWorkflowExecution`, `QueryWorkflow`, `DescribeWorkflowExecution` and `GetWorkflowExecutionHistory`, the request's workflow id (`WorkflowExecution.WorkflowId` / `Execution.WorkflowId`) must carry the `t:<tenant>:` prefix. Missing or empty is denied. This closes F2-4 because `temporal-sys-worker-deployment:<name>` has no `t:` prefix.

`nava-operator` stays outside D9: inspecting system workflows is its job, and it is restricted to the operator network (T28).

The consolidated statement for Annex A.2, after E2 + A2:

> **Every workflow id that an allow-listed `nava-relay`, `nava-core` or `nava-worker` API names, including the one inside a `token.v1.Task` task token, must carry the `t:<tenant>:` prefix. Every task queue that such an API names, including the one inside a `token.v1.QueryTask` query token and inside a `WorkerHeartbeat`, must be a platform queue or, where D9b permits a sticky queue, sticky-shaped. A request field that names a workflow id or a task queue and that no rule classifies is a CI failure (drift golden), not an allow.**

This is Security's shape (1), scoped by identity and extended to task queues.

### D9b+ (amendment A2): polls must be unversioned (F2-1)

For `nava-worker` `PollWorkflowTaskQueue` and `PollActivityTaskQueue`, in addition to D9b: `DeploymentOptions.WorkerVersioningMode` must be `UNSPECIFIED` or `UNVERSIONED` (absent counts as unspecified), and `WorkerVersionCapabilities.UseVersioning` must be `false`. Anything else is denied with an alert.

Verified at source: matching registers a deployment version (and so starts `temporal-sys-worker-deployment*` workflows) only when `worker_versioning.DeploymentFromCapabilities` returns non-nil, which happens only for mode `VERSIONED` or `UseVersioning && DeploymentSeriesName != "" && BuildId != ""` (`server@v1.29.7 common/worker_versioning/worker_versioning.go:159-181`, called from `service/matching/physical_task_queue_manager.go:372`). Our SDK v1.49.0 always sends `WorkerVersionCapabilities{BuildId, UseVersioning: false}` and sends `DeploymentOptions` only when a deployment version is configured (`sdk@v1.49.0 internal/worker.go:586`, `internal_task_pollers.go:1411`). So the rule denies nothing our workers send today.

**Drift guard (makes the rules allow-list shaped), widened in round 1:** a CI-C test in `temporal-server/authz/policy_test.go`, next to `TestAllowListGolden` (same mechanism, same package, no new CI step), holds a golden of the full field paths of the **request message of every API allow-listed for `nava-relay`, `nava-core` and `nava-worker`**, read from the proto descriptors, recursing into nested messages (so `WorkerHeartbeat.TaskQueue` and `DeploymentOptions.*` are covered). The golden is generated from the allow-list itself, so adding an API to an identity's allow-list adds its request message automatically. Any `go.temporal.io/api` bump that adds, removes or renames a field fails CI until a policy PR classifies the field (names a workflow id / task queue / workflow type → which rule checks it; or inert). This answers the objection that D9b+ is a deny-list on named fields, and it catches the drift that produced F2-6 (`ShutdownWorkerRequest` grew `TaskQueue`, `TaskQueueTypes`, `WorkerHeartbeat`, `WorkerInstanceKey` across recent api releases).

### D6-K row (amendment A2): token-namespace enforcement (F2-3b)

| Key | Go symbol (CI-E) | Refuse start if | Shipped |
|---|---|---|---|
| `frontend.enableTokenNamespaceEnforcement` | `dynamicconfig.EnableTokenNamespaceEnforcement` | not `true` (missing also refuses) | `true` |

Verified: key and default `true` at `server@v1.29.7 common/dynamicconfig/constants.go:789`, a global setting. `checkSecurityConfig` (`temporal-server/cmd/server/check.go:85`) only implements must-be-false today; `guardedKey` gains a `want bool` field. I require the explicit value, like the other `requireExplicit` rows, so that an upstream default change cannot silently turn it off.

### R-CORE (amendment A2, stated rule): `nava-core` derives the tenant of every workflow id from the JWT

> **R-CORE.** `nava-core` builds every workflow id it sends to Temporal as `t:<tenant>:<target_key>`, where `<tenant>` is the tenant resolved by D26 from the request's validated access token (through `tenant_identity_binding`), and `<target_key>` is read from `workflow_ref` under RLS in that tenant's context. It never takes a full workflow id, or the tenant segment of one, from the request path, query string, body, headers other than the bearer token, or any stored column. A record that is not visible under RLS yields the same "not found" as a missing one (I5) and no Temporal call is made.
>
> **R-CORE, opaque tokens (added round 1).** `nava-core` never sends Temporal a page token (`NextPageToken`) or any other opaque Temporal token that came from the client. It keeps the cursor server-side, or seals it with AEAD bound to the tenant and the workflow id, and treats a token that fails to open, or opens to a different tenant or workflow id, as "not found" (I5), with no Temporal call.

Why the token clause is needed (Security, round 1, Known from source, not tested live): when `GetWorkflowExecutionHistory` carries a `NextPageToken` with `WaitNewEvent=false`, history reads events by the token's `BranchToken`; the request's workflow id only picks the shard and is never re-checked against the token (`server@v1.29.7 service/history/api/getworkflowexecutionhistory/api.go:140-165`, `api/get_history_util.go:179-183`). So D9e and D9f bind the id in the request, not the history that comes back. A pass-through would let tenant A, holding a leaked page token of tenant B, read B's history through A's own record. Low today (no caller), High the day a pass-through ships.

Why it must be a rule: D9f's prefix check proves a `nava-core` call names *a* tenant's workflow, never *which* tenant, because the `nava-core` certificate carries no tenant. The only thing between tenant A's JWT and tenant B's workflow is R-CORE. It is I2 applied to the `nava-core` → Temporal path, mirroring the relay's existing rule (Annex A.2: *"builds every workflow id as `t:<outbox.tenant_id>:<target_key>`, and never reads a full id from a column"*).

Enforcement: the Authorizer cannot check it, so it is enforced by (a) a single helper in `nava-core` that is the only constructor of Temporal workflow ids (lint/review: no other code formats `t:`), and (b) two acceptance cases: **R-CORE-1**, tenant A's JWT on a path naming tenant B's record makes no Temporal call and returns the same 404 as a missing record; **R-CORE-2**, tenant A's JWT on A's own record with a client-supplied page token minted for tenant B's workflow (or any raw Temporal `NextPageToken`) makes no Temporal call with that token and returns the same 404 as a missing record. Security reviews the first `nava-core` change that calls these APIs against R-CORE (Security's own offer on TICK-122). Today no `nava-core` product code calls them, so this binds future work.

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
| I2 | **Strengthened.** R-CORE states I2 for the `nava-core` → Temporal path, including opaque tokens; D9d/D9e/D9f stop service certificates reaching non-tenant workflows; D9g and the D9d query clause stop them naming non-platform task queues. F2-6 is availability-only, not tenant-to-tenant. |
| I6 | Unchanged. No new path starts or signals a workflow; D9b+ removes one server-side start path (worker-deployment workflows). |
| I5 | R-CORE reuses I5's "not visible = not found". |
| D22 | Enforced more completely. |
| I1, I3, I4, I7, I8, I9 | Not touched. |

## 7. Acceptance (for the `core-dev` task the Technical Lead plans)

- **CI-I extension:** every PR #36 test skipped with F2-1, F2-2, F2-3, F2-3b, F2-4, F2-5 or F2-6 is un-skipped and passes. Each one was shown red on `main` @ `84d9bf0` by Security. By name, the round-1 tests in `temporal-server/authz/adversarial_f2_worker_reach_test.go` (PR #36, commit `92d0863`):

  | Test | Skip id | Rule | Red → green cases | Allow-path control (must stay green) |
  |---|---|---|---|---|
  | `TestAdversarialF2WorkerHistoryTenantPrefix` | F2-5 | D9e | system workflow ids, `""`, `t:`, `t::x`, `tenantA:x` denied | `t:`-prefixed `wfA` allowed |
  | `TestAdversarialF2WorkerShutdownTaskQueue` | F2-6 | D9g | `StickyTaskQueue` = platform queue, `temporal-sys-per-ns-tq`, `temporal-sys-x`, `nava-tq:not-a-uuid` denied; heartbeat queue `temporal-sys-per-ns-tq` denied on `ShutdownWorker` **and** `RecordWorkerHeartbeat` | **the SDK-shaped `ShutdownWorker`**: sticky `worker-host:<uuid>` and `""`, heartbeat on the platform queue, allowed |
  | `TestAdversarialF2WorkerQueryTokenTaskQueue` | F2-6 | D9d query clause | query token naming `temporal-sys-per-ns-tq` denied | query tokens naming the platform queue and an SDK sticky queue allowed |

- **New tests:** D9d allow path for each token API; D9b+ allow path for the exact poll our SDK v1.49.0 sends; D9g undecodable / empty-`TaskQueue` heartbeat denied; D9d query token that does not decode denied.
- **CI-E:** the new D6-K row asserted by symbol, including "missing refuses" and "false refuses".
- **CI-C:** the widened request-field golden (every request message allow-listed for Relay, Core and Worker) in `authz/policy_test.go` next to `TestAllowListGolden`; `allowlist.golden.md` regenerated with D9b+, D9d (both clauses), D9e, D9f, D9g.
- **CI-D / CI-G:** the stack smoke still passes, including a graceful worker shutdown (proves D9b+/D9d/D9e/D9g deny nothing legitimate).
- **R-CORE:** tracked acceptance cases **R-CORE-1** (cross-tenant path → same 404, no Temporal call) and **R-CORE-2** (client-supplied or cross-tenant page token → same 404, no Temporal call with that token) for the first `nava-core` change that calls Temporal; not part of this task, because there is no caller yet. The Technical Lead records both as a dependency of that future task.

## 8. Residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS6 | R-CORE is enforced by code and tests, not by the Authorizer. Covers both clauses: workflow ids built from the JWT tenant (R-CORE-1), and no client-supplied page token or other opaque Temporal token forwarded (R-CORE-2). The Authorizer cannot see inside a history page token, so D9e/D9f do not bind the history returned for one. | Security & Multi-Tenancy Reviewer | First `nava-core` Temporal call; any `nava-core` API that paginates Temporal results; any `nava-core` path that runs tenant-supplied code |
| RS7 | Under D9g, one `nava-worker` pod can unload a sibling pod's SDK-shaped sticky queue (`stickyQueueName` checks shape, not ownership). Harmless: sticky dispatch falls back to the normal queue. Accepted by Security in round 1. | Security & Multi-Tenancy Reviewer | Worker certificates become per-pod or per-tenant; sticky queues gain a tenant meaning |
| RS2 (unchanged, widened) | The new D6-K row is startup-only, like the rest | Security & Multi-Tenancy Reviewer | Any runtime write path to dynamic config |

Technical Architect round-1 items #7 (heartbeat registry) and #8 (query token) are **not** residuals: CTO ruling, closed by D9g and the D9d query clause.

## 9. Assumptions (status after round 1)

1. **A1, resolved (Known).** Only the poll path registers a deployment version: the single matching caller of `RegisterTaskQueueWorker` (`physical_task_queue_manager.go:645`) is reached only from `PollTask`. `RecordWorkerHeartbeat` and `ShutdownWorker` write only to the in-memory worker registry (Technical Architect, source + spike).
2. **A2, resolved (Known).** All five D9d APIs deserialize `token.v1.Task`; only `RespondQueryTaskCompleted` uses `token.v1.QueryTask` (Technical Architect).
3. **A3, resolved (Known).** F2-5 confirmed by spike and by `TestAdversarialF2WorkerHistoryTenantPrefix`; Low (Security).
4. **A4, open (Inferred).** We keep `enableDeploymentVersions` at its default, so CI-G needs no new config; CI-G gains the graceful-shutdown step only.
5. **A5, new (Known, from source).** D9g's sticky check reuses `stickyQueueName` (`authz/policy.go:233`, `^[^/]+:<uuid>$`) and `checkStickyTaskQueue`'s "not a platform queue" test (`authz/authorizer.go:256-259`) unchanged; Security's test cases (`nava-tq:not-a-uuid` denied, `worker-host:<uuid>` allowed) define its accepted behaviour.

## 10. Questions for reviewers (round 2: blocking items only)

- **Technical Architect:** do D9g and the D9d query clause read as you drafted them, and does the widened golden (recursive field paths, generated from the allow-list) match your item #9?
- **Security:** does the R-CORE token clause and R-CORE-2 close item 4, and is RS7 recorded as you accepted it?
- **Technical Lead (not blocking):** one `core-dev` task or split the guard (`cmd/server`) from the Authorizer (`authz`); the Technical Architect's view is one cohesive `authz` change plus the guard.

**Numbering assumed:** RFC-003 (highest in repo is RFC-002), erratum E2 (highest is E1 in ADR-001), amendment A2 (highest is A1).
