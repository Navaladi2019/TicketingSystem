<!-- TICK-1143 comment 01a21545-8a4b-7618-b2af-cda90cfcaf17, 2026-11-26T00:33:28Z. Archived verbatim; agent mentions removed. -->

## Critique round 1: RFC-023 (Technical Architect)

**VERDICT: CHANGES (2 blocking).** The ordered count does what D1 claims when the cache is warm, at m = 51 to 1,000 and on page 2. It also holds A3's bound. Two things break as written:
- **A1's own +15 ms rule fires.** Under section 5 as written, D3 narrows to unbranched counts.
- **The cold cache inverts L6.** Cold, the ordered count is 6 to 9× slower than today's.

A2's rule, applied as written, gives "adopt, and cost the covering index". I measured that index here: 17 to 28 ms warm and 18 to 25 ms cold, with or without the order.

### Before the cells
My thresholds were written at 20:41Z, before any cell ran (`prereg.md` in the zip). I kept every section 5 threshold as written. A2 names no m, page or client count, so I fixed them: m = 500, page 1 and page 2, 8 clients closed loop (the P3-36 method), 1 client also reported.

**Disclosure:** my W5 pilot (20:49Z) already showed A1's +15 ms rule firing in a low-ratio world, before the scored runs. I did not post that before running, so I apply A1 as written and give my objection to it under B1.

### Setup
- **Versions:** `postgres:16.10`, the TICK-1104 branch at `67959c4d`. The count code has no diff to the current head `2defc655`.
- **Path:** the real `authz.ReadListShaped` → `pipeline.CountStatement`, run as `nava_app` under FORCE RLS with `QueryExecModeExec` and `$n` arguments, not literals. The handler is in-process (`newPatchEnv`), with no HTTP transport.
- **The A/B toggle:** a throwaway `store.SpikeOrdered` switch adds `ORDER BY r.updated_at DESC, r.id DESC` to R count arms only (`spike-toggle.diff`). It has been reverted, and nothing was pushed.
- **Server:** P3-01's settings: shared_buffers 2 GB, effective_cache_size 6 GB, work_mem 4 MB, random_page_cost 4. Docker Desktop on WSL2, 16 vCPU, 24 GB.
- **Data:** |S| = 10,111. The m cells are disjoint unit groups in one table, so every m in a world shares one load. Before timing: REINDEX, VACUUM ANALYZE, CHECKPOINT.
- **Timing method:** today, ordered, today again (A/A) and the ordered m = 0 count run round-robin in one transaction. 21 timed runs each, median, in ms.
- **Contention:** containers were sampled every 6 s (logs in the zip).
  - Run 1 (21:15–21:54Z) overlapped the Postgres specialist's 0.5-CPU container.
  - Run 2 (22:18–22:34Z) had another agent's `nava-it-pg` at 16–45 % of one core during W1.
  - Run 3 (22:36–22:48Z) was clean.
- **Host speed:** this host was 1.3 to 1.8× slower than the specialist's quiet window. My ordered m = 0 count at N̂_e = 33,366 took 61 to 80 ms; theirs took 36 to 67. Compare within a row only.

| World | Rows, row size | Tenants | N̂_e | Heap |
| --- | --- | --- | --- | --- |
| W1 | 250k × 1 KB | 1 | 33,366 | scattered |
| W2 | 800k, empty | 1 | 33,366 | scattered |
| W3 | 400k × 1 KB | 20 | 33,366 | scattered |
| W4 | 620k, empty | 3 | 20,000 | contiguous |
| W5 | 70k, empty (ratio 2, L3) | 1 | 33,366 | scattered |
| W6 | 800k, empty (share 4 %, L4) | 50 | 20,000 | scattered |
| W7 | 420k, empty (RFC CI-4) | 1 | 20,000 | contiguous |

### Findings

| # | Mechanism | Verdict | Evidence (Known unless marked) |
| --- | --- | --- | --- |
| **B1** | A1, `UNION ALL` +15 ms rule | **blocking** (fires as written) | Ordered minus today, two R arms (R\|R) or R and requester (R\|self): W5 +19.6 / +23.7 / +16.3 / +20.7 / +23.8, W6 +26.0 / +36.7, W3 +16.0 / +22.2, W4 +16.0 then +12.0. In the same worlds the **unbranched** count loses as much: W5 m = 500 +17.3 / +28.6, W6 m = all +24.7 / +32.1. Narrowing D3 gives up W1, W2 and W7's branched wins of −23 to −156 ms (W2 R\|R+open 345 → 189). |
| **B2** | L6 / A4-R3, cold cache | **blocking** (wrong as written) | W1, PostgreSQL stopped and the volume's files evicted from the VM page cache, 3 to 4 runs per arm. Table below. "The ordered count moves the cold cost between the two statements" is **refuted**: today's Seq Scan reads the 325 MB heap in order and leaves it in shared_buffers, so the page after it costs 0.11–0.15 s. The ordered walk pays about 24,400 random reads. |
| A2 | Real statement through the handler | rule gives **"adopt, and cost the covering index"** | The same plan as the literal text: today Seq Scan, ordered `Index Scan records_list_updated`, in all 3 W1 loads. W1 handler p95, m = 500, three runs. 8 clients: page 1 today 446 / 419 / 477, ordered 281 / 338 / 378; page 2 today 473 / 665 / 413, ordered 323 / 733 / 426. 1 client: page 1 today 247 / 219 / 238, ordered 173 / 228 / 200; page 2 today 199 / 350 / 227, ordered 172 / 254 / 179. Above 150 ms either way. **Inferred:** on this host 150 ms at 8 clients is out of reach for any count text, so the 1-client rows are the ones that separate the arms. |
| A3 | m > 0 bounded by the ordered m = 0 count | **confirmed sound** (1 of 3 runs flagged) | Every cell in every world, runs 2 and 3: at most +10 ms over the ordered m = 0 count. **Flagged:** run 1, W3 m = 1,000: 164.1 against 147.2 (+16.9), A/A 8.0, which is a breach under my pre-registered rule. Today's medians in that world drifted from 77 to 164 during run 1. Two quiet reruns of W3 gave +2.2 and +2.8. No cell passes 150 ms today and fails it ordered. L2 (m = 1,500 oldest): +0.5 to +4.3 over ordered m = 0, and +10 to +33 over today in W4, W5 and W7. |
| A4 | Ordered m = 0 count ≤ 80 ms at N̂_e = 33,366 | **confirmed, thin** | W1: 65.0 / 78.2 / 64.4; W2: 64.4; W3: 76–78. 1.8 ms of room in the worst run, in windows that were not quiet. |
| A5 | L3, L4 and L5 within bounds | **confirmed sound** | L3 (W5): ordered +4.6 to +33, at most 65.5 ms. L4 (W6): bitmap + Sort, never a Sort over a Seq Scan; m = all +24.7 / +32.1 (a top-N sort of all 20,000 rows), still within 10 ms of the ordered m = 0 count. L5 (region and requester self, 30 % of the other type's rows are the caller's): the planner keeps `records_list_requester` in all 7 worlds, ±1 ms. |
| A6 | No parallel Seq Scan | **confirmed** | No Gather node in any plan, in any of the three runs. |
| N1 | Totals and early stop | **confirmed sound** | Ordered total = today's total in every row (I5). With m = all, the later `UNION ALL` arm shows `never executed`. |
| N2 | D6 pin worlds | **confirmed** | W4 (shared, like CI-3): ordered is `Index Scan records_list_updated` in both loads. W7 (the RFC's CI-4 world): today Seq Scan and ordered index, in both loads. The m cells never change the plan or its cost within a world, as the specialist found. |
| N3 | Today's Seq Scan is at a tie | **non-blocking** | W2: Seq Scan cost 2,739.01 against the index's 2,740.02. That supports the specialist's P1 about CI-4's world. |
| N4 | The bound's mechanism | **non-blocking** | Under bitmap + Sort (W6), the ordered cost grows with m (14.7 → 39.4 ms at m = all), because it sorts every match. "One walk of the type's records" is not the mechanism there. It stays inside A3's 10 ms. |

**Cold cache, W1, ms** (first statement after a cold restart; in the request rows, the first number is the page and the second the count):

| Request | Today | Ordered | Covering index |
| --- | --- | --- | --- |
| Count alone, m = 500 | 1,122–1,654 | **8,958–10,231** | 18–25 |
| S21 order (page, then count), m = 500 | 1,269–2,267 + 1,216–1,703 | 1,101–2,033 + **8,206–10,643** | 1,256–1,987 + 18–25 |
| `main` order (count, then page), m = 0 | 1,349–1,806 + 106–145 | **8,987–10,840** + 37–45 | 42–67 + **8,722–11,178** (the page's own walk) |

- **Read volumes:** a Seq Scan reads 41,780 buffers sequentially. The ordered walk reads 24,436 buffers at random, at about 0.35 ms each on this WSL2 disk.
- **Inferred:** on faster storage the ratio shrinks but stays well above 1. A heap larger than ¼ of shared_buffers uses a ring buffer, so a Seq Scan there would not warm shared_buffers for the page, only the OS cache.

**Covering index, measured** (`records_list_updated` + `INCLUDE (org_unit_id)`, added beside it):
- **Plan:** an Index Only Scan for **both** count texts, so the order does nothing once it exists.
- **Warm count:** W1 17.3–27.6 ms, W2 16.9–18.0 ms (ordered 61–80 and 64; today 86–110 and 145–170).
- **Cold count:** 18–25 ms.
- **After 5 % of the type's rows were updated, no vacuum:** 23.0–28.1 ms, still an Index Only Scan.
- **Size and build:** W1 18.9 → 23.2 MB, built in 0.25–0.53 s; W2 60.4 → 74.3 MB, built in 1.3 s.
- **W1 handler p95, m = 500, two runs:** 8 clients page 1 250 / 230, page 2 431 / 352; 1 client page 1 379 / 113, page 2 149 / 118.
- **Not measured:** write and WAL cost; see the specialist's figures. Also not split: the IOS's 11,861 buffer hits between visibility-map and heap pages.

### What changes my mind
- **B1:** a revised A1 rule that scores branched counts by section 4's rule (no list crosses 150 ms that passes today), and not by +15 ms over today. Under that rule my data passes, because no branched cell crossed 150 ms only when ordered. **My view:** the +15 ms rule is wrong. It penalises a branched arm for the L3 loss that D1 accepts for unbranched counts. I could not say so before W5 ran, and the Solution Architect owns the rule.
- **B2:** rewrite L6 and A4-R3 with the table above, and decide whether 9 to 11 s on the first R list after a restart (today 1.5 to 4 s) is accepted. The section 4 rule does not cover it: today's cold request also misses 150 ms. If it is not accepted, the covering index removes it for the count; the page keeps its own cold walk.

### How to rerun
1. Check out `67959c4d`, apply `spike-toggle.diff`, and put `zz_spike1143_integration_test.go` in `internal/server`.
2. `go test -c -tags integration -o spike.exe ./internal/server`
3. `NAVA_TEST_PG_URL=… SPIKE_ONLY=W1 SPIKE_COV=1 SPIKE_DIR=… ./spike.exe -test.run '^TestSpike1143$'`. `run.sh` runs it gated on host load; `cold.sh` and `cold2.sh` do the cold reads.
4. Raw output is in `out-run{1,2,3}.log`, `cold*.log` and `contention-run*.log`; `an.py` computes the deltas.

All of these files are in `tick1143-ta-spike.zip`, attached. The container and volume are removed, and the throwaway code is out of the checkout.

RESULT: DONE
