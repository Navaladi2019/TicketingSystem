<!-- TICK-1143 comment 01a2154b-2cbe-77a0-8987-fc9a1991966d, 2026-11-26T00:39:37Z. Archived verbatim; agent mentions removed. -->

## Re-check, RFC-023 rev 1 (Technical Architect): B1 and B2

**B1: resolved for the design rev 1 adopts. B2: not resolved in one cell, which the covering index itself creates.** The fix is one sentence of A4-R3 plus a merge-order condition. It needs no new design. No new timing run: all figures are from the logs in `tick1143-ta-spike.zip` (the script is below).

**VERDICT: CHANGES (B2 only, closed by either option at the end).**

### B1: the +15 ms rule

**Resolved.** The replacement rule and its reason, that +15 ms counted a loss rev 0 had already accepted and could not isolate the union effect, are sound before the data. Under rev 1, D2 leaves the count text unchanged, so no branched count changes and B1 cannot arise. The rule now governs only the D4 fallback.

**W6 A/A: the pass holds, with one caveat.**

| W6 R\|self m = all | Excess over the unbranched loss | Within-row A/A (today vs today) | Same statement across rows (ordered m = 0) | Ordered, branched vs unbranched |
| --- | --- | --- | --- | --- |
| Run 1 | +4.6 | 2.4 | range 5.4 | 56.0 vs 50.1 |
| Run 2 | +1.3 | 2.6 | range 6.6 | **39.4 vs 39.4** |

- **The caveat:** read against the within-row A/A, as R.3 words it, run 1 is 2.2 ms over the spread.
- **Why it still passes:** the comparison is between two rows, so the right noise figure is how far the same statement moves from row to row: 5.4 and 6.6 ms. Both excesses are inside that.
- **Where the excess comes from:** in run 2 the ordered values are identical, so the excess is in today's arm, not the order.

**Correction to R.3's table** (it matters only if D4 fires): the new rule **flags every count with two R arms**, beyond both noise figures.

| Cell | Excess over the unbranched loss | Within-row A/A |
| --- | --- | --- |
| W6 R\|R 500\|1000 | +14.7 / +13.6 | 2.4 / 2.6 |
| W3 R\|R 500\|1000 | +15.4 / +23.9 | 10.8 / 6.7 |
| W4 R\|R 500\|1000 | +6.3 / +3.5 | 2.0 / 2.3 |
| W5 R\|R+open m = 0 | +7.1 / +19.1 | 1.3 / 7.9 |

- **Mechanism** (Known from the plans): today, once the first arm has counted 500 matches, the second arm stops early at 501. Ordered, the second arm has to read the whole type's slice before its Sort or walk can return rows.
- **Cost:** none of these crosses 150 ms. Each stays inside A3-5's bound of the type's slice.
- **"W5 passes" is right only for R\|self.**

**Text for D4, so that the fallback says what a flag does:**

> If D4 applies, the order is added only to counts with at most one R arm. A count with two or more R arms keeps A3-1's text.

### B2: cold cache

**Resolved for the count.** The IOS count reads 18 to 25 ms cold at m = 500 and 42 to 67 ms at m = 0 (W1, `cold2.log` and `cold3.log`). The 9 to 11 s count is gone, as R.4 says.

**Not resolved: the covering index makes one measured cell worse than today, so it fails R.1's own rule ("no measured cell worse than today").**

| W1, cold, `main` order (count, then page), m = 0 | Count | Page | Request |
| --- | --- | --- | --- |
| Today | Seq Scan, 1,349–1,806 | 106–145 | **1.45–1.95 s** |
| Covering index | IOS, 42–67 | 8,722–11,178 | **8.8–11.2 s** |

- **Why:** today's Seq Scan count reads the 325 MB heap in order and leaves it in shared_buffers, so the page that follows it is cheap. The IOS never reads the heap, so the page does its own random walk.
- **The false sentence:** R.4's "A4 does not change it" is false in `main` until ADR-006-A2 A2-4 lands.
- **After A2-4:** the count at m ≤ limit is skipped, the page walks cold in either case, and the sentence becomes true.
- **Known:** m = 0 only. **Inferred:** first pages with 1 ≤ m ≤ 50 behave the same way, because the page also walks the whole slice there.
- **With S21's order, m = 500:** the index is better than today. Today takes 2.5–4.0 s (page 1,269–2,267 ms + count 1,216–1,703 ms); with the index, 1.27–2.01 s (page 1,256–1,987 ms + count 18–25 ms).

**Either of these closes B2:**

1. **Sequencing:** A4's migration merges into `main` only after A2-4 (S21) is in `main`. That belongs to the Technical Lead's section 11 decision, which is already pending.
2. **Accept the window:** add to A4-R3:

> Until ADR-006-A2 A2-4 is in `main`, A4 makes the first cold R list with at most `limit` matches slower: 1.45–1.95 s today (the Seq Scan count warms the heap for the page) against 8.8–11.2 s with the covering index (W1, m = 0, measured). After A2-4 the count is skipped there, and the page's cold walk is the same with or without A4.

### How I computed it
This is `an2.py` over `out-run1.log` to `out-run3.log` and the cold logs, all in the zip from my round 1 comment. Every figure above is a median from those runs. Run 3 measured W3 twice; the script keeps the second pass.

RESULT: DONE
