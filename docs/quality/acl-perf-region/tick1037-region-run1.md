# TICK-1037 region rows (A2-11)

## Sets (as each row ran)

| Set | Type | \|S\| | m | N̂_e | K* | Shape (A2-5) |
| --- | --- | --- | --- | --- | --- | --- |
| area (\|S\| 10) | ticket | 10 | 220 | 400000 | 3613 | C3 (m <= K*) |
| district (\|S\| 101) | ticket | 101 | 3379 | 400000 | 3613 | C3 (m <= K*) |
| region (\|S\| 1,011) | ticket | 1011 | 37930 | 400000 | 3613 | W0 (m > K*) |
| root (\|S\| 10,111) | ticket | 10111 | 380048 | 400000 | 3613 | W0 (m > K*) |
| district hot | ticket | 101 | 22996 | 400000 | 3613 | W0 (m > K*) |
| district thin | ticket | 101 | 600 | 400000 | 3613 | C3 (m <= K*) |
| uniform m=1001 | ticket | 101 | 1001 | 400000 | 3613 | C3 (m <= K*) |
| uniform m=3000 | ticket | 101 | 3000 | 400000 | 3613 | C3 (m <= K*) |
| uniform m=4500 | ticket | 101 | 4500 | 400000 | 3613 | W0 (m > K*) |
| uniform m=10000 | ticket | 101 | 10000 | 400000 | 3613 | W0 (m > K*) |
| bench_rare at root | bench_rare | 10111 | 500 | 500 | 1000 | A2-5a rare: W0, no probe |
| bench_conc, regions 0-8 (\|S\| 9,099) | bench_conc | 9099 | 0 | 20000 | 1000 | C3 (m <= K*) |

## Rows: handler p95 against 150 ms, paired base beside it (1 client)

| Row | Attempt | n | p50 | **p95** | Base n | Base p50 | Base p95 | Δp95 vs base | Floor p50 before / after | Gate | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| region: area (\|S\| 10), agents | 1 | 686 | 49.70 | **68.26** | 686 | 34.20 | 51.12 | +17.14 | 1.71 / 1.37 |  | pass |
| region: area (\|S\| 10), count-only | 1 | 680 | 49.18 | **72.32** | 679 | 35.29 | 52.83 | +19.49 | 1.83 / 1.13 |  | pass |
| region: district (\|S\| 101), agents | 1 | 559 | 66.38 | **85.94** | 559 | 37.99 | 52.02 | +33.91 | 1.11 / 2.00 |  | pass |
| region: district (\|S\| 101), count-only | 1 | 573 | 64.57 | **86.62** | 573 | 36.99 | 52.86 | +33.76 | 2.11 / 1.82 |  | pass |
| region: region (\|S\| 1,011), agents | 1 | 625 | 54.74 | **72.01** | 624 | 37.00 | 52.78 | +19.23 | 1.70 / 6.75 | floor p50 1.70 / 6.75 > 3 ms | void |
| region: region (\|S\| 1,011), agents | 2 | 680 | 50.40 | **67.56** | 679 | 34.19 | 50.73 | +16.83 | 2.38 / 1.58 |  | pass |
| region: region (\|S\| 1,011), count-only | 1 | 635 | 52.92 | **77.10** | 635 | 36.00 | 56.30 | +20.80 | 1.29 / 3.19 | floor p50 1.29 / 3.19 > 3 ms | void |
| region: region (\|S\| 1,011), count-only | 2 | 518 | 67.71 | **87.29** | 517 | 46.82 | 62.73 | +24.56 | 3.21 / 2.89 | floor p50 3.21 / 2.89 > 3 ms | void |
| region: region (\|S\| 1,011), count-only | 3 | 569 | 61.01 | **81.90** | 569 | 41.54 | 59.41 | +22.49 | 2.99 / 1.83 |  | pass |
| region: root (\|S\| 10,111), agents | 1 | 619 | 54.49 | **79.54** | 618 | 37.20 | 57.83 | +21.71 | 2.02 / 1.50 |  | pass |
| region: root (\|S\| 10,111), count-only | 1 | 668 | 52.23 | **68.62** | 667 | 35.37 | 49.98 | +18.63 | 1.33 / 1.68 |  | pass |
| region: district hot, agents | 1 | 706 | 49.03 | **65.21** | 705 | 33.69 | 48.68 | +16.54 | 1.18 / 1.69 |  | pass |
| region: district hot, count-only | 1 | 712 | 49.25 | **64.46** | 711 | 33.11 | 47.98 | +16.48 | 1.44 / 1.59 |  | pass |
| region: district thin, agents | 1 | 744 | 46.54 | **64.68** | 744 | 31.30 | 46.65 | +18.04 | 1.15 / 1.08 |  | pass |
| region: district thin, count-only | 1 | 740 | 45.66 | **65.49** | 740 | 31.07 | 47.05 | +18.45 | 1.12 / 1.27 |  | pass |
| region: uniform m=1001, agents | 1 | 724 | 48.53 | **66.44** | 724 | 32.09 | 44.94 | +21.50 | 1.10 / 1.08 |  | pass |
| region: uniform m=1001, count-only | 1 | 709 | 49.42 | **67.76** | 709 | 32.65 | 47.31 | +20.44 | 1.15 / 1.79 |  | pass |
| region: uniform m=3000, agents | 1 | 615 | 60.37 | **79.92** | 615 | 33.44 | 47.61 | +32.31 | 1.15 / 1.66 |  | pass |
| region: uniform m=3000, count-only | 1 | 590 | 61.84 | **86.83** | 590 | 35.39 | 52.57 | +34.27 | 1.25 / 1.69 |  | pass |
| region: uniform m=4500, agents | 1 | 730 | 47.57 | **65.65** | 729 | 31.64 | 47.82 | +17.83 | 1.11 / 1.69 |  | pass |
| region: uniform m=4500, count-only | 1 | 640 | 57.88 | **76.71** | 640 | 33.18 | 47.89 | +28.82 | 1.28 / 1.70 |  | pass |
| region: uniform m=10000, agents | 1 | 738 | 46.88 | **64.51** | 737 | 31.64 | 46.91 | +17.60 | 1.14 / 1.13 |  | pass |
| region: uniform m=10000, count-only | 1 | 821 | 44.20 | **63.48** | 820 | 24.61 | 40.57 | +22.91 | 1.11 / 1.11 |  | pass |
| region: bench_rare at root, agents | 1 | 1091 | 31.56 | **44.98** | 1091 | 19.63 | 29.84 | +15.14 | 1.55 / 1.18 |  | pass |
| region: bench_rare at root, count-only | 1 | 1170 | 30.42 | **40.84** | 1169 | 19.37 | 27.26 | +13.58 | 0.77 / 1.14 |  | pass |
| region: bench_conc, regions 0-8 (\|S\| 9,099), agents | 1 | 492 | 68.19 | **161.57** | 491 | 27.27 | 87.33 | +74.24 | 0.64 / 1.16 |  | **FAIL** |
| region: bench_conc, regions 0-8 (\|S\| 9,099), count-only | 1 | 455 | 99.14 | **123.53** | 454 | 30.82 | 44.65 | +78.87 | 1.36 / 1.26 |  | pass |

## Host

```
### 2026-11-22T07:56:45Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
```

## docker ps every 10 s during the run (nava-it-/nava-ci-/nava-aclperf- only; empty lines = none)

246 samples, 0 with a foreign container.

## bench_conc per-statement times (ms, from the raw report)

### region: bench_conc, regions 0-8 (|S| 9,099), agents, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f6a98994-5926-4403-94b5-f7ca32d83e74', true)` | 492 | 1.90 | 8.78 | 15.54 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 492 | 2.25 | 8.26 | 13.32 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 492 | 1.90 | 7.56 | 14.50 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 492 | 2.03 | 9.32 | 15.99 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 492 | 6.95 | 14.69 | 21.32 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 492 | 1.97 | 6.49 | 14.98 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 492 | 20.42 | 39.15 | 56.62 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 492 | 3.89 | 12.78 | 19.17 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 492 | 20.84 | 43.98 | 63.76 |
| 10 | `commit` | 492 | 1.71 | 8.12 | 13.45 |

### region: bench_conc, regions 0-8 (|S| 9,099), count-only, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f6a98994-5926-4403-94b5-f7ca32d83e74', true)` | 455 | 2.24 | 4.21 | 4.82 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 455 | 2.50 | 4.47 | 5.88 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 455 | 2.10 | 3.98 | 4.98 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 455 | 2.34 | 4.03 | 5.22 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 455 | 7.31 | 10.33 | 11.76 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 455 | 2.28 | 4.22 | 5.12 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 455 | 25.39 | 33.81 | 38.43 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 455 | 28.30 | 37.36 | 42.50 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 455 | 20.13 | 26.55 | 29.44 |
| 10 | `commit` | 455 | 2.11 | 3.68 | 4.65 |

