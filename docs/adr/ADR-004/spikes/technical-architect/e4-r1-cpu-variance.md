# Spike: does `commit-check-budget` vary < 1.2× within one CPU model? (TICK-526)

ADR-004-E4 follow-up 6. Evidence for AR-TICK-517-01 (E4-R1). Technical Architect, 2026-11-05.

## Verdict

**Pass.** For every CPU model with at least 3 runs, max/min < 1.2 on every row. The worst is 1.127 (`AMD EPYC 7763`, clause per call at N = 200, one 258.0 µs outlier). Every other cell is ≤ 1.07.

**But the E4-R1 trigger as written does not work, for a reason the variance test cannot catch.** `warn` is an absolute threshold, not a per-model regression cue:

- **Always on for EPYC 7763.** Its import row sits at 189–194 µs against the 175 µs threshold, so 7 of 7 runs on it were `warn`. The baseline in `docs/budgets.md` (194.9 µs, run 41679998550) is already over the threshold. In this sample 7763 drew 7 of 16 runners, so "two `warn` runs in a row on `main` on a recorded model" fires with no regression at all.
- **Blind on the fast models.** On `AMD EPYC 9V45` (import 78.5–81.1 µs, N = 200 125.9–128.6 µs), only a uniform regression of about 2.2× (import) or 2.3× (per call) reaches a threshold.

Measured noise is ≤ 1.13× per model, so a **per-model relative band** around the recorded baseline would catch a ≥ 1.2× uniform regression on every model. See "Proposed E4-R1 trigger".

## Hypothesis and criterion

On GitHub-hosted `ubuntu-latest`, for each `cpu=` model with ≥ 3 runs, max/min < 1.2 for clause per call at N = 200, 1,000 and 5,000, and for import µs per probe. Pass if this holds for every such model, otherwise fail. Criterion fixed in the ticket before collection.

## Setup

- `main` @ `21106dfe` (first `main` build with TICK-509's `cpu=` logging), CI run [41964296547](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547), job `commit-check-budget` (`postgres:16.10`, `ubuntu-latest`, 4 vCPU, alone on its runner, 38 partitions).
- 16 samples: attempt 1 is the push build. Attempts 2–16 are re-runs of only that job (`gh run rerun 41964296547 --job <id>`), one after another, on 2026-11-05 between 04:38 and 05:03 UTC. Each attempt gets a fresh hosted runner. All 16 succeeded. The only dependent job, `main-red`, is gated on `failure()` and stayed skipped.
- Why re-runs rather than 16 separate pushes: only one `main` build had `cpu=` when the spike was released. Re-running one SHA also holds code and fixture constant, so the spread is runner noise only.

## Limitation (Inferred, partly checked)

All samples come from one SHA within about 70 minutes (03:54 push build, re-runs 04:38–05:03), so day-to-day drift (runner image updates, noisy neighbours at other hours) is not sampled directly. Partial cross-day check against the 2026-11-03/04 baselines in `docs/budgets.md`, which come from different SHAs:

| model | baseline (budgets.md) | this spike | baseline / spike range |
|---|---|---|---|
| EPYC 9V45, N = 200 | 126.1 | 125.9–128.6 | within range |
| EPYC 9V45, import | 80.3 | 78.5–81.1 | within range |
| EPYC 7763, N = 200 | 235.7 | 229.0–258.0 | within range |
| EPYC 7763, import | 194.9 | 189.0–193.5 | 1.007× the max |
| "CPU not logged", N = 200 / import | 170.5 / 127.7 | 9V74: 170.8–183.2 / 128.6–129.3 | matches 9V74 |

The "CPU not logged" baseline (run 41673342312) is almost certainly `AMD EPYC 9V74` (Inferred: it matches to within 1%), so `docs/budgets.md` can relabel it once a 9V74 run confirms.

## Summary (max/min per model and row)

| CPU model | runs | N=200 | N=1,000 | N=5,000 | import | counts for verdict |
|---|---|---|---|---|---|---|
| `AMD EPYC 9V74 80-Core Processor` | 2 | 1.073 | 1.034 | 1.037 | 1.005 | no (n<3) |
| `AMD EPYC 7763 64-Core Processor` | 7 | 1.127 | 1.035 | 1.030 | 1.024 | yes |
| `INTEL(R) XEON(R) PLATINUM 8573C` | 3 | 1.023 | 1.026 | 1.023 | 1.038 | yes |
| `AMD EPYC 9V45 96-Core Processor` | 3 | 1.021 | 1.046 | 1.069 | 1.033 | yes |
| `Intel(R) Xeon(R) 6973P-C` | 1 | 1.000 | 1.000 | 1.000 | 1.000 | no (n<3) |

The single-subject `verdict` reads `warn` on 7763 even though every per-call value is < 300 µs. That field reports the run's combined verdict, which the import row drives (Known from the lines: `over_threshold=false` on the single row and `true` on the import row).

## Proposed E4-R1 trigger

Replace "two `warn` runs in a row on `main` on a CPU model already recorded in a baseline" with:

> Two `main` runs in a row on a CPU model recorded in a `docs/budgets.md` baseline, each with any row (clause per call at N = 200, 1,000 or 5,000, or import per probe) more than **1.2×** that model's recorded baseline value; or an over-threshold baseline re-take.

Why 1.2×: the worst within-model noise measured is 1.127×, and ≤ 1.07× apart from one N = 200 outlier. "Two in a row" absorbs a single outlier like attempt 13. The check is manual, the same as today (a reader compares the logged line with the table). Automating it is optional follow-up work and is not part of this proposal.

The EPYC 7763 import baseline is already over threshold. Under the ADR's own rule, that already "goes to the Technical Lead as a possible regression". It is not a regression (the ratio is 8.58, the same as on the other models). That needs recording as a known-slow-host baseline so it stops counting as a trigger.

## How to rerun

```sh
R=Navaladi2019/TicketingSystem RUN=<a main run with cpu= lines>
JOB=$(gh api repos/$R/actions/runs/$RUN/jobs --jq '.jobs[]|select(.name=="commit-check-budget")|.id')
gh run rerun $RUN -R $R --job $JOB && sleep 15 && gh run watch $RUN -R $R --exit-status
A=$(gh api repos/$R/actions/runs/$RUN --jq .run_attempt)
J=$(gh api repos/$R/actions/runs/$RUN/attempts/$A/jobs --jq '.jobs[]|select(.name=="commit-check-budget")|.id')
gh api repos/$R/actions/jobs/$J/logs --allow-escape-sequences | grep -a COMMIT-CHECK-BUDGET
```

Pitfall (Known, observed in this spike): `gh run view --job <id> --log` on a re-run job can return the **latest attempt's** log under an older job id. Fetch by job id through `actions/jobs/<id>/logs` instead.

## Per model: raw values and max/min

### `AMD EPYC 9V74 80-Core Processor` (2 runs)

| attempt | job | clause µs/call N=200 | N=1,000 | N=5,000 | import µs/probe | verdict (single / import) |
|---|---|---|---|---|---|---|
| 1 | [127037731885](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127037731885) | 170.8 | 166.7 | 171.8 | 129.3 | pass / pass |
| 10 | [127056716926](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127056716926) | 183.2 | 172.3 | 165.6 | 128.6 | pass / pass |
| **max/min** | | 1.073 (170.8–183.2) | 1.034 (166.7–172.3) | 1.037 (165.6–171.8) | 1.005 (128.6–129.3) | |

### `AMD EPYC 7763 64-Core Processor` (7 runs)

| attempt | job | clause µs/call N=200 | N=1,000 | N=5,000 | import µs/probe | verdict (single / import) |
|---|---|---|---|---|---|---|
| 2 | [127053006071](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053006071) | 234.4 | 232.3 | 230.6 | 190.4 | warn / warn |
| 5 | [127054555284](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127054555284) | 233.0 | 226.4 | 229.5 | 190.8 | warn / warn |
| 6 | [127055037736](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055037736) | 232.5 | 226.2 | 232.0 | 192.1 | warn / warn |
| 12 | [127058210643](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127058210643) | 229.0 | 224.5 | 226.1 | 189.0 | warn / warn |
| 13 | [127058793248](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127058793248) | 258.0 | 228.7 | 231.6 | 191.2 | warn / warn |
| 15 | [127059751351](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127059751351) | 236.6 | 228.6 | 232.9 | 193.5 | warn / warn |
| 16 | [127060193703](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127060193703) | 233.5 | 232.2 | 230.4 | 191.7 | warn / warn |
| **max/min** | | 1.127 (229–258) | 1.035 (224.5–232.3) | 1.030 (226.1–232.9) | 1.024 (189–193.5) | |

### `INTEL(R) XEON(R) PLATINUM 8573C` (3 runs)

| attempt | job | clause µs/call N=200 | N=1,000 | N=5,000 | import µs/probe | verdict (single / import) |
|---|---|---|---|---|---|---|
| 3 | [127053469280](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053469280) | 223.8 | 219.3 | 219.9 | 145.9 | pass / pass |
| 4 | [127053923380](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053923380) | 228.9 | 224.9 | 224.9 | 151.4 | pass / pass |
| 8 | [127055851276](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055851276) | 227.5 | 220.6 | 222.6 | 149.3 | pass / pass |
| **max/min** | | 1.023 (223.8–228.9) | 1.026 (219.3–224.9) | 1.023 (219.9–224.9) | 1.038 (145.9–151.4) | |

### `AMD EPYC 9V45 96-Core Processor` (3 runs)

| attempt | job | clause µs/call N=200 | N=1,000 | N=5,000 | import µs/probe | verdict (single / import) |
|---|---|---|---|---|---|---|
| 7 | [127055451827](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055451827) | 126.8 | 125.8 | 125.8 | 78.5 | pass / pass |
| 9 | [127056311690](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127056311690) | 128.6 | 124.5 | 121.8 | 81.1 | pass / pass |
| 11 | [127057622460](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127057622460) | 125.9 | 120.3 | 117.7 | 79.6 | pass / pass |
| **max/min** | | 1.021 (125.9–128.6) | 1.046 (120.3–125.8) | 1.069 (117.7–125.8) | 1.033 (78.5–81.1) | |

### `Intel(R) Xeon(R) 6973P-C` (1 runs)

| attempt | job | clause µs/call N=200 | N=1,000 | N=5,000 | import µs/probe | verdict (single / import) |
|---|---|---|---|---|---|---|
| 14 | [127059304692](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127059304692) | 167.4 | 156.6 | 154.3 | 103.8 | pass / pass |
| **max/min** | | 1.000 (167.4–167.4) | 1.000 (156.6–156.6) | 1.000 (154.3–154.3) | 1.000 (103.8–103.8) | |

## Raw `COMMIT-CHECK-BUDGET` lines

Timestamps stripped; `::warning::` annotation duplicates omitted.

```text
# attempt 1, job 127037731885, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127037731885
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 9V74 80-Core Processor" clause_us_n200=170.8 clause_us_n1000=166.7 clause_us_n5000=171.8 max_min=1.03 ref_us_n200=121.7 ref_us_n1000=97.8 ref_us_n5000=215.5 ratio_n200=1.40 ratio_import=9.52 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 9V74 80-Core Processor" subjects=10000 clause_us_per_probe=129.3 clause_runs=[129.3 129.7 128.5] ref_us_per_probe=13.6 ref_runs=[13.6 13.7 13.4] ratio_import=9.52 ratio_n200=1.40 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=14.192s
# attempt 2, job 127053006071, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053006071
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=234.4 clause_us_n1000=232.3 clause_us_n5000=230.6 max_min=1.02 ref_us_n200=165.0 ref_us_n1000=104.9 ref_us_n5000=264.2 ratio_n200=1.42 ratio_import=9.63 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=190.4 clause_runs=[192.6 189.4 190.4] ref_us_per_probe=19.8 ref_runs=[19.7 19.8 20.3] ratio_import=9.63 ratio_n200=1.42 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=18.765s
# attempt 3, job 127053469280, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053469280
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" clause_us_n200=223.8 clause_us_n1000=219.3 clause_us_n5000=219.9 max_min=1.02 ref_us_n200=133.4 ref_us_n1000=96.2 ref_us_n5000=244.2 ratio_n200=1.68 ratio_import=9.55 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" subjects=10000 clause_us_per_probe=145.9 clause_runs=[145.3 149.5 145.9] ref_us_per_probe=15.3 ref_runs=[15.3 15.3 15.3] ratio_import=9.55 ratio_n200=1.68 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=16.131s
# attempt 4, job 127053923380, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127053923380
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" clause_us_n200=228.9 clause_us_n1000=224.9 clause_us_n5000=224.9 max_min=1.02 ref_us_n200=131.1 ref_us_n1000=97.4 ref_us_n5000=250.3 ratio_n200=1.75 ratio_import=9.51 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" subjects=10000 clause_us_per_probe=151.4 clause_runs=[151.4 151.0 151.9] ref_us_per_probe=15.9 ref_runs=[15.9 16.0 15.8] ratio_import=9.51 ratio_n200=1.75 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=16.343s
# attempt 5, job 127054555284, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127054555284
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=233.0 clause_us_n1000=226.4 clause_us_n5000=229.5 max_min=1.03 ref_us_n200=158.0 ref_us_n1000=104.9 ref_us_n5000=268.7 ratio_n200=1.48 ratio_import=9.89 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=190.8 clause_runs=[192.5 189.2 190.8] ref_us_per_probe=19.3 ref_runs=[19.3 19.4 19.3] ratio_import=9.89 ratio_n200=1.48 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=18.758s
# attempt 6, job 127055037736, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055037736
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=232.5 clause_us_n1000=226.2 clause_us_n5000=232.0 max_min=1.03 ref_us_n200=152.9 ref_us_n1000=107.8 ref_us_n5000=266.3 ratio_n200=1.52 ratio_import=9.22 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=192.1 clause_runs=[195.6 192.1 190.7] ref_us_per_probe=20.8 ref_runs=[21.3 20.8 19.6] ratio_import=9.22 ratio_n200=1.52 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=19.133s
# attempt 7, job 127055451827, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055451827
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" clause_us_n200=126.8 clause_us_n1000=125.8 clause_us_n5000=125.8 max_min=1.01 ref_us_n200=84.1 ref_us_n1000=55.5 ref_us_n5000=139.6 ratio_n200=1.51 ratio_import=8.55 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" subjects=10000 clause_us_per_probe=78.5 clause_runs=[79.3 78.5 77.4] ref_us_per_probe=9.2 ref_runs=[9.1 9.2 9.7] ratio_import=8.55 ratio_n200=1.51 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=9.385s
# attempt 8, job 127055851276, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127055851276
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" clause_us_n200=227.5 clause_us_n1000=220.6 clause_us_n5000=222.6 max_min=1.03 ref_us_n200=128.4 ref_us_n1000=96.6 ref_us_n5000=251.4 ratio_n200=1.77 ratio_import=9.64 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="INTEL(R) XEON(R) PLATINUM 8573C" subjects=10000 clause_us_per_probe=149.3 clause_runs=[149.7 148.6 149.3] ref_us_per_probe=15.5 ref_runs=[15.5 15.4 15.5] ratio_import=9.64 ratio_n200=1.77 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=16.199s
# attempt 9, job 127056311690, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127056311690
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" clause_us_n200=128.6 clause_us_n1000=124.5 clause_us_n5000=121.8 max_min=1.06 ref_us_n200=87.9 ref_us_n1000=56.5 ref_us_n5000=142.9 ratio_n200=1.46 ratio_import=8.29 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" subjects=10000 clause_us_per_probe=81.1 clause_runs=[81.1 80.5 81.4] ref_us_per_probe=9.8 ref_runs=[9.8 9.8 9.7] ratio_import=8.29 ratio_n200=1.46 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=9.624s
# attempt 10, job 127056716926, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127056716926
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 9V74 80-Core Processor" clause_us_n200=183.2 clause_us_n1000=172.3 clause_us_n5000=165.6 max_min=1.11 ref_us_n200=121.6 ref_us_n1000=80.7 ref_us_n5000=210.3 ratio_n200=1.51 ratio_import=9.58 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 9V74 80-Core Processor" subjects=10000 clause_us_per_probe=128.6 clause_runs=[128.6 129.4 127.3] ref_us_per_probe=13.4 ref_runs=[13.4 13.4 13.6] ratio_import=9.58 ratio_n200=1.51 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=13.699s
# attempt 11, job 127057622460, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127057622460
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" clause_us_n200=125.9 clause_us_n1000=120.3 clause_us_n5000=117.7 max_min=1.07 ref_us_n200=88.0 ref_us_n1000=58.4 ref_us_n5000=141.7 ratio_n200=1.43 ratio_import=8.35 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 9V45 96-Core Processor" subjects=10000 clause_us_per_probe=79.6 clause_runs=[79.1 79.6 79.6] ref_us_per_probe=9.5 ref_runs=[9.7 9.5 9.5] ratio_import=8.35 ratio_n200=1.43 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=9.45s
# attempt 12, job 127058210643, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127058210643
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=229.0 clause_us_n1000=224.5 clause_us_n5000=226.1 max_min=1.02 ref_us_n200=152.2 ref_us_n1000=107.8 ref_us_n5000=267.7 ratio_n200=1.50 ratio_import=9.81 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=189.0 clause_runs=[192.6 187.3 189.0] ref_us_per_probe=19.3 ref_runs=[19.1 19.3 21.9] ratio_import=9.81 ratio_n200=1.50 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=18.704s
# attempt 13, job 127058793248, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127058793248
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=258.0 clause_us_n1000=228.7 clause_us_n5000=231.6 max_min=1.13 ref_us_n200=150.2 ref_us_n1000=105.0 ref_us_n5000=269.5 ratio_n200=1.72 ratio_import=9.78 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=191.2 clause_runs=[192.7 191.2 190.1] ref_us_per_probe=19.6 ref_runs=[21.0 19.6 19.3] ratio_import=9.78 ratio_n200=1.72 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=18.998s
# attempt 14, job 127059304692, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127059304692
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="Intel(R) Xeon(R) 6973P-C" clause_us_n200=167.4 clause_us_n1000=156.6 clause_us_n5000=154.3 max_min=1.08 ref_us_n200=102.0 ref_us_n1000=73.1 ref_us_n5000=181.4 ratio_n200=1.64 ratio_import=8.89 threshold_us=300 flat_below=1.5 over_threshold=false verdict=pass
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="Intel(R) Xeon(R) 6973P-C" subjects=10000 clause_us_per_probe=103.8 clause_runs=[103.1 103.8 106.3] ref_us_per_probe=11.7 ref_runs=[11.8 11.7 11.5] ratio_import=8.89 ratio_n200=1.64 threshold_us=175 over_threshold=false verdict=pass
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=12.332s
# attempt 15, job 127059751351, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127059751351
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=236.6 clause_us_n1000=228.6 clause_us_n5000=232.9 max_min=1.03 ref_us_n200=170.4 ref_us_n1000=104.8 ref_us_n5000=264.3 ratio_n200=1.39 ratio_import=9.92 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=193.5 clause_runs=[197.3 193.5 190.9] ref_us_per_probe=19.5 ref_runs=[20.0 19.5 19.0] ratio_import=9.92 ratio_n200=1.39 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=18.946s
# attempt 16, job 127060193703, https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547/job/127060193703
commit_check_budget_test.go:100: COMMIT-CHECK-BUDGET row=single_subject partitions=38 cpu="AMD EPYC 7763 64-Core Processor" clause_us_n200=233.5 clause_us_n1000=232.2 clause_us_n5000=230.4 max_min=1.01 ref_us_n200=153.8 ref_us_n1000=112.3 ref_us_n5000=308.2 ratio_n200=1.52 ratio_import=9.78 threshold_us=300 flat_below=1.5 over_threshold=false verdict=warn
commit_check_budget_test.go:101: COMMIT-CHECK-BUDGET row=import partitions=38 cpu="AMD EPYC 7763 64-Core Processor" subjects=10000 clause_us_per_probe=191.7 clause_runs=[192.2 190.6 191.7] ref_us_per_probe=19.6 ref_runs=[19.4 22.4 19.6] ratio_import=9.78 ratio_n200=1.52 threshold_us=175 over_threshold=true verdict=warn
commit_check_budget_test.go:102: COMMIT-CHECK-BUDGET wall_time=19.196s
```
