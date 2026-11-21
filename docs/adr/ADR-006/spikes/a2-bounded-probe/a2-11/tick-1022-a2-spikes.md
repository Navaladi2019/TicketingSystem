# TICK-1022: ADR-006-A2-11 spikes on the compiler branch

**Verdict against A2-11's pre-stated rules:**

| Spike | Result |
| --- | --- |
| **S-A2-skew** | **Pass.** The pre-count is an Index Only Scan on `rfv_num_value`: 14.7 ms with 0% of pages all-visible, 0.6 ms with 100%. The limit is 20 ms. |
| **S-A2-anti** | **Fail at K = 30,000.** The 6% agent's anti-correlated cell is +341 / +517 ms over "before". The 6% partly correlated cell is +150 / +557 ms. Lowering `K` cannot fix it for this agent shape (see below). |
| **S-A2-multi** | **Fail.** Three broad fields are +118 / +252 ms over "before". |
| R13 step | Measured: 19,999 side rows cost +0.6 / +94 ms over "before", and 20,000 cost +230 / +466 ms. Crossing the cap moves single-client latency by about 225–370 ms. |

## Setup

- **Code.** Branch `TICK-1022/count-probe` at `64a6160e`: PR #812 with guard A2-3a (`K` = 30,000) and the ordered pre-count.
- **Harness.** `TestA2Spikes` (`internal/server/records_fieldindex_a2_spikes_perf_integration_test.go`), run with `scratch/tick-1022-a2-spikes/run.sh`.
- **Data.** P7-07's 800k seed: 400 tenants, one of them with 400,000 records of one type.
- **Agents.** Two agents whose visible sets are exactly one group's records:
  - a6 sees 24,055 records (6%);
  - a50 sees 200,379 (50%).
- **Fixtures on num field `pk`**, one side row per record:

  | Value | Side rows | Visible to the agents |
  | --- | --- | --- |
  | 10 | 26,593 | none |
  | 20 | 21,487 | 494 visible to a6 |
  | 30 | 19,999 | none |
  | 31 | 20,000 | none |
  | 77 | 50 here, plus 200k in tenant B | – |
  | 3 / 4 | the rest, half each | – |

  Text field `pt` and bool field `pb` are half each.
- **Arms.**
  - "before": today's SQL. The caps are set so nothing is broad.
  - "A2": the production caps.
- **States.** Visibility map `none` (ANALYZE, no VACUUM, 0% all-visible) and `vacuum`. Autovacuum is off on the side tables.
- **Host.** Windows 11, i7-1260P, Docker 29.9.2 on WSL2 (16 CPUs, 24.5 GiB), `postgres:16.10` with P3-01's settings (`work_mem` 4 MB, `random_page_cost` 4, `nava_app` at `force_custom_plan`). The S21 compose stack was active during the run (its worker at up to 28% CPU). Server times are comparable within this run, not with run 5: the run-5 pre-count read 11–18 ms, here 9–110 ms.

## Cells: single-client server time (planning + execution, ms)

Each cell shows the count before, then the A2 total of pre-counts + guard + count, then the difference.

| Spike | Cell | `none`: before → A2 (Δ) | `vacuum`: before → A2 (Δ) |
| --- | --- | --- | --- |
| anti | a6, value on 26k invisible records | 122.6 hash → 82.9 + 88.8 + 468.0 = 639.7 probe (**+517.1**) | 135.3 hash → 37.1 + 72.4 + 367.0 = 476.5 probe (**+341.1**) |
| anti | a50, the same value | 513.5 hash → 523.1, guard keeps the hash (+9.6) | 317.6 hash → 204.9, guard keeps the hash (−112.8) |
| anti | a6, partly correlated (494 visible) | 92.8 hash → 650.2 probe (**+557.4**) | 54.5 hash → 204.2 probe (**+149.8**) |
| step | a6, 19,999 invisible rows | 65.7 hash → 160.0, not broad (+94.3) | 28.9 hash → 29.5, not broad (+0.6) |
| step | a6, 20,000 invisible rows | 144.3 hash → 609.9 probe (+465.5) | 25.3 hash → 254.7 probe (+229.5) |
| ref | a6, broad 50% value | 256.0 hash → 172.6 probe (−83.4) | 33.8 join → 51.3 probe (+17.4) |
| ref | a50, broad 50% value | 38.7 join → 190.8, guard keeps the join (+152.1) | 100.1 hash → 183.9, guard keeps the hash (+83.8) |
| multi | a6, three broad fields | 384.8 hash → 637.1 probe (**+252.3**) | 137.2 hash → 255.0 probe (**+117.8**) |

8-client p95 for each cell, the page's own time and every plan: `tick-1022-a2-spikes/raw.md` and `plans.md`.

## What the cells show

1. **The probe's walk costs about 15–19 µs per visible record here**, against the 4–7 µs the rev-0 projection assumed. Each record is one `rfv_num_pkey` seek of four buffer hits. For a value the agent's records don't carry, the count walks all 24,055 of them: 367–468 ms, against a hash of 123–135 ms.
2. **No `K` works for the agent shape TICK-1022 is about.** The kit's Alice sees about 5% (20k records), close to a6.
   - A `K` above the visible set lets the anti-correlated walk through (+341–517 ms).
   - A `K` below it never probes, so the broad cell keeps today's hash (256 ms in `none`).
   - For one agent, the visible-set size can't tell "the value is on my records" from "it is on someone else's".
3. **The guard and pre-count cost more than measured in run 5.**
   - The visible-set count of 24k records costs 24–112 ms (branched arms over `records_list_group`), against the 1.5 ms projected for 4k.
   - The pre-count costs 9–110 ms (20k index entries with heap fetches at 0% all-visible).
   - Those two alone break "+10 ms" on the cells where A2 then keeps the old plan (a50 `ref`: +84 / +152 ms).
4. **The R13 step is large.** At 6% visibility, an invisible value with 20,000 side rows answers about 230–470 ms slower than one with 19,999. That signal is easy to read from outside.
5. **The page also walks.** The anti-correlated value on a50 reads 2.2–3.4 s in the page statement (51 matches never come). That is TICK-992's page half, outside A2. It is recorded because the same fixture shows it.

## Reshape candidate (for the Solution Architect; not measured)

Replace the pre-count and the visible-set guard with one **visible sample**: probe the first N (say 256) records the caller can see, in the page's index order, and probe in the count only if at least a share r of them match (say 25%).

- **Worst case.** The probing count then stops at 1,001 matches within about 1,001 / r = 4k probes, about 60–80 ms at the rates above, whatever the side table holds.
- **Anti-correlated and partly correlated values** keep today's plan for the cost of N probes (about 4–5 ms).
- **R13 goes away:** the decision reads only records the caller may read. No pre-count of hidden rows remains.
- **Residual: sample bias.** If the first N visible records carry the value and the rest don't, the walk can run long. That needs its own spike.

A2-11's fallback, option (C), stays open if the reshape fails too.
