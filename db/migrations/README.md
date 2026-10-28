# db/migrations

New migrations use a 14-digit UTC timestamp version,
`YYYYMMDDHHMMSS_<name>.sql`: create one with `goose create <name> sql` (no
`-s`). `00001` to `00007`, and the sequential numbers TICK-284 and TICK-285
assign, keep their 5-digit versions; never add another one. Timestamps from two
branches do not collide, and the migrate job allows out-of-order versions, so a
branch's older timestamp still applies after a newer one has merged
(TICK-288, `docs/delivery/merge-safety.md`).

`go test ./db/` (part of `task test`, no Docker) fails on a malformed name, a
duplicate version (naming both files), a gap in the sequential versions, an
invalid timestamp, and any file goose cannot parse, such as one missing
`-- +goose Up`.

## Re-stamp before merging

Before a pull request merges, every migration it adds must have a version later
than the newest migration on its base branch. When the base has moved past
yours, re-stamp yours: rename it to a fresh timestamp (`goose create <name> sql`
gives one, then move your SQL into it). This is safe because no shared database
ever applies a branch's migration. It keeps fresh CI databases (version order)
and deployed ones (merge order) applying migrations in the same order;
out-of-order stays enabled as a safety net only (TICK-299).

The ci workflow's unit job enforces it: `TestNewMigrationsLaterThanBase` fails
and names each added file, its version and the base's newest version. Locally
it runs only with `BASE` set: `git fetch origin main && BASE=origin/main go test
./db/`.
