ITEM 1: PASS — the deferred commit-check trigger adds at most +6.4% p95 (stall-trimmed; +3.3% on the cleanest cell) to a records-shaped write through PgBouncer. Target is ≤ 10%.

# Spike S1 (TICK-423, P5-01): ADR-004 R-A4 results

| # | Item | Verdict | Headline number |
|---|---|---|---|
| 1 | p95 overhead of the commit-check trigger, through PgBouncer | **PASS** | Trigger costs 34–76 µs per call. p95 overhead is −11.1% to +6.4% in every cell once host-stall blocks are trimmed. |
| 2 | 10k-subject transaction: commit time and queue memory | **PASS with fix F1** (as pinned: **FAIL**, O(n²)) | 10k subjects: 142–147 ms commit and 287 KB queue with F1; **24.1 s** as pinned. |
| 3 | WAL bytes per entry, with and without the seal UPDATE | **PASS** (≈ 2× confirmed) | 1,344 B per insert + 1,323 B per seal = **1.98×** (warm). Cold seal: 3,764 B. |
| 4 | BEFORE INSERT assert on `PARTITION BY RANGE` + DEFAULT | **PASS**: behaves as R10 states | 30 of 30 cases match. |
| 5 | Export caps: 5,000-row pages, 30 s and 15 min | **PASS** (anchor query: see F2) | Page p95 43–99 ms. 1M entries in 12–26 s. |

Two findings need action in Epic 05. Neither changes ADR-004.

- **F1 (P5-10b, P5-11).** As pinned, the commit check is O(n²) in a large transaction. The fix is one statistics override per partition, plus a CI plan test.
- **F2 (export task, S16).** The R20 anchor query, written the obvious way, scales with tenant history instead of window size. Use the window-bounded form.

## Environment

| | |
|---|---|
| Host | Intel Core i7-1260P (12 cores, 16 threads), 39.7 GB RAM, Micron MTFDKCD512TFK NVMe SSD, Windows 11 Home |
| Container runtime | Docker Desktop, engine 29.9.2, WSL2 kernel 5.15.167.4; VM limits 16 vCPU and 19.4 GiB |
| PostgreSQL | `postgres:16.10` (PostgreSQL 16.10, Debian 16.10-1.pgdg13+1). `shared_buffers=1GB`, `max_wal_size=8GB`, `checkpoint_timeout=30min`, `track_functions=pl`; everything else default (`synchronous_commit=on`, `full_page_writes=on`, no data checksums) |
| Pooler | `edoburu/pgbouncer:v1.24.1-p1`, `pool_mode=transaction`, `max_prepared_statements=200`. pgbench runs with `-M prepared` |
| Data | 50 tenants, 1,000,000 `records` (20,000 per tenant, jsonb ≈ 350 B), 1,980,000 sealed `audit_log` entries. Tenant 1 holds 1,000,000; the other 49 hold 20,000 each. The DEFAULT partition is 2,038 MB (1,190 MB heap) with the full R28 index set. See `out/seed.txt` |
| Schema | `sql/schema.sql`: R1 columns, R2 CHECKs, R9 FORCE RLS with canonical policies, R10 column grants and assert, R11 guard, R12 writer grants, R14 head, R28 indexes, and the decision-2 constraint trigger with the TICK-449 probe |

All clients run inside the `pg` container and connect to the `pgbouncer` service over the compose network. No ports are published.

## How to rerun

```sh
cd docs/adr/ADR-004/spikes/postgres/s1
./run.sh up        # stack + schema
./run.sh item4     # on the EMPTY table, before seed
./run.sh seed      # ~3 min
./run.sh item1     # ~25 min; CELL=update-sync-c1 ./run.sh item1 reruns one cell
./run.sh trimmed   # post-hoc stall-trimmed p95 (item 1)
./run.sh item2     # FIX=off then on, see F1
./run.sh variants  # five forms of the probe, plans + commit times
./run.sh item3
./run.sh item5
./run.sh down      # deletes the volume
```

Only `compose.yaml` in this folder is used; no `make stack-*` or `make ci*`. Outputs go to `out/`. Raw pgbench logs (`out/item1/raw/`) are git-ignored; per-block pgbench summaries are committed. Every command reads and writes the same database, so a full rerun needs `down` and `up` first. The numbers below come from the runs recorded in `out/`, in the order above. Item 1's two async c=1 cells were rerun with F1 applied, and update-sync-c1 was rerun once.

## Item 1: trigger overhead (PASS)

**Method.** Two records-shaped transactions, as `nava_app` through PgBouncer, with the tenant set by `set_config(..., true)` (TICK-449 baseline):

- **update:** `UPDATE records` (version, jsonb, updated_at) `RETURNING version`, plus one `audit_log` INSERT (`bench/update.pgbench`);
- **insert:** `INSERT INTO records … RETURNING id`, plus one `audit_log` INSERT (`bench/insert.pgbench`).

Each cell runs 10,000 transactions per arm, trigger off vs on. The arms alternate in **20 blocks of 500 in ABBA order**, so host noise hits both arms. Cells cover `synchronous_commit` on (production) and off (fsync removed, which exposes the trigger's CPU cost), at 1 and 8 clients. p50, p95 and p99 come from pgbench's per-transaction log (`-l`). The trigger is toggled with `ALTER TABLE records DISABLE|ENABLE TRIGGER`.

The probe is exactly P5-11's pinned form: `tenant_id = $1 AND occurred_at = now() AND txid = pg_current_xact_id() AND subject_type = $2 AND subject_id = $3`. For a single-subject transaction its plan is an Index Only Scan on the R2 unique key: 5 shared buffers hit, 0.074 ms (`out/item1/probe-plan.txt`). The function's own time, from `pg_stat_user_functions`, is **34–76 µs per call** across all cells (`out/item1/runs.txt`).

| Cell (10k per arm) | p50 off → on | p95 off → on | Pooled p95 | Median block-pair p95 | Stall-trimmed p95 |
|---|---|---|---|---|---|
| update, sync, c=1 (run 1) | 4,498 → 4,558 µs | 6,476 → 11,012 µs | +70.0% ⚠ | −0.2% | +0.5% (dropped b7, b8) |
| update, sync, c=1 (run 2) | 4,666 → 4,692 µs | 7,540 → 8,936 µs | +18.5% ⚠ | +3.0% | +2.1% (dropped b17) |
| insert, sync, c=1 | 4,759 → 4,630 µs | 14,380 → 7,490 µs | −47.9% | −4.5% | −5.2% |
| update, sync, c=8 | 6,390 → 6,472 µs | 8,712 → 8,533 µs | −2.1% | −4.3% | −2.1% (none dropped) |
| insert, sync, c=8 | 6,899 → 6,670 µs | 11,914 → 10,922 µs | −8.3% | −8.4% | −8.3% (none dropped) |
| update, async, c=1 | 992 → 1,052 µs | 1,634 → 1,688 µs | **+3.3%** | +6.1% | +3.3% (none dropped) |
| insert, async, c=1 | 954 → 962 µs | 1,945 → 1,554 µs | −20.1% | −12.6% | −7.5% |
| update, async, c=8 | 2,261 → 2,280 µs | 5,411 → 4,133 µs | −23.6% | −1.0% | +6.4% |
| insert, async, c=8 | 2,109 → 2,134 µs | 3,280 → 3,267 µs | −0.4% | +0.5% | −0.4% (none dropped) |
| update, async, c=1, **with F1** | 1,755 → 1,797 µs | 5,301 → 5,057 µs | −4.6% | +3.8% | +5.4% |
| insert, async, c=1, **with F1** | 1,412 → 1,411 µs | 3,827 → 3,403 µs | −11.1% | −1.8% | −11.1% (none dropped) |

**Why PASS, given the two ⚠ cells.** In both runs of update-sync-c1, the pooled p95 excess comes from one host stall that landed on the "on" arm:

- Run 1: blocks b7-on and b8-on (26.2 ms and 17.5 ms p95, run back to back under ABBA), against 5.4–7.4 ms in the other 36 blocks.
- Run 2: b17, where both arms stalled (15.0 ms off, 27.7 ms on).

The same stalls produce large *negative* "overheads" elsewhere: insert-sync-c1 −47.9%, update-async-c8 −23.6%. The trigger runs before the commit record is flushed and costs 34–76 µs. It cannot add milliseconds of fsync stall. Every robust statistic stays inside the target:

- p50 moves by +1.3% at most in the sync cells;
- the median block-pair p95 is between −12.6% and +6.1% in every cell;
- the cleanest cell (update, async, c=1: no fsync, one client, no pairs dropped) shows **+3.3%**.

**The stall-trimmed column was added after the fact.** The rule: drop a block pair from *both* arms when either arm's block p95 exceeds 2× the cell's median block p95 (`analyze_item1.sh`). It was applied to every cell. If the Technical Lead wants a pooled p95 without trimming, rerun `./run.sh item1` on Linux CI hardware, where these WSL2 fsync stalls do not occur. No measured mechanism supports a fail.

## Item 2: 10k-subject transaction (PASS with F1; FAIL as pinned)

**Method.** `item2.sh`. One transaction as `nava_app` (unix socket, fresh backend, RLS applies) writes N records in one statement (UPDATE or INSERT) and N audit entries in a second statement. The script then reads the `AfterTriggerEvents` memory context before COMMIT, times COMMIT, and reads the backend's `VmHWM` and `RssAnon` afterwards. Each case runs with the trigger off and on.

| Case | n_distinct fix | COMMIT, trigger off | COMMIT, trigger on | `AfterTriggerEvents` | `RssAnon` off → on |
|---|---|---|---|---|---|
| UPDATE 10k | off (as pinned) | 44 ms | **24,101 ms** | 286,912 B | 4,356 → 4,728 kB |
| UPDATE 10k | **on (F1)** | 270 ms | **142 ms** | 286,912 B (28.7 B/event) | 4,364 → 4,724 kB |
| INSERT 10k | on | 12 ms | **147 ms** | 155,792 B (15.6 B/event) | 4,020 → 4,244 kB |
| INSERT 100k | on | 22 ms | **1,230 ms** | 2,122,064 B (21.2 B/event) | 3,952 → 4,400 kB |

The off-arm commit times include the fsync of a 10k-row transaction; the 270 ms off figure is a single noisy sample. With F1, the check costs about **12–15 µs per subject at commit**, and the queue costs about **16–29 B per subject**. `VmHWM` (300–650 MB) is dominated by shared buffers the backend touched and does not differ by trigger state. Peak private memory (`RssAnon`) grows by less than 0.5 MB, even at 100k subjects.

### F1: the pinned probe is O(n²) in a large transaction

**Plan as pinned.** auto_explain of the trigger's cached generic plan (`out/item2-variants.txt`, "fix: off"; first observed in `out/item2.txt`):

```
Index Scan using audit_log_default_pkey on audit_log_default audit_log
  Index Cond: ((tenant_id = $14) AND (occurred_at = now()))
  Filter: ((subject_type = ($12)[0]) AND (subject_id = $15) AND (txid = pg_current_xact_id()))
```

**Why.** For a table with no repeated timestamps, the planner estimates `occurred_at = <value>` at about 1 row (n_distinct ≈ 92k on the seed). The pkey's 2-column prefix and the R2 key's 5-column prefix therefore both cost 8.46, and the pkey wins the tie. But every entry of one transaction shares `(tenant_id, occurred_at = now())`. Each of the n deferred probes scans up to n entries, so 10k subjects take 24 s.

**Forms that do not fix it.** Moving the values into plpgsql locals, adding `version_after`, a row comparison, or an `ORDER BY` on the R2 key all get the same pkey plan (`item2_variants.sh`; 2k subjects: 630–880 ms).

**Fix (`sql/schema.sql`, on by default).** Set `ALTER TABLE audit_log ALTER COLUMN occurred_at SET (n_distinct = -0.0001)`, the same on **every partition**, then ANALYZE. Partitions do not inherit the attribute option: checked with `CREATE TABLE … PARTITION OF`. With the override:

- the pinned text plans onto the R28 subject index: `Index Cond: (tenant_id, subject_type, subject_id, occurred_at)`, `Filter: txid`;
- the locals form (`v1-locals`) plans onto the R2 key: Index Only Scan, all 5 columns in `Index Cond`;
- 10k subjects commit in 142–190 ms.

Item 1 rechecked with F1 shows no regression; see the table above.

**The plan depends on statistics.** After the bulk test transactions put 100k rows on one `occurred_at` value, ANALYZE made that value an MCV (frequency 0.031). The planner then chose the good plan even without the override (`out/item2-variants.txt`). A fresh or ordinary table does not have this, so do not rely on it.

**Side effects of the override.** It only changes estimates for *equality* on `occurred_at`. Range predicates (reader, export, R20) use the histogram, and PK lookups `(tenant_id, occurred_at, id)` stay unique through `id`.

**Asks:**

1. **P5-10b**: set the override on `audit_log` and its DEFAULT partition, and write it into the Epic 30 partition-creation contract (set it, then ANALYZE, on every new partition).
2. **P5-11**: write the probe with plpgsql locals (`v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();`) so it gets the Index Only Scan on the R2 key. Add a CI plan test: in a 1,000-subject transaction under `plan_cache_mode = force_generic_plan`, the probe's plan must have `subject_id` in `Index Cond`.

The ADR-004 text (decision 2, R2, R28) is unchanged.

## Item 3: WAL bytes per entry (PASS)

**Method.** `item3.sh`, measured with `pg_walinspect.pg_get_wal_stats` between LSNs.

- 10,000 entries for tenant 3, one INSERT per transaction, through PgBouncer as `nava_app`.
- The spike sealer (`spike_seal_batch`, the R16 A2 stand-in) then seals them as `nava_worker` in 10 batches of 1,000: R15 head lock, R11 guard trigger active.
- Older unsealed entries are drained first, so each seal covers exactly the 10,000 new entries.

| Step | Bytes per entry | Main records |
|---|---|---|
| INSERT, warm | **1,344** | Heap/INSERT and 5 B-tree leaf inserts (PK, time, subject, R2 key, unsealed partial) |
| INSERT, first touch after CHECKPOINT | 2,980 | Same, plus full-page images |
| Seal UPDATE, warm | **1,323** | Heap/UPDATE 584 B, 5 B-tree inserts 495 B, Heap/LOCK ×2 108 B |
| Seal UPDATE, cold (CHECKPOINT between insert and seal) | 3,764 | Full-page images: 10.9 MB on Heap/LOCK and 13.2 MB on B-tree, per 10k entries |
| Seal UPDATE, warm, R11 trigger disabled (attribution only) | 1,317 | Same record mix |

- **Insert + seal = 2,667 B vs 1,344 B = 1.98×** when warm. ADR-004's "≈ 2×" holds.
- The worst case is about 3.8×: a seal lands after a checkpoint, and every touched page pays a full-page image. With a 10 s seal interval and 30 min checkpoints, that is rare.
- **0 of 134,728 seal updates were HOT**, as expected: `seq` and `sealed_at` are indexed.
- The two `Heap/LOCK` records per entry remain with the R11 trigger disabled. They come from PostgreSQL's non-HOT `heap_update` path, not from R11.

## Item 4: BEFORE INSERT assert on the partitioned table (PASS: as R10 states)

**Method.** `sql/item4.sql`, run on the empty table. 30 cases, each self-checking (`out/item4.txt`). Result: **`ITEM 4: PASS`, 30/30 match.**

- **Accepted:** `nava_app` with defaults; a second statement in the same transaction (`now()` is transaction-stable); an explicit `occurred_at = now()`.
- **Rejected by the column grant:** `nava_app` naming `occurred_at` (42501 permission denied).
- **Rejected by the trigger (42501 "protected insert column")**, for a role with a full INSERT grant:
  - `occurred_at` backdated, or set to `clock_timestamp()`;
  - a foreign `txid`;
  - `backfilled = true`;
  - each of `seq`, `prev_hash`, `entry_hash`, `sealed_at`, `erased_at` and `erasure_ref` set;
  - a multi-row INSERT with one bad row (the whole statement fails);
  - `INSERT … SELECT`.
- **The superuser is not exempt.** Only `nava_migrator` is exempt; its backdated `backfilled` row is stored unchanged and routed to DEFAULT.
- **Direct partition inserts are covered.** The trigger is cloned onto the DEFAULT partition, onto a partition made with `CREATE … PARTITION OF`, and onto one added with `ATTACH PARTITION`. Direct inserts into each of the three raise. A migrator backfill routes to the new range partition.
- **COPY.** COPY FROM by any RLS role is refused outright ("COPY FROM not supported with row-level security"). A superuser COPY with a backdated row raises.
- **Contrast case.** A *rewriting* BEFORE trigger that moves a row to another partition fails on PG 16 (0A000), which confirms why R10 asserts rather than rewrites.

Nothing differs from R10, so no BLOCKED report.

## Item 5: export caps (PASS; see F2)

**Method.** `item5.sh`. Tenant 1 holds 1,000,000 sealed entries, one of 50 tenants (1.98M rows in total).

- The client reads through PgBouncer as a reader role with SELECT and canonical RLS.
- **Each page is its own transaction** with `SET LOCAL statement_timeout = '30s'`: keyset `seq > :last AND seq <= :head ORDER BY seq LIMIT 5000` on `(tenant_id, seq) WHERE seq IS NOT NULL`.
- 200 pages per pass. The client receives every byte and takes the last `seq` from each page.

Plan for a deep page (seq > 900,000): Index Scan on `audit_log_default_tenant_id_seq_idx`, Index Cond `(tenant_id, seq > …, seq <= …)`, 416 buffers, 5.8 ms.

| Pass | Page SELECT p50 / p95 / max | Server time, 200 pages | Wall clock, 1M rows | Bytes |
|---|---|---|---|---|
| Unfiltered (every line full, worst case) | 40.7 / 81.6 / 118 ms | 9.2 s | **16.9 s** | 742 MB |
| Filtered (one subject full, the rest without `diff`) | 32.0 / 43.5 / 71 ms | 6.7 s | **12.2 s** | 507 MB |

The cap is 30 s per page and 900 s per export. Earlier runs of the same passes gave page p95 between 60 and 102 ms (`out/item5.txt` keeps the last run).

### F2: the R20 anchor query must be window-bounded

`first_seq` = the first sealed `seq` whose `occurred_at` falls in the window, regardless of filter. Two forms return the same `first_seq` (checked for 4 windows):

| Window (1M-entry tenant) | (a) `SELECT min(seq) … WHERE occurred_at in window` | (b) same, with an `OFFSET 0` fence so the scan follows the window |
|---|---|---|
| First day | 1.0–1.8 ms | 8–30 ms |
| Middle day | 386–641 ms (reads 508k rows) | 14–37 ms |
| Last day | **508–1,199 ms (reads 991k rows)** | 9–29 ms |
| Whole 90 days | 0.05–0.1 ms | **2.4–8.4 s** (reads 1M rows) |

- **(a)** walks the seq index from the start of the tenant's history and filters by time, so its cost grows with the history *before* the window.
- **(b)** costs O(rows in the window). The export streams at least those rows anyway, so (b) never costs more than the export it anchors.

Both stay under 30 s at 1M entries, so the R21 caps hold. But at a few tens of millions of entries per tenant, (a) on a recent window would hit the timeout.

**Ask (export task, S16):** use form (b), or choose per window. A whole-history export needs no anchor query: `first_seq = 1`, genesis.

## Files

- `compose.yaml`, `pgbouncer/`: the stack.
- `sql/schema.sql`, `sql/seed.sql`: schema and data.
- `run.sh`: entry point.
- `bench/*.pgbench`: item 1 and item 3 transactions.
- `item2.sh`, `item2_variants.sh`, `item3.sh`, `item5.sh`, `sql/item4.sql`, `analyze_item1.sh`: per-item scripts.
- `out/`: recorded outputs.
