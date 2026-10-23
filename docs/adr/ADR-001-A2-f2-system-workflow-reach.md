---
type: adr-amendment
id: ADR-001-A2
title: Erratum E2 and amendment A2 to ADR-001-A1, closing system-workflow and task-queue reach via service certificates (F2)
status: PROPOSED
date: 2026-10-23
issue: TICK-129
rfc: docs/adr/ADR-001/rfc/RFC-003-a1-f2-system-workflow-reach.md (rev 1, archived; thread under TICK-129 comment 01a16771-d2e9-75b0-9a17-dc83bc9d3097)
supersedes: "the ADR-001-A1 sentence \"D9 doesn't change ... `nava-core`\" (for `nava-core` only; `nava-operator` stays outside D9)"
amends: "[[ADR-001-A1-temporal-server-1-29-7-workload-allow-list]] D9, D6-K; [[ADR-001-platform-blueprint]] Annex A.2"
source: TICK-122 findings (Security & Multi-Tenancy Reviewer), PR #36 skipped tests (incl. commit 92d0863), RFC-003 rounds 1 and 2
deciders: Solution Architect (author), Technical Architect (APPROVE, round 2), Security & Multi-Tenancy Reviewer (APPROVE, round 2), CTO
signs: pending (human sign-off)
---

# ADR-001 amendment A2: close F2 (erratum E2 + amendment A2)

> **Status: PROPOSED, 2026-10-23.** RFC-003 rev 1 was approved in round 2 by the Technical Architect and the Security & Multi-Tenancy Reviewer, and the CTO closed it (TICK-129). This record waits for human sign-off.

> **Invariants source.** The project description defines no invariants. As in A1, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

## Decision

One record, in two labelled parts. **Erratum E2** fixes cases where A1's wording already required the check and the D9 table and the code left it out. **Amendment A2** adds new rules.

The test for erratum vs amendment: A1's governing sentence is *"A request that names a workflow type, a task queue or a workflow id is allowed only if **every** such field is allow-listed."* If a finding is a field that names one of those, it is an erratum. If A1 is silent, or explicitly excludes the case, it is an amendment.

| Finding | Part | Rule | Severity (Security) |
|---|---|---|---|
| **F2-1** a versioned poll makes matching start `temporal-sys-worker-deployment*` workflows in `nava` | A2 | **D9b+** | Low |
| **F2-2** `nava-core` can act on any `temporal-sys-*` workflow id | A2 | **D9f** | Low, bites at scale |
| **F2-3** forged task tokens on `RespondActivityTask*`, `RecordActivityTaskHeartbeat`, `RespondWorkflowTaskFailed` | E2 | **D9d** | Low |
| **F2-3b** the frontend acts on the token's namespace, guarded only by a default | A2 | **D6-K row** | Medium, bites at scale |
| **F2-4** `nava-core` `UpdateWorkflowExecution` on `temporal-sys-worker-deployment:*` changes `nava-tq` routing | A2 | **D9f** | Low, bites at scale |
| **F2-5** `nava-worker` `GetWorkflowExecutionHistory` reads any workflow id | E2 | **D9e** | Low |
| **F2-6** `nava-worker` `ShutdownWorker.StickyTaskQueue` force-unloads the `nava-tq` root partition; heartbeat and query-token queues are unchecked | E2 | **D9g**, **D9d query clause** | Low (availability only) |
| at-scale: a `t:` prefix check can't tell tenants apart for `nava-core` | A2 | **R-CORE** | — |

None of these is a tenant-to-tenant gap. Every one needs a stolen service certificate. Urgency is normal.

## D9 additions (Annex A.2 sub-table)

| Part | Identity / API | Rule |
|---|---|---|
| **D9b+** (A2) | `nava-worker`: `PollWorkflowTaskQueue`, `PollActivityTaskQueue` | In addition to D9b: `DeploymentOptions.WorkerVersioningMode` must be `UNSPECIFIED` or `UNVERSIONED` (absent counts as unspecified), and `WorkerVersionCapabilities.UseVersioning` must be `false`. Anything else is denied with an alert. |
| **D9d** (E2) | `nava-worker`: `RespondWorkflowTaskFailed`, `RespondActivityTaskCompleted`, `RespondActivityTaskFailed`, `RespondActivityTaskCanceled`, `RecordActivityTaskHeartbeat` | Decode `TaskToken` as `token.v1.Task`. Deny, with an alert, if it does not decode or if `WorkflowId` lacks the `t:<tenant>:` prefix. This extends E1's first bullet from `RespondWorkflowTaskCompleted` to every task-token API. |
| **D9d query clause** (E2) | `nava-worker`: `RespondQueryTaskCompleted` | Decode `TaskToken` as `token.v1.QueryTask`. Deny, with an alert, if it does not decode, or if `TaskQueue` is neither a platform queue nor sticky-shaped (`stickyQueueName`). |
| **D9e** (E2) | `nava-worker`: `GetWorkflowExecutionHistory` | `Execution.WorkflowId` must carry the `t:<tenant>:` prefix. |
| **D9f** (A2) | `nava-core`: `RequestCancelWorkflowExecution`, `UpdateWorkflowExecution`, `QueryWorkflow`, `DescribeWorkflowExecution`, `GetWorkflowExecutionHistory` | The request's workflow id (`WorkflowExecution.WorkflowId` / `Execution.WorkflowId`) must carry the `t:<tenant>:` prefix. Missing or empty is denied. |
| **D9g** (E2) | `nava-worker`: `ShutdownWorker`, `RecordWorkerHeartbeat` | `StickyTaskQueue` must be empty or pass the name half of the D9b sticky rule (`stickyQueueName`, and not a platform queue). Every `WorkerHeartbeat.TaskQueue` must be a platform queue. Anything else is denied with an alert. An absent heartbeat is allowed; a present one with an empty `TaskQueue` is denied. |

A1's sentence *"D9 doesn't change … `nava-core` / `nava-operator`"* now reads *"D9 doesn't change … `nava-operator`"*. `nava-operator` stays outside D9: inspecting system workflows is its job, and the operator network restricts it (T28).

**Consolidated Annex A.2 statement after A2:**

> Every workflow id that an allow-listed `nava-relay`, `nava-core` or `nava-worker` API names, including the one inside a `token.v1.Task` task token, must carry the `t:<tenant>:` prefix. Every task queue that such an API names, including the one inside a `token.v1.QueryTask` query token and inside a `WorkerHeartbeat`, must be a platform queue or, where D9b permits a sticky queue, sticky-shaped. A request field that names a workflow id or a task queue and that no rule classifies is a CI failure (drift golden), not an allow.

**Drift golden (CI-C).** A test in `temporal-server/authz/policy_test.go`, next to `TestAllowListGolden`, holds a golden of the full field paths of the request message of **every API allow-listed for `nava-relay`, `nava-core` and `nava-worker`**. It reads them from the proto descriptors, recursing into nested messages, and generates them from the allow-list itself. Any `go.temporal.io/api` bump that adds, removes or renames a field fails CI until a policy PR classifies that field. This is what makes D9b+ and D9g allow-list shaped, not deny-lists on named fields.

**Known at api v1.53.0.** sdk-go v1.49.0 (api v1.63.5) also sends `ShutdownWorkerRequest.TaskQueue`, `TaskQueueTypes` and `WorkerInstanceKey`. The server's api v1.53.0 has only fields 1–5, so the Authorizer can't see them today. The api bump that adds them fails the golden, and that policy PR must classify `TaskQueue` as "platform queue".

## D6-K addition (A2, closes F2-3b)

| Key | Go symbol (CI-E) | Refuse start if | Shipped |
|---|---|---|---|
| `frontend.enableTokenNamespaceEnforcement` | `dynamicconfig.EnableTokenNamespaceEnforcement` | not `true` (missing also refuses) | `true` |

`checkSecurityConfig` (`temporal-server/cmd/server/check.go`) implements only must-be-false today. `guardedKey` gains a `want bool` field. This can't live in the Authorizer: namespace validation runs after the auth interceptor (`frontend/fx.go` chain), and the token's namespace wins over the request's (`common/rpc/interceptor/namespace_validator.go:226-249`).

## R-CORE (A2, stated rule)

> **R-CORE.** `nava-core` builds every workflow id it sends to Temporal as `t:<tenant>:<target_key>`. `<tenant>` is the tenant D26 resolves from the request's validated access token, through `tenant_identity_binding`. `<target_key>` is read from `workflow_ref` under RLS in that tenant's context. `nava-core` never takes a full workflow id, or the tenant segment of one, from the request path, query string, body, headers other than the bearer token, or any stored column. A record that is not visible under RLS yields the same "not found" as a missing one (I5), and no Temporal call is made.
>
> **R-CORE, opaque tokens.** `nava-core` never sends Temporal a page token (`NextPageToken`), or any other opaque Temporal token, that came from the client. It keeps the cursor server-side, or seals it with AEAD bound to the tenant and the workflow id. A token that fails to open, or that opens to a different tenant or workflow id, is treated as "not found" (I5), with no Temporal call.

**Why it is a rule.** D9f proves that a `nava-core` call names *a* tenant's workflow, never *which* tenant, because the `nava-core` certificate carries no tenant. History also reads a page by the token's branch token. The request's workflow id only picks the shard (`server@v1.29.7 service/history/api/getworkflowexecutionhistory/api.go:140-165`). So D9e and D9f bind the id in the request, not the history that comes back. R-CORE is I2 on the `nava-core` → Temporal path, and it mirrors the relay's rule in Annex A.2.

**Enforcement.** There is one helper in `nava-core` that is the only constructor of Temporal workflow ids, and no other code formats `t:`. Two acceptance cases bind the first `nava-core` change that calls Temporal:
- **R-CORE-1:** tenant A's JWT on a path naming tenant B's record returns the same 404 as a missing record, with no Temporal call.
- **R-CORE-2:** tenant A's JWT on A's own record, with a client-supplied page token minted for B's workflow or with any raw `NextPageToken`, returns the same 404, with no Temporal call made with that token.

Security reviews that change against R-CORE.

## Rejected options

| Option | Why rejected |
|---|---|
| F2-1 by a D6-K row `system.enableDeploymentVersions = false` | The Authorizer is the wall and config is defence in depth (A1 already rejected "close S1 by config only"). The key is namespace-scoped and D6-K checks only at startup (RS2). With D9b+, adopting versioning later is one policy PR, not a server-wide flag flip. |
| Both D9b+ and that D6-K row | Little extra protection, because D9b+ is checked per request and the drift golden guards it. Revisit if a second identity gets poll rights. |
| The Authorizer compares the token's `NamespaceId` with `nava`'s (instead of the F2-3b row) | It would add a namespace registry, which is new state in the auth path. The frontend already does this check; we only guarantee it stays on. |
| Deny-list `temporal-sys-*` ids or queues | Everything Temporal adds is allowed until someone notices. The prefix and platform-queue rules are allow-lists. |
| A per-tenant certificate for `nava-core` | Issuance and rotation grow with the number of tenants, and the Authorizer would gain tenant state. R-CORE plus RLS gives the same guarantee in code we already test. Revisit if `nava-core` ever runs tenant-supplied code. |
| Leave F2-2 and F2-4 as residuals until `nava-core` calls Temporal | It's a few lines on a path with no callers. Without it, the first caller would land with no wall in place. |
| Record the Technical Architect's round-1 #7 (heartbeat queue) and #8 (query token) as residuals | CTO ruling: take the rules. D9g and the D9d query clause are cheap and close both. |

## Invariants

| Invariant | Effect |
|---|---|
| I2 | **Strengthened.** R-CORE states I2 for the `nava-core` → Temporal path, including opaque tokens. D9d, D9e and D9f stop service certificates reaching non-tenant workflows. D9g and the D9d query clause stop them naming non-platform task queues. |
| I5 | R-CORE reuses "not visible = not found". |
| I6 | Unchanged. No new start or signal path. D9b+ removes one server-side start path (worker-deployment workflows). |
| D22 | Enforced more completely. |
| I1, I3, I4, I7, I8, I9 | Not touched. |

## Consequences and accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS6 | R-CORE is enforced by code and tests, not by the Authorizer. That covers both clauses: R-CORE-1 (ids built from the JWT tenant) and R-CORE-2 (no client-supplied opaque token is forwarded). The Authorizer can't see inside a history page token. | Security & Multi-Tenancy Reviewer | The first `nava-core` Temporal call; any `nava-core` API that paginates Temporal results; any `nava-core` path that runs tenant-supplied code |
| RS7 | Under D9g, one `nava-worker` pod can still unload a sibling pod's SDK-shaped sticky queue, because `stickyQueueName` checks the shape, not the owner. This is harmless: sticky dispatch falls back to the normal queue. | Security & Multi-Tenancy Reviewer | Worker certificates become per-pod or per-tenant; sticky queues gain a tenant meaning |
| RS2 (widened) | The new D6-K row is checked only at startup, like the rest. | Security & Multi-Tenancy Reviewer | Any runtime write path to dynamic config |

## Acceptance (`core-dev` task)

- **CI-I:** every PR #36 test that is skipped with F2-1, F2-2, F2-3, F2-3b, F2-4, F2-5 or F2-6 is un-skipped and passes. The round-1 tests are in `temporal-server/authz/adversarial_f2_worker_reach_test.go` (PR #36 @ `92d0863`):
  - `TestAdversarialF2WorkerHistoryTenantPrefix`: D9e.
  - `TestAdversarialF2WorkerShutdownTaskQueue`: D9g. Keep its allow-path control for the SDK-shaped `ShutdownWorker`: sticky `worker-host:<uuid>` and `""`, with the heartbeat on the platform queue.
  - `TestAdversarialF2WorkerQueryTokenTaskQueue`: the D9d query clause.
- **New tests:**
  - the D9d allow path for each token API;
  - the D9b+ allow path for the exact poll sdk-go v1.49.0 sends;
  - D9g denies a heartbeat with an empty `TaskQueue`;
  - D9d denies a query token that doesn't decode.
- **CI-E:** the new D6-K row asserted by symbol, including "missing refuses" and "false refuses".
- **CI-C:** the widened request-field golden, and `allowlist.golden.md` regenerated with D9b+, D9d (both clauses), D9e, D9f and D9g.
- **CI-D / CI-G:** the stack smoke still passes, including a graceful worker shutdown.
- **R-CORE-1 / R-CORE-2:** tracked as a dependency of the first `nava-core` change that calls Temporal. They are not part of the `core-dev` task.

**Implementation note (Technical Architect, round 2; it doesn't change the decision).** The drift-golden walk must stop at any message type it has already visited on the current path. `temporal.api.failure.v1.Failure` contains itself through `cause` (`api@v1.53.0 failure/v1/message.pb.go:669`), and it sits under `RespondActivityTaskFailed` and `RespondWorkflowTaskFailed`, so a naive walk never ends. Print a cycle as `failure.cause → (Failure)` so the golden stays stable.

## Evidence

- **Security**, on TICK-122 and in RFC-003 round 1: every F2 item is red on `main` @ `84d9bf0` with its skip removed (PR #36, including `92d0863`). The controls (what sdk-go v1.49.0 sends) pass.
- **Technical Architect**, in RFC-003 round 1: an in-process Authorizer spike ALLOWs every F2 request today. It also shows that the `ShutdownWorker` unload key equals the `nava-tq` root WORKFLOW key, and that only the poll path registers a deployment. The program and its raw output are in `docs/adr/ADR-001/spikes/technical-architect/rfc-003-round-1/`.
- The RFC is archived at `docs/adr/ADR-001/rfc/RFC-003-a1-f2-system-workflow-reach.md` (final, rev 1), with `rev-0/` and `rev-1/` beside it.
