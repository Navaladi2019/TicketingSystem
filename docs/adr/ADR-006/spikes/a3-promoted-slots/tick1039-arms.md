# TICK-1039 arms (copy of docs/data/field-index-measured.md, TICK-1039 sections, PR #920 head 60d1f0dd, scored on 660e243a)

## TICK-1009 and TICK-1039: scored on a fully idle host (2026-11-28)

**Host:** fully idle. The only other containers were the `multica-*` platform's: `multica-postgres-1` peaked at 15.9% CPU and the others stayed under 7%. There were no `nava`, `nava-*`, `nava-it-*` or bench containers, and no other agent's Go work. The transport floor read p50 0.54 ms after the seed, 0.61 / 1.68 ms before and after TICK-1009, and 0.69 / 1.52 ms before and after TICK-1039, so the p50 ≤ 3 ms gate passed at every check. The no-filter controls read 46.3–82.7 ms p95.


### TICK-1039 (A1-6): which arm passes

**Arm (b), the promoted column with `assignment_group_id` in its index, meets the exit criterion in every fixture: 11 of 11, 27.2–131.3 ms.** The other arms fall short:

- **(a), the adaptive shape:** 2 of 11. The pre-count sent every request to the walk-and-probe shape, because every fixture value has more than 4,500 side rows. When the value is anti-correlated or invisible, the walk probes `rfv_num_pkey` once for every visible record: 24,339 to 392,102 inner loops and 1.18–1.96M buffers, for 1,178–4,377 ms. (1) of A1-6 bounds the page only when matches are independent of `updated_at`, and these fixtures are built so that they are not.
- **D7:** 2 of 11. Its worst case is the a50 agent with a value none of its records carry: 2,223 ms, a walk over 200,113 rows of `records_list_updated`.
- **(b) without the group key:** 10 of 11. Its one miss is the a50 agent with the broad 50% value, at 163.6 ms (provisional), where the count joins all 200,000 index entries.

Neither arm needs a change to RLS, to CI-2 or to D7's `EXISTS` semantics. Arm (b) has a different cost, which is a design call and not mine to make. Each field that needs the band takes per-field DDL on `records`: a column, plus an index of 66 MB per 800k rows. I didn't measure the write amplification of that extra index on `records`. This is the schema-less trade-off ADR-006-A1 lists under Rejected. The ADR-006 amendment, if one follows, is the Technical Lead's to request from the Solution Architect.

| Fixture (p95 ms, line 150) | D7 | (a) | (b) | (b) + group |
| --- | --- | --- | --- | --- |
| anti-correlated 2%, a6 | 192.9 | 1,178.0 | 71.7 | 66.8 |
| anti-correlated 2%, a50 | 216.1 | 2,362.3 | 63.6 | 69.4 |
| anti-correlated 2%, no ACL | 355.2 | 4,376.6 | 50.9 | 76.6 |
| none visible, a6 | 124.8 | 1,249.9 | 78.6 | 27.2 |
| none visible, a50 | 2,223.2 | 2,662.0 | 79.8 | 30.5 |
| partly correlated, a6 | 211.3 | 349.7 | 99.7 | 62.9 |
| partly correlated, a50 | 420.9 | 534.9 | 101.8 | 74.1 |
| 19,999 invisible, a6 | 135.3 (provisional) | 1,204.6 | 71.9 | 29.6 |
| 19,999 invisible, a50 | 242.0 | 2,301.6 | 69.9 | 31.0 |
| broad 50%, a6 | 405.6 | 84.9 | 79.4 | 131.3 |
| broad 50%, a50 | 212.4 | 85.8 | 163.6 (provisional) | 60.9 |
| **cells that pass** | **2 / 11** | **2 / 11** | **10 / 11** | **11 / 11** |

### How the cells were built

Run: `sh scratch/field-index-measured/spikes.sh` (`NAVA_PERF_PHASES=edges,arms`
on the TICK-992 harness). One 800k seed serves both. `edges` (TICK-1009) runs
first, because `arms` (TICK-1039) rewrites the records heap. Cell code:
`internal/server/records_fieldindex_spikes_perf_integration_test.go`.

| Spike | Cells | Line |
| --- | --- | --- |
| TICK-1009 S1, band edge | `num lt` (`score` lt 500 / 1000 / 2000) and `text eq` (`tier` `v5` / `v10` / `v20`) at 5 / 10 / 20%, with the agents' ACL (6%) and without: 12 | 5%: 350 ms; 10%, 20%: 150 ms (over is a new band, reported) |
| TICK-1009 S2, ACL breadth | 2% `num lt` and `text eq`, agents seeing 6% (control), 26% and 50%: 6 | 350 ms |
| TICK-1039 arms | D7, (a) adaptive, (b) promoted column, (b) with `assignment_group_id` in the index, × 11 `pk` fixtures (below): 44 | 150 ms |
| Both | no-filter controls and transport floor before and after each phase | floor p50 ≤ 3 ms |

Fixture changes, none of which move a TICK-992 band:

- **`tier`** gains `v5` 5%, `v10` 10% and `v20` 20%, taken from the 97
  others. Those keep the remaining 22.74%. `v40` / `v2` / `v025` / `v001`
  keep 40 / 2 / 0.25 / 0.01%, and the run logs every share.
- **Broader agents.** `a25-N` and `a50-N` use `agent-N`'s principal, with 13
  and 25 of the 50 assignment groups. The existing `agent-N` (3 groups) is
  unchanged. The group sets nest, 6% inside 26% inside 50%. 25% is not a
  multiple of one group's 2%, so a25 sees 26%. Measured: agent-1 sees 24,666
  records (6.2%), a25-1 sees 104,490 (26.1%) and a50-1 sees 200,402 (50.1%).
- **`pk`** (num, large type only, one side row per record). TICK-1039's a6
  is `agent-1` and its a50 is `a50-1`:

  | Value | Side rows | Visible to a6 / a50 | Fixture |
  | --- | --- | --- | --- |
  | 40 | 8,000, the oldest by `updated_at` | 484 / 4,011 | A1-6 (3): anti-correlated 2%; a6's count stays under the cap |
  | 10 | 26,593 | 0 / 0 | none visible |
  | 20 | 21,487 | 494 / 494 | partly correlated |
  | 30 | 19,999 | 0 / 0 | the R13 step's lower side |
  | 3 | 200,000 | 15,819 / 133,409 | broad 50% |
  | 4 | the remaining 123,921 | – | – |

  Unlike A2-11, a50 is a superset of a6, so a50 also sees the 494.

Arms. Each arm goes through the real handler, ACL and RLS, with
`WithListIndexTerm`, a compiler seam that exists only under the
`integration` tag.

- **(a), adaptive.** A capped pre-count (`LIMIT 4501`) of the value's side
  rows runs in the list's transaction. If it finds at most 4,500, the list
  keeps D7's `EXISTS`, which drives from the side index. If it finds more,
  the `EXISTS` gets `OFFSET 0`. That blocks the semi-join pull-up, so the
  list walks the records index in list order and probes `rfv_num_pkey` once
  per record. The count does the same per group-visible record, up to the
  cap. The shape forces the probe, with no `enable_*` and no hints. The cap
  was fixed before any run: sqrt(51 × 400,000) ≈ 4,500 is where a 51-row
  walk (51 / s probes) costs the same as reading s × 400k side rows.
- **(b), promoted column.** `records.spike_pk` with an index on (tenant_id,
  entity_definition_id, spike_pk, updated_at), then the same with
  `assignment_group_id` before `updated_at`. This is scratch DDL inside the
  perf test and is dropped at the end. The column is filled before any arm
  runs, and `VACUUM FULL records` follows, so every arm reads the same heap.
  Index sizes: 52 MB and 66 MB for 800k rows.

## TICK-992: scored on the reference machine (2026-11-19)

**Host condition: `nava` stack running at idle, waived by the Technical Lead
on TICK-992.** These numbers were not taken on a fully empty host. The
`nava-s18` stack (TICK-669, done) was stopped first (`docker compose -p
nava-s18 down`, volumes kept). The owner's `nava` stack (15 containers) kept
running. A `docker stats` sample before each run put it at 21% of one core
