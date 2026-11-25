---
title: "Migration history"
section: database
status: delivered
stories: [S31]
sources: [db/migrations, db/migrations/README.md, docs/data/entity-versions-org-unit-id-reserved.md, docs/handbook/src/database/migration-history.json, docs/handbook/build/build.py]
adrs: [ADR-001, ADR-010]
---

Migrations live in `db/migrations` and run with goose, as `nava_migrator`,
in the one-shot `migrate` job. Every table, index, policy and grant on the
other database pages is the state after the last of them. Dropped objects
appear only here, in the migration that dropped them.

## Two naming eras

- **`00001` to `00012`**: five-digit sequential versions, from the first
  platform tables to TICK-285. No new migration may take one.
- **`YYYYMMDDHHMMSS_<name>.sql`**: a 14-digit UTC timestamp, from
  `20261028175712` on (TICK-288). Timestamps from two branches do not
  collide. Before a pull request merges, each migration it adds must be newer
  than the newest one on `main`, or be re-stamped (`db/migrations/README.md`).

## Online steps

Migrations run against a live database, so the slow parts are split off:

- **`NOT VALID`, then `*_validate`.** A new `CHECK` or foreign key is added
  `NOT VALID`, which takes a brief lock and checks only new rows. A second
  migration, whose name ends in `_validate`, runs `VALIDATE CONSTRAINT` in its
  own transaction. That scan takes `SHARE UPDATE EXCLUSIVE`, so reads and
  writes go on while it runs. The exception is a migration that also creates
  temporary policies for the scan: `20261121000300` holds `ACCESS EXCLUSIVE`
  on `records` for it (112 ms at 1,000,000 rows, `docs/data/org-units.md`). The table pairs them both ways.
- **`CONCURRENTLY`.** An index on a large table is built with `CREATE INDEX
  CONCURRENTLY`, alone in a `-- +goose NO TRANSACTION` file. A failed build
  leaves an invalid index, which the file drops and rebuilds when it runs
  again.
- **`drops`** marks a migration that removes a table, index, policy, column,
  constraint, function or trigger.

## Upgrading to S31

The S31 upgrade can refuse stored data in two places. Check a database
**before** the upgrade, on the release before S31, as a superuser. The SQL is in
one file, which the tests run:
[entity-versions-org-unit-id-reserved.md](../../../data/entity-versions-org-unit-id-reserved.md#pre-upgrade-check-and-repair-tick-1125).

| Migration | Refuses when | Before the upgrade | If it stopped the upgrade |
|---|---|---|---|
| `20261120235900`, S31's first | A stored entity-type version has a top-level property `org_unit_id` (23514) | Run the [check](../../../data/entity-versions-org-unit-id-reserved.md#check). A row: follow the [order of work](../../../data/entity-versions-org-unit-id-reserved.md#order-of-work), which stops `nava-core` and `nava-worker`; the [repair](../../../data/entity-versions-org-unit-id-reserved.md#repair) is its step 3. 10,000 versions or more, or 256 MiB or more: stop and ask the Technical Lead | Nothing of S31 is applied, and the old release keeps working. The same order of work |
| `20261121000100` | A stored access document names, under `under`, an id that is not a live org unit of its tenant (`AR-S30-01`) | Run the AR-S30-01 query of the same [check](../../../data/entity-versions-org-unit-id-reserved.md#check). A row: do not migrate, ask the Technical Lead | `20261120235900` and `20261121000000` stay applied. Access publishing works; the old group sync deletes no removed org unit until the upgrade finishes. Run the AR-S30-01 query again and ask the Technical Lead. Do not delete `acl_versions` rows, whatever the error's HINT says: the newest row is the access in force |

- **The repair refuses** some types (R1 to R11 in
  [what the repair refuses](../../../data/entity-versions-org-unit-id-reserved.md#what-the-repair-refuses)).
  R1 is a field mask that names the property: never tell the tenant's admin to
  drop the mask, because that shows the value to everyone it was hidden from.
- **A stop after `20261121161036`** blocks access publishing for every tenant
  with `42501`. Run `migrate` again
  ([if the upgrade stops later](../../../data/entity-versions-org-unit-id-reserved.md#if-the-upgrade-stops-later)).
  A successful upgrade has the same gap for about a second (0.7 to 1.0 s in
  three runs, PostgreSQL 16.10, 100,000 records).
- **Accepted risks:** `AR-S31-02` (one tenant can stop the upgrade),
  `AR-S31-03` (the publishing gap) and `AR-S31-04` (no mask for the old name
  after a repair), in the [register](../security/accepted-risks.md).

## Every migration

"Introduced by" names the story, tasks and pull request of the commit that
first added the file to `main`. The build takes these from
`docs/handbook/src/database/migration-history.json`, so it needs no git
history. After a migration merges to `main`, refresh the file with
`python -I docs/handbook/build/build.py --capture-migrations`. That command
needs `origin/main`. The build fails when the file and `db/migrations` list
different files. The summary is the first sentence of the file's header
comment.

<!-- gen:migrations -->
