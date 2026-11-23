# Review: TICK-913 / PR #864 — handbook:check ignores the hand-authored examples/

Reviewer: Quality, Testing & CI Advisor · Base: `main` · Code head: `59f2c40f`

## Verdict

APPROVE. No Bug, Breaks-under or Spec-drift findings.

Spec challenge: done. Consumers checked: none (a local generator check; no task reads its output). Boundaries: none (no service, tenant data or identity code). Result: no conflict.

## What was checked

- `task handbook:check`'s command exits 0 on the branch. With `examples` removed from `HAND_AUTHORED_DIRS` it exits 1 and lists the 7 files.
- `test_build.py`: 97 tests pass. `test_examples_are_hand_authored_source` fails under each of two mutations: `examples` removed from the tuple, and the exclusion applied at every depth instead of the top level only.
- The exclusion hides no generated output: `examples` is not in `SECTIONS`, and `is_generated_name` never matches a path under it.
- `write_output` loses only the "leaving ... alone" message for these files. It never deleted them.
- The sensitivity scan is unchanged: it reads build output, `src/database/schema.json` and `src/diagrams/*.svg`, and never read `examples/`.

## Findings

1. **Missing test** (non-blocking, predates this PR): no CI step runs `handbook:check`. `make test-handbook` runs the generator against temporary trees only, so a file added under `docs/handbook/` can break the check on `main` with CI green, as TICK-789 did. A `make` target called from the lint job would catch it. Whether stale HTML should block every PR is the Technical Lead's call.

## CI

`ci` run 45109187531 (`ci:run`) on `59f2c40f`: success. `changes`, `lint` (with the handbook generator tests and the page-coverage gate), `unit`, `integration`, `stack`, `coverage` and `commit-check-budget` passed. `temporal-server`, `sdk` and `isolation` were skipped: this diff does not need them.
