# TICK-753: S22 and S29 adversarial isolation suite

Author: Security & Multi-Tenancy Reviewer. Covers TICK-753 (S22 P7-13) and
TICK-926 (S29 P10-11, absorbed). ADR-006 D6, D11, D12, R3; ADR-009 D2, D10,
D12, D15 (RFC-021 §10 CI-3).

Threat model: tenant A holds a valid token, or a nava-worker task token for
tenant A, or can make the relay deliver a forged or mis-kinded outbox row.
The suite looks for the request that reads, writes, starts or steers anything
of tenant B.

## Where it lives

| File | What |
|---|---|
| `internal/tenancy/isolation/s22_s29_attack_paths.go` | The isolation suite rows (`s22s29AttackPaths`), registered after the S21 rows |
| `temporal-server/authz/adversarial_s22_s29_test.go` | The Temporal halves: `schema.backfill` start-only, `field_index.remove.run` child-only |
| `internal/tenancy/isolation/registry_isolation_test.go` | `TestCursorPathsHold` now skips a pending `Check`, the way `TestRegisteredPaths` does |
| `internal/fieldindex/d7_test.go` | The new file is on the D7 allow-list (adversarial `rfv_*` SQL, as for the S21 file) |

## Day 1 and day 2

On 2026-11-26 none of the code under test had landed: TICK-750, TICK-752,
TICK-916, TICK-919 and TICK-920 were all open. The suite is written from their
contracts. A row whose code has not landed names its task in `Pending`. It is
listed in the isolation report and not run. The PR that lands the task
removes `Pending`, and the row must then pass. The rows that need Go APIs
that do not exist yet (`OpMaterialize`, `FieldIndexRemove`, the relay
dispatch, the schema.backfill workflow) have a `s22s29NotWritten` body. If
`Pending` is removed before the body is written, the row fails.

The authz tests switch on by themselves. Their deny cases run today: the
types are not allow-listed, so the commands are denied for
`workflow_type_not_allowed`. Once the types are listed as start-only or
child-only, the reason stays the same. The cases that need the type listed
(positive controls, tenant checks on a run of the type) skip until
`platformWorkflowTypes` or `childOnlyWorkflowTypes` lists it.

## Rows against the scope

| Scope item | Row | Runs today | Pending on |
|---|---|---|---|
| S22 1: tenant A cannot remove B's field or start a removal with B's `field_id` | S22 attack 1 (DB: every D11 statement as nava_app and nava_worker with B's field id; the publish 404) | yes | — |
| | S22 attack 1b (crafted `field_index.remove` outbox row through the coordinator) | no body | TICK-750 |
| | authz `TestRelayCannotStartRemovalDirectly`, `TestInterpreterCannotStartRemoval` | yes | — |
| | authz `TestCoordinatorRemovalKeepsTenantChecks` | skips | TICK-750 |
| S22 2: no orphan side rows under writes from every writer role | S22 attack 2 (DB: a removing field takes no row from any manifest records writer; D11 3–5 then pass the FK; B's removal does not move A) | yes | — |
| | S22 attack 2b (R3 race against the real workflow and barrier) | no body | TICK-750 |
| S22 3: `tenant_limits` not writable by tenant roles; operator change audited | S22 attack 3 (grants manifest + SQL probes, no-tenant read) | written | TICK-916 |
| | S22 attack 3b (audit entry with actor and reason; tenant admin 403) | no body | TICK-916 |
| S22 4: no route evaluates `records.data` JSONB for a filter | S22 attack 4 (AST scan of non-test Go: JSONB operators on `data` in string literals; `DataFilterTimeout`, `WithDataFilterTimeout`, `LegacyScanFields`) | written | TICK-752 |
| S29 CI-3 (1), (2): no-prefix start denied; child and continue-as-new denied | authz `TestWorkerCommandsCannotNameSchemaBackfill`, `TestSchemaBackfillStartedByRelayOnly`, `TestSchemaBackfillQueueIsNotASideDoor` | yes (prefix and positive cases skip) | TICK-920 |
| S29 CI-3 (3): job `t:A:…` changes zero B rows; history has no value | S29 attack 5 | no body | TICK-920 |
| | authz `TestSchemaBackfillRunKeepsTenantChecks` | skips | TICK-920 |
| S29 CI-3 (4): `OpMaterialize` forged patch, source or actor refused | S29 attack 4 | no body | TICK-916 |
| S29 CI-3 (5), (6): no `app.tenant_id` reads zero rows; B invisible, B inserts refused, all three tables, nava_app and nava_worker | S29 attack 1 (also: no value column; B's active job does not block A's admission) | written | TICK-916 |
| S29 CI-3 (7): job and outbox rows match their definition's tenant | S29 attack 2 (publish half; the rollout migration's own test is TICK-925's, in TICK-916) | written | TICK-919 |
| S29 API: 404 same bytes on B's `getEntityType`, failures, publish; plain user 404 | S29 attack 2 | written | TICK-919 |
| S29 filters: no cross-tenant leak in either D12 shape or `total`; ACL holds in the UNION ALL arm | S29 attack 3 | written | TICK-919 |
| S29 forged outbox payload with `tenant_id` rejected by the relay | S29 attack 6 | no body | TICK-920 |

## Broken variants

Each row must be red against a broken variant of the defence it proves.

| Variant | Row | Result |
|---|---|---|
| A1: `schema.backfill` added to `platformWorkflowTypes` but not to `startOnlyWorkflowTypes` | `TestWorkerCommandsCannotNameSchemaBackfill` | **Run locally: red.** The interpreter's child and continue-as-new are allowed |
| A2: `field_index.remove.run` added to `platformWorkflowTypes` but not to `childOnlyWorkflowTypes` | `TestRelayCannotStartRemovalDirectly`, `TestInterpreterCannotStartRemoval` | **Run locally: red.** The relay starts a removal directly, and an interpreter starts one as a child |
| S4: today's tree, with the legacy scan still in place | S22 attack 4 | **Run locally: red.** It flags exactly the 8 sites TICK-752 deletes: `records_list.go:296` (`data @>`), `:303` (`data ?`), `LegacyScanFields` ×2, `DataFilterTimeout` ×3, `WithDataFilterTimeout` |
| D1: `record_index_fields` DELETE policy for nava_worker `USING (true)` | S22 attack 1, "registry delete" naming B | Traced: tenant B's note row is reached (1 row, want 0) |
| D2: `rfv_text` DELETE policy `USING (true)` | S22 attack 1, "batched delete by field id alone" | Traced: tenant B's 2 rows are reached |
| D3: the D4 trigger copies fields in `removing` too (`state IN ('backfilling','ready','removing')`) | S22 attack 2 | Traced: "write added 1 side rows to tenant A's note while it is removing" |
| D4: `GRANT UPDATE ON tenant_limits TO nava_app` | S22 attack 3 | Traced: the manifest check fails before any SQL runs |
| D5: `schema_backfills` policy without the tenant predicate | S29 attack 1 | Traced: "read 1 rows of another tenant" |
| D6: the partial unique index on `(state)` instead of `(tenant_id)` | S29 attack 1, admission | Traced: tenant A's admission gets 23505 while tenant B's job runs |

D1 to D6 are traced, not run. The isolation suite runs only in CI
(`scripts/ci/isolation.sh`). Agents do not run it locally. The pending rows
run for the first time in the PR that lands their task. That PR must show the
row red against D4 to D6 and against the variants of the rows it fills in.

## Findings

None. No isolation gap was found in the code on `TICK-740/search-finish`. The
S22 attack 1 and 2 rows exercise the field-index tables and trigger that are
already there.

One design note for TICK-750 (not a gap today). D11 step 4's `NOT EXISTS` on a
field id is true for another tenant's field, because RLS hides its rows. The
removal is safe only because the step 5 registry delete is also under RLS,
and because the workflow takes its tenant from `workerguard.TenantOf`, never
from the payload. Row 1 pins both. TICK-750 should keep an explicit
`tenant_id = $1` in every D11 statement, as row 3b requires for the backfill.
