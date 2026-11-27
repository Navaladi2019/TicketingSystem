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

On 2026-11-26 (day 1) none of the code under test had landed. The suite was
written from the contracts of TICK-750, TICK-752, TICK-916, TICK-919 and
TICK-920, and each row whose code had not landed named its task in `Pending`.

On 2026-11-27 (day 2, PR #943) TICK-916 and TICK-920 had merged, and their
rows run: S22 3 and 3b, and S29 1, 4, 5 and 6. Their bodies are in
`s29_attack_isolation_test.go` (integration tag), because they drive the
records pipeline, the real `schema.backfill` workflow on the Temporal SDK test
environment, and the relay. S22 3b drives the real operator limits route: the
suite's API handler is now built with `WithOperatorLimits`. The authz suite
now fails, rather than skips, if `schema.backfill` is not a start-only
platform type.

Still `Pending`:

| Rows | Pending on |
|---|---|
| S22 1b, S22 2b, and the `field_index.remove.run` authz cases | TICK-750 |
| S22 4 | TICK-752 |
| S29 2 and 3 | TICK-919 |

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

Each row must be red against a broken variant of the defence it proves. The
DB variants were run in GitHub CI, each on a throwaway `SPIKE, NEVER MERGE`
branch (since deleted) with one `gh workflow run ci`. A variant that weakens a
policy also needs an entry in the CI-1 allow-list (`internal/db/catalog`), or
the catalogue gate refuses the migration before the suite runs.

| Variant | Row | Result |
|---|---|---|
| A1: `schema.backfill` platform-listed, not start-only | `TestWorkerCommandsCannotNameSchemaBackfill` | Local run: red |
| A2: `field_index.remove.run` platform-listed, not child-only | `TestRelayCannotStartRemovalDirectly`, `TestInterpreterCannotStartRemoval` | Local run: red |
| S4: the tree before TICK-752 | S22 attack 4 | Local run: red on exactly the 8 legacy-scan sites |
| D1: `record_index_fields` worker DELETE `USING (true)` | none of the S22 rows | The CI-1 gate refuses it (run 45840586980). Past the gate, the generic per-table probe catches it (run 45844195746). S22 attack 1 stays green, because PostgreSQL also applies the SELECT policy to a `DELETE … WHERE`. |
| D1': D1, plus worker SELECT `USING (true)` | S22 attack 1 | Red (run 45845475310): tenant A reaches tenant B's registry row, and only the D3 FK stops the delete |
| D2: `rfv_text` worker DELETE `USING (true)` | none of the S22 rows | As D1: the gate (run 45840594226), then the generic probe (run 45844240150) |
| D2': D2, plus worker SELECT `USING (true)` | S22 attack 1 | Red (run 45845504565): the batched delete reaches tenant B's 2 side rows |
| D3: the D4 trigger also copies fields in `removing` | S22 attack 2 | Red, on this row only (run 45840602023) |
| V4: `nava_app` granted UPDATE on `tenant_limits` | S22 attack 3 | The CI-A3 audited-writers gate refuses it (run 45913136815). With `nava_app` also listed as a writer, S22 attack 3 is red (run 45918771787) |
| V5: `schema_backfills_select_app USING (true)` | S29 attack 1 | Red (run 45913188778), together with the generic probe and TICK-916's own row |
| V6: the one-active index on `(state)`, not `(tenant_id)` | S29 attack 5 | Red (run 45913295758): tenant A's own job cannot run while tenant B's runs. S29 attack 1 stays green, because B's seeded job is `running` and A's goes to `filling`, a different key |
| V7: the limits audit actor is `system:`, not `operator:` | S22 attack 3b | Red (run 45913351311) |
| V8: `OpMaterialize` accepts a patch | S29 attack 4 | Red (run 45913406847): "a non-empty patch = written" |
| V9: the job's statements drop `tenant_id = $1`, and the worker SELECT and UPDATE policies on `schema_backfills` are open | generic probe | The generic probe is red (run 45913464508). The suite step then failed without writing its report, so S29 attack 5's own result under V9 is not confirmed |
| V10: the relay skips `ValidateKindPayload` | S29 attack 6 | Red (run 45913509592): the forged row is dispatched |

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
