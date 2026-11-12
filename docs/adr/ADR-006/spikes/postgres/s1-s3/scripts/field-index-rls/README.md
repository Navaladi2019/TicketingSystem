# scratch/field-index-rls (TICK-714, throwaway)

S1–S3 benchmark scripts for RFC-015 rev 1. Report: `docs/data/field-index-spike.md`.
Not a migration and not imported by anything. Runs inside the P3-01 bench
container (`bench/records-list/run.sh setup custom`), copied to `/bench/fi`.

| File | What |
| --- | --- |
| `10_leakproof.sql` | `pg_proc.proleakproof` catalogue check |
| `20_s1_data.sql` | S1/S2 fields at fixed selectivities; 400k tenant as one type |
| `21_s1_schema.sql` | rev 1 data layer as written (D2, D3, D5, D12, D19, D20) |
| `22_s1_populate.sql`, `22b_…_noterms.sql` | registry rows, initial copy, indexes, FKs, triggers |
| `23_split.sql` | finding 1: four per-type side tables + MCV statistics, trigger routed by type |
| `24_coerce_inline.sql` | D19 without function-level SET (A/B only) |
| `fq.sh` | EXPLAIN / pgbench driver (`FILTERS=` picks the filter file) |
| `s1_filters.txt`, `s1u_filters.txt`, `s2_filters.txt` | the measured predicates |
| `s1_load.sh`, `s1u_load.sh`, `s2_load.sh`, `s1_writes.sh`, `s1_ab.sh` | S1 / S2 runs |
| `s3.sh`, `s3_run.sh`, `races.sh` | flip, D17 barrier, D10 backfill, D11 removal, diff; R1/R3 |
| `rebuild_and_run.sh` | one-shot rebuild + R10 A/B + S3 |
