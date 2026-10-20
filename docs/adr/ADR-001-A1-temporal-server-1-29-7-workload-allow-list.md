---
type: adr-amendment
id: ADR-001-A1
title: Temporal server v1.29.7 / api v1.53.0, the D6-K startup guard and the D9 workload allow-list
status: PROPOSED
date: 2026-10-20
amends: "[[ADR-001-platform-blueprint]] §3 (pin block), R7 (revisit trigger), Annex A.2 (allow-list)"
source: RFC-002 rev 1.1 (archived), TICK-115, comment thread under the CTO's 2026-10-20 rulings
deciders: Solution Architect (author), Technical Architect (APPROVE), Security & Multi-Tenancy Reviewer (approve on the V1 rule as written, per CTO ruling), CTO
signs: nava
---

# ADR-001 amendment A1: Temporal server v1.29.7, D6-K, D9

> **Invariants source.** The project description defines no invariants. I checked this against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

## Decision

1. **Pins (replaces the two Temporal lines in ADR-001 §3):** `go.temporal.io/server` **v1.29.7** and `go.temporal.io/api` **v1.53.0** in `temporal-server/go.mod`. If `go mod tidy` resolves a higher api version, pin that and record it here. Compose runs **`temporalio/auto-setup:1.29.7`**. The platform-side SDK stays at v1.49.0 with api v1.63.5.
2. **Schema:** main **1.17 → 1.18**, visibility stays at **1.9**. Run `temporal-sql-tool update-schema` before the 1.29.7 binary starts anywhere we keep data. The binary refuses to start on 1.17.
3. **No vulncheck exception.** `make vulncheck-temporal-server` is a required `ci.yml` step that must report zero findings. If CI on 1.29.7 fails, that is a new decision for the triad, not a pre-approved exception.
4. **D6-K startup guard** (`checkSecurityConfig`). See the table below.
5. **D9 workload allow-list in the Authorizer.** See the D9 section below. It becomes a sub-table of Annex A.2.
6. **R7 revisit trigger widened:** *"or any Go vulnerability DB advisory against the pinned `go.temporal.io/server` (the CI gate fails)"*.

**Why.** v1.28.4 is **not affected** by GO-2026-5766: the vulnerable `BatchActivityWithProtobuf` path doesn't exist there, which both reviewers showed by source and test. We move because govulncheck flags the module version and the gate must be green. Urgency is normal. v1.29.7 rather than v1.29.5 because 1.29.5 reopens GO-2026-5578. D9 is needed on **every** version: Security's live spike showed a `nava-relay` start of `temporal-sys-batch-workflow` terminating another tenant's workflow inside `nava` on both 1.28.4 and 1.29.7 (S1, High).

## D6-K: guarded keys at v1.29.7

The guard refuses to start if any value of a key breaks its rule, including per-namespace constrained values. `config/dynamicconfig/development.yaml` sets every key explicitly. CI-E asserts each key **by Go symbol**, so a renamed or removed key fails to compile instead of leaving a dead check.

| Key | Go symbol (CI-E) | Refuse start if | Shipped |
|---|---|---|---|
| `system.disableStreamingAuthorizer` | `dynamicconfig.DisableStreamingAuthorizer` | `true` | `false` |
| `frontend.exposeAuthorizerErrors` | `dynamicconfig.ExposeAuthorizerErrors` | `true` | `false` |
| `system.enableCrossNamespaceCommands` | `dynamicconfig.EnableCrossNamespaceCommands` | not `false` (missing also refuses) | `false` |
| `worker.enableNamespaceBatcher` | **`dynamicconfig.EnableBatcherNamespace`** (the symbol isn't derived from the key name) | not `false` | `false` |
| `worker.enableScheduler` | `dynamicconfig.WorkerEnableScheduler` | not `false` | `false` |
| `frontend.enableSchedules` | `dynamicconfig.FrontendEnableSchedules` | not `false` | `false` |
| `frontend.enableBatcher` | `dynamicconfig.FrontendEnableBatcher` | not `false` | `false` |
| `services.frontend.rpc.httpPort` (static config) | — | non-zero | `0` |

All seven symbols and key strings are verified in `common/dynamicconfig/constants.go` at v1.29.7 (key lines 148, 153, 800, 853, 960, 2646, 2696). `authorization.NewInterceptor` at v1.29.7 ends with `(exposeAuthorizerErrors, enableCrossNamespaceCommands, disableStreamingAuthorizer)`. The test harness passes all three, in that order.

## D9: workload allow-list (Annex A.2 sub-table)

A request that names a workflow type, a task queue or a workflow id is allowed only if **every** such field is allow-listed. There is no deny-list. `System` is exempt, as it is for the API allow-list. The contents are compiled into `authz/policy.go` and printed into `allowlist.golden.md`. Adding an entry is a policy PR with Security review.

```go
var platformTaskQueues    = set("nava-tq")
var platformWorkflowTypes = set("nava.smoke.Echo")
```

| Part | Identity / API | Rule |
|---|---|---|
| **D9a** starts | `nava-relay`: `StartWorkflowExecution`, `SignalWithStartWorkflowExecution` | `WorkflowType.Name` ∈ types; `TaskQueue.Name` ∈ queues; `TaskQueue.Kind` NORMAL or UNSPECIFIED; `WorkflowId` has the `t:<tenant>:` prefix. |
| **D9a** signals | `nava-relay`: `SignalWorkflowExecution` | `WorkflowId` has the `t:<tenant>:` prefix. |
| **D9b** polls | `nava-worker`: `PollWorkflowTaskQueue`, `PollActivityTaskQueue` | NORMAL or UNSPECIFIED: `TaskQueue.Name` ∈ queues. STICKY: a STICKY `Name` must match `^[^/]+:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` and must not be in `platformTaskQueues`; `NormalName` must be in `platformTaskQueues`, and an empty `NormalName` is denied. STICKY on `PollActivityTaskQueue` is denied. |
| **D9c** commands | `nava-worker`: `RespondWorkflowTaskCompleted` (`checkCommand`) | `ScheduleActivityTask`: `TaskQueue` empty (inherits) or ∈ queues. `StartChildWorkflowExecution` / `ContinueAsNewWorkflowExecution`: `WorkflowType` ∈ types, an empty `WorkflowType` is denied (not inherited), `TaskQueue` empty or ∈ queues. `StickyAttributes` follows the same rule as D9b STICKY. |

The sticky rule exists because, at v1.29.7, sticky and normal partitions of the same name share a key (`common/tqid/task_queue_id.go`). The regex matches the SDK's own sticky name (`sdk-go@v1.49.0` `getWorkerTaskQueue` = `host:uuid`), and the SDK always sets `NormalName`.

D9 doesn't change the API allow-list, the namespace rule, erratum E1's header and search-attribute checks, or `nava-core` / `nava-operator`.

## Rejected options

| Option | Why rejected |
|---|---|
| Stay on 1.28.4 with a time-boxed vulncheck exception (D) | It's a false positive on 1.28.4, so there is no live risk to time-box, and a fallback exception is one more moving part. Security's dissent was withdrawn. |
| 1.29.5 / 1.29.6 | 1.29.5 reopens GO-2026-5578. 1.29.6 has no advantage over 1.29.7. |
| 1.30.8 (B) | Visibility schema 1.9 → 1.13, no auto-setup image, 16 new APIs to classify. It's the next step under RS1. |
| 1.31.4 (C) | More Authorizer surface (`CallTarget.Principal`) to re-verify. |
| Backport onto 1.28.4 (E) | We'd own a fork, and the gate still flags the version. |
| Rely on topology (F) | The gate stays red, and S1 shows the topology isn't a wall. |
| Close S1 by config only | Other per-namespace components still serve `temporal-sys-per-ns-tq`. Kept only as D6-K defence in depth. |
| Deny-list `temporal-sys-*` (including the "no `temporal-sys-` prefix" sticky fallback) | Every name Temporal adds is allowed until someone notices. It already misses `default-worker-tq` and `/_sys/` partitions. |

## Invariants

| Invariant | Effect |
|---|---|
| I2 | Upheld **only with D9**. The upgrade alone changes nothing; S1 breaks I2 on every version until D9 lands. |
| I6 | Not weakened. Schema-before-binary is enforced by the binary. A sequential 1.28.4 → 1.29.7 swap kept an in-flight execution. |
| I8 | Unchanged. |
| D22 | Enforced more completely: API allow-list plus workload allow-list. |
| I1, I3, I4, I5, I7, I9 | Not touched. |

## Consequences and accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS1 | 1.29's support horizon is unknown | Technical Lead | Unfixed advisory, Temporal announces 1.29 end of support, or M2 planning |
| RS2 | D6-K checks only at startup | Security & Multi-Tenancy Reviewer | Any runtime write path to dynamic config |
| RS3 | SDK v1.49.0 against server 1.29.7 is proven only by CI-D | Technical Architect | CI-D failure |
| RS4 | Turning off system features could break an internal workflow | Technical Architect | CI-G failure: drop that one D6-K key and record why. Never widen D9 to compensate. |
| RS5 | Every new workflow type or task queue needs a policy PR and an image rebuild | Technical Lead | More than about one change a week |

## Acceptance (T27d PR)

CI-A (vulncheck, zero findings), CI-B (pins), CI-C (drift, five new methods in `deniedByDefault`, golden table with the D9 sub-table), CI-D (CI-6 suite on 1.29.7), CI-E (D6-K by symbol), CI-F (cross-namespace denied either way), CI-G (stack smoke on 1.29.7 with D6-K values), CI-H (live replay of S1, plus one live sticky case), CI-I (`37b7bf4` and `f39e0a3` red → green, plus empty-`WorkflowType` and STICKY-activity-poll cases). The full definitions are in RFC-002 §8.

## Follow-ups (not part of A1)

- **F1:** mixed-version 1.28/1.29 node smoke before any multi-node deployment (Technical Architect).
- **F2** (Security & Multi-Tenancy Reviewer): scheduler and worker-deployment workflows on the shared queue; `t:` prefix on `nava-core` workflow-id calls; `RespondActivityTask*` task-token prefix; `nava-core` `UpdateWorkflowExecution` reaching `temporal-sys-worker-deployment:*`.
- **E4:** whether any data on 1.28.4 must be kept (Technical Lead, at planning).
