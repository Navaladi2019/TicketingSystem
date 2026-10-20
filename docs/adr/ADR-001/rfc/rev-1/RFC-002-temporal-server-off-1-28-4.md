---
type: rfc
id: RFC-002
title: Move go.temporal.io/server off v1.28.4 (GO-2026-5766), and close the reserved-system-workflow path (D9). ADR-001 §3 amendment
status: REVISED (rev 1, the triad's one revision)
date: 2026-10-20
issue: TICK-115
amends: "[[ADR-001-platform-blueprint]] §3 pin block, R7 revisit wording, Annex A.2 (workload allow-list)"
author: Solution Architect
supersedes: RFC-002 rev 0 (2026-10-19)
---

# RFC-002 (rev 1): Move the custom Temporal server off v1.28.4, and close S1

> **Invariants source.** The project description is still empty, so it defines no invariants. I checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, that list is missing and these checks are incomplete against it.

## Changelog from rev 0

| Item | Source | Change |
|---|---|---|
| S1 | Security (blocking), CTO ruling 1 | **New D9.** A deny-by-default workload allow-list in the Authorizer, covering workflow type, task queue and workflow id. It applies to relay starts, worker polls **and worker commands**. The commands part goes beyond what Security specified; see D9c. The four system-workflow features are off and guarded. New tests CI-I, plus CI-H replaying the live spike. |
| D6 / CI-E / D4 | TA (required), CTO ruling 2 | The key set is corrected and checked against **v1.29.7** source. **One deviation:** `system.disableStreamingAuthorizer` **stays guarded**. It was removed in 1.29.0–1.29.5, but **re-added in 1.29.6 and is live in 1.29.7**, our target. See D6 for the evidence. D4 now lists all three trailing parameters in order. |
| §1 / §4 / §6 | CTO ruling 3 | v1.28.4 is **not affected** by GO-2026-5766. We move because of the vulncheck gate, at normal urgency. I2 is upheld **only with D9**. §4 records the settled facts. |
| Option D | CTO ruling 4 | **Removed.** If the 1.29.7 CI run fails, that comes back as a new decision. Security's dissent is recorded in §5. |
| Follow-ups | CTO | F1, mixed-version rolling upgrade, and F2, the untested scheduler and deployment workflows, go to §11. Neither is in scope. |

## 1. Summary

- **Pin:** amend ADR-001 §3 to **`go.temporal.io/server` v1.29.7** and **`go.temporal.io/api` v1.53.0**, with compose on **`temporalio/auto-setup:1.29.7`**. Not 1.29.5, because 1.29.5 reopens GO-2026-5578.
- **Why move:** **v1.28.4 is not affected by GO-2026-5766.** Both reviewers showed the vulnerable `BatchActivityWithProtobuf` path does not exist in 1.28.4. We move because govulncheck flags the module *version* and the gate must report zero findings (D3). **Urgency: normal.**
- **Why the PR is bigger than a version bump (D9):** Security found an isolation gap that exists on **every** version, 1.28.4 and 1.29.7 alike. A `nava-relay` start of `temporal-sys-batch-workflow` on `temporal-sys-per-ns-tq` runs as `temporal-system` and terminated another tenant's workflow inside `nava`. D9 closes it in our Authorizer, the lowest layer we own, and turns off the system features M1 doesn't use.
- **No vulncheck exception, no fallback.** If CI on 1.29.7 fails, it returns to the triad as a new decision.
- **What nava accepts:** schema 1.17 → 1.18 on the main store; a workload allow-list that new workflow types and task queues must be added to, with Security review; and residual RS1, an unknown support horizon for 1.29.

## 2. Context

| Fact | Source | Status |
|---|---|---|
| GO-2026-5766 published range `< 1.29.5`. The bug is in `BatchActivityWithProtobuf`, introduced in 1.29.0 and fixed by `90738c6200` in 1.29.5. | GHSA, TA and Security source reads, the TA's upstream regression test (FAIL on 1.29.4, PASS on 1.29.5) | **Settled.** v1.28.4 has only the legacy `BatchActivity`, which checks the namespace by name. The TA test passes on 1.28.4. |
| GO-2026-5578 (streaming authorizer bypass): affected `< 1.28.4`, `1.29.0 – < 1.29.6`, `1.30.0 – < 1.30.4`. | `vuln.go.dev` | Fetched |
| `system.disableStreamingAuthorizer`: present in 1.28.4, **absent in 1.29.4 and 1.29.5**, **present again in 1.29.6, 1.29.7, 1.30.8 and 1.31.4**. In 1.29.7, `interceptor.go:183` reads `bypassAuth := a.disableStreamingAuthorizer()`. | `raw.githubusercontent.com/temporalio/temporal/<tag>/common/{dynamicconfig/constants.go,authorization/interceptor.go}`, fetched 2026-10-20 | **Verified by me.** This is where the TA's critique (checked at 1.29.4/1.29.5) and this revision differ. |
| A relay-started `temporal-sys-batch-workflow` terminates a `t:B:*` workflow in `nava` (`SuccessCount:1`) on stock 1.28.4 and 1.29.7. | Security live spike, round 1 | **Settled (S1, High).** |
| `temporalio/auto-setup` stops at 1.29.7. | Docker Hub | Fetched |

## 3. Decisions

| # | Decision | Why |
|---|---|---|
| **D1** | `temporal-server/go.mod`: `go.temporal.io/server v1.29.7`, `go.temporal.io/api v1.53.0`. If `go mod tidy` resolves a higher api version, pin that and record it. Toolchain stays as T27d set it. `gomod_test.go` asserts the pins. | Clears the GO-2026-5766 gate finding and keeps GO-2026-5578 fixed. |
| **D2** | Compose `temporal`: `auto-setup:1.29.7`. Main schema **1.17 → 1.18**, visibility stays at **1.9**. Anywhere we must keep data, run `temporal-sql-tool update-schema` **before** the 1.29.7 binary starts. The binary refuses to start otherwise: *"Expected version: 1.18 cannot be greater than Actual version: 1.17"* (confirmed by TA and Security). | One version in both places. The ordering is enforced mechanically. |
| **D3** | **No vulncheck exception.** `make vulncheck-temporal-server` becomes a required `ci.yml` step that must report zero findings. | The gate is the reason we move. An exception would only defer that reason. |
| **D4** | **Authorizer and ClaimMapper interfaces are unchanged** between 1.28.4 and 1.29.7. `authorization.NewInterceptor`'s trailing dynamic-config parameters change from two in 1.28.4, `(enableCrossNamespaceCommands, disableStreamingAuthorizer)`, to **three** in 1.29.7, **`(exposeAuthorizerErrors, enableCrossNamespaceCommands, disableStreamingAuthorizer)`**. `exposeAuthorizerErrors` is new and goes first; the other two keep their meaning. `newTestInterceptor` in `authz/adversarial_test.go` passes all three, in that order, each returning `false`. Only the test harness is affected. | The rev 0 wording was right for 1.29.7 but didn't spell this out. The TA's "swap" reading is right for 1.29.4/1.29.5 only. Naming all three stops an engineer from wiring them in the wrong order. |
| **D5** | **Allow-list drift is handled by classifying, not by widening.** Five new `WorkflowService` methods (`DescribeWorker`, `FetchWorkerConfig`, `SetWorkerDeploymentManager`, `UpdateTaskQueueConfig`, `UpdateWorkerConfig`) go to `deniedByDefault`, and the golden table is regenerated. The command enum is unchanged. A method moves to `nava-worker` only if CI-D proves SDK v1.49.0 needs it, by name, with Security sign-off. | D22 survives the upgrade. |
| **D6** | **Startup guard (`checkTLS` is renamed `checkSecurityConfig`, and the existing `checkConfig` dynamic-config check is folded in).** It refuses to start, with a named error, if **any** value of a key, including per-namespace constrained values, breaks a rule below. `config/dynamicconfig/development.yaml` sets every key explicitly. Today it is `{}`. See table D6-K. | It keeps every authorizer-off switch, info-leak switch and system-workflow feature from being enabled by config drift. |
| **D7** | **The platform-side SDK matrix is unchanged:** SDK v1.49.0 + api v1.63.5 in the root module. *Server 1.29.7 ↔ SDK v1.49.0* is proven by CI-D. | Proven by test, not assumed. |
| **D8** | **ADR-001 amendment A1**, not a new ADR. It replaces the two Temporal pin lines in §3 and widens R7's revisit trigger: *"or any Go vulnerability DB advisory against the pinned `go.temporal.io/server` (the CI gate fails)"*. It **also adds the D9 workload allow-list to Annex A.2** as a sub-table under the API allow-list. `docs/pins.md`, `docs/temporal-server.md` and the golden-table header move to match. nava signs A1. | D9 refines how D22 is enforced. It doesn't change any decision or invariant, so it is still an amendment. If a reviewer reads D9 as a new decision, I'll split it into ADR-002. |
| **D9** | **The workload allow-list in the Authorizer** (closes S1). See §3.1. | S1: platform identities could hand work to Temporal's own system workers, which run as `temporal-system` and bypass every tenant check. |

### D6-K: guarded keys at v1.29.7

| Key | Rule (refuse start if…) | Default at 1.29.7 | Shipped value | Why |
|---|---|---|---|---|
| `system.disableStreamingAuthorizer` | `true` | `false` | `false` | Live at `interceptor.go:183`. If `true`, the streaming endpoint skips the Authorizer, which is the GO-2026-5578 class. **Kept, unlike the TA's proposal:** the key exists at our target version. |
| `frontend.exposeAuthorizerErrors` | `true` | `false` | `false` | It would return Authorizer internals to callers. |
| `system.enableCrossNamespaceCommands` | not `false` (missing also refuses) | `true` | `false` | The GO-2026-5766 bug class. We have one tenant namespace. |
| `worker.enableNamespaceBatcher` | not `false` | `true` | `false` | S1 defence in depth. With it off, a forged batch fails with `WorkflowTaskFailed` (Security, live, 1.29.7). |
| `worker.enableScheduler` | not `false` | `true` | `false` | Same primitive through `temporal-sys-scheduler-workflow`. M1 has no schedules. |
| `frontend.enableSchedules` | not `false` | `true` | `false` | Closes the schedule API surface. |
| `frontend.enableBatcher` | not `false` | `true` | `false` | Closes the batch API surface. `StartBatchOperation` is already in `deniedByDefault`. |
| `services.frontend.rpc.httpPort` (static config) | non-zero | — | `0` | The HTTP API stays off until it is proven to go through the same mTLS identity and interceptor. |

All seven dynamic-config keys exist with these names and defaults in `common/dynamicconfig/constants.go` at v1.29.7 (lines 148, 153, 800, 853, 960, 2646, 2696), which I verified. The four feature flags have the same names in 1.28.4.

### 3.1 D9: the workload allow-list

**Rule.** A request that names a workflow type, a task queue or a workflow id is allowed only if **every** such field matches a value in the allow-list. Unlisted means denied. There is no deny-list. `temporal-sys-*`, `default-worker-tq`, `/_sys/` and every name Temporal adds later are denied because they aren't listed, not because they're named. `System` is exempt, as it is for the API allow-list.

**Contents** live in `authz/policy.go`, compiled in, next to `allowList`, and printed into `allowlist.golden.md`:

```go
// platformTaskQueues and platformWorkflowTypes are the only task queues and
// workflow types a platform identity may name (D9). Adding one is a policy
// change: regenerate the golden table; Security reviews.
var platformTaskQueues    = set("nava-tq")
var platformWorkflowTypes = set("nava.smoke.Echo")
```

M1 has no product workflows yet. `nava.smoke.Echo` is the stack smoke's workflow, which gets registered under that name. Each task that adds a product workflow type or task queue adds it here, in its own PR, with Security sign-off. This is the same process as D5. I chose compiled-in values over a config file because the allow-list is a security boundary: it must match the binary that enforces it, and a code review is the approval step we want.

| Part | Identity / API | Checks (all must pass) | Deny reason (log + metric tag) |
|---|---|---|---|
| **D9a**: starts | `nava-relay`: `StartWorkflowExecution`, `SignalWithStartWorkflowExecution` | `WorkflowType.Name` ∈ types. `TaskQueue.Name` ∈ queues. `TaskQueue.Kind` is NORMAL or UNSPECIFIED. `WorkflowId` has the `t:<tenant>:` prefix (`tenantOf`). | `workload_type`, `workload_queue`, `workflow_id_no_tenant` |
| **D9a**: signals | `nava-relay`: `SignalWorkflowExecution` | `WorkflowExecution.WorkflowId` has the `t:<tenant>:` prefix. Temporal's own workflow ids, such as `temporal-sys-scheduler:*`, never have it. | `workflow_id_no_tenant` |
| **D9b**: polls | `nava-worker`: `PollWorkflowTaskQueue`, `PollActivityTaskQueue` | NORMAL or UNSPECIFIED: `TaskQueue.Name` ∈ queues. STICKY (workflow polls only): `TaskQueue.NormalName` ∈ queues (see V1). | `workload_queue` |
| **D9c**: commands *(beyond Security's spec)* | `nava-worker`: `RespondWorkflowTaskCompleted`, inside the existing `checkCommand` | `ScheduleActivityTask`: `TaskQueue` is empty (inherits) or its name ∈ queues. `StartChildWorkflowExecution` and `ContinueAsNewWorkflowExecution`: `WorkflowType` ∈ types, `TaskQueue` empty or ∈ queues. `StickyAttributes.WorkerTaskQueue`, if set: kind STICKY (see V1). | `command_workload_type`, `command_workload_queue` |

**Why D9c.** With only D9a and D9b, a stolen `nava-worker` certificate could still reach the system worker by scheduling an activity on `temporal-sys-per-ns-tq`, or by starting a child or continuing-as-new as `temporal-sys-batch-workflow`. `checkCommand` checks headers and target ids today, not types or queues (`authz/commands.go:checkCommand`). It is the same S1 primitive through the worker's command channel, so it belongs in the same fix.

**V1 (verify-by-spike, inside CI-I):** can a STICKY-kind poll or sticky attribute whose `Name` is a reserved queue (for example `temporal-sys-per-ns-tq`) receive or redirect NORMAL-kind tasks? I believe matching keys sticky and normal partitions separately, but I haven't verified it. If the CI-I case shows it can, sticky names must also be absent from `platformTaskQueues` and must not carry a `temporal-sys-` prefix. The sticky name is SDK-generated (`host:uuid`), so it can't be allow-listed exactly. That would be the one place with a prefix check, and only on a field the platform doesn't choose.

**What D9 does not change:** the API allow-list, the namespace rule, erratum E1's header and search-attribute checks, and `nava-core` / `nava-operator`. Core's workflow-id calls reaching system workflows is follow-up F2.

## 4. Exposure (settled in critique round 1)

| # | Question | Answer | Evidence |
|---|---|---|---|
| **E1** | Is the GO-2026-5766 code in 1.28.4? | **No.** | Both reviewers' source reads. The TA's tests at 1.28.4 PASS, 1.29.4 FAIL, 1.29.5 PASS. |
| **E2** | Does our build meet the precondition? | **Yes, on an affected binary.** `systemWorker` is `temporal-system`, which is allowed everywhere. Never land on 1.29.0–1.29.4. | `config/development.yaml`, `authz/policy.go` |
| **E3** | Can a platform identity reach another tenant? | **Across namespaces: no.** There is one tenant namespace and `StartBatchOperation` is denied. **Inside `nava`: yes, S1, on every version.** D9 fixes that. | TA source read. Security live spike plus red test `37b7bf4`. |
| **E4** | Is any data on 1.28.4 that we must keep? | **Open, Technical Lead.** Doesn't block. The stock image never had our Authorizer. | — |

## 5. Options considered

| Option | Verdict | Why |
|---|---|---|
| **A. server v1.29.7 + api v1.53.0, plus D9** | **Recommended** | The smallest move that clears the gate and keeps 5578 fixed. D9 is needed whichever version we pick. |
| A'. 1.29.5 / 1.29.6 | Rejected | 1.29.5 reopens 5578. 1.29.6 has no advantage over 1.29.7. |
| B. 1.30.8 | Rejected for now; it's the next step under RS1 | Visibility schema 1.9 → 1.13, no auto-setup image, 16 new APIs to classify. |
| C. 1.31.4 | Rejected | `CallTarget.Principal` and more Authorizer changes to re-verify. |
| E. Backport onto 1.28.4 | Rejected | We'd own a fork, and the gate still flags the version. |
| F. Rely on topology alone | Rejected | Leaves the gate red. S1 shows the topology isn't a wall. |
| S1 via config only (turn off batcher, scheduler, schedules) | Rejected as the **only** fix, kept as D6 defence in depth | Other per-namespace components still serve `temporal-sys-per-ns-tq`. This is Security's live finding. |
| S1 via a deny-list of `temporal-sys-*` | Rejected | Every name Temporal adds is allowed until someone notices. That fails the D22 deny-by-default principle. |
| ~~D. Time-boxed exception~~ | **Removed (CTO ruling)** | It is a false positive on 1.28.4, so there is no live risk to time-box. If CI fails, that's a new decision. *Dissent: the Security & Multi-Tenancy Reviewer would keep D as a CI-D fallback.* |

## 6. Invariants touched

| Invariant | Effect |
|---|---|
| **I2** (tenant only from verified source; Temporal is a path like any other) | **Upheld only with D9.** The upgrade alone doesn't change I2's exposure. S1 breaks I2 on every version until D9 lands. D9 restores it, and D6 adds config-level walls. I make no "strengthened" claim. |
| **I6** (deterministic, versioned durable path) | Not weakened. Schema-before-binary is enforced. A sequential 1.28.4 → 1.29.7 swap kept an in-flight execution (Security, live). Mixed-version nodes are F1. |
| **I8** (tenant code never reaches Temporal) | Unchanged. |
| D22 | **Enforced more completely.** It was the API allow-list only; it is now the API allow-list plus the workload allow-list (D8 / Annex A.2). |
| R7 | Trigger widened (D8). D9 adds policy code but no fork, so the module stays minimal. |
| I1, I3, I4, I5, I7, I9 | Not touched. |

## 7. Risks and accepted residuals

| # | Risk / residual | Mitigation | Owner | Revisit trigger |
|---|---|---|---|---|
| **RS1** | We don't know how long 1.29 will be supported. | The D3 gate. Option B is pre-analysed. | Technical Lead | An unfixed advisory, Temporal announcing 1.29 end of support, or M2 planning |
| **RS2** | D6 checks only at startup, and dynamic config reloads at runtime. | The file ships in the image, read-only. | Security & Multi-Tenancy Reviewer | Any runtime write path to dynamic config |
| **RS3** | SDK v1.49.0 against server 1.29.7 is unverified. | CI-D. D5 governs any added API. | Technical Architect | CI-D failure |
| **RS4** | Turning off the cross-namespace commands, batcher, scheduler and schedules could break an internal system workflow. **Partly verified:** with the batcher off, system workflows started and `nava` stayed usable (Security, live). | CI-G stack smoke. | Technical Architect | CI-G failure: drop that one key from D6 and record why. **Never widen the D9 allow-list to compensate.** |
| **RS5 (new)** | Every new workflow type or task queue needs a policy PR and an image rebuild. | It's the same process as D5 and is intentional. The deny reasons `workload_type` / `workload_queue` show up in logs on the first failing run. | Technical Lead | If more than about 1 change per week happens in practice, revisit a signed config file. |

## 8. Test / acceptance spec

| # | Test | Catches |
|---|---|---|
| **CI-A** | `make vulncheck-temporal-server`: a required `ci.yml` step, zero findings | D3 |
| **CI-B** | `gomod_test.go`: server v1.29.7 and api v1.53.0 (or the tidy-resolved version), no `replace` | D1 |
| **CI-C** | `policy_test.go` drift checks at api v1.53.0. The five new methods are in `deniedByDefault`. The golden table is regenerated **and includes the D9 sub-table**. | D5, D8 |
| **CI-D** | The full CI-6 suite (Annex B) against the 1.29.7 build | D4, D7, RS3 |
| **CI-E** | Startup-guard table test, re-anchored to **D6-K**: one sub-case per row (a forbidden value refuses start with the named error, and a missing value refuses for the "not `false`" rows). One sub-case per constrained (per-namespace) value. The shipped `development.yaml` starts. Every D6-K key name is asserted against `dynamicconfig` constants by symbol (`dynamicconfig.DisableStreamingAuthorizer.Key()` and so on), so a renamed or removed key fails to compile instead of the clause silently going dead. | D6, and the "dead clause" failure the TA raised |
| **CI-F** | A cross-namespace child, signal or cancel is denied with `enableCrossNamespaceCommands` both `true` and `false` | The Authorizer doesn't depend on the server flag |
| **CI-G** | Stack smoke on `auto-setup:1.29.7` (schema 1.18) **with the shipped D6-K values**, including `nava` registration by `temporal-system`. The smoke workflow is registered as `nava.smoke.Echo` on `nava-tq`; today it's `Echo` on `t28-<nanos>`. | D2, RS4, D9 contents |
| **CI-H** | **Replays Security's live spike against our 1.29.7 image:** a `nava-relay` `StartWorkflowExecution` of `temporal-sys-batch-workflow` on `temporal-sys-per-ns-tq` with `BatchType: terminate` aimed at `t:B:victim` gets `PermissionDenied`, and `t:B:victim` is still running afterwards. Same for `SignalWithStart`. | S1 end to end |
| **CI-I** | `authz/adversarial_reserved_test.go` from `37b7bf4`: the 14 start/signal-with-start subtests plus the worker-poll test. **One edit:** the control case reads its type and queue from `platformWorkflowTypes` / `platformTaskQueues` instead of the literal `TicketWorkflow` / `nava-tq`. **Added for D9c:** `ScheduleActivityTask` on `temporal-sys-per-ns-tq`, and `StartChild` / `ContinueAsNew` with type `temporal-sys-batch-workflow`, are each denied. **Added for D9a:** a relay `Signal` to `temporal-sys-scheduler:x` is denied. **V1:** a STICKY poll named `temporal-sys-per-ns-tq` (with `NormalName: nava-tq`) receives no NORMAL task. This runs as part of CI-H against the live server. | D9a/b/c, V1 |

**Done means** CI-A to CI-I are green in the T27d PR, `37b7bf4`'s test went from red to green on that branch, and Security re-checks D9.

## 9. Sequencing

1. Round-1 reviewers re-check this revision: Security for D9, the TA for D4/D6/CI-E. The TA should re-check D6-K at **v1.29.7** in particular.
2. **T27d PR** (Principal Software Engineer): `6c16dc0` + D1, D2, D4, D5, D6, **D9** + docs + CI-A to CI-I, as one PR. Cherry-pick `37b7bf4` first so the red test sits on the branch before the fix.
3. ADR-001 amendment A1 (D8, including the Annex A.2 workload sub-table) goes to nava. T27d merges after A1 is accepted.

## 10. Open questions

- **TA:** do you agree that `system.disableStreamingAuthorizer` exists at v1.29.7 (`constants.go:153`, `interceptor.go:93/108/183`)? If you have contrary evidence at **that** tag, D6-K drops the row. Otherwise it stays.
- **Security:** is D9c (the command-channel extension) correct, and is V1 the right way to handle sticky queues?
- **Technical Lead:** E4 (doesn't block). Will product tasks accept RS5's add-it-to-the-allow-list step?
- **nava:** at sign-off, accept A1 (pins, the R7 trigger, the Annex A.2 workload sub-table), RS1 and RS5.

## 11. Follow-ups (not in this revision)

| # | Item | Owner |
|---|---|---|
| **F1** | A mixed-version 1.28/1.29 node smoke during a rolling upgrade. M1 is single-node. | Technical Architect, before any multi-node deployment |
| **F2** | Scheduler and worker-deployment system workflows share `temporal-sys-per-ns-tq` and weren't exercised. D9 covers the start path structurally. Still to check: whether `nava-core`'s workflow-id calls (`RequestCancel`, `Update`, `Query`, `Describe`) should also require the `t:` prefix. | Security & Multi-Tenancy Reviewer |

## Verification notes (2026-10-20)

- Fetched `common/dynamicconfig/constants.go` and `common/authorization/interceptor.go` at v1.28.4, v1.29.4, v1.29.5, v1.29.6 and v1.29.7 from `raw.githubusercontent.com/temporalio/temporal/<tag>`, plus the constants at v1.30.8 and v1.31.4. `"system.disableStreamingAuthorizer"`: present at 1.28.4, 1.29.6, 1.29.7, 1.30.8 and 1.31.4; absent at 1.29.4 and 1.29.5. `NewInterceptor` at 1.29.7 ends `exposeAuthorizerErrors, enableCrossNamespaceCommands, disableStreamingAuthorizer`.
- Read `temporal-server/authz/{policy,authorizer,commands}.go` and `cmd/server/check.go` on `main`, and `adversarial_reserved_test.go` at `37b7bf4`. `checkCommand` checks no task queue or workflow type, which is why D9c exists.
- I haven't built or run anything against 1.29.7, and I didn't re-run either reviewer's spike. CI-A to CI-I are the proof, in the T27d PR.
