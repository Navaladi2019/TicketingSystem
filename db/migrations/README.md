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
