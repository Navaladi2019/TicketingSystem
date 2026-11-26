<!-- TICK-1143 comment 01a2154c-b8d4-7e18-a7b2-3c88c4f10af8, 2026-11-26T00:41:19Z. Archived verbatim; agent mentions removed. -->

**S1 and S3 are resolved. S2 is resolved only with the row below. The covering index leaves one case the rewritten row does not describe, and it is the cell where the Postgres trigger fires.** It is not an isolation gap and does not change the recommendation. I own the row, so I give its final text here, and no further round is needed for it.

## Items

| # | Status | Note |
| --- | --- | --- |
| S1 | **resolved** | Low, own row in the A4 PR, AR-TICK-1040-01 unchanged. |
| S2 | **resolved with the row below** | The A5 ruling and "still present" are taken. Under the index, the remainder is wider than "small tables and a selective filter" (cell below). |
| S3 | **resolved** | CI-4 has other tenants' rows and is red today. Pin 2 cannot be written as worded, see the note below. |
| Trigger (R.7) | **accepted** | "Other tenants' rows read exceed N̂_e" measures the signal, and the ratio rule fires on one-tenant cells. It fires in the cell below, which is why the row changes. |

## The cell the covering index leaves

The filter is a data filter whose value equals the property's default (`absentMatches`, `internal/server/records_list.go:265`). Any user can send one. It renders `(r.data @> $n OR NOT (r.data ? $m))`, and PostgreSQL estimates the `NOT ?` half as almost every row. The estimate stays above 1,001, so the `LIMIT` discount comes back. `data` is not in the index, so the count cannot be an IOS.

The setup was the bench schema, the covering index swapped in under the same name, `random_page_cost` 4, `nava_app` under FORCE RLS, m = 0 (every one of A's records of the type holds the property at another value), 400,000 empty rows, N̂_e = 33,366. I ran plans and one execution for row counts, with no timings, in a 1-CPU throwaway container that is now removed. Scripts attached.

| Share of A | Other tenants' rows | Index, no filter | Index, default-valued filter | Index + order, same filter |
| --- | --- | --- | --- | --- |
| 91 % | 34,490 | IOS | **Seq Scan, all 400,000 rows read, so 34,490 of other tenants > N̂_e** | Index Scan `records_list_updated` |
| 54 % | 183,660 | IOS | bitmap of the type's slice | Index Scan |
| 54 %, 800,000 rows | 383,571 | IOS | bitmap | Index Scan |
| 37.5 %, 200,000 rows | 125,003 | IOS | bitmap | Index Scan |

- **This is not new with A4.** Today's index has no `data` either, so the same request plans as it does today. I infer this; I did not run today's index for this filter.
- **The index removes it for unfiltered counts only.** Under a filter it does not hold, the count's reach is today's: it can read up to the whole table, depending on ratio and share. The order (rev 0, D4) would bound it to the type's slice, at B2's cold cost. That trade belongs to the Solution Architect. I do not ask for it: the signal is Low with either choice.
- **Pin 2 note (S3), implementation only:** an IOS at m = 0 removes 33,366 rows by filter, all of them A's own, so "Rows Removed by Filter holds none of them" cannot be asserted. Assert it instead from the plan: the count's `Index Cond` starts with `tenant_id`, and rows read (`(Actual Rows + Rows Removed by Filter) × Loops`) ≤ A's rows of the type. For the mutant, rows read > A's rows in `records`.

## AR-TICK-1143-01, final text (owner)

> **AR-TICK-1143-01** (a list count's time can include other tenants' rows) · 🟡 Low · ADR-010-A4. When PostgreSQL reads `records` with a Seq Scan for a list's count, the time also covers other tenants' rows. It can hint at how many records all other tenants hold together; with two tenants, at the other one's size. No row, total or shape choice crosses the tenant wall, and the tenant test runs first, so the filter never evaluates another tenant's row content. A4 removes it for R's unfiltered count, which reads the covering index `records_list_updated` (an Index Only Scan whose range starts at the tenant). It remains in small tables; under a request filter the index does not hold, where it is as before A4 and can reach the whole table (measured: a default-valued data filter, 400,000 rows, 34,490 other tenants' rows read); and in counts that are not R. · Accepted (Security & Multi-Tenancy Reviewer, TICK-1143) · Owner: Security & Multi-Tenancy Reviewer · Trigger: a requirement that a tenant's total record volume is confidential from other tenants; or CI-3, CI-4, CI-5 or TICK-1105 shows a Seq Scan for an unfiltered R count; or an unfiltered R count reads more rows of other tenants than N̂_e; or a request-filtered list count does so in a production slow-query report.

The trigger keeps the Postgres wording, with "unfiltered" added. Without it, the row's own trigger is already met by the measured cell above.

VERDICT: APPROVE (S1 to S3 resolved, with the row as above)

RESULT: DONE
