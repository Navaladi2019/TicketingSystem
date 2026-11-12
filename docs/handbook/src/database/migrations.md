---
title: "Migration history"
section: database
status: delivered
sources: [db/migrations, db/migrations/README.md, docs/handbook/src/database/migration-history.json, docs/handbook/build/build.py]
adrs: [ADR-001]
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
  writes go on while it runs. The table pairs them both ways.
- **`CONCURRENTLY`.** An index on a large table is built with `CREATE INDEX
  CONCURRENTLY`, alone in a `-- +goose NO TRANSACTION` file. A failed build
  leaves an invalid index, which the file drops and rebuilds when it runs
  again.
- **`drops`** marks a migration that removes a table, index, policy, column,
  constraint, function or trigger.

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
