| Cell | rpc4 count / page ms (count join) | rpc1.1 count / page ms (count join) |
| --- | --- | --- |
| text eq 40%, ACL | 31.9 / 9.1 (Merge) | 30.8 / 14.4 (Merge) |
| text eq 40%, no ACL | 13.5 / 3.3 (NL) | 27.6 / 4.1 (NL) |
| text eq 2%, ACL | 33.8 / 90.8 (Hash) | 22.0 / 64.1 (Hash) |
| text eq 2%, no ACL | 24.2 / 76.1 (Hash) | 36.0 / 125.3 (Hash) |
| text eq 0.25%, ACL | 16.3 / 15.0 (Hash) | 32.4 / 20.2 (Hash) |
| text eq 0.25%, no ACL | 11.6 / 19.9 (-) | 9.8 / 18.4 (-) |
| text eq 0.01%, ACL | 14.1 / 5.5 (Hash) | 18.4 / 5.1 (Hash) |
| text eq 0.01%, no ACL | 0.8 / 2.2 (-) | 1.0 / 2.6 (-) |
| text in (2 values) 40%, ACL | 108.0 / 8.5 (Hash+spill) | 91.3 / 8.2 (Hash+spill) |
| text in (2 values) 40%, no ACL | 16.5 / 2.3 (NL) | 27.6 / 3.6 (NL) |
| text in (2 values) 2%, ACL | 26.2 / 72.0 (Hash) | 23.9 / 51.6 (Hash) |
| text in (2 values) 2%, no ACL | 29.2 / 68.1 (Hash) | 75.2 / 164.3 (Hash) |
| text in (2 values) 0.25%, ACL | 18.4 / 15.3 (Hash) | 12.7 / 15.7 (Hash) |
| text in (2 values) 0.25%, no ACL | 8.1 / 12.5 (-) | 7.3 / 17.5 (-) |
| text in (2 values) 0.01%, ACL | 16.8 / 5.4 (Hash) | 21.6 / 6.2 (Hash) |
| text in (2 values) 0.01%, no ACL | 1.2 / 2.1 (-) | 1.9 / 3.1 (-) |
| num lt 40%, ACL | 98.7 / 7.1 (Hash+spill) | 79.8 / 12.0 (Hash+spill) |
| num lt 40%, no ACL | 18.0 / 2.7 (NL) | 17.8 / 3.9 (NL) |
| num lt 2%, ACL | 32.2 / 78.5 (Hash) | 29.5 / 51.8 (Hash) |
| num lt 2%, no ACL | 34.0 / 88.5 (Hash) | 24.4 / 78.0 (Hash) |
| num lt 0.25%, ACL | 21.0 / 15.5 (Hash) | 18.3 / 13.8 (Hash) |
| num lt 0.25%, no ACL | 9.0 / 15.5 (-) | 13.4 / 16.1 (-) |
| num lt 0.01%, ACL | 5.1 / 2.9 (Hash) | 2.9 / 7.5 (Hash) |
| num lt 0.01%, no ACL | 1.0 / 1.7 (-) | 1.9 / 3.0 (-) |
| ts range 40%, ACL | 114.4 / 10.9 (Hash+spill) | 97.5 / 11.3 (Hash+spill) |
| ts range 40%, no ACL | 30.9 / 4.2 (NL) | 23.3 / 3.6 (NL) |
| ts range 2%, ACL | 23.6 / 60.4 (Hash) | 36.1 / 87.3 (Hash) |
| ts range 2%, no ACL | 28.6 / 82.8 (Hash) | 40.2 / 106.2 (Hash) |
| ts range 0.25%, ACL | 17.0 / 14.5 (Hash) | 17.0 / 14.2 (Hash) |
| ts range 0.25%, no ACL | 9.7 / 20.6 (-) | 13.6 / 18.3 (-) |
| ts range 0.01%, ACL | 4.0 / 7.2 (Hash) | 4.3 / 7.8 (Hash) |
| ts range 0.01%, no ACL | 0.9 / 2.8 (-) | 0.8 / 1.4 (-) |
| bool eq 40%, ACL | 40.8 / 18.8 (Merge) | 40.0 / 23.2 (Merge) |
| bool eq 40%, no ACL | 19.7 / 3.4 (NL) | 23.1 / 4.3 (NL) |
| bool eq 2%, ACL | 27.1 / 85.0 (Hash) | 32.7 / 67.3 (Hash) |
| bool eq 2%, no ACL | 32.2 / 93.1 (Hash) | 35.1 / 76.0 (Hash) |
| bool eq 0.25%, ACL | 23.8 / 22.9 (Hash) | 26.3 / 22.9 (Hash) |
| bool eq 0.25%, no ACL | 7.2 / 16.8 (-) | 12.5 / 8.8 (-) |
| bool eq 0.01%, ACL | 9.9 / 12.2 (Hash) | 10.2 / 14.0 (Hash) |
| bool eq 0.01%, no ACL | 1.8 / 2.5 (-) | 2.2 / 4.2 (-) |
