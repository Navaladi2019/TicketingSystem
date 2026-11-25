# A later run on the same commit hides a red required check (TICK-1146)

**Result: yes.** On a pull request into `main` with a red `isolation`, GitHub moved
`mergeStateStatus` from `BLOCKED` to `CLEAN` without any code change in three ways:
adding any label other than `ci:run` or `ci:full` (b), switching `ci:full` to `ci:run` (d),
and closing and reopening the PR with no label (f). Adding a label while the
full run was still in progress (e) also gave `CLEAN`, even though the red result
finished after the label run's skipped one.

GitHub takes each required check from the **newest run** (check suite) on the head commit,
not from the newest result. Under every `if:` that skips a job, the job still reports
its required name as `skipped`, and a skipped required check counts as passed.

**Fix, proven on the spike:** name the job with an expression, `name: ${{ 'isolation' }}`.
A job that ran reports the evaluated name `isolation`. A skipped job reports the
unevaluated text of the expression. So a skipped run never supplies the required check
`isolation`, and only a run that executed the suite can turn it green (g, h).

## Method

Spike PR [#898](https://github.com/Navaladi2019/TicketingSystem/pull/898), a draft into `main`,
closed without merging, with its branch deleted. Branch protection was not changed. Its `ci.yml` was
`main`'s, except that the steps of the nine suite jobs were stubbed: `isolation` exits 1,
the others exit 0 (`evidence/stub.py`). Every trigger, `if:`, `needs`, job name and
concurrency group stayed real, so the event handling under test is production's.
Each case waited for every run on the head to finish, then waited 45 s, then read GraphQL
`mergeStateStatus` (the verdict), REST `mergeable_state`, the runs, all check runs, and
`gh pr checks --required` (`evidence/snap.sh`). A read counts only when the head was
0 commits behind `main`: strict status checks report `BEHIND`, which hides
`BLOCKED`/`CLEAN`. Three reads taken after `main` moved were discarded and redone.

A draft PR reports `CLEAN`/`BLOCKED` normally (case 0, case a), so the PR stayed a draft.

## Results

| Case | Sequence on one commit | `mergeStateStatus` | Newest `isolation` per run |
|---|---|---|---|
| 0 | opened, no label | `CLEAN` | skipped (lint-only, AR-TICK-563-01) |
| a | + `ci:full` (control) | **`BLOCKED`** | failure |
| b | + `question` after the run | **`CLEAN`** (re-read after 60 s: `CLEAN`) | skipped (label run) |
| c | `ci:full` removed and added again | `BLOCKED` | failure |
| d | `ci:full` removed, + `ci:run` | **`CLEAN`** (re-read: `CLEAN`) | skipped (fast run) |
| e | new head; + `ci:full`, + `question` 12 s later | **`CLEAN`** | failure started 22:40:53 in the older run, skipped 22:40:41 in the newer run |
| f | + `ci:full` again (`BLOCKED`), then remove labels, close, reopen | **`CLEAN`** | skipped (lint-only) |
| g | fix applied: `name: ${{ 'isolation' }}`, opened run | `BLOCKED` | none named `isolation` (skipped job's name is the raw expression) |
| h1 | fix; + `ci:full`, + `question` during the run | `BLOCKED` | failure |
| h2 | fix; + `wontfix` after the run | `BLOCKED` | failure |

Raw output per case: `evidence/<case>-pr.json` (GraphQL), `-rest.json`, `-runs.json`,
`-checkruns-all.json` (`filter=all`, reduced to id, name, conclusion, times, suite),
`-pr-checks-required.txt`, `-time.txt` (GitHub `Date`), `-behind.txt`.

**`gh pr checks` is not evidence.** In e it listed `isolation fail` while the PR was `CLEAN`.

## What else this means

- Every required suite job (`lint`, `unit`, `integration`, `stack`, `coverage`, ...) can be
  hidden the same way. The `MergePr` label that agents add before the owner merges is case b.
- Story branches have no required checks in GitHub at all (ruleset "Mina branch" has
  only deletion and non-fast-forward rules), so `full-suite` protects nothing there by itself.
  An unrelated label also gives it a skipped run.
- AR-TICK-563-01 ("the push to `main` then runs every job") is out of date: since TICK-652
  a push to `main` runs a fast run without isolation. The daily run is the first to catch it.

## Recommendation

1. **Pin it (this change).** `make check-ci-steps` gains a `checks:` list in
   `tools/ci/cisteps/required.yaml`. A listed job must be named `${{ '<check>' }}`.
   The `isolation` entry fails on `main` today and passes with the one-line fix.
2. **Fix (`ci.yml`), two edits that only work together:**
   - `isolation`: `name: ${{ 'isolation' }}`.
   - `isolation`'s `if:`: run whenever the run is not label-only or lint-only, not just
     on full runs. Otherwise no `ci:run` PR into `main` can ever report `isolation`
     and every one stays `BLOCKED`. Cost: one isolation job of about 6 min per labelled run.
     This reverses part of AR-TICK-652-01, so it is the CTO's decision.
   The same name trick on the other required jobs would make every PR run every suite
   before merging (reversing AR-TICK-563-01 and the diff-based skipping). Recommended only
   for `isolation` now; the rest is a cost decision.
3. **Rejected candidates.**
   - A check of earlier results when the label is added: case e shows the red can arrive
     after the label run is created.
   - Re-running or dispatching from a separate label workflow: a re-run keeps the old
     payload's labels, and a dispatch runs on the branch head, not the merge ref.

## Until it is fixed (reviewers and mergers)

Do not trust the merge button, `mergeStateStatus` or `statusCheckRollup`. Before approving
or merging a PR into `main` or a story branch, list every result on the head commit and refuse
when any required name has a `failure`, `cancelled` or `timed_out` that no later
**executed** (non-skipped) success of the same name follows:

```sh
gh api --paginate "repos/Navaladi2019/TicketingSystem/commits/<head sha>/check-runs?filter=all&per_page=100" \
  -q '.check_runs|sort_by(.started_at)[]|select(.conclusion!="skipped")|"\(.started_at) \(.name) \(.conclusion)"'
```

Read the last line per name. Do not add labels other than `ci:run`/`ci:full` to a PR
whose latest full run is red.
