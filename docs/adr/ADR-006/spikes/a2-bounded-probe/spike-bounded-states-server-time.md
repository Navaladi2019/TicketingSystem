| Cell | bounded pre + count / page ms (count join) |
| --- | --- |
| analyze pk=3 (48%) | 18.0 + 19.0 / 7.2 (-) |
| analyze pk=4 (50%) | 13.7 + 17.5 / 8.2 (-) |
| analyze pk=1 (1%) | 7.4 + 22.6 / 66.1 (Hash) |
| churn5 pk=3 (48%) | 14.3 + 17.0 / 12.6 (-) |
| churn5 pk=4 (50%) | 12.2 + 14.9 / 7.0 (-) |
| churn5 pk=1 (1%) | 6.4 + 24.7 / 59.8 (Hash) |
| vacuum pk=3 (48%) | 22.2 + 26.8 / 11.8 (-) |
| vacuum pk=4 (50%) | 16.1 + 20.6 / 8.7 (-) |
| vacuum pk=1 (1%) | 2.0 + 15.1 / 48.9 (Hash) |
