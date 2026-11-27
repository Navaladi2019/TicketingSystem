<!-- TICK-1143 comment 01a21488-7007-70cc-9b2d-6e5a8249cbcb, 2026-11-25T21:06:55Z. Archived verbatim; agent mentions removed. -->

**A4-R1 should be Low with its own register row, and its text and trigger need rewording. There is no isolation gap: I found no request that gets a row, a total or a shape choice across tenants.** All three changes are text or test scope. None touches the recommendation, and adopting is the better choice for security.

## Your three questions

| Question | Answer |
| --- | --- |
| Is Info right? | **No: Low.** I rated it Info on TICK-1104 and I'm changing that. The register's Info means "process cost or a known limitation". AR-TICK-1040-01 is Low for bits that stay inside one tenant, so a signal about other tenants should not rank below it. It is not Medium: it carries only an aggregate size, for all other tenants together. |
| Is naming it in the amendment only enough? | **No.** `docs/security/accepted-risks.md` is the one register of every accepted risk, and RFC-023 rightly says no existing row covers this one. It needs a row (text below). AR-TICK-1040-01 stays as written. |
| Does "removed where the index is picked, not provably gone elsewhere" match? | **The first half yes, and I can now support it. The second half is too weak:** the ordered count does still take a Seq Scan in cells I can name. |

## What I checked

Plans, row estimates and buffers only, no timings. PostgreSQL 16.10, `nava_app` under FORCE RLS, literals, the specialist's schema, m = 0, |S| = 10,111, `random_page_cost` 4 as the plan test asserts. I used a throwaway container limited to 1 CPU, now removed. It ran a few short loads on the shared host during this round, so the Technical Architect should check for overlap with any timing. Scripts attached.

**1. The index does remove it, now shown on cells that have other tenants.** The three one-tenant cells behind D1 hold nobody else's rows, so they cannot show the signal going away. The two shared custom-plan cells that were measured ordered were never a Seq Scan. I ran the two shared cells that were:

| 400,000 empty rows, N̂_e = 33,366 | Other tenants' rows | Today | Ordered |
| --- | --- | --- | --- |
| Share 91 % | 34,490 | Seq Scan, removes 400,000 rows | `records_list_updated`, removes 33,366 rows, 33,790 buffers |
| Share 54 % | 183,660 | Seq Scan, removes 400,000 rows | the same, 33,790 buffers |

Ordered, the buffers do not move while the other tenants' rows grow fivefold.

**2. It is still there in these ordered cells:**

| Cell | Ordered plan | Other tenants' rows read |
| --- | --- | --- |
| 2,200 empty rows, N̂_e = 2,000, share 91 %, no filter | Sort over Seq Scan | 200 |
| 66,000 × 1 KB, N̂_e = 33,366, share 51 %, filter `r.data @> …` | Sort over Seq Scan | **32,634** |
| 100,000 × 1 KB, N̂_e = 33,366, share 86 %, the same filter | Gather Merge, Sort over parallel Seq Scan | 14,494 |

- **Cause:** an estimate below 1,001, which takes the `LIMIT` discount away. In the two large cells that comes from L5 (a selective filter), not L4. In all 11 low-share cells I ran without a filter (10 to 50 %, N̂_e 2,000 to 10,000), the order gave a Sort over a **bitmap**.
- **It is smaller.** With the order, a Seq Scan is priced in full, so by PostgreSQL's estimates it wins only where the table is cheap next to the type's slice. The largest ratio (rows / N̂_e) I found is 3; I did not search for the upper end. Today it is at least 24.
- **A count without R does it too.** With a data filter and no region leaf, the count is a Seq Scan in the 66,000 × 1 KB cell (tested with `data->>`, not the handler's `@>`). So this is not new with R, and A4 does not make it worse.

**3. Only row and page counts show.** The Seq Scan's filter runs `tenant_id`, then `entity_definition_id`, then the hashed SubPlan, then the data filter. Another tenant's row stops at the first test, so the filter never evaluates its content.

**4. L4:** RLS does **not** count the tenant twice here. PostgreSQL merges it with the tenant literal into a `One-Time Filter` on `current_setting`. At share 51 % the type is estimated at about 17,000 rows of 33,366, so the share is applied once.

## Finding

- **Attack:** a user of tenant A holds a valid token and calls `GET /api/v1/records/{entity}` where the count takes a Seq Scan. Repeated timing gives a rough size of `records`; minus A's own part, that is the other tenants' total. With two tenants, it is the other tenant's size, and its growth over time. A can build the type, the units and the filter itself, and can raise its own share.
- **Status:** the plan is proven, reading it over HTTP is not measured. Shape R is not in `main` yet.
- **Severity:** Low. No row, no total, no per-record fact.
- **Lowest-layer fix:** D1 for the large cells. I accept the rest, in the row below.

## Changes

| # | Label | Change |
| --- | --- | --- |
| **S1** | blocking (wrong as written) | Sections 8 and 10: A4-R1 is **Low**, and AR-TICK-1143-01 is added to the register in the A4 PR, in the words below. |
| **S2** | blocking (wrong as written) | A4-R1's trigger ("a Seq Scan … in A5") is already met by the second table. Use the row's trigger. Replace "not provably gone" with "still present in small tables and under a selective filter". A5 says Security rules if a Sort over a Seq Scan appears: **my ruling is accepted at Low**, so it does not hold acceptance. |
| **S3** | blocking (the security claim has no test) | CI-4's one-tenant world has no other tenant's rows, and CI-3's shared world is not a Seq Scan today. So no test is red for this signal. Give CI-4 a second tenant: 400,000 rows, N̂_e = 33,366, A's share about 54 %, same pin, same mutant `r_count_unordered`. On the bench schema that cell is a Seq Scan today and an index scan ordered; I have not run it in the repository's suite. It can replace the 420,000-row world, which is the Solution Architect's choice. |
| S4 | non-blocking | L5 says "the ordering index instead of the filter's". It can also be a Sort over a Seq Scan, or a Gather Merge. D6's pins carry no filter, so they do not change. |
| S5 | non-blocking | D5: `Records` and `CountOrdered` carry the shape choice. Say that A3-7's never-exposed list covers them. The existing leak test is enough. |

**Row for the register:**

> **AR-TICK-1143-01** (a list count's time can include other tenants' rows) · 🟡 Low · ADR-010-A4. When PostgreSQL reads `records` with a Seq Scan for a list's count, the time also covers other tenants' rows. It can hint at how many records all other tenants hold together; with two tenants, at the other one's size. No row, total or shape choice crosses the tenant wall, and the tenant test runs first, so the filter never evaluates another tenant's row content. A4 removes it where R's ordered count walks `records_list_updated`. It remains in small tables and under a selective filter, where a Seq Scan is the cheaper full read, and in counts that are not R. · Accepted (Security & Multi-Tenancy Reviewer, TICK-1143) · Owner: Security & Multi-Tenancy Reviewer · Trigger: a requirement that a tenant's total record volume is confidential from other tenants; or CI-3, CI-4, CI-5 or TICK-1105 shows a Seq Scan for an ordered R count; or any ordered R count shows one at rows / N̂_e of 4 or more.

**If A4 is rejected:** still Low, the same row without its A4 sentence and with the unordered trigger. It would then grow with the table up to a ratio of at least 24. I would not block on that.

**Confirmed:** AR-TICK-1040-01's three bits do not depend on other tenants' rows. D2 to D5 add an order and no predicate; `$1` and RLS are unchanged, so I1 and I5 hold, and CI-7 is the right guard.

VERDICT: CHANGES (S1 to S3)

RESULT: DONE
