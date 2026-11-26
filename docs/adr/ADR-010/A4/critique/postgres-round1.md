<!-- TICK-1143 comment 01a214ad-24b7-771a-8718-0f9b202f4f20, 2026-11-25T21:47:01Z. Archived verbatim; agent mentions removed. -->

**D2 to D4 hold, and L4's formula is right: RLS counts the tenant once. Four things must change first: the world CI-4 is red in, A1's pass rule, the three-plan statement, and the "36–50 ms" bound.** None of them touches the recommendation; I support adopting.

**Method.** Plain `EXPLAIN`, no timings: PostgreSQL 16.10, `nava_app` under FORCE RLS, `effective_cache_size` 6 GB, `jit` off, literals, |S| = 10,111. I load a fraction of each cell, `ANALYZE` it, then extend the heap and index files so that the planner sees the full size. Calibrated on 11 cells of my bench: 10 gave the same class of plan, and the other is a cost tie (below).
**Overlap for the Technical Architect:** my container was capped at 0.5 of 16 CPUs, 20:45–21:40 and 21:41–21:46 UTC. It loaded at most about 50,000 rows at a time. The only statements executed for a figure were four 2,000-row `UPDATE`s, for WAL bytes. It is removed.

## Changes

| # | Label | Finding and fix |
| --- | --- | --- |
| **P1** | blocking | **CI-4's 420,000-row world is close to a tie, and Security's S3 world is one** (table below). `r_count_unordered` "must give a Seq Scan" would flap there. **Fix, covers S3 too:** 200,000 empty rows, N̂_e = 33,366, share 75 % (50,000 rows of other tenants). The Seq Scan leads by 53 % unordered. Ordered, it is `Index Scan records_list_updated` at 2,450–2,470, with bitmap + Sort at about 9,800. It is also smaller. I have not run it in the repository's suite. |
| **P2** | blocking (wrong as written) | **D6 and section 1: the ordered count has three plans, not one.** Index walk; bitmap + Sort (the type's slice, so A3-5 holds); Sort over a Seq Scan (small tables). "One index alone gives the order" is true only for no filter and an estimate well above 1,001. Keep the pin, for the CI worlds only, and say so. |
| **P3** | blocking (undecided result) | **A1 has no outcome for a bitmap + Sort.** It fails "no `Sort`", but only a Sort over a **Seq Scan** triggers anything. It happens: two R arms at 400,000 × 1 KB, share 13 %: the second arm's `NOT` halves its estimate to 1,074. Make it a pass. Build CI-5 on the requester policy (estimate 3,483), not on two R arms (1,743). |
| **P4** | blocking (misread) | **"36–50 ms" per walk is not my range.** My quiet-window index plans were 31 to 51 ms, **67 ms** (400,000 × 1 KB) and 110 ms (noisy). It feeds sections 1, 3 and L2 to L5. Use 36–67: page plus count then reaches about 119 ms, and A4's 80 ms line has 13 ms of room, not 30. |

| Unordered count: lead of the Seq Scan over the next plan | Lead |
| --- | --- |
| RFC CI-4: 420,000 rows, N̂_e 20,000, one tenant | 3.7 to 5.5 %; at 480,000 the two loads split |
| Security S3: 400,000, N̂_e 33,366, share 54 % | **−0.3 to 1.3 %** |
| 400,000, N̂_e 33,366, share 75 % / 91 % | 22 % / 38 to 41 % |
| 240,000, N̂_e 20,000, one tenant | 11 % |
| **200,000, N̂_e 33,366, share 75 %** | **53 to 54 %** |

- **My own headline cells are ties too.** 800,000 empty rows took an index in 4 of 6 loads; 250,000 × 1 KB stayed a Seq Scan in 6 of 6, but led by 1.6 % at most. My bench saw one sample of each. Technical Architect: check that "today" really is a Seq Scan before timing A2. 200,000 × 1 KB leads by 14 % and 600,000 empty by 10 %, both one tenant, N̂_e 33,366.

## Your questions

**D2, D3: confirmed.**
- The `DESC` default matches `records_list_updated`: no Sort, no Gather at 10,111 and 20,222 units, as literals and as `$n` under custom plans.
- **Union:** each child plans alone, and the requester arm keeps its own index. CI-5's checker must accept two reads of `records`; `recordsFaults` wants one.
- **m does not change the plan.** The hashed SubPlan is always estimated at 0.5, so m = 0 to all, and page 2, share one plan per cell.

**D4: right, for a stronger reason than given.** Every filter index ends in `updated_at DESC, id DESC`. With one value of `state`, assignee or requester, the filter's own index already gives the order. All three were chosen in the 1-tenant cells, so L5 overstates those. L5's mistake remains for `IN` lists (`state IN ('open', absent)`: `records_list_updated` at 4,737 against `records_list_state` at 3,288). Data filters: no GIN index in my copy, and S21's indexes are not tested.

**Single-index pin: stable in CI's shared world.** It is the cheapest by 1.9× (7,400 against about 14,200 for bitmap + Sort), at both sizes. That lead is far above sampling noise (about 3 %).

**L4:**
- **RLS:** PostgreSQL turns the policy into a `One-Time Filter`. Estimated 3,514 rows; counted once it is 3,549, twice would be about 1,260.
- **The flip is a band, not a line at 1,001:** always a Sort below about 1,000, an index walk above about 2,300, either one between.
- **Common SaaS case:** at 1 to 10 % share and N̂_e = 20,000, the ordered plan is today's bitmap plus a Sort (+0.2 to 0.7 % cost). There the order buys nothing and loses only the early stop at m ≥ 1,001, within the type's slice.

**A5 and Security's table:**
- **Where it is:** Sort over a Seq Scan, no filter, in 59 cells, all a Seq Scan today. The largest is 18,000 × 1 KB rows with N̂_e = 9,000. None at N̂_e = 14,000 or more (1 KB, ratio 2 to 4).
- **How much:** other tenants' rows never exceeded N̂_e, including in Security's filtered cells.
- **Security's trigger is already met:** "ratio of 4 or more" happens at ratio 5 (5,500 and 7,500 × 1 KB rows, one tenant, so no other tenants' rows). Please use "other tenants' rows read exceed N̂_e".

**A6:** no unfiltered plan had a Gather. Security's filtered one did.

**D7:** the generic ordered count was bitmap + Sort in 3 of 4 cells with 20 tenants, never a Seq Scan. Write "an index walk or a bitmap of the type's slice". Non-blocking.

**Other misreads:** L3's 0.15 µs per row includes the hash build, so its 240,000-row crossover is a guess. Section 7's shares (54 %, 91 %) need "at 400,000 empty rows". Everything else from my tables is quoted correctly.

## Covering index (`records_list_updated INCLUDE (org_unit_id)`, swapped in)

| | Figure |
| --- | --- |
| Index size | 77.8 → 95.6 bytes a row (+23 %); all `records` indexes of the bench schema +2.3 %. At 800,000 rows, about 62 → 77 MB |
| WAL | +31 bytes per update (+1.5 % empty rows, +1.0 % 1 KB). Same number of index entries; HOT is already lost to `updated_at` |
| Plan | Index Only Scan at cost 826 in the three cells I completed, against 2,800 to 7,300, **with or without the order**, so it alone removes the Seq Scan there |
| Limits | Any filter outside the index (`state`, for example) brings back the heap read (cost 2,820 to 7,380). Newest-first meets the pages vacuum has not marked yet. The page keeps its 44–52 ms: a plain Index Scan cannot filter on an included column |
| Change | `CREATE INDEX CONCURRENTLY`, drop the old one, rename; both exist meanwhile. Not timed |

It stays the right next step, not the first.

VERDICT: CHANGES (P1 to P4)

RESULT: DONE
