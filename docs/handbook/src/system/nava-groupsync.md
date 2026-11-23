---
title: "nava-groupsync"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-groupsync/main.go, db/grants.yaml, internal/groupsync/apply.go, internal/groupsync/closure.go, internal/groupsync/groupsync.go, deploy/compose/prometheus/rules/groupsync.yml, docs/data/org-units.md, docs/data/org-unit-closure.md]
adrs: [ADR-001, ADR-002, ADR-010, ADR-010-A2]
stories: [S31]
tables: [org_units, org_unit_members, org_unit_closure]
packages: [cmd/nava-groupsync, internal/groupsync]
invariants: [I3]
tests: [TestE5GroupsyncCannotRekeyGroups, TestTwoSyncersNeverSyncOneTenantConcurrently, TestGroupsyncAccountFloor, TestOrgMembersAndPrimaryWritten, TestOrgUnitWithMembersRemovedAndRevived, TestOrgSubtreeMaxReported, TestClosureFollowsTheTree, TestClosureDriftCheckComparesSetsNotCounts, TestClosureUnrootedUnitIsReported, TestUnderClosureAgreementNestedMovedAndTombstoned]
---

Mirrors each tenant's users, groups, memberships and org units under `/t/<tenant_id>/` from Keycloak into the identity tables, every `NAVA_GROUPSYNC_INTERVAL` (default 5m, at most 5m). A principal resolves (`GET /api/v1/me`) only after a cycle has synced it. `--once` runs one cycle and exits.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only |
| Host port | none (tunnel 19092) |
| DB role | `nava_groupsync` through PgBouncer |
| Talks to | keycloak (Admin API read, client `nava-groupsync`); pgbouncer; otel-collector |
| Introduced by | PLAN-001 T21b (TICK-234) |

## Org units (S31)

Since S31 the org-unit tree drives the `under` access leaf, so group sync
writes three tables per tenant, in the same transaction as the rest of that
tenant's sync ([Org units](../database/org-units.md)):

- **`org_units`.** A unit that vanished from Keycloak is tombstoned
  (`deleted_at` set), and so are its descendants. Its memberships are deleted
  first, because the live foreign key refuses a member of a tombstoned unit.
  A unit that comes back is revived on the same id. Units are never deleted.
- **`org_unit_members` and the primary unit (D-11-6).** Each user's units,
  with one primary: the user attribute `nava_primary_org_unit`, else the
  user's only unit, else none. The attribute is declared in the `nava` realm
  user profile with admin-only view and edit, so users cannot set it
  ([Keycloak](keycloak.md)).
- **`org_unit_closure`.** A run that touched units replaces only the closure
  rows of the touched subtrees (TICK-1054); the first import rebuilds the
  tenant. An **applied** run (the directory changed) that touched no unit compares
  the stored and the live closure as sets and rebuilds on drift. An
  unchanged-snapshot cycle does not compare the closure. The cursor is in
  memory, so a restart makes the next cycle apply. Both run under the per-tenant advisory lock.
- **Unrooted check (TICK-1058).** After every rebuild call, and on every
  unchanged-snapshot cycle, it counts the tenant's live units with no closure
  self row. Such a unit is outside every region. A count above 0 is reported
  and **does not fail the sync**: failing would repair nothing and would hold
  back that tenant's tombstones and group removals.
- **Never refuses a sync on size (TICK-1072).** A large tree raises an alert,
  nothing else, so deprovisioning is never held back.

| Metric (no tenant label, T01) | What it is | Alert |
|---|---|---|
| `groupsync_org_subtree_max` | Histogram, once per applied tenant sync: the most live units under any one unit a user belongs to | None. A tenant at 8,000 or more is named in a WARN log line |
| `groupsync_org_units_live_max` | Gauge, set once per cycle: the largest live-unit count of any tenant visited | `GroupsyncOrgUnitsAboveCertified`: above 10,111 for 10 minutes, the tree ADR-010-A2 is certified to (A2-7) |
| `groupsync_org_closure_unrooted_units` | Counter: live units found with no closure self row | Alert on any increase of the counter, not on the WARN line, which repeats every cycle |

The tenant over a threshold is named in the log line, never in a metric. A
`--once` run exits with its own metrics, so `groupsync_org_units_live_max`
has no sample until the long-running service's first cycle; the cycle's log
line carries `org_subtree_max` and `org_closure_rows`.

## Boundary it enforces

**I3.** `nava_groupsync` may write only the identity tables, and cannot re-key a group: `TestE5GroupsyncCannotRekeyGroups`. Its Keycloak account holds only the roles the sync needs: `TestGroupsyncAccountFloor`.

[Back to the system map](index.md)
