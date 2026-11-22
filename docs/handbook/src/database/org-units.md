---
title: "Org units, membership and the closure"
section: database
status: delivered
tables: [org_units, org_unit_members, org_unit_closure]
adrs: [ADR-010, ADR-010-A2, ADR-004-E5]
stories: [S31]
sources: [docs/data/org-units.md, docs/data/org-unit-closure.md, db/migrations/20261121000000_org_unit_members.sql, db/migrations/20261121050000_org_unit_closure.sql, db/migrations/20261121144100_org_unit_closure_unrooted_warning.sql, db/grants.yaml]
---

Org units are a tenant's company tree (regions, countries, teams), mirrored
from Keycloak groups under `/t/<tenant_id>/ou/` by
[group sync](../system/nava-groupsync.md). Since S31 they drive the `under`
access leaf ("tickets in my region", ADR-010 D7 as amended by
[ADR-010-A2](../adr/adr-010-a2.md)). Three tables hold them:

- **`org_units`**: the tree, one row per unit, with `parent_id`. Units are
  **tombstoned, never deleted**: `deleted_at` is set, and the generated
  `live` column is `deleted_at IS NULL`. `nava_groupsync` holds no `DELETE`,
  so a record that names a unit never dangles. A unit that comes back in
  Keycloak is revived on the same id.
- **`org_unit_members`**: who belongs to which unit, with `is_primary`. At
  most one primary unit per user (`org_unit_members_one_primary`). Group
  sync picks it: the Keycloak attribute `nava_primary_org_unit`, else the
  user's only unit, else none (D-11-6). A member of a tombstoned unit is
  refused by the live foreign key (`CHECK (live)`, 23503), so sync deletes a
  unit's memberships before it tombstones the unit.
- **`org_unit_closure`**: for each live unit, every live unit at or below it,
  itself included. A row exists only if every unit from the root down to the
  descendant is live, so a tombstoned unit and its whole subtree have no
  rows, live descendants included. Group sync is the only writer and keeps it
  current in the same transaction as the tree change. Access reads it; no
  request walks the tree.

`records.org_unit_id` points at `org_units` ([Records](records.md)). Every key
carries `tenant_id`, so tenant A's membership, closure row or record cannot
name tenant B's unit.

## How the closure is kept right

| When | What group sync does |
|---|---|
| A run that touched units | Replaces only the closure rows of the touched units' subtrees (TICK-1054). The first import rebuilds all of the tenant's rows |
| A run that touched no unit | Compares the stored and the live closure as sets, and rebuilds on drift |
| After every rebuild call, and on every unchanged-snapshot cycle | Counts the tenant's live units that have no `(u, u)` self row (TICK-1058). Such a unit is outside every region. A count above 0 adds to `groupsync_org_closure_unrooted_units` and writes a WARN line naming the tenant. The sync does not fail |

Failing the sync on an unrooted unit would repair nothing, because the
rollback keeps the stored closure. It would also hold back that tenant's
user tombstones and group removals, so deprovisioning would wait. Alert on
the counter, not on the WARN line, which repeats every cycle.

`org_unit_closure_rebuild_all()` (the migration backfill, run as
`nava_migrator`) does the same self-row check and raises a `WARNING` per
affected tenant (`20261121144100_org_unit_closure_unrooted_warning.sql`).

## Measured cost

From `docs/data/org-unit-closure.md`, PostgreSQL 16.10, 400 tenants, one
tenant with an 11,111-unit tree:

| Measure | Result |
|---|---|
| Closure size | 53,778 rows for the large tenant (about 4.9 per unit); 9.7 MB with its primary key, for all tenants |
| `Allows` probe (is unit d under unit a?) | primary-key probe, 4 buffers, 0.06 ms |
| A level-1 unit's subtree (1,000 units) | index-only scan, 0.68 ms |
| Full rebuild of the large tenant | 1.26 s, most of it the two foreign-key checks; the self-row check adds about 2% |

The primary key `(tenant_id, ancestor_id, descendant_id)` is the closure's
only index. It serves the probe, the subtree read and the per-tenant delete.

## Grants

| Role | `org_units` | `org_unit_members` | `org_unit_closure` |
|---|---|---|---|
| `nava_groupsync` | `SELECT`, `INSERT`, `UPDATE (parent_id, name, deleted_at)`; no `DELETE` | `SELECT`, `INSERT`, `UPDATE (is_primary)`, `DELETE` | `SELECT`, `INSERT`, `DELETE`; no `UPDATE` (the only writer) |
| `nava_app` | `SELECT` | `SELECT` | `SELECT` |
| `nava_worker` | `SELECT` on `tenant_id, id, parent_id, deleted_at` | `SELECT` on `tenant_id, org_unit_id, principal_id, is_primary` | `SELECT` (for `under` in workflow policy tests) |

All three tables are identity mirrors and exempt from audit. The exact
grants are generated on [Roles and grants](roles-and-grants.md).

## Diagram

<!-- gen:er area=org-units -->

## Tables

<!-- gen:tables area=org-units -->
