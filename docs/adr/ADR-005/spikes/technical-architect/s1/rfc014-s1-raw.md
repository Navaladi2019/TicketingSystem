# RFC-014 spike S1 raw output (2026-11-09/10, ta-s1-pg, PG 16.10, pgx v5.11.0)

## Quiet host, 8 clients x2 reps + 1 client
```
mode=L0 clients=8 delay_oneway=0s creates=2455 errors=0 rate=122.8/s
  transport SELECT 1 p50/p95 3.04/5.12 ms; bare 1-row commit txn p50/p95 6.16/8.06 ms
  create  p50/p95/p99  48.80 / 170.62 / 244.43 ms
  wait    p50/p95/p99  26.77 / 150.31 / 221.74 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 4.25 / 5.80 ms  [n=2455]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 7.97 / 9.96 ms [n=2450]
[load] multica-postgres-1 7.46%
mode=L1 clients=8 delay_oneway=0s creates=2840 errors=0 rate=142.0/s
  transport SELECT 1 p50/p95 1.74/2.92 ms; bare 1-row commit txn p50/p95 5.01/6.99 ms
  create  p50/p95/p99  41.09 / 141.56 / 198.57 ms
  wait    p50/p95/p99  18.37 / 119.70 / 172.90 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 2.34 / 3.45 ms  [n=2840]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 6.98 / 9.75 ms [n=2836]
[load] multica-backend-1 0.96%
mode=L2 clients=8 delay_oneway=0s creates=2862 errors=0 rate=143.1/s
  transport SELECT 1 p50/p95 1.75/3.16 ms; bare 1-row commit txn p50/p95 4.67/6.97 ms
  create  p50/p95/p99  46.48 / 120.16 / 266.20 ms
  wait    p50/p95/p99  6.03 / 30.59 / 50.83 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.06 / 0.14 ms  [n=2862]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 5.99 / 10.83 ms [n=2859]
[load] multica-postgres-1 2.25%
mode=L2s clients=8 delay_oneway=0s creates=2314 errors=0 rate=115.7/s
  transport SELECT 1 p50/p95 7.66/16.29 ms; bare 1-row commit txn p50/p95 12.17/20.36 ms
  create  p50/p95/p99  62.49 / 113.96 / 196.23 ms
  wait    p50/p95/p99  0.31 / 0.99 / 1.53 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.05 / 0.12 ms  [n=2314]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 6.13 / 24.98 ms [n=2313]
[load] multica-backend-1 0.90%
mode=L2p clients=8 delay_oneway=0s creates=2685 errors=0 rate=134.2/s
  transport SELECT 1 p50/p95 3.41/7.49 ms; bare 1-row commit txn p50/p95 7.40/12.88 ms
  create  p50/p95/p99  57.12 / 87.11 / 101.07 ms
  wait    p50/p95/p99  3.29 / 21.80 / 37.40 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.07 / 0.15 ms  [n=2685]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 6.75 / 14.53 ms [n=2684]
[load] multica-backend-1 0.48%
mode=L0 clients=8 delay_oneway=0s creates=2049 errors=0 rate=102.5/s
  transport SELECT 1 p50/p95 3.07/6.08 ms; bare 1-row commit txn p50/p95 6.53/10.93 ms
  create  p50/p95/p99  56.12 / 207.23 / 310.12 ms
  wait    p50/p95/p99  29.07 / 178.68 / 278.31 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 5.40 / 8.02 ms  [n=2049]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 9.37 / 12.80 ms [n=2045]
[load] multica-backend-1 0.27%
mode=L1 clients=8 delay_oneway=0s creates=2980 errors=0 rate=149.0/s
  transport SELECT 1 p50/p95 1.68/2.58 ms; bare 1-row commit txn p50/p95 4.49/6.04 ms
  create  p50/p95/p99  39.43 / 132.58 / 205.86 ms
  wait    p50/p95/p99  17.40 / 108.46 / 182.72 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 2.15 / 3.37 ms  [n=2980]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 6.33 / 9.77 ms [n=2977]
[load] multica-backend-1 0.90%
mode=L2 clients=8 delay_oneway=0s creates=3591 errors=0 rate=179.6/s
  transport SELECT 1 p50/p95 1.74/2.89 ms; bare 1-row commit txn p50/p95 4.59/6.62 ms
  create  p50/p95/p99  41.59 / 69.49 / 95.94 ms
  wait    p50/p95/p99  7.10 / 36.16 / 59.84 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.06 / 0.13 ms  [n=3591]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 5.71 / 7.41 ms [n=3590]
[load] multica-backend-1 1.08%
mode=L2s clients=8 delay_oneway=0s creates=5166 errors=0 rate=258.3/s
  transport SELECT 1 p50/p95 1.71/2.89 ms; bare 1-row commit txn p50/p95 4.57/6.79 ms
  create  p50/p95/p99  29.28 / 39.45 / 48.01 ms
  wait    p50/p95/p99  0.25 / 0.84 / 1.20 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.04 / 0.09 ms  [n=5166]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 2.58 / 11.88 ms [n=5165]
[load] multica-backend-1 0.51%
mode=L2p clients=8 delay_oneway=0s creates=2681 errors=0 rate=134.1/s
  transport SELECT 1 p50/p95 2.75/16.92 ms; bare 1-row commit txn p50/p95 6.20/53.03 ms
  create  p50/p95/p99  47.82 / 132.54 / 217.19 ms
  wait    p50/p95/p99  5.41 / 50.80 / 113.00 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.06 / 0.12 ms  [n=2681]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 5.75 / 14.88 ms [n=2680]
[load] multica-backend-1 0.80%
mode=L0 clients=1 delay_oneway=0s creates=689 errors=0 rate=45.9/s
  transport SELECT 1 p50/p95 1.70/2.66 ms; bare 1-row commit txn p50/p95 4.06/6.20 ms
  create  p50/p95/p99  19.97 / 31.52 / 61.22 ms
  wait    p50/p95/p99  0.15 / 0.27 / 0.42 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 3.77 / 6.61 ms  [n=689]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 20.05 / 31.38 ms [n=688]
mode=L1 clients=1 delay_oneway=0s creates=1211 errors=0 rate=80.7/s
  transport SELECT 1 p50/p95 2.64/6.41 ms; bare 1-row commit txn p50/p95 5.96/9.30 ms
  create  p50/p95/p99  12.10 / 15.33 / 16.98 ms
  wait    p50/p95/p99  0.10 / 0.15 / 0.30 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 1.08 / 1.53 ms  [n=1211]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 12.06 / 15.37 ms [n=1210]
mode=L2 clients=1 delay_oneway=0s creates=789 errors=0 rate=52.6/s
  transport SELECT 1 p50/p95 1.73/2.95 ms; bare 1-row commit txn p50/p95 4.54/6.94 ms
  create  p50/p95/p99  18.50 / 23.61 / 27.83 ms
  wait    p50/p95/p99  0.11 / 0.19 / 0.38 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.02 / 0.03 ms  [n=789]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 18.57 / 23.74 ms [n=788]
mode=L2s clients=1 delay_oneway=0s creates=1029 errors=0 rate=68.6/s
  transport SELECT 1 p50/p95 1.14/2.32 ms; bare 1-row commit txn p50/p95 3.56/5.17 ms
  create  p50/p95/p99  14.24 / 18.92 / 21.51 ms
  wait    p50/p95/p99  0.12 / 0.19 / 0.35 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.02 / 0.03 ms  [n=1029]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 14.19 / 18.77 ms [n=1028]
```
## Contended host (nava-aclperf-pg-601 at 85-233% CPU) - for reference only
```
mode=L0 clients=8 delay_oneway=0s creates=2791 errors=0 rate=139.6/s
  transport SELECT 1 p50/p95 1.66/2.64 ms; bare 1-row commit txn p50/p95 3.99/5.13 ms
  create  p50/p95/p99  42.01 / 150.10 / 228.62 ms
  wait    p50/p95/p99  23.93 / 130.42 / 207.56 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 3.27 / 4.52 ms  [n=2791]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 7.04 / 8.39 ms [n=2786]
mode=L1 clients=8 delay_oneway=0s creates=1759 errors=0 rate=88.0/s
  transport SELECT 1 p50/p95 2.79/6.05 ms; bare 1-row commit txn p50/p95 6.27/9.76 ms
  create  p50/p95/p99  59.97 / 225.26 / 520.59 ms
  wait    p50/p95/p99  21.85 / 177.65 / 435.49 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 3.10 / 6.07 ms  [n=1759]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 9.05 / 21.16 ms [n=1755]
mode=L2 clients=8 delay_oneway=0s creates=1197 errors=0 rate=59.9/s
  transport SELECT 1 p50/p95 14.88/85.96 ms; bare 1-row commit txn p50/p95 23.63/284.63 ms
  create  p50/p95/p99  113.92 / 281.31 / 394.71 ms
  wait    p50/p95/p99  0.71 / 14.85 / 26.19 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.07 / 0.15 ms  [n=1197]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 7.88 / 60.14 ms [n=1196]
mode=L2s clients=8 delay_oneway=0s creates=820 errors=0 rate=41.0/s
  transport SELECT 1 p50/p95 20.00/90.42 ms; bare 1-row commit txn p50/p95 32.44/300.15 ms
  create  p50/p95/p99  166.89 / 401.08 / 509.50 ms
  wait    p50/p95/p99  0.33 / 1.20 / 1.79 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.05 / 0.11 ms  [n=820]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 11.26 / 111.49 ms [n=819]
mode=L2p clients=8 delay_oneway=0s creates=1631 errors=0 rate=81.5/s
  transport SELECT 1 p50/p95 5.08/19.25 ms; bare 1-row commit txn p50/p95 12.01/126.40 ms
  create  p50/p95/p99  76.33 / 202.03 / 323.19 ms
  wait    p50/p95/p99  2.25 / 22.72 / 48.40 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.07 / 0.16 ms  [n=1631]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 7.30 / 39.53 ms [n=1630]
mode=L0 clients=8 delay_oneway=0s creates=1309 errors=0 rate=65.5/s
  transport SELECT 1 p50/p95 11.08/28.36 ms; bare 1-row commit txn p50/p95 16.38/36.68 ms
  create  p50/p95/p99  80.14 / 331.20 / 510.67 ms
  wait    p50/p95/p99  28.73 / 279.42 / 461.90 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 8.59 / 17.39 ms  [n=1309]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 14.02 / 24.77 ms [n=1304]
mode=L1 clients=8 delay_oneway=0s creates=1673 errors=0 rate=83.7/s
  transport SELECT 1 p50/p95 5.69/20.06 ms; bare 1-row commit txn p50/p95 10.13/29.56 ms
  create  p50/p95/p99  80.68 / 200.70 / 302.93 ms
  wait    p50/p95/p99  16.75 / 142.52 / 245.94 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 3.44 / 8.21 ms  [n=1673]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 10.14 / 20.87 ms [n=1670]
mode=L2 clients=8 delay_oneway=0s creates=2081 errors=0 rate=104.0/s
  transport SELECT 1 p50/p95 4.92/18.75 ms; bare 1-row commit txn p50/p95 8.86/19.92 ms
  create  p50/p95/p99  73.78 / 117.78 / 147.03 ms
  wait    p50/p95/p99  2.75 / 17.20 / 31.48 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.07 / 0.15 ms  [n=2081]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 7.05 / 26.13 ms [n=2079]
mode=L2s clients=8 delay_oneway=0s creates=3243 errors=0 rate=162.2/s
  transport SELECT 1 p50/p95 3.66/9.42 ms; bare 1-row commit txn p50/p95 7.80/13.41 ms
  create  p50/p95/p99  48.66 / 68.72 / 81.94 ms
  wait    p50/p95/p99  0.29 / 0.97 / 1.97 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.05 / 0.13 ms  [n=3243]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 4.04 / 20.25 ms [n=3242]
mode=L2p clients=8 delay_oneway=0s creates=3031 errors=0 rate=151.6/s
  transport SELECT 1 p50/p95 2.55/4.47 ms; bare 1-row commit txn p50/p95 6.35/8.95 ms
  create  p50/p95/p99  51.00 / 72.03 / 88.03 ms
  wait    p50/p95/p99  4.30 / 25.86 / 42.92 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.07 / 0.17 ms  [n=3031]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 6.62 / 9.43 ms [n=3030]
mode=L0 clients=1 delay_oneway=0s creates=968 errors=0 rate=64.5/s
  transport SELECT 1 p50/p95 1.17/1.85 ms; bare 1-row commit txn p50/p95 3.94/5.31 ms
  create  p50/p95/p99  15.24 / 18.18 / 20.46 ms
  wait    p50/p95/p99  0.12 / 0.21 / 0.36 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 2.80 / 3.67 ms  [n=968]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 15.27 / 18.06 ms [n=967]
mode=L1 clients=1 delay_oneway=0s creates=867 errors=0 rate=57.8/s
  transport SELECT 1 p50/p95 1.16/1.80 ms; bare 1-row commit txn p50/p95 3.94/5.20 ms
  create  p50/p95/p99  17.12 / 20.17 / 22.37 ms
  wait    p50/p95/p99  0.14 / 0.26 / 0.34 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 1.63 / 2.31 ms  [n=867]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 17.03 / 20.18 ms [n=866]
mode=L2 clients=1 delay_oneway=0s creates=629 errors=0 rate=41.9/s
  transport SELECT 1 p50/p95 1.75/2.50 ms; bare 1-row commit txn p50/p95 5.26/6.83 ms
  create  p50/p95/p99  23.46 / 28.67 / 31.20 ms
  wait    p50/p95/p99  0.13 / 0.23 / 0.33 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.02 / 0.04 ms  [n=629]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 23.41 / 28.71 ms [n=628]
mode=L2s clients=1 delay_oneway=0s creates=566 errors=0 rate=37.7/s
  transport SELECT 1 p50/p95 1.80/2.86 ms; bare 1-row commit txn p50/p95 5.46/6.99 ms
  create  p50/p95/p99  26.33 / 33.10 / 39.83 ms
  wait    p50/p95/p99  0.21 / 0.41 / 0.63 ms
  hold (acq->commit_ts, excl WAL flush) p50/p95 0.03 / 0.08 ms  [n=566]
  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 26.42 / 32.80 ms [n=565]
```
## A4 crash check (L2s, wal_writer_delay=10s, wal_writer_flush_after=1GB)
```
=== crash iteration 1 (port 59528)
  CRASH: docker kill -s KILL ta-s1-pg at 2026-11-09T23:59:44.8166483+05:30 -> ta-s1-pg <nil> (max allocated so far 26548)
mode=L2s crash run: acked creates=1176 errors=15615 max_acked_n=26546 max_allocated_n=26548 first_error="claim: FATAL: terminating connection due to unexpected postmaster exit (SQLSTATE 57P01)"
2026-11-09 18:29:50.473 UTC [29] LOG:  database system was not properly shut down; automatic recovery in progress
2026-11-09 18:29:50.482 UTC [29] LOG:  redo starts at 0/3C48CE8
2026-11-09 18:29:51.385 UTC [29] LOG:  redo done at 0/971E748 system usage: CPU: user: 0.73 s, system: 0.16 s, elapsed: 0.90 s
2026-11-09 18:29:51.697 UTC [1] LOG:  database system is ready to accept connections
verify: counter next-1=26548 max_record_n=26546 records=26546 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
mode=L2s clients=8 delay_oneway=0s creates=886 errors=0 rate=177.2/s
  transport SELECT 1 p50/p95 2.31/3.05 ms; bare 1-row commit txn p50/p95 5.38/7.20 ms
verify: counter next-1=27437 max_record_n=27437 records=27435 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
=== crash iteration 2 (port 55034)
  CRASH: docker kill -s KILL ta-s1-pg at 2026-11-10T00:00:16.4203736+05:30 -> ta-s1-pg <nil> (max allocated so far 28090)
mode=L2s crash run: acked creates=650 errors=13714 max_acked_n=28087 max_allocated_n=28090 first_error="alloc: unexpected EOF"
2026-11-09 18:30:23.724 UTC [29] LOG:  database system was not properly shut down; automatic recovery in progress
2026-11-09 18:30:23.733 UTC [29] LOG:  redo starts at 0/971E7E8
2026-11-09 18:30:23.849 UTC [29] LOG:  redo done at 0/A2BE9E8 system usage: CPU: user: 0.09 s, system: 0.01 s, elapsed: 0.11 s
2026-11-09 18:30:24.051 UTC [1] LOG:  database system is ready to accept connections
verify: counter next-1=28087 max_record_n=28087 records=28085 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
mode=L2s clients=8 delay_oneway=0s creates=459 errors=0 rate=91.8/s
  transport SELECT 1 p50/p95 8.05/16.30 ms; bare 1-row commit txn p50/p95 12.54/21.38 ms
verify: counter next-1=28547 max_record_n=28547 records=28545 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
=== crash iteration 3 (port 52609)
  CRASH: docker kill -s KILL ta-s1-pg at 2026-11-10T00:00:49.7847236+05:30 -> ta-s1-pg <nil> (max allocated so far 29639)
mode=L2s crash run: acked creates=1088 errors=16113 max_acked_n=29635 max_allocated_n=29639 first_error="write: FATAL: terminating connection due to unexpected postmaster exit (SQLSTATE 57P01)"
2026-11-09 18:30:56.970 UTC [29] LOG:  database system was not properly shut down; automatic recovery in progress
2026-11-09 18:30:56.981 UTC [29] LOG:  redo starts at 0/A2BEA88
2026-11-09 18:30:57.115 UTC [29] LOG:  redo done at 0/AE75688 system usage: CPU: user: 0.12 s, system: 0.01 s, elapsed: 0.13 s
2026-11-09 18:30:57.329 UTC [1] LOG:  database system is ready to accept connections
verify: counter next-1=29637 max_record_n=29636 records=29634 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
mode=L2s clients=8 delay_oneway=0s creates=837 errors=0 rate=167.4/s
  transport SELECT 1 p50/p95 3.43/6.96 ms; bare 1-row commit txn p50/p95 7.29/10.50 ms
verify: counter next-1=30482 max_record_n=30482 records=30479 duplicate_numbers=0 invariant(max_record_n<=next-1 && no dups)=true
```
