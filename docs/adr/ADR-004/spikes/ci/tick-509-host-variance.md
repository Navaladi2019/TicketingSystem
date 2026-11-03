# TICK-509 CI host variance: raw commit-check budget lines

Source: GitHub Actions job `commit-check-budget` (`postgres:16.10`, `ubuntu-latest`, nproc 4, `TestCommitCheckBudget` alone on the runner, 38 partitions). Retrieved with `gh run view <id> --log` on 2026-11-03 for ADR-004-E4. The CPU model was not logged by these runs; that gap is ADR-004-E4 Decision 3.

## Run 41676703902 (push, SHA 25ca403): E3 verdict pass

```
commit-check-budget (postgres:16.10): cluster healthy after 63s; nproc 4
COMMIT-CHECK-BUDGET row=single_subject partitions=38 clause_us_n200=170.4 clause_us_n1000=170.6 clause_us_n5000=165.1 max_min=1.03 ref_us_n200=117.5 ref_us_n1000=79.5 ref_us_n5000=208.7 ratio_n200=1.45 ratio_import=9.76 ceiling_us=300 flat_below=1.5 verdict=pass
COMMIT-CHECK-BUDGET row=import partitions=38 subjects=10000 clause_us_per_probe=129.5 clause_runs=[130.5 129.5 127.1] ref_us_per_probe=13.3 ref_runs=[13.3 13.3 13.3] ratio_import=9.76 ratio_n200=1.45 ceiling_us=175 verdict=pass
COMMIT-CHECK-BUDGET wall_time=13.403s
```

## Run 41676713796 (pull_request, SHA 25ca403): E3 verdict rerun (job red)

```
commit-check-budget (postgres:16.10): cluster healthy after 70s; nproc 4
COMMIT-CHECK-BUDGET row=single_subject partitions=38 clause_us_n200=233.2 clause_us_n1000=234.6 clause_us_n5000=228.8 max_min=1.03 ref_us_n200=151.4 ref_us_n1000=108.5 ref_us_n5000=274.5 ratio_n200=1.54 ratio_import=9.98 ceiling_us=300 flat_below=1.5 verdict=pass
COMMIT-CHECK-BUDGET row=import partitions=38 subjects=10000 clause_us_per_probe=190.9 clause_runs=[190.7 191.0 190.9] ref_us_per_probe=19.1 ref_runs=[18.8 19.6 19.1] ratio_import=9.98 ratio_n200=1.54 ceiling_us=175 verdict=rerun
COMMIT-CHECK-BUDGET wall_time=18.829s
```

## Run 41673342312 (SHA 2073e3c): E3 verdict pass

```
commit-check-budget (postgres:16.10): cluster healthy after 85s; nproc 4
COMMIT-CHECK-BUDGET row=single_subject partitions=38 clause_us_n200=170.5 clause_us_n1000=163.6 clause_us_n5000=164.8 max_min=1.04 ref_us_n200=133.5 ref_us_n1000=82.1 ref_us_n5000=212.0 ratio_n200=1.28 ratio_import=9.75 ceiling_us=300 flat_below=1.5 verdict=pass
COMMIT-CHECK-BUDGET row=import partitions=38 subjects=10000 clause_us_per_probe=127.7 clause_runs=[127.4 127.7 129.3] ref_us_per_probe=13.1 ref_runs=[13.1 13.1 13.3] ratio_import=9.75 ratio_n200=1.28 ceiling_us=175 verdict=pass
```

## Derived (same SHA, slow host / fast host)

| Quantity | Fast | Slow | Slow / fast |
|---|---|---|---|
| Clause per call, N = 200 | 170.4 | 233.2 | 1.37 |
| Ref per call, N = 200 | 117.5 | 151.4 | 1.29 |
| Clause import | 129.5 | 190.9 | 1.47 |
| Ref import | 13.3 | 19.1 | 1.44 |
| Ratio N = 200 | 1.45 | 1.54 | 1.06 |
| Ratio import | 9.76 | 9.98 | 1.02 |
| Flatness (max/min) | 1.03 | 1.03 | 1.00 |

Headroom under E3 (absolute): import 175 / 129.5 = 1.35× on the fast host, below the ≈1.44–1.47× spread; per call 300 / 234.6 = 1.28× on the slow host.
Headroom under E4 (within-run): import ratio 15 / 9.98 = 1.50×; per-call ratio 5 / 1.54 = 3.2×; flatness 1.5 / 1.04 = 1.44×.
