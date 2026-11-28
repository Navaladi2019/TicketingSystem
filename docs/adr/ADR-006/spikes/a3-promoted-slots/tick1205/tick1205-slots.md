> Archived copy of `docs/data/field-index-measured.md` § "Arm (b) write path" from PR #995 at commit `0b733511` (TICK-1205). The 3.4 MB plan file `tick1205-slots-explain.md` stays in `docs/data/field-index-measured/` on that PR and is not duplicated here.

## Arm (b) write path: promoted filter slots, ADR-006-A3 draft (TICK-1205, 2026-11-28)

TICK-1205 (S21b, T-A1-6b) answers A3-10 items 2 to 4 and the Technical Architect's findings 6 to 8 on TICK-1039. It measures arm (b) in the slot form that the draft [ADR-006-A3](https://github.com/Navaladi2019/TicketingSystem/pull/993) proposes. It costs the write path, the DDL and the sizes against D7 on the same seed, and it runs S-b1 to S-b3 and S-c1 to S-c3. The DDL is scratch only. Nothing is in `db/migrations/`. RLS, CI-2 and D7's `EXISTS` are unchanged.

**Host:** the gate passed. No `nava`, `nava-*`, `nava-it-*` or bench containers were up, only the harness's own `nava-fiperf-pg-61858` and the `multica-*` platform. `multica-postgres-1` exceeded 10% of one core in 93 of 594 samples, with a peak of 37.3% of one core at 06:11Z, which is 2.3% of the VM's 16 CPUs. `multica-backend-1` peaked at 17.4% and `multica-frontend-1` at 7.8%. The transport floor read p50 1.68 ms after the seed, 1.66 ms before the slots phase and 2.58 ms after it, so p50 stayed at or under 3 ms at every check.

**Run:** commit `e1f1da99`, `postgres:16.10` with P3-01's settings (`shared_buffers` 2 GB, `work_mem` 4 MB, `random_page_cost` 4, `jit` off, `fsync` and `synchronous_commit` on, `full_page_writes` on, `wal_compression` off, `nava_app` at `force_custom_plan`). It ran on Docker 29.9.3 on WSL2 (16 CPUs, 24.5 GiB). The seed was 800k records with a 400k large type. The run used 8 clients, 20 s per list cell, 30 s per write cell and 60 s per live-mix control. It took 7,523 s, from 04:54Z to 07:00Z. Raw report: `docs/data/field-index-measured/tick1205-slots-raw.md`. Plans: `tick1205-slots-explain.md`, one `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)` per cell, as `nava_app`. Contention: `tick1205-slots.contention`. Entry point: `sh scratch/field-index-measured/slots.sh`. Cell code: `internal/server/records_fieldindex_slots_perf_integration_test.go`.

**Design under test, mirrored in scratch DDL.** A3-2 is one `ALTER TABLE records` that adds k = 2 typed slots per value type: `fx_num_1..2`, `fx_text_1..2 COLLATE "C"`, `fx_ts_1..2` and `fx_bool_1..2`. Each slot gets `CREATE STATISTICS (mcv)` on (tenant_id, entity_definition_id, slot) and one partial index `(tenant_id, entity_definition_id, slot, assignment_group_id, updated_at) WHERE slot IS NOT NULL`, the "group index", built `CONCURRENTLY`. The field-to-slot map is a scratch table that stands in for `record_index_fields.slot`. A `BEFORE INSERT OR UPDATE OF data` trigger fills mapped slots. The A3-5 backfill runs 1,000-record batches of `UPDATE records SET slot = record_field_coerce(data -> key)` with no version or `updated_at` change. Promoted fields keep their D7 side rows. The "plain index" in S-b2 (ii) and Item 1 "both" is the same key without `assignment_group_id`.

### Verdict

| Check | Result | Line | Verdict |
| --- | --- | --- | --- |
| 1. Write amplification | Slot vs D7 at the same n: insert p95 equal or lower; promoted-field update +2 to +18 ms p95 and 1.4 to 3.1x the WAL; unrelated update unchanged with the group index. No update is HOT in any arm, D7 0 included | report | reported |
| 2. Size per field | Group index 37 MB fresh and 55 to 78 MB after the backfill per 400k-per-type field; plain index 29 MB; 74 / 58 MB for a slot filled across 892k rows; value bytes in heap 0.4 to 3.5 MB | report | reported |
| 3. DDL | `ADD COLUMN` × 8: 30 ms wall, `AccessExclusiveLock` seen for 6 ms, 6 writers waited at most 30 ms. `CREATE INDEX CONCURRENTLY`: 1.5 to 2.1 s on an empty slot, 2.3 to 4.6 s on a filled one, `ShareUpdateExclusiveLock` only. Live p95 104.3 → 107.3 ms during the migration; 0 errors | report | reported |
| A3-5 backfill (D10) | Live p95 +18.7% (101.7 → 120.8 ms) during the score backfill | +25% | **pass** |
| 4. Slots per table | Insert p95 62.9 / 78.6 / 99.5 / 131.5 / 192.6 ms at 0 / 8 / 16 / 32 / 64 extra filled, indexed slots | curve | reported |
| S-b1, per-group merge | Anti-correlated merge cells bounded (≤ 33 rows per slot scan, Sort input ≤ 3). Broad merge cells never use the slot index: each arm walks `records_list_group` and filters 1,993 to 3,618 rows, so they fail on rows read as well as p95. 1 of 6 meets p95 | ≤ 51 + ε rows per arm, p95 ≤ 150 ms | **fail** |
| S-b2, no-ACL broad 50% | (i) group only 290.0 ms; (ii) group + plain 386.0 ms. Neither page uses a slot index: the planner chose the `updated_at` walk | p95 ≤ 150 ms | **fail on both sets**; the index set is open (SP-2, SP-3) |
| S-b3, 2.6M-record type | Per-group merge 105.3 ms; the compiled shape 6,762.4 ms | p95 ≤ 150 ms | **pass** (merge shape) |
| Heap fetches under churn | Count plans 215 to 478, page plans 0, at 73% all-visible | report | reported |
| S-c1, shared slot | Estimates within 2.1x except a 13-row tenant (est. 1, 13x). No plan changed; the shared slot is never slower than the slot of its own | 10x, band | **fail** on the 13x estimate only, p95 36.7 ms |
| S-c2, operators | Selective (2%) `lt`, `gte`, range, `text eq`, `in`: pass for a6 and without an ACL. `num in` at 17% misses at 283.5 to 1,170.8 ms. a50 misses 4 of 5 compiled | p95 ≤ 150 ms | **mixed**, below |
| S-c3, 10% and 20% | Compiled shape: 7 of 8 pass (a6 `text eq` 20% misses at 159.7 ms, provisional). Per-group merge: all 4 a50 cells miss | p95 ≤ 150 ms | **mixed**, below |

Three findings bear on the amendment. None of them changes the A3-2 DDL.

1. **The slot write path costs about what D7 already costs.** The group index adds no insert latency over D7 at the same n. It adds 2 to 18 ms p95 to an update of the promoted field. The migration and the index builds are online: one `AccessExclusiveLock` of 6 ms, and no write errors. The D10 backfill passes at +18.7%. The carry curve gives the ceiling. 8 filled slots (A3-2's k = 2) cost +25% insert p95, and insert p95 doubles at about 32. That makes k = 2 cheap, and k = 4 (16 slots, +58%) the most I would propose without a new measurement.
2. **A broad value, 17% to 50% of the type, misses when the newest rows of the type do not carry it.** In this run, the ~100k rows the write cells inserted or patched carry no pk 3: inserts have no pk, and every PATCH target has pk 4. These live rows sit at the head of `updated_at`. The page walks `records_list_updated` and filters out 100,514 to 126,698 of them before it finds 51 matches (Rows Removed by Filter, S-b2 and S-b1, which counts visible tuples only). TICK-1039's 131.3 ms was measured with pk spread evenly over `updated_at`. This is the anti-correlation case, like pk 40, and not heap churn: `VACUUM` does not change it. It is a real production shape, for example a value that was common and is rare now; the remedy is plan shape or plan steering, not heap maintenance (spike SP-1 confirms the cause). **The index set is open.** S-b2 fails on both sets because the planner chose the `updated_at` walk, not for lack of an index. Whether the plain index would bound the no-ACL page if the planner chose it is open (SP-2). The plain index's unrelated-update cost is confounded by run order (see Item 1; SP-3).
3. **The per-group merge is a shape for the compiler to choose, not a default.** It rescues the cases where the compiled shape collapses: a50 anti-correlated 1,485.1 → 267.0 ms, and broad at 2.6M records 6,762.4 → 105.3 ms. It loses whenever the agent has many groups and the value is selective, because 25 arms cost more than the one bitmap: a50 at 2% to 20%, 114 → 313 ms and similar. Neither shape alone meets the band across the matrix. The merge bounds the page only when its arms use the slot index. The planner put them there at 2.6M and not at 800k. PostgreSQL 16 has no plan hints, so "the compiler chooses the merge" also needs a way to keep the arms on the slot index; that is a design question for the Solution Architect.

### Item 1: write amplification (8 clients through the real handler, 30 s per cell)

D7 n: n of (score, tier, opened, zone, b2) flagged, side rows only. Slot n: the same n flagged *and* mapped to slots, each with its index; the side rows stay (A3-5). The update's PATCH body carries the n promoted values. Cells: p95 ms / WAL bytes per op (`pg_current_wal_lsn` diff over the cell). HOT: 0% in every update cell of every arm, D7 0 included. Records already carries indexes on `updated_at` and `version`, so no PATCH through the pipeline is HOT, with or without a slot.

| Write | D7 0 | D7 1 | D7 3 | D7 5 | slot 1, group | slot 3, group | slot 5, group | slot 1, both | slot 3, both | slot 5, both |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| insert | 42.4 / 9,231 | 78.9 / 13,111 | 100.9 / 14,081 | 80.6 / 17,922 | 78.9 / 13,158 | 78.9 / 17,631 | 69.5 / 16,577 | 77.4 / 13,215 | 61.2 / 16,416 | 67.9 / 34,158 |
| update promoted field (score) | 122.2 / 32,006 | 123.5 / 20,234 | 134.4 / 13,269 | 121.7 / 14,590 | 132.8 / 29,326 | 136.2 / 41,506 | 139.7 / 32,749 | 120.8 / 36,684 | 111.0 / 30,540 | 144.4 / 30,555 |
| update unrelated field (b001) | 93.2 / 19,074 | 121.1 / 16,922 | 120.7 / 35,453 | 120.5 / 11,313 | 123.0 / 10,197 | 107.6 / 10,214 | 131.5 / 10,023 | 167.8 / 19,187 | 160.8 / 9,307 | 203.5 / 11,471 |

- **Promoted-field update, slot vs D7 at the same n:** +9.3 / +1.8 / +18.0 ms p95, and 1.4x / 3.1x / 2.2x the WAL. The extra WAL is the slot index entry plus full-page images: 4.1 to 6.2 FPI per op against 0.9 to 1.9 for D7.
- **Insert:** no slot cost over D7 at the same n. Slot 5 reads 69.5 ms against D7 5's 80.6 ms, which is within run-to-run noise (D7 3 reads 100.9 ms). Each mapped slot adds 0.5 to 1 kB of WAL per insert, except "slot 5, both" at 34 kB, which carries 3.6 FPI per op from a checkpoint inside the cell.
- **Unrelated update:** the group index leaves it where D7 has it (107.6 to 131.5 ms against 120.5 to 121.1 ms), with *less* WAL (10 kB against 11 to 35 kB). Adding the plain index puts it at 160.8 to 203.5 ms p95 with no change in WAL. Those cells ran last, on the most-churned heap, their WAL per op is equal to or lower than group-only, and the same arm's promoted-field update is faster. So this run does not measure the plain index's update cost; SP-3 does.
- WAL records per op: 30.3 to 54.0 for D7, 31.8 to 58.8 for the group index, 32.5 to 70.1 for both. Errors: 0 in every cell.

### Item 2: size per promoted field

| Slot (field) | Filled rows | Value bytes in heap | Group index, as grown by the backfill | Group index, fresh build | Plain index, fresh build |
| --- | --- | --- | --- | --- | --- |
| `fx_num_1` (score) | 451,222 | 3,525 kB | 78 MB | 37 MB | 29 MB |
| `fx_text_1` (tier) | 451,222 | 1,721 kB | 56 MB | 37 MB | 29 MB |
| `fx_ts_1` (opened) | 451,222 | 3,525 kB | 56 MB | 37 MB | 29 MB |
| `fx_text_2` (zone) | 451,222 | 1,891 kB | 56 MB | 37 MB | 29 MB |
| `fx_bool_1` (b2) | 451,222 | 441 kB | 55 MB | 37 MB | 29 MB |
| `fx_num_2` (pk) | 400,000 | 3,125 kB | 43 MB | 33 MB | 26 MB |

- **400k per type:** 37 MB for the group index on a fresh build, and 1.5 to 2.1x that (55 to 78 MB) when the backfill grows it under writes. A `REINDEX CONCURRENTLY` after the backfill would recover the difference. The plain key is 29 MB.
- **800k:** `fx_num_2` shared by the large type, ticket2 and 399 small tenants (892,000 filled) is 74 MB group and 58 MB plain on a fresh build.
- **Heap:** the slot values add only 0.4 to 3.5 MB, because a NULL slot costs one bit in the null bitmap. The backfill itself grows the heap: six backfills took `records` from 1,291 to 1,738 MB (+35%) and left 59k to 328k dead tuples per pass, all non-HOT. Autovacuum (scale factor 0.2) reclaims the space but does not shrink the file.

### Item 3: DDL cost on the 800k seed, under the live mix (8 clients, POST + PATCH)

Locks were sampled from `pg_locks` every 2 ms.

| Statement | Wall | Lock on `records` | Writers waiting: max / longest | Live p95 / max ms overlapping |
| --- | --- | --- | --- | --- |
| `ALTER TABLE records ADD COLUMN` × 8, one statement (A3-2) | 30 ms | `AccessExclusiveLock`, seen for 6 ms (17 ms queued behind writers) | 6 / 30 ms | 102.9 / 102.9 |
| `ADD COLUMN` × 1, arm (b) literally, per field | 74 ms | `AccessExclusiveLock` (66 ms queued) | 6 / 55 ms | 150.2 / 150.2 |
| `CREATE STATISTICS (mcv)` × 8 | 9–15 ms each | `ShareUpdateExclusiveLock` | 0 | 82.8–106.7 |
| `CREATE TRIGGER` (slot sync) | 29 ms | `ShareRowExclusiveLock` | 3 / 20 ms | 107.4 |
| `CREATE INDEX CONCURRENTLY`, group key, empty slot × 8 | 1.53–2.09 s | `ShareUpdateExclusiveLock` | 0 | 93.8–123.7 / ≤ 130.9 |
| `CREATE INDEX CONCURRENTLY`, plain key, 400k filled, live mix | 4.15 s | `ShareUpdateExclusiveLock` | 1 / 7 ms | 136.1 / 158.1 |
| `CREATE INDEX CONCURRENTLY`, group key, 400k filled, live mix | 3.88 s | `ShareUpdateExclusiveLock` | 0 | 132.9 / 167.7 |
| `CREATE INDEX CONCURRENTLY`, plain key, 400k filled, alone × 5 | 2.33–4.61 s | `ShareUpdateExclusiveLock` | – | – |

| Live mix window | Length | ops/s | p50 ms | p95 ms | p99 ms |
| --- | --- | --- | --- | --- | --- |
| before the A3-2 migration | 60 s | 119.2 | 70.4 | 104.3 | 118.7 |
| during it (first `ALTER` to last CIC) | 20 s | 116.4 | 70.7 | 107.3 | 119.7 |
| after it | 60 s | 123.7 | 67.2 | 99.6 | 157.2 |
| before two filled-slot builds | 60 s | 108.8 | 77.7 | 121.8 | 158.4 |
| during them | 8 s | 86.9 | 94.4 | 135.0 | 143.4 |

0 live errors in every window. Building the empty slot index costs about 2 s, so building it on all-NULL columns, as A3-2 does, is the cheap order. The A3-5 backfill then pays per field:

| Field → slot | Load | Duration | Batch p50 / p95 / max ms | WAL | WAL B / record | HOT / updates |
| --- | --- | --- | --- | --- | --- | --- |
| score → `fx_num_1` (451,222) | live mix, 500 ms pause | 11m11s | 1,133 / 1,606 / 1,896 | 3.92 GB | 9,335 | 2,846 / 452,428 |
| tier → `fx_text_1` | alone | 3m12s | 404 / 626 / 1,031 | 2.23 GB | 5,317 | 0 / 451,222 |
| opened → `fx_ts_1` | alone | 3m4s | 379 / 616 / 904 | 1.46 GB | 3,467 | 0 / 451,222 |
| zone → `fx_text_2` | alone | 2m50s | 377 / 623 / 1,107 | 2.31 GB | 5,487 | 0 / 451,222 |
| b2 → `fx_bool_1` | alone | 3m17s | 386 / 717 / 1,310 | 2.31 GB | 5,504 | 0 / 451,222 |
| pk → `fx_num_2` (400,000) | alone | 2m41s | 338 / 520 / 1,142 | 1.63 GB | 3,867 | 12,256 / 451,222 |

D10 live mix around the score backfill: 101.7 ms p95 alone before, 120.8 ms during (671 s), 131.9 ms alone after. That is **+18.7% against before (pass, budget +25%)** and −8.4% against after, with 0 errors. The quick run's +27.1% was taken beside a busy `nava-s21` stack. A backfill writes 3.5 to 9.3 kB of WAL per record, mostly full-page images. Every backfill `UPDATE` is non-HOT because the slot is indexed.

### Item 4: how many filled, indexed slots `records` can carry

64 extra `double precision` slots, filled for the large type's 400k and each indexed with the group key, were switched on in steps. The A3 slots were unmapped and the five D7 fields stayed flagged throughout.

| Extra slots (indexes on `records`) | Insert p50 / p95 ms | Insert WAL B/op | Unrelated update p50 / p95 ms | Update WAL B/op | List a6 pk 3, merge: p95 / planning ms | List a6, no filter p95 | `records` indexes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 0 (18) | 47.8 / 62.9 | 16,557 | 85.7 / 176.6 | 13,855 | 119.3 / 5.35 | 75.1 | 2,068 MB |
| 8 (26) | 59.1 / 78.6 | 15,909 | 109.6 / 242.3 | 10,251 | 120.7 / 5.93 | 75.6 | 2,418 MB |
| 16 (34) | 77.0 / 99.5 | 26,030 | 111.4 / 140.2 | 66,280 | 120.9 / 6.80 | 87.9 | 2,768 MB |
| 32 (50) | 95.0 / 131.5 | 21,821 | 165.0 / 254.3 | 11,965 | 170.2 / 8.96 | 121.1 | 3,460 MB |
| 64 (82) | 160.4 / 192.6 | 55,786 | 231.9 / 310.9 | 260,548 | 144.0 / 8.38 | 95.3 | 4,809 MB |

- **The cost is in the writes, not the planner.** Insert p50 rises by about 1.7 ms per filled, indexed slot (47.8 → 160.4 ms over 64), and p95 doubles at about 32 slots. Unrelated-update p50 rises 85.7 → 231.9 ms. Its p95 is noisy (140 to 311 ms), and the WAL spikes at 16 and 64 come from checkpoints inside the cell. Planning time for the list rises only 5.35 → 8.38 ms, and the no-filter list stays at 75 to 121 ms.
- **Each filled, indexed slot costs about 43 MB of index at 400k rows** (2,068 → 4,809 MB over 64).
- A3-2's 8 slots: +25% insert p95, +350 MB of index once filled.

### S-b1: per-group merge page, group index only (A3-2's set)

The merge shape gives each visible group one `UNION ALL` arm, `ORDER BY updated_at DESC, id DESC LIMIT 51`, under a Merge Append. Table churn at this point, with no `VACUUM FULL` since the fixture: 3,194,291 updates (15,102 HOT), 60,889 dead tuples, 162,701 of 222,358 pages all-visible.

| Fixture | Agent | Compiled (b)+group p95 | Per-group merge p95 | Merge: slot-index scans / max rows per scan / max Sort input | S-b1 |
| --- | --- | --- | --- | --- | --- |
| broad 50% (pk 3) | a6 | 466.0 | 159.0 (provisional) | 0 / – / 3 | **fail** (p95) |
| broad 50% (pk 3) | a25 | 434.0 | 314.7 | 0 / – / 2 | **fail** (p95) |
| broad 50% (pk 3) | a50 | 413.2 | 559.3 | 0 / – / 1 | **fail** (p95) |
| anti-correlated 2% (pk 40) | a6 | 107.1 | 128.8 | 3 / 33 / 0 | pass |
| anti-correlated 2% (pk 40) | a25 | 118.3 | 178.8 | 13 / 33 / 0 | **fail** (p95) |
| anti-correlated 2% (pk 40) | a50 | 1,485.1 | 267.0 | 25 / 33 / 0 | **fail** (p95) |

- **The anti-correlated arms are bounded; the broad arms are not.** Where a merge arm scans the slot index (the anti-correlated cells), it reads at most 33 rows, inside 51 + ε. The broad merge cells never use the slot index (0 scans): each arm walks `records_list_group` and filters 1,993 to 3,618 rows, so they fail on rows read as well as p95. No arm top-N sorts the visible matches: the largest Sort input in any merge plan is 3 rows. TICK-1039's 15,482-row sort is gone.
- **For the broad value the planner leaves the slot index unused.** Each arm walks `records_list_group` in `updated_at` order and filters on `fx_num_2`. The arms return 53 to 75 rows, but they read 8,876 / 42,200 / 81,768 buffers for a6 / a25 / a50, against 113k for the compiled shape. The misses come from those filtered rows, the recent pk-4 and pk-less rows at the head of each group's `updated_at` order (finding 2), and from the merge's fixed cost: count p95 rises from 6 to 8 ms to 10 to 19 ms with 3 to 25 arms.
- **Verdict: fail.** No cell sorts more than about 1,000 rows, but 5 of 6 merge cells miss the band, and the 3 broad cells also read more than 51 + ε rows per arm. At 800k the merge cuts the a6 broad cell by 3x and a50 anti-correlated by 5.6x. It makes a25 anti-correlated and a50 broad worse.

**Heap fetches under churn** (`Heap Fetches` in one `EXPLAIN ANALYZE` each, at 73% all-visible): count plans 215 to 478 per count, out of a 1,001-row capped count. Page plans: 0 in every cell, because the page reads the heap anyway. The count's index-only scan does up to one heap fetch per two rows here. TICK-1039 read 0 on a `VACUUM FULL`ed heap.

### S-b2: the no-ACL broad 50% value, index sets compared

| Cell (no ACL) | (i) group only | (ii) group + plain | Page plan in both |
| --- | --- | --- | --- |
| broad 50% (pk 3) | **290.0 miss** | **386.0 miss** | `records_list_updated` walk, 51 rows, 86,090 / 111,945 buffers |
| `num in` 3 values (17%) | **283.5 miss** | **371.4 miss** | same walk, 86,171 / 112,026 buffers |
| `num lt` 2% | 63.5 pass | 70.8 pass | walk, 1,493 / 1,097 buffers |
| `num gte` 2% (newest end) | 111.0 pass | 130.6 pass | slot index, 7,680 / 7,675 rows |
| `num` range 2% | 161.4 miss (provisional) | 127.0 pass | walk, 46,671 / 25,314 buffers |
| `text eq` 2% | 79.1 pass | 78.5 pass | walk / plain index (52 rows, 216 buffers) |
| `text in` 3 values (2.26%) | 82.4 pass | 98.5 pass | walk |

**Verdict: fail on both index sets.** Both fail for the same reason, the planner's estimate, not a missing index. The broad page's plan costs the `records_list_updated` walk at 120.68 for the `LIMIT`, assuming pk 3 is spread evenly (estimate 203,108 rows), and then filters out 100,514 (i) / 125,697 (ii) recent rows that carry pk 4 or no pk (finding 2). For the no-ACL broad page, `(tenant, def, slot, updated_at)` could return 51 rows directly; the group-only index cannot bound it at all, since with no group predicate it either sorts every match or does the same walk. **The index set is open:** whether the plain index bounds this page when chosen is SP-2, and its write cost is SP-3.

### S-b3: the large type at 2.6M records

The large type was copied 4 times with the same values, ACL and groups, to 2,616,075 records. That is more than the 2M asked for, because the copies include the rows Items 1 and 4 inserted. a6 sees 160,318 of them, 79,083 with pk 3. The heap was 4,911 MB, indexes 4,364 MB.

| Shape | p95 ms | p50 ms | Server page ms | Page plan |
| --- | --- | --- | --- | --- |
| compiled (b)+group | **6,762.4 miss** | 4,766.2 | 1,535.5 | `records_list_updated` walk, 456,738 hit + 274,754 read buffers |
| per-group merge | **105.3 pass** | 85.0 | 13.4 | `spike_fx_num_2_grp`, 4 scans, at most 56 rows each, 3,937 buffers |

**Verdict: pass with the S-b1 shape.** At 2.6M records the merge arms switch to the slot index and stay bounded. The compiled shape degrades much worse than linearly: 466 ms at 400k becomes 6,762 ms at 6.5x the rows, once the walk falls out of `shared_buffers`. Two observations, with an inference: the merge shape is the one that scales; and at 800k the planner preferred `records_list_group` for the same arms, so the plan choice depends on table size. The plan choice is what S-b1 leaves open.

### S-c1: one slot shared by types with different distributions (R-d)

`fx_num_2` holds pk for the large type (skewed: 50% / 2% values), for ticket2 (100k, uniform) and for all 399 small tenants (uniform): 892,000 filled. The control is ticket2 on a slot of its own (`spike_fx_own`).

| Predicate | Own slot: estimate / actual | Shared slot: estimate / actual | Off by (shared) |
| --- | --- | --- | --- |
| large type, pk 3 (50%) | 200,745 / 200,000 | 198,930 / 200,000 | 1.0x |
| large type, pk 40 (2%) | 8,019 / 8,000 | 8,091 / 8,000 | 1.0x |
| large type, pk 3, a6's groups | 7,301 / 15,482 | 7,202 / 15,482 | 2.1x |
| ticket2, pk 3 (2%, uniform) | 2,147 / 2,032 | 2,149 / 2,032 | 1.1x |
| ticket2, pk 40 (2%, uniform) | 2,248 / 2,021 | 2,283 / 2,021 | 1.1x |
| ticket2, pk 3, a6's groups | 78 / 111 | 78 / 111 | 1.4x |
| tenant s1's ticket, pk 3 (13 rows) | – | 1 / 13 | **13.0x** |

| Cell (p95 ms) | Own slot | Shared slot | Same plan? |
| --- | --- | --- | --- |
| large type, pk 3, a6 | 507.9 | 456.9 | yes, both miss (the broad-value walk) |
| large type, pk 40, a6 | 115.5 | 109.8 | yes |
| ticket2, pk 3, a6 | 108.9 | 78.5 | yes, own vs shared slot index |
| ticket2, pk 40, a6 | 111.7 | 79.1 | yes |
| ticket2, pk 3, no ACL | 68.6 | 72.3 | yes |
| tenant s1, pk 3, admin | – | 36.7 | – |

**Verdict: fail on the letter, on one estimate.** Sharing changed no plan, and the shared slot is never slower than its own beyond noise. The extended MCV statistics on (tenant_id, entity_definition_id, slot) keep each type's estimate within 1.1x. The exception is a tenant whose 13 rows fall outside the MCV list (estimate 1, 13x off). Its page is still a 13-row index scan at 36.7 ms. The large type's broad-value miss is the same on its own slot, so it is not a sharing effect. The tail estimate gets worse as more tenants share a slot, because `default_statistics_target` 100 caps the MCV list. A per-slot `ALTER TABLE ... ALTER COLUMN ... SET STATISTICS` would tighten it. That is a tuning item, not a design change.

### S-c2 and S-c3: operators, types and 10% / 20% values, with an ACL on the slot

p95 ms against 150. a6 sees 6% of the type and a50 sees 50%.

| Cell | a6 compiled | a6 merge | a50 compiled | a50 merge |
| --- | --- | --- | --- | --- |
| S-c2 `num lt` 2% (score < 200) | 150.3 miss (prov.) | 145.7 pass (prov.) | 113.8 pass | 312.5 miss |
| S-c2 `num gte` 2% (score ≥ 9800) | 132.2 pass | 152.4 miss (prov.) | 159.0 miss (prov.) | 332.8 miss |
| S-c2 `num` range 2% | 148.3 pass (prov.) | 156.0 miss (prov.) | 182.4 miss | 361.6 miss |
| S-c2 `text eq` 2% (tier v2) | 136.5 pass (prov.) | 137.3 pass (prov.) | 150.8 miss (prov.) | 261.2 miss |
| S-c2 `text in` 3 values (2.26%) | 136.8 pass (prov.) | 144.1 pass (prov.) | 200.0 miss | 441.4 miss |
| S-c2 `num in` 3 values (17%) | 586.3 miss | 187.6 miss | 611.8 miss | 1,170.8 miss |
| S-c3 `num lt` 10% | 140.6 pass (prov.) | 138.1 pass (prov.) | 122.6 pass | 275.3 miss |
| S-c3 `num lt` 20% | 131.8 pass | 145.5 pass (prov.) | 122.9 pass | 290.0 miss |
| S-c3 `text eq` 10% | 128.0 pass | 113.6 pass | 103.9 pass | 245.9 miss |
| S-c3 `text eq` 20% | 159.7 miss (prov.) | 111.4 pass | 89.2 pass | 229.4 miss |

- **S-c2:** every operator and type uses the slot index when the value is selective. The a6 cells sit on the line, 132 to 156 ms. Without an ACL, 5 of 7 operators pass (S-b2 table). The miss is a broad `in` list: 17% of the type in three values gets the broad-value walk on every shape. a50 compiled misses 4 of 5 selective operators by 1 to 33%.
- **S-c3:** with an ACL on the slot, the compiled shape passes 7 of 8 cells. TICK-1009 measured 208.7 to 421.5 ms at 10% and 20% on D7, so the slot fixes that band. The one miss is a6 `text eq` 20% at 159.7 ms (provisional).
- **The merge shape loses every a50 cell** (229 to 1,171 ms). 25 arms plus a count over 25 groups cost 30 to 43 ms of server time before the first row. The compiler should choose the merge only when the compiled shape would sort or walk a large visible set: broad values, anti-correlated values, large types. Choosing it on the number of visible groups alone would not work.

### What this changes and what it leaves

- **For the ADR-006-A3 draft (Solution Architect):** the A3-2 DDL and the A3-5 backfill hold. They are online and inside D10, Items 1 to 4 give the cost per slot. Do not rely on conclusions 1 (index set) and 2 (cause of the broad-value miss) until the open spikes below have run. Open: (1) the broad-value page when the newest rows lack the value, which misses with either index set (S-b2, S-b1 broad, S-c2 `in`), and the index set itself (group only vs group + plain); (2) when the compiler should choose the per-group merge (S-b1, S-b3, S-c3); (3) the backfill's index bloat (1.5 to 2.1x), which a `REINDEX CONCURRENTLY` step at `ready` would recover.
- **Open spikes** (Technical Architect's review; the Technical Lead decides whether they run on this PR or as a precondition of TICK-1206). Each is a short rerun on this harness:
  - **SP-1, recency vs churn.** Record `count(*) FILTER (WHERE fx_num_2 = 3)` over the type's newest 100k rows by `updated_at`. Then rerun S-b2 (i)/(ii) broad and S-b1 broad a6/a25/a50 on the same churned heap, after giving the inserted and patched rows pk values drawn from the fixture distribution. *Pass (confirms finding 2):* the walk filters fewer than 1,000 rows and p95 is ≤ 150 ms. *Fail:* p95 still misses with fewer than 1,000 rows filtered, which makes churn the cause.
  - **SP-2, can the index bound the page.** On the current fixture, single-shot `EXPLAIN ANALYZE` inside `BEGIN; DROP INDEX records_list_updated, records_list_group; … ROLLBACK;` (scratch only) for S-b2 (ii) broad and S-b1 merge broad a6/a25/a50. *Pass:* the plain-index page and each merge arm read ≤ 51 + ε rows, with server page time ≤ 20 ms; then the open problem is plan steering, and group-only has to be justified on cost (SP-3). *Fail:* group-only stands on S-b2.
  - **SP-3, plain-index write cost without the confound.** The unrelated-update cell alternating group-only and group + plain (ABAB, 3 reps each, same heap state). *Pass for "material cost":* the mean difference exceeds the spread within each arm.
- **Not changed:** no budget line, no ADR text, no migration. Scratch DDL is dropped at the end of the test.
- **Handbook:** the benchmark page changes. It gets this section's numbers as an arm (b) write-path section next to the A1-6 table.

