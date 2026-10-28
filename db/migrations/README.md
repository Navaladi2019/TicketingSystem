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
than the newest migration on `main`. This holds for every pull request, also
one into a story branch: the order that matters is the one deployed databases
applied, and only `main` ships (TICK-309). A migration counts as added when
`main` does not have it, so `main`'s own migrations merged into a story branch
never count. When `main` has moved past yours, re-stamp yours: rename it to a
fresh timestamp (`goose create <name> sql` gives one, then move your SQL into
it). This is safe because no shared database ever applies a branch's
migration. It keeps fresh CI databases (version order) and deployed ones (merge
order) applying migrations in the same order; out-of-order stays enabled as a
safety net only (TICK-299).

A story may renumber its own unshipped migrations: their order against the
story branch protects no deployed database. A long-lived database built from a
story branch must be rebuilt after the story renumbers its migrations
(accepted risk, `docs/delivery/decision-log.md`, 2026-10-28).

The ci workflow's unit job enforces it: `TestNewMigrationsLaterThanBase` fails
and names each added file, its version and `main`'s newest version. The job
sets `BASE=origin/main` for every event, never the pull request's base. Locally
it runs only with `BASE` set: `git fetch origin main && BASE=origin/main go test
./db/`.
