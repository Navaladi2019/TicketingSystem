| Tenant | Scenario | Query | Visible rows | p50 ms | p95 ms | max ms | Buffers (EXPLAIN) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| L (1M) | default/end-user | count exact | 1114 | 2013.6 | **2993.1** | 3499.7 | shared hit=1011226 |
| L (1M) | default/end-user | count capped 1001 | 1114 | 1515.5 | **3258.3** | 3711.0 | shared hit=912006 |
| L (1M) | default/end-user | page 1 | 1114 | 2325.2 | **3989.6** | 4130.0 | shared hit=1011231 dirtied=5 |
| L (1M) | default/end-user | page 20 | 1114 | 1950.3 | **3320.8** | 3376.5 | shared hit=1011703 |
| L (1M) | default/agent-3-teams | count exact | 40248 | 1619.2 | **2027.0** | 2106.2 | shared hit=1012029 |
| L (1M) | default/agent-3-teams | count capped 1001 | 40248 | 41.3 | **64.7** | 64.9 | shared hit=25604 |
| L (1M) | default/agent-3-teams | page 1 | 40248 | 1675.9 | **2223.0** | 2267.7 | shared hit=1012034 |
| L (1M) | default/agent-3-teams | page 20 | 40248 | 2386.2 | **4300.0** | 4672.1 | shared hit=1012034 |
| L (1M) | default/service-desk | count exact | 1000000 | 472.5 | **713.8** | 748.4 | shared hit=175716 |
| L (1M) | default/service-desk | count capped 1001 | 1000000 | 5.1 | **7.4** | 7.6 | shared hit=179 |
| L (1M) | default/service-desk | page 1 | 1000000 | 2184.3 | **3289.7** | 4000.8 | shared hit=175721 |
| L (1M) | default/service-desk | page 20 | 1000000 | 1677.6 | **2085.8** | 2273.6 | shared hit=175721 |
| L (1M) | share-heavy/partner | count exact | 51377 | 2541.3 | **3633.4** | 4131.8 | shared hit=1012549 |
| L (1M) | share-heavy/partner | count capped 1001 | 51377 | 162.7 | **288.3** | 330.2 | shared hit=21378 |
| L (1M) | share-heavy/partner | page 1 | 51377 | 4470.3 | **6204.2** | 6680.4 | shared hit=1012554 |
| L (1M) | share-heavy/partner | page 20 | 51377 | 4149.7 | **4826.3** | 4880.0 | shared hit=1012554 |
| M (100k) | default/end-user | count exact | 304 | 247.8 | **336.3** | 352.8 | shared hit=101189 |
| M (100k) | default/end-user | count capped 1001 | 304 | 232.0 | **274.1** | 279.1 | shared hit=101189 |
| M (100k) | default/end-user | page 1 | 304 | 237.8 | **266.7** | 279.1 | shared hit=101194 |
| M (100k) | default/agent-3-teams | count exact | 8220 | 259.4 | **368.3** | 393.8 | shared hit=101496 |
| M (100k) | default/agent-3-teams | count capped 1001 | 8220 | 36.4 | **48.1** | 48.6 | shared hit=13383 |
| M (100k) | default/agent-3-teams | page 1 | 8220 | 275.9 | **420.5** | 472.5 | shared hit=101501 |
| M (100k) | default/agent-3-teams | page 20 | 8220 | 273.5 | **494.1** | 546.6 | shared hit=101501 |
| M (100k) | default/service-desk | count exact | 100000 | 218.8 | **290.6** | 305.4 | shared hit=101083 |
| M (100k) | default/service-desk | count capped 1001 | 100000 | 5.1 | **6.8** | 7.4 | shared hit=1017 |
| M (100k) | default/service-desk | page 1 | 100000 | 434.2 | **546.2** | 580.8 | shared hit=101088 |
| M (100k) | default/service-desk | page 20 | 100000 | 497.0 | **712.7** | 750.8 | shared hit=101088 |
| M (100k) | share-heavy/partner | count exact | 5270 | 255.3 | **383.8** | 417.3 | shared hit=101243 |
| M (100k) | share-heavy/partner | count capped 1001 | 5270 | 56.4 | **76.1** | 85.1 | shared hit=19231 |
| M (100k) | share-heavy/partner | page 1 | 5270 | 327.1 | **720.9** | 805.0 | shared hit=101248 |
| M (100k) | share-heavy/partner | page 20 | 5270 | 281.6 | **726.8** | 861.2 | shared hit=101248 |
