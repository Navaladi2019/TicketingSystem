<!-- TICK-1143 comment 01a2154c-8280-7a5b-9f0c-e476f7e5733c, 2026-11-26T00:41:05Z. Archived verbatim; agent mentions removed. -->

**P2 and P4 are resolved. P1 is resolved in its world, but the 1.3× guard is not sound as written. P3 (CI-5) is left open by the covering index itself.** Plans only, as before. My 0.5-CPU container ran 00:38:39–00:39:44 and 00:40:31–00:40:44 UTC, and is removed.

| # | Status | Basis |
| --- | --- | --- |
| P1 | **resolved, guard to fix** | The world is right. With the index, the count is `Index Only Scan records_list_updated` in 4 of 4 loads (contiguous and scattered), at 10,111 and 20,222 units. Mutant `r_count_no_covering` is a Seq Scan in 4 of 4. The guard is wrong; see below. |
| P2 | resolved | D5 limits the pins to the CI worlds and names every plan. |
| P3 | **not resolved, caused by the index** | The failing cell is below. The bitmap + Sort pass and the requester policy are taken. |
| P4 | resolved | 36–67 ms; the L3 and section 7 corrections. |

## The 1.3× guard: it measures the closure, not the margin

`enable_indexonlyscan = off` also takes the Index Only Scan away from the hashed SubPlan's read of `org_unit_closure`. That SubPlan costs 655 with an Index Only Scan and 2,257 without one, and the same extra cost lands on every runner-up.

| CI-4 world, 10,111 units | Chosen | IOS off | Guard reads |
| --- | --- | --- | --- |
| Covering index | IOS, 832 | Seq Scan, 3,010 | 3.62× |
| **Today's index (the mutant)** | Seq Scan, 1,414 | **the same Seq Scan**, 3,016 | **2.13×** |
| CI-3, today's index | bitmap, 4,673 | **the same bitmap**, 6,276 | 1.34× |

The guard passes 2.13× when the chosen plan and the runner-up are the same plan, so it would also pass at a real tie.

**Fix:** compare only the `records` scan node's own run cost, total minus startup. The startup holds the hash build. On this world that gives IOS 1,724 against Seq Scan 8,905, **5.2×**, the same at both sizes. The world itself is sound.

**Smaller point:** R.5 pin 2 ("`Rows Removed by Filter` holds no other tenant's rows") cannot be read from a count. On an IOS that figure is the type's own non-matching entries. Instead, assert that the node's `Index Cond` holds `tenant_id`.

## P3 / CI-5: the R arm is not an IOS when it carries an earlier branch's negation

The arm's predicate then names a column the index does not carry, here `requester_id`.

| Policy, with the covering index | CI-4 world | CI-3 world |
| --- | --- | --- |
| `under` first, then requester (no negation on R) | R arm: **IOS** | R arm: **IOS** |
| **requester first, then `under`**: R arm has `(requester_id = …) IS NOT TRUE` | R arm: **Seq Scan**, estimated 12,465 | R arm: **Bitmap Heap Scan** |
| Same, with D3's order on the R arm | R arm: `Index Scan records_list_updated`, with heap reads | the same |

- **What fails:** D5's CI-5 pin, and R.1's "branched: the same plan class as today with one cheaper path".
- **It reaches Security's item too:** in the CI-4 world that R arm reads 50,000 rows of other tenants. Any policy that puts the region leaf after another branch (requester, assignee, group) is affected.
- **Ways out:**
  1. Fire D4 now for R arms that carry a negation. That is "both" for branched counts, and B2's random cold walk returns there.
  2. `INCLUDE` each column that a negation can name.
  3. Record it as a residual with its own trigger.

That choice is the Solution Architect's.

VERDICT: CHANGES (the P1 guard, the P3 cell above)

RESULT: DONE
