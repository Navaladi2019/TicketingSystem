# Review: TICK-992, PR #775 (idle-host scoring of D13 and D10)

- **Change:** `TICK-992/idle-host-scoring` → `TICK-739/filterable-fields`, head `37e8b89c` (3 commits, 0 behind the story branch)
- **Reviewed against:** TICK-992 scope and acceptance criteria, ADR-006 D10/D13/R-a, the Technical Lead's waiver and decisions on TICK-992
- **Reviewer:** Quality, Testing & CI Advisor, 2026-11-19
- **CI:** GitHub Actions `ci` run 44620227259 green: `lint`, `unit`, `integration`, `stack`, `coverage`, `commit-check-budget`, `full-suite`
- **Local:** `go vet -tags 'integration perf' ./internal/server/` clean. The perf-tagged file is not compiled by `unit` or `integration`, so this is the only build check it gets.

## VERDICT: APPROVE

There are no must-change findings.

## Harness fix 1: the ts two-`EXISTS` comparison is removed

`internal/server/records_fieldindex_perf_integration_test.go`. This fix is correct and does not change what a run measures.

- `mergedRange` called `t.Fatalf` when a ts statement had no `>= … AND EXISTS (… <)` pair to merge. After F1 (TICK-989, one `EXISTS` per field) the compiler no longer emits that pair, so every ts cell stopped the list phase. This bug was real.
- The comparison ran after each ts cell's `runOne` pair had already recorded its p95. It was a separate diagnostic table, not an input to any D13 cell, the miss list or the controls. Removing it removes one table and roughly 20 s of extra load per ts cell. The scored numbers do not change.
- `sqlLoad`, `explainExec`, `fiRangePair`, `mergeRange` and the `tenancy` import had no other callers. `fiPlanning` and `fiExecution` are still used by the plan column. Vet is clean.

## Harness fix 2: `NAVA_PERF_*_BATCHES=0` now runs the whole backfill

This fix is correct. The only behaviour change is the intended one.

- `perfEnvInt` accepts only `v > 0`, so `0` fell back to the default (40 / 60) and the documented whole run never happened. The new `fiBatchCap` accepts `v >= 0`.
- `0` reaches `b.cap`, and the batch hook closes `capped` only when `b.cap > 0` (line 719). The workflow therefore runs to completion and `runBackfill` takes the `done` branch, which is the uncapped path that also diffs side rows. That matches the recorded 626 s whole backfill with diff = 0.
- With the env unset, defaults are unchanged (40 / 60). Negative values still fall back to the default.

## Findings

1. **Preference.** `docs/budgets.md`, D13 row, still says "decision with the Technical Lead: options (a), (b), (c)". The decision is now made: (c) as ADR-006-A1, with (a) gated on R-a's trigger. PR #777 (TICK-1001) rewrites this row, so no change is needed here.
2. **Preference.** The raw report headers in `docs/data/field-index-measured/tick992-*.md` read "TICK-747 field-index measurements". The harness prints that title. A reader who opens the raw file alone could take it for the P7-07 run. The date, the host lines and the summary in `docs/data/field-index-measured.md` disambiguate it.
3. **Preference.** `scratch/field-index-measured/run.sh` does not set `NAVA_PERF_FIX_BATCHES=0`. The whole run is a manual override, documented at `docs/data/field-index-measured.md:417`. That is fine for a scratch runner. Default it only if every future scoring run should be a whole run.

## Not covered by this review

- The numbers themselves are measurements, not code. The waiver, the borderline rule and the provisional cells are decided on TICK-992 (Technical Lead). The four provisional cells are rerun under TICK-1002.
- The D13 variance is TICK-1001 (PR #776, #777). It is not part of this change.
- Handbook: the field-index benchmark page needs the TICK-992 numbers. The Technical Lead owns that under TICK-982.
