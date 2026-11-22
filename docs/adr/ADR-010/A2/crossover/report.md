<!-- A2-12 (a) crossover cells and R2-B1 (Technical Architect), archived verbatim from TICK-1040 comment 01a1fdfb-00d3, 2026-11-21. -->

## A2-12 (a) crossover cells: C3 and W0 cross at m â‰ˆ 3,650, so c â‰ˆ 0.8. Rule 1 holds. Rule 2, as a groupsync tree refusal, doesn't work

**Verdict:**
- **(a) passes the A2-12 test.** C3 and W0 cross inside the clamp, at m â‰ˆ 3,650 for N_e = 400k and page 51, which gives **c â‰ˆ 0.81**. The worst cell at K\* for the region-only caller is **+25.5 ms median / +27.2 ms Î”p95** (uniform m = 4,500). SP-1e measured +85 ms for the same caller at K = 1,000.
- **R2-B1 rule 1 holds.** A rare type at root drops from +71 ms to +7.7 ms median.
- **R2-B1 rule 2 doesn't work as a groupsync refusal.** The cost grows with units in S only when the type is sparse there. Any tree-size cap low enough to bound it would refuse the bench's own 10,111-unit tree, where dense types cost +8 ms at root. That part needs your decision; see below.

All the cells are done. **The container `nava-aclperf-pg-2660` and the worktree have been removed**, and no `nava-aclperf-*`, `nava-it-*` or `nava-ci-*` container is up. [@Quality, Testing & CI Advisor](mention://agent/35ba247c-fae0-4437-b202-18b5012c0bbe): the host is free for your two baseline runs on TICK-946. But read the harness note at the end first, because as things stand those runs cannot score.

### Setup and scoring
- **Code:** `TICK-946/in-my-region` at `79e8c7b1` (with PR #820). The SP-1e harness is reused, plus one throwaway phase file, attached as `sp1x.go.txt`.
- **Data:** the fresh 800k seed, SP-1c's districts and SP-1e's districts, with the B1 key. Added for these cells:
  - a time-uniform m = 4,500 district;
  - `bench_rare`: 500 records, one per random leaf across the tree;
  - `bench_conc`: 20,000 records in a single district of region 9.
- **Scoring:** paired wall-clock, 300 pairs per cell, behind P3-36's floor gate.
  - Every reported chunk passed the gate at its start and its end (p50 1.08 to 1.68 ms).
  - One chunk was refused at its end (p50 3.25 ms) and was rerun in full. Only the rerun is reported; both passes agreed within about 4 ms.
  - The `nava` dev stack (12 containers) was up throughout, as it was in SP-1e. Î” cells within Â±3 ms are not separable.
- **Equality:** every mode returns the reference's ids, in order, and its capped count in all 17 sets. That covers C3 forced, W0 forced, MC3 at K\* and rule 1, for 20 agents and for the count-only caller.

### X1: the crossover (count-only caller, page shape forced, CC count, median / Î”p95 in ms)

| uniform m | C3 forced | W0 forced | cheaper |
|---|---|---|---|
| 1,001 | +9.5 / +12.2 | +61.5 / +92.5 | C3 |
| 3,000 | +28.1 / +32.7 | +37.9 / +45.7 | C3 |
| 4,500 | +31.8 / +40.0 | +19.1 / +26.6 | W0 |
| 10,000 | +38.4 / +46.5 | +12.5 / +16.0 | W0 |

The crossover, interpolated linearly on the medians, is m â‰ˆ 3,650, so **c = 3,650 / âˆš(51 Ã— 400,000) = 3,650 / 4,516 â‰ˆ 0.81**. At the switch, both shapes cost about +30 ms in this pass.

### X2: MC3 at K\* = 4,516 (c = 1), all eleven ticket sets (median / Î”p95 in ms)

| Set (\|S\|, m) | count-only (region-only caller) | agents | SP-1e count-only at K = 1,000 |
|---|---|---|---|
| area (10, 220) | +6.2 / +11.5 | +3.6 / +5.1 | +6.1 |
| district (101, 3,379) | +19.2 / +23.2 | +20.3 / +22.5 | +31.5 |
| thin (101, 600) | +8.7 / +13.5 | +9.1 / +12.5 | +9.1 |
| hot (101, 22,996) | +10.8 / +13.1 | +8.3 / +9.2 | +8.2 |
| region (1,011, 37,930) | +9.7 / +14.5 | +6.3 / +10.6 | +9.4 |
| root (10,111, 380,048) | +8.4 / +13.1 | +4.9 / +9.7 | +5.5 |
| mixed (101, 900) | +9.9 / +12.5 | +9.4 / +12.7 | +9.5 |
| uniform m = 1,001 | +9.7 / +12.6 | +8.4 / +13.5 | **+85.4** |
| uniform m = 3,000 | +18.8 / +22.7 | +18.1 / +24.2 | +47.7 |
| **uniform m = 4,500** | **+25.5 / +27.2** | **+22.6 / +25.6** | not run |
| uniform m = 10,000 | +15.2 / +20.0 | +5.2 / +7.4 | +20.5 |

- **c = 0.81 cross-check** (K\* = 3,658; count-only; a noisier pass with baseline p95 up to 43 ms):
  - district +23.3, thin +10.0, uniform m = 1,001 +10.1, m = 3,000 +24.8, m = 4,500 +22.0, m = 10,000 +20.7 (medians).
  - The worst cell is +24.8, against +25.5 at c = 1. That difference is within host noise.
- **What to record:** A2-12 (a) says to set c to the measured crossover, so record **c = 0.8** (K\* â‰ˆ 3,600 at bench scale). It does not measurably change the worst case: **about +25 ms median at the switch, for both callers.**
  - That is the top of your inferred +15 to +25 ms.
  - These are deltas in the probe harness against a baseline p95 of about 19 ms. They are not handler numbers. The budget verdict is still TICK-1037's.

### X3: R2-B1 (count-only caller, median / Î”p95 in ms)

| Set (\|S\|, m, N_e, K\*) | MC3 at K\* | Rule 1 (W0 page + W0 count, no probe) |
|---|---|---|
| rare, district (101, 6, 500, 1,000) | +9.3 / +27.9 | +6.8 / +10.5 |
| rare, region (1,011, 46, 500, 1,000) | +15.2 / +24.9 | +7.1 / +8.8 |
| **rare, root (10,111, 500, 500, 1,000)** | **+71.1 / +122.8** | **+7.7 / +15.4** |
| conc, district (101, 0, 20,000, 1,009) | +5.0 / +8.5 | not applicable (N_e > K\*) |
| conc, region (1,011, 0, 20,000, 1,009) | +11.0 / +12.4 | not applicable |
| **conc, regions 0â€“8 (9,099, 0, 20,000, 1,009)** | **+93.9 / +151.6** | not applicable |

- **Server-side diagnostic** (EXPLAIN): rare at root costs P3 23.9 + CC 28.0 + C3 17.7 = 69.6 ms under MC3, and 2.7 + 1.4 ms under rule 1. W0 is *not* a fallback for the conc type: its W0 count takes 64 to 288 ms. So rule 1 must stay gated on NÌ‚_e â‰¤ K\*, as you wrote it.
- **Rule 2 sizing:**
  - The conc cost rises at about **10 Âµs per unit in S** (+11.0 ms at 1,011 units, +93.9 ms at 9,099).
  - Keeping it under the K\* worst (+25 ms) needs |S| â‰¤ about 2,400 units. A groupsync refusal at that size would refuse the bench's own tree, which A2 certifies for dense types (+8.4 ms at root).
  - The cost comes from how densely a type sits in S, not from the size of the tree. So a tree-size refusal is the wrong tool.
- **Options, your call:**
  1. **Cap at the certified size.** Keep rule 2 as a groupsync refusal at the size A2 measured: 10,111 units in the tenant tree. Add a residual row, A2-R6: "a type sparse under a large S costs about 10 Âµs per unit in S, +94 ms at 9,099 units for a region-only caller". Its trigger leads to option (iii).
  2. **Find a sparse-type rendering.** Look for a rendering that doesn't visit empty units, for example a hash join on the closure side when the type is sparse. That is a new spike, not measured here.

  I recommend option 1 for A2 now. Option 2 can be a follow-up if A2-R6 fires.

### Harness note for QA (Known)
The P3-36 handler row in `run.sh` waited 30 minutes and then failed with "host not isolated". The reason is that since PR #820, the 12 `nava` dev-stack containers count as intruders: they are not in `perfAllowedContainers`. **So no list or get row can score while the dev stack is up.** Either the dev stack is stopped for the baseline window (it isn't mine to stop), or the allow-list question goes back to the Technical Lead.

### Attached
- `a212-raw.md`: every table, gate line and plan.
- `sp1x.go.txt`: the phase code, throwaway. To rerun it: put it at `bench/acl-perf/region/probe/sp1x.go`, add a `case "x-setup", "x-check", "x-plans", "x-wall"` dispatch in `main.go`, then run `probe.sh setup`, `e-setup`, `x-setup`, `x-check`, `x-plans` and `x-wall`. `SP1C_ONLY` selects cells, and `SP1X_C` sets c.

Nothing was committed.