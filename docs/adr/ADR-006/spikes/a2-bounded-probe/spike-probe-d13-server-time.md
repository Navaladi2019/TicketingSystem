| Cell | base count / page ms (count join) | spike count / page ms (count join) |
| --- | --- | --- |
| text eq 40%, ACL | 31.9 / 9.1 (Merge) | 23.8 / 7.8 (-) |
| text eq 40%, no ACL | 13.5 / 3.3 (NL) | 20.9 / 3.4 (NL) |
| text eq 2%, ACL | 33.8 / 90.8 (Hash) | 186.0 / 67.4 (-) |
| text eq 2%, no ACL | 24.2 / 76.1 (Hash) | 28.0 / 74.0 (Hash) |
| text eq 0.25%, ACL | 16.3 / 15.0 (Hash) | 138.8 / 15.4 (-) |
| text eq 0.25%, no ACL | 11.6 / 19.9 (-) | 8.5 / 14.1 (-) |
| text eq 0.01%, ACL | 14.1 / 5.5 (Hash) | 102.3 / 4.9 (-) |
| text eq 0.01%, no ACL | 0.8 / 2.2 (-) | 1.3 / 2.4 (-) |
| text in (2 values) 40%, ACL | 108.0 / 8.5 (Hash+spill) | 22.5 / 9.1 (-) |
| text in (2 values) 40%, no ACL | 16.5 / 2.3 (NL) | 25.9 / 2.7 (NL) |
| text in (2 values) 2%, ACL | 26.2 / 72.0 (Hash) | 225.3 / 57.7 (-) |
| text in (2 values) 2%, no ACL | 29.2 / 68.1 (Hash) | 19.8 / 67.9 (Hash) |
| text in (2 values) 0.25%, ACL | 18.4 / 15.3 (Hash) | 177.6 / 11.7 (-) |
| text in (2 values) 0.25%, no ACL | 8.1 / 12.5 (-) | 13.9 / 20.7 (-) |
| text in (2 values) 0.01%, ACL | 16.8 / 5.4 (Hash) | 179.3 / 7.0 (-) |
| text in (2 values) 0.01%, no ACL | 1.2 / 2.1 (-) | 1.9 / 2.8 (-) |
| num lt 40%, ACL | 98.7 / 7.1 (Hash+spill) | 26.5 / 10.7 (-) |
| num lt 40%, no ACL | 18.0 / 2.7 (NL) | 15.0 / 2.4 (NL) |
| num lt 2%, ACL | 32.2 / 78.5 (Hash) | 7669.2 / 49.3 (-) |
| num lt 2%, no ACL | 34.0 / 88.5 (Hash) | 34.1 / 84.7 (Hash) |
| num lt 0.25%, ACL | 21.0 / 15.5 (Hash) | 940.0 / 11.7 (-) |
| num lt 0.25%, no ACL | 9.0 / 15.5 (-) | 8.9 / 14.1 (-) |
| num lt 0.01%, ACL | 5.1 / 2.9 (Hash) | 150.4 / 5.1 (-) |
| num lt 0.01%, no ACL | 1.0 / 1.7 (-) | 1.0 / 2.0 (-) |
| ts range 40%, ACL | 114.4 / 10.9 (Hash+spill) | 27.3 / 11.7 (-) |
| ts range 40%, no ACL | 30.9 / 4.2 (NL) | 12.3 / 2.4 (NL) |
| ts range 2%, ACL | 23.6 / 60.4 (Hash) | 8342.3 / 59.3 (-) |
| ts range 2%, no ACL | 28.6 / 82.8 (Hash) | 33.7 / 79.6 (Hash) |
| ts range 0.25%, ACL | 17.0 / 14.5 (Hash) | 1070.2 / 16.9 (-) |
| ts range 0.25%, no ACL | 9.7 / 20.6 (-) | 9.5 / 14.2 (-) |
| ts range 0.01%, ACL | 4.0 / 7.2 (Hash) | 136.2 / 3.7 (-) |
| ts range 0.01%, no ACL | 0.9 / 2.8 (-) | 1.2 / 2.0 (-) |
| bool eq 40%, ACL | 40.8 / 18.8 (Merge) | 18.8 / 12.1 (-) |
| bool eq 40%, no ACL | 19.7 / 3.4 (NL) | 22.3 / 5.0 (NL) |
| bool eq 2%, ACL | 27.1 / 85.0 (Hash) | 98.1 / 64.7 (-) |
| bool eq 2%, no ACL | 32.2 / 93.1 (Hash) | 66.3 / 181.6 (Hash) |
| bool eq 0.25%, ACL | 23.8 / 22.9 (Hash) | 178.4 / 23.1 (-) |
| bool eq 0.25%, no ACL | 7.2 / 16.8 (-) | 8.2 / 14.5 (-) |
| bool eq 0.01%, ACL | 9.9 / 12.2 (Hash) | 90.9 / 9.5 (-) |
| bool eq 0.01%, no ACL | 1.8 / 2.5 (-) | 1.0 / 2.2 (-) |
