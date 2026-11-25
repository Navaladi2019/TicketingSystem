# S31 org units: isolation review and attack tests (P11-19)

TICK-952, story TICK-946 (S31), ADR-010 I1, I3, D7, D8. The attacker holds a valid token for tenant A, or controls tenant A's Keycloak directory (its groups and the `nava_primary_org_unit` user attribute). They try to see or write tenant B's org-unit rows, to give a record of A a unit of B, or to widen `under` access after a directory change. Every attack below is a runnable `//go:build integration` test, and the passing ones stay as regression guards.

**Verdict: no exploitable isolation gap.** Every guard holds at the database (FORCE RLS, grants, foreign keys that carry `tenant_id`). The `under` leaf's closure reads (TICK-1034; the set-S query before it) and the primary-unit lookup filter by tenant on top of RLS. The reviewer's verdict on TICK-947, TICK-948 and TICK-949 is in the PR description of this change.

## Paths to data

| Path | Tenant comes from | Lowest-layer guard |
| --- | --- | --- |
| `org_unit_members` reads (`nava_app`, `nava_worker`, `nava_groupsync`) | `app.tenant_id`, set by `InTenantTx` from the token or the sync's tenant | FORCE RLS on SELECT. Only `nava_groupsync` may write, through RLS `WITH CHECK` |
| `org_units` writes (`nava_groupsync`) | the sync's tenant | FORCE RLS. There is no DELETE grant or policy. The parent FK is `(tenant_id, parent_id)` |
| `records.org_unit_id` on create (`pipeline.resolveUnit`, `store.PrimaryUnit`) | the write transaction's tenant | `primaryUnitSQL` filters `m.tenant_id`, RLS, and the FK `(tenant_id, org_unit_id)` |
| `records.org_unit_id` from a body (API, workflow `update_record`) | n/a | `refuseOrgUnit`, `field_not_writable` on create and update, for every source |
| The `under` leaf (`authz.ResolveUnder`, `resolveSQL`; `compiler.under`, `RegionDriver`), TICK-1034 | `tenancy.TenantOf(tx)` | every read of `org_unit_members` and `org_unit_closure` filters `tenant_id` explicitly, on top of FORCE RLS on both tables. The closure is built per tenant under that tenant's RLS (`org_unit_closure_rebuild_all`, group sync), and its FKs carry `tenant_id`. `scopeFor` drops a binding made for another tenant, principal or groups |

```mermaid
sequenceDiagram
    participant A as Tenant A (token, or its directory)
    participant GS as groupsync (nava_groupsync)
    participant DB as org_units / org_unit_members (FORCE RLS, tenant FKs)
    participant PL as write pipeline
    participant AZ as under leaf (ResolveUnder, closure)
    A->>GS: snapshot naming B's unit ids, B's paths, B's path as primary
    GS->>DB: only rows of A, under A's app.tenant_id
    DB-->>GS: B's rows unreachable (RLS), cross-tenant keys refused (23503)
    A->>PL: create or update with org_unit_id = B's unit
    PL-->>A: 422 field_not_writable, nothing written
    PL->>DB: primary unit of the requester, tenant A only
    A->>AZ: under caller / under of [B's unit]
    AZ->>DB: org_unit_closure, tenant_id = A, live paths only
    AZ-->>A: no unit of B and no tombstoned subtree is under any U
```

## Attack table

| # | Attack | Result | Test |
| --- | --- | --- | --- |
| 1 | Read, add, move or drop tenant B's `org_unit_members` rows as `nava_app`, `nava_worker` or `nava_groupsync`, with B's predicate or none | Each role reads only A's rows. Every write is refused (42501 by grant or RLS, or 23503 by the tenant FK), or reaches 0 rows | `internal/tenancy/isolation/registry.go` (`org_unit_members:` paths, TICK-947 and TICK-952) |
| 2 | Give a record of A a unit of B: in the body (create, update, workflow `update_record`), or through a forged primary membership that the pipeline's lookup reads | The body is refused with `field_not_writable` for every source. The forged lookup fails closed: 23503 on `records_org_unit_fkey` (asserted), nothing written. No record of A names B's unit | `internal/records/pipeline/org_unit_isolation_integration_test.go` `TestRecordNeverGetsAnotherTenantsOrgUnit`; `org_unit_integration_test.go` `TestOrgUnitNeverReadsAnotherTenantsMembership`, `TestCreateRefusesOrgUnitInBody`; `internal/db/dbtest/org_units_test.go` `TestTICK947RecordOrgUnitIsTenantScoped`; `internal/tenancy/isolation/org_unit_fk_paths.go` (both roles, TICK-1126) |
| 3 | A tombstoned unit grants through `under`, including to a member whose row was not removed | The live FK refuses a member of a tombstoned unit, and refuses tombstoning a unit that still has a member, for every role (23503). groupsync deletes members before the tombstone, in one transaction. A forged member row in a tombstoned unit adds nothing to S, and neither does its live subtree | `TestUnderSetIgnoresForgedCrossTenantAndTombstonedRows`; `TestTICK947MembershipConstraints`; `acl_under_integration_test.go` `TestUnderSetFromARealTreeAgreesInSQLAndGo` |
| 4 | Set S contains a unit of B: a forged membership of an A principal in B's unit, B's unit hung under A's tree, A's unit hung under B's, `under of [B's unit]`, or S bound in A and evaluated for a principal that claims B | S never holds a unit of B (SQL, Go and branch split agree). A binding made for another tenant voids the policy, `not under` included | `TestUnderSetIgnoresForgedCrossTenantAndTombstonedRows`, `TestUnderSetBoundInOneTenantGrantsNothingToAnother` |
| 5 | groupsync deletes an org unit. A crafted directory payload with B's Keycloak ids or paths un-tombstones, renames or reparents B's unit | No DELETE grant (42501). B's tombstoned unit keeps its `deleted_at`, and B's live unit, members and rows are unchanged. A gets units of its own, never on B's ids | `internal/groupsync/orgunit_isolation_integration_test.go` `TestDirectoryPayloadNeverUntombstonesAnotherTenantsUnit`; registry `org_units:` paths; `TestTICK947GroupsyncCannotDeleteOrgUnits` |
| 6 | The 10,000 cap error leaks unit ids | Gone with the cap (TICK-1034): there is no size refusal and no `access_scope_too_large` problem, so no error names a unit. A region of more than 10,000 units is a normal answer | `acl_under_integration_test.go` `TestUnderAboveTenThousandUnitsIsANormalAnswer` |
| 7 | `nava_primary_org_unit` names a group path that exists only in B | When the user is in no A unit, or in two, no primary is set, and the pipeline's lookup returns NULL, so a new ticket's `org_unit_id` stays NULL. B's own primary is unchanged. (When the user is in exactly one A unit, D-11-6 falls back to that A unit, which is the user's own unit) | `TestPrimaryOrgUnitPathOnlyInAnotherTenantSetsNoPrimary`; end to end through the real group sync and a real pipeline create, with a control that stores A's unit: `internal/records/pipeline/org_unit_isolation_integration_test.go` `TestCreateAfterSyncWithForeignPrimaryPathLeavesOrgUnitNull`; `orgmembers_isolation_integration_test.go` `TestPrimaryOrgUnitNeverResolvesAcrossTenants` |

**Orphaned units (attack 4, TICK-1034).** A unit whose `parent_id` is missing from its tenant, which only a forged row can produce, has no closure rows. So `under` is false for it and `not under` is true, like a tombstoned subtree. This is a deliberate change from TICK-950's `expandSQL`, which seeded it as a root. Decided by the Security & Multi-Tenancy Reviewer on TICK-1034. The rows `listed Z: orphaned by a foreign parent, outside every region` and `not under listed Z: the orphan reads as outside` in `TestUnderSetIgnoresForgedCrossTenantAndTombstonedRows` pin both sides. A rebuild check that counts live units with no self row is TICK-1058.

The forged rows in attacks 2 to 4 are written by the superuser with `session_replication_role = replica`, which skips every FK and trigger. That attacker is stronger than the owner role: the owner is held by FORCE RLS and cannot disable the RI triggers. These tests therefore show that the `under` leaf and the lookup do not depend on the FKs alone. Since TICK-1034 the forged world's closure is rebuilt from the forged rows, so the leaf reads what the forger wrote: the per-tenant walk under RLS reaches neither B's unit hung under A's tree nor A's unit hung under B's.

## Red check (QA review round 1)

The `Denied` mode passes on any class-23 error, so each registry path has to reach the guard it names, not a key that fails first. With the guards removed after migration (`org_unit_members` RLS disabled, DELETE granted and allowed on `org_units`, UPDATE (`is_primary`) granted to `nava_app`), these paths go Red: the group sync DELETE of the unreferenced leaf `OrgUnitA2`; both `SET is_primary = false WHERE tenant_id = $1 AND is_primary` paths; and the group sync INSERT of B's valid keys `(TenantB, OrgUnitB1, PrincipalB2)`, which only `org_unit_members_insert WITH CHECK` refuses. The pre-fix forms (DELETE of `OrgUnitA1`, `SET is_primary = NOT is_primary`, INSERT with mixed-tenant keys) stay green under the same removal: 23503 or 23505 hides them. Dropping `records_org_unit_fkey` turns attack 2 Red. That includes the two INSERT paths of `org_unit_fk_paths.go`, which accept only 23503 from that constraint. The registry's UPDATE path does not see it: since TICK-1120 the missing grant refuses it first. Letting group sync fall back to a user's first unit turns attack 7's pipeline test Red. A guard removed by a migration never reaches the tests: the catalogue gate (ADR-001 CI-1) refuses a disabled or unforced RLS and a policy that does not test `app.tenant_id`.

## Cross-references

- **TICK-1029 (attack 4).** `scopeFor` matches a binding on tenant and principal id. Attack 4's cross-tenant binding test shows that the tenant half holds. TICK-1029 adds the groups to the binding, so that a principal with the same id but other groups cannot reuse S. That is a same-tenant gap with no call site today, and the fix is left to TICK-1029. Attack 7 does not touch the binding.
- **R3 (revocation window).** A user moved out of a unit keeps `under` access until the next sync. This is accepted as [AR-TICK-901-01](accepted-risks.md) (ADR-010 R3).

## Recorded, not blocking

- `primaryUnitSQL` does not join `org_units` to check that the unit is live. It relies on the live FK of `org_unit_members`. Only a forged row can name a tombstoned unit, and `under` excludes tombstoned units anyway, so this grants nothing. Recorded as AR-S31-01.
