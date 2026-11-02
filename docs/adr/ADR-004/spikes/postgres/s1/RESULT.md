ITEM 1: PASS — the commit check costs 34–76 µs per call, ≤ 4.7% of the smallest baseline p95 (1,634 µs, async) and ≤ 1.2% of any sync p95. That is a measured additive bound, immune to host noise. Supporting evidence: p50 moves ≤ +60 µs, and the median block-pair p95 is ≤ +6.1%. Pooled p95 is inconclusive on this host, because its noise floor is larger than the signal. Target ≤ 10%.

# Spike S1 (TICK-423, P5-01): ADR-004 R-A4 results

| # | Item | Verdict | Headline number |
|---|---|---|---|
| 1 | p95 overhead of the commit-check trigger, through PgBouncer | **PASS** | Trigger self time 34–76 µs per call: ≤ 4.7% of the smallest baseline p95, and ≤ 1.2% of sync p95. p50 moves ≤ +60 µs. Median block-pair p95 ≤ +6.1%. Pooled p95 is inconclusive on this host. |
| 2 | 10k-subject transaction: commit time and queue memory | **PASS with fix F1** (as pinned: **FAIL**, O(n²)) | 10k subjects with F1: 172–198 ms commit and a 156–287 KB queue. As pinned: **24.1 s**. Without F1 the plan is unstable over fresh ANALYZE samples: pkey 9/10 (pinned form) and 10/10 (locals form). |
| 3 | WAL bytes per entry, with and without the seal UPDATE | **PASS** (≈ 2× confirmed) | 1,344 B per insert + 1,323 B per seal = **1.98×** (warm). Cold seal: 3,764 B. |
| 4 | BEFORE INSERT assert on `PARTITION BY RANGE` + DEFAULT | **PASS**: behaves as R10 states | 30 of 30 cases match. |
| 5 | Export caps: 5,000-row pages, 30 s and 15 min | **PASS** (anchor query: see F2) | Page p95 43–99 ms. 1M entries in 12–26 s. |

Two findings need action in Epic 05. Neither changes ADR-004.

- **F1 (P5-10b, P5-11).** As pinned, the commit check is O(n²) in a large transaction. Moving the probe's values into plpgsql locals changes the tie-break, not the costs. The fix is one statistics override per partition, plus a CI plan test.
- **F2 (export task, S16).** The R20 anchor query, written the obvious way, scales with tenant history instead of window size. Its plan must scan the `occurred_at` range.

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
./run.sh f1        # F1 on clean stats, right after seed: out/f1-fix-off.txt, out/f1-fix-on.txt, out/item2.txt
bash ./f1_census.sh > out/f1-census.txt   # plan census + 10 fresh ANALYZE samples per form, fix off and on
bash ./f1_vm.sh > out/f1-vm.txt           # locals form after VACUUM and after scattered writes, fix off and on
./run.sh item1     # ~25 min; CELL=update-sync-c1 ./run.sh item1 reruns one cell
./run.sh trimmed   # post-hoc stall-trimmed p95 (item 1), supporting only
./run.sh item3
./run.sh item5
./run.sh down      # deletes the volume
```

Only `compose.yaml` in this folder is used; no `make stack-*` or `make ci*`. Outputs go to `out/`. Raw pgbench logs (`out/item1/raw/`) are git-ignored; per-block pgbench summaries are committed. Every command reads and writes the same database, so a full rerun needs `down` and `up` first.

**Which stack each output comes from.**
- Items 1, 3 and 5, and `out/item2-run1.txt` (the 24.1 s as-pinned result), come from the first stack. That stack ran item4, seed, item1, item2, item3 and item5, in that order. Item 1's two async c=1 cells were rerun with F1 applied, and update-sync-c1 was rerun once.
- **Most `out/item1` cells were recorded with F1 off.** Only the two async c=1 reruns had it on. `sql/schema.sql` now applies F1 by default, so a fresh `./run.sh item1` measures the F1 configuration.
- For the TICK-460 review, a fresh stack ran `down`, `up`, item4 (30/30 again), seed, `f1`, `f1_census.sh` and `f1_vm.sh`. That run produced `out/item4.txt`, `out/seed.txt`, `out/f1-*.txt` and the current `out/item2.txt`.

**The host drifted between the first item 1 run and the F1 reruns.** On update-async-c1, baseline p50 rose from 992 to 1,755 µs (+77%), and the function's self time from 36.7 to 67.8 µs per call. Both arms moved, the trigger-off baseline included, so read this as host load and not as a cost of F1. F1 changes only the probe's plan, and for a one-subject transaction that plan is a single small index probe either way.

## Item 1: trigger overhead (PASS)

**Method.** Two records-shaped transactions, as `nava_app` through PgBouncer, with the tenant set by `set_config(..., true)` (TICK-449 baseline):

- **update:** `UPDATE records` (version, jsonb, updated_at) `RETURNING version`, plus one `audit_log` INSERT (`bench/update.pgbench`);
- **insert:** `INSERT INTO records … RETURNING id`, plus one `audit_log` INSERT (`bench/insert.pgbench`).

Each cell runs 10,000 transactions per arm, trigger off vs on. The arms alternate in **20 blocks of 500 in ABBA order**, so host noise hits both arms. Cells cover `synchronous_commit` on (production) and off (fsync removed, which exposes the trigger's CPU cost), at 1 and 8 clients. p50, p95 and p99 come from pgbench's per-transaction log (`-l`). The trigger is toggled with `ALTER TABLE records DISABLE|ENABLE TRIGGER`.

The probe is exactly P5-11's pinned form: `tenant_id = $1 AND occurred_at = now() AND txid = pg_current_xact_id() AND subject_type = $2 AND subject_id = $3`. 

**The single-subject probe plan is an Index Scan on the R28 subject index, with `txid` as a Filter** (`out/item1/probe-plan.txt`):
- `Index Cond: (tenant_id, subject_type, subject_id, occurred_at = now())`;
- `Filter: (txid = pg_current_xact_id())`;
- 5 buffers, 0.074 ms.

It is not an Index Only Scan on the R2 key. That file is an EXPLAIN with literal values, taken at the end of the item 1 run.

The trigger's cached generic plan for the same one-subject shape was captured later on the fresh stack, with auto_explain on the 8th call of a session:
- **Without F1:** a `(tenant_id, occurred_at)`-prefix Index Scan on the R28 time index (`out/f1-fix-off.txt`).
- **With F1:** an Index Only Scan on the R2 key (`out/f1-fix-on.txt`).

Every one of these plans touches 1–2 entries for a single subject. The per-call bound below covers whichever plan item 1 actually ran.

The function's own time, from `pg_stat_user_functions`, is **34–76 µs per call** across all cells (`out/item1/runs.txt`).

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

**Why PASS.** The verdict rests on three statistics that host noise cannot inflate. The pre-declared runs collected all three before any result was read.

1. **The function's self time is an additive upper bound** on what the trigger adds to each transaction: at most 75.7 µs per call (`out/item1/runs.txt`). That is ≤ 1.2% of the smallest sync off-arm p95 (6,476 µs) and ≤ 4.7% of the smallest async off-arm p95 (1,634 µs). Item 2 measures the deferred-queue bookkeeping around the call separately: 16–29 B per event.
2. **p50 moves by at most +60 µs** (update-sync-c1 run 1 and update-async-c1), and by at most +6.0% in any cell.
3. **The median block-pair p95 ratio is at most +6.1%** in every cell (range −12.6% to +6.1%).

**Pooled p95 is inconclusive on this host.** The pre-declared statistic was pooled p95, on vs off (`run.sh`, item 1 header). `out/item1/summary.txt` prints FAIL for update-sync-c1 in both runs: +70.0% and +18.5%. On this host, the noise floor is larger than the signal:
- across cells running the same code, pooled p95 swings from −47.9% to +70.0%;
- the excess in update-sync-c1 comes from single stalls: b7-on and b8-on at 26.2 and 17.5 ms in run 1, and b17 in run 2 (15.0 ms off, 27.7 ms on). Every other block ran at 5.4–7.4 ms.

So pooled p95 neither confirms nor refutes ≤ 10%. The PASS rests on points 1–3.

**Stall-trimmed column: post-hoc, supporting only.** The rule drops a block pair from both arms when either arm's block p95 exceeds 2× the cell's median block p95 (`analyze_item1.sh`). It was chosen after the results came in. It appears here because it agrees with points 1–3; the verdict does not rest on it.

## Item 2: 10k-subject transaction (PASS with F1; FAIL as pinned)

**Method.** `item2.sh`. One transaction as `nava_app` (unix socket, fresh backend, RLS applies) writes N records in one statement (UPDATE or INSERT) and N audit entries in a second statement. The script then reads the `AfterTriggerEvents` memory context before COMMIT, times COMMIT, and reads the backend's `VmHWM` and `RssAnon` afterwards. Each case runs with the trigger off and on.

| Case | Source | COMMIT, trigger off | COMMIT, trigger on | `AfterTriggerEvents` | `RssAnon` off → on |
|---|---|---|---|---|---|
| UPDATE 10k, **as pinned** | `out/item2-run1.txt` (first stack) | 44 ms | **24,101 ms** | 286,912 B | 4,356 → 4,728 kB |
| UPDATE 10k, **F1** | `out/item2.txt` (fresh stack, clean ANALYZE) | 42 ms | **172 ms** | 286,912 B (28.7 B/event) | 3,968 → 4,184 kB |
| INSERT 10k, F1 | `out/item2.txt` | 52 ms | **198 ms** | 155,792 B (15.6 B/event) | 4,012 → 4,236 kB |
| INSERT 100k, F1 | `out/item2.txt` | 37 ms | **1,984 ms** | 2,122,064 B (21.2 B/event) | 3,940 → 4,396 kB |

- **When the F1 rows ran.** On the fresh stack, after every post-seed row was deleted and the table analysed (`f1.sh`, last step), in this order: 10k UPDATE, 10k INSERT, 100k INSERT.
- **Cost with F1.** About **13–20 µs per subject at commit**, and **16–29 B per subject** in the deferred queue.
- **Memory.** `VmHWM` is dominated by shared buffers the backend touched, and does not differ by trigger state. Peak private memory (`RssAnon`) grows by less than 0.5 MB, even at 100k subjects.
- **First-stack F1 rows** (142 / 147 / 1,230 ms) are in `out/item2-run1.txt`.

### F1: the pinned probe is O(n²) in a large transaction; only the override fixes it

**Evidence on clean statistics.** Fresh stack, right after seed. Every post-seed row was deleted, then `VACUUM (ANALYZE)`; at most 50 rows share an `occurred_at` value.

1. **The trigger's generic plan as pinned, with the fix off** (`out/f1-fix-off.txt`). It has `Index Cond: ((tenant_id = $14) AND (occurred_at = now()))`, with `subject_type`, `subject_id` and `txid` in `Filter`. The index is the pkey or the R28 time index: the two tie on cost, and either way each probe scans every entry the transaction has written. 2,000 subjects commit in 752–1,011 ms (`out/f1-census.txt`). On the first stack, 10k subjects took 24.1 s.
2. **Fresh ANALYZE samples, fix off** (`out/f1-census.txt`, stability section). After each of 10 fresh ANALYZEs, a new session reports the index of the cached generic plan:
   - pinned form (v0): **pkey 9/10**, R2 key 1/10;
   - plpgsql-locals form (v1): **pkey 10/10**.
3. **The locals form after VACUUM and after ordinary writes, fix off** (`out/f1-vm.txt`): **pkey 10/10**, both at 100.0% and at 97.8% all-visible.
4. **The good plan can still turn up by chance.**
   - In the census, 2 v1 sessions run straight after a `VACUUM (ANALYZE)` got the R2-key plan on all 2,000 probes and committed in 34–39 ms (`out/f1-census.txt`, census section).
   - v0 got the R2-key plan once in the stability test.
   - In `out/f1-fix-off.txt`, the v1–v4 plan-capture sessions show the R2 plan, yet their separately timed 2,000-subject commits took 433–920 ms, as slow as v0 (880 ms). The plan differs from session to session. That is the instability: the pkey prefix and the R2 key both cost 8.46, and small differences between ANALYZE samples decide the tie.
5. **Fix on** (`out/f1-fix-on.txt`, `out/f1-census.txt`, `out/f1-vm.txt`). Every trial gets an index whose `Index Cond` contains both `subject_id` and `occurred_at`: 10/10 + 10/10 fresh samples, 10/10 after VACUUM and after writes, and 4 census sessions.
   - The locals forms (v1–v4) get the R2-key Index Only Scan on all 5 columns, cost 4.46.
   - The pinned text gets either the R2 key or the R28 subject index `(tenant_id, subject_type, subject_id, occurred_at)`.
   - 2,000 subjects commit in 37–107 ms.

**Why: a cost tie.** Without the override, the R2-key path and any `(tenant_id, occurred_at)`-prefixed index (pkey, `tenant_time`, `unsealed`) have the same estimated cost: **8.46 vs 8.46**. The plan is then decided by tie-break. The Technical Architect found that it goes wrong once the partition is mostly not all-visible, which is the steady state under sealing (item 3: 0 HOT updates). With the override, the prefix path costs **214.98 against 8.46** (50 tenants). The margin figures come from a generic `PREPARE`/`EXPLAIN` in the Technical Architect's review of PR #371 (TICK-423 comment `01a199d9`, findings 4 and 5).

Every entry of one transaction shares `(tenant_id, occurred_at = now())`. Once the prefix path wins, each of the n deferred probes therefore scans up to n entries. With the override, `occurred_at` equality is estimated at about 200 rows per tenant, so the R2 key is strictly cheaper.

The rule "it goes wrong only when the partition is mostly not all-visible" does not cover every run here. In `out/f1-vm.txt`, the locals form also lost the tie 5/5 straight after `VACUUM (ANALYZE)`, at 100.0% all-visible. The tie-break also depends on other state, so a CI test should not assume that a freshly vacuumed table is safe.

**The locals form changes the tie-break, not the costs; only the override makes the R2 key strictly cheaper.** It won the tie in the Technical Architect's rerun. Here it lost in 20 of 20 fresh-ANALYZE trials (points 2 and 3) and won in 2 census sessions (point 4). **F1 is needed.**

**Fix (`sql/schema.sql`, on by default).** Run `ALTER TABLE … ALTER COLUMN occurred_at SET (n_distinct = -0.0001)` on `audit_log` **and on every partition**, then ANALYZE. Partitions do not inherit the attribute option (checked with `CREATE TABLE … PARTITION OF`).

**Side effects of the override.** It changes estimates only for *equality* on `occurred_at`. Range predicates (reader, export, R20) use the histogram. PK lookups `(tenant_id, occurred_at, id)` stay unique through `id`.

**Asks.** The Technical Lead has added the catalogue and plan tests to TICK-450 and TICK-428.

1. **P5-10b:** set the override on `audit_log` and every partition. Write it into the Epic 30 partition-creation contract: set it, then ANALYZE, on every new partition.
2. **P5-11:** write the probe with plpgsql locals (`v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();`). With F1 this gets the R2-key Index Only Scan (cost 4.46, against 8.59 for the subject index). The plan test must require both `subject_id` and `occurred_at` in `Index Cond`.

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
- **(b)** costs O(rows in the window). The export streams at least those rows, so in practice (b) costs no more than the export it anchors. The testable form of this claim is the plan assertion below.

Both stay under 30 s at 1M entries, so the R21 caps hold. But at a few tens of millions of entries per tenant, (a) on a recent window would hit the timeout.

**Ask (export task, S16, TICK-444):**
- Use form (b), or choose the form per window. A whole-history export needs no anchor query: `first_seq = 1`, genesis.
- Test it as a plan assertion: the anchor query's plan scans the `occurred_at` range (`occurred_at` in `Index Cond`, or in a bitmap index condition). It never walks the `(tenant_id, seq)` index from the start of the tenant's history.

## Files

- `compose.yaml`, `pgbouncer/`: the stack.
- `sql/schema.sql`, `sql/seed.sql`: schema and data.
- `run.sh`: entry point.
- `bench/*.pgbench`: item 1 and item 3 transactions.
- `item2.sh`, `item3.sh`, `item5.sh`, `sql/item4.sql`, `analyze_item1.sh`: per-item scripts.
- `f1.sh`, `f1_census.sh`, `f1_vm.sh`, `item2_variants.sh` (the six probe forms): F1 evidence.
- `out/`: recorded outputs.
