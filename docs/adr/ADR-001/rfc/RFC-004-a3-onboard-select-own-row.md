---
type: rfc
id: RFC-004
title: ADR-001 Annex A.3 erratum, nava_onboard SELECT is "own row", not "own provisioning row" (erratum E3)
status: ARCHIVED (rev 1, approved in round 1 2026-10-25: Technical Architect no blocking finding, Security & Multi-Tenancy Reviewer APPROVE; recorded as ADR-001-E3)
supersedes: RFC-004 rev 0 (2026-10-25)
date: 2026-10-25
issue: TICK-170
source: TICK-71 second review of PR #80, comment 01a17125-65dc-7a14-986a-8a953fbe1c49 (Postgres & Data Modeling Specialist); CTO framing request, TICK-170 comment 01a17126-ea84-7ce8-83a8-7e228eb5beba; round-1 critiques 01a171f7-0383 (Technical Architect), 01a171fc-fb0e (Security); CTO close 01a171fe-a15d
amends: "[[ADR-001-platform-blueprint]] Annex A.3, Onboarding row"
author: Solution Architect
---

# RFC-004: A.3 says "own provisioning row"; the code and PostgreSQL say "own row"

## Changelog (round 1 → rev 1)

| # | Change | Raised by | Where |
|---|---|---|---|
| 1 | O2c cost restated: both policies would be pinned in `PinnedTenantPolicies`, not the allow-list. | Technical Architect F8 (non-blocking) | §4 O2c |
| 2 | O2b fragility made precise: a `WHERE`, `RETURNING` or `SET` that *references a column* reopens 42501; `WHERE true` does not. | Technical Architect F6 | §4 O2b |
| 3 | T21 condition added to RS8: the onboarding `app.tenant_id` is server-derived, never caller-supplied. | Security round 1 | §5 |
| 4 | D4 evidence: PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState`. | Security round 1 | §2 D4, §6, §7 |
| 5 | "the one T21 will run" marked as the expected shape; no production activation statement exists yet. | Technical Architect F9 | §3 |
| 6 | Column-list hardening of `db/grants.yaml` left to Delivery, outside this record. | Security round 1, CTO | §5 |

> **Invariants source.** The project description is empty, so it defines no invariants. As in A1 and A2, this is checked against [[ADR-001-platform-blueprint]] §1 **I1–I9**. If the project is meant to carry its own list of eight, the invariant check in §5 is incomplete against it.

> **Numbering.** RFC-001 to RFC-003 and errata E1, E2 exist. This is **RFC-004**, and the record it produces is **ADR-001-E3**. It is not called "A3": that name would collide with Annex A.3, the section it corrects.

## 1. Summary

- **Recommendation: fix the ADR, not the code.** Annex A.3's Onboarding row changes from "`SELECT` on its own provisioning row" to "`SELECT` on its own row (the tenant of its transaction, in either state)". The policy `tenants_onboard_select` stays as shipped.
- The wording cannot be implemented as written. PostgreSQL applies the `SELECT` policy to the **new** row of an `UPDATE` whenever the statement reads a column of the table, which `UPDATE tenants SET state = 'active' WHERE id = $1` does. A provisioning-only `SELECT` policy makes activation fail with SQLSTATE 42501. Cited in §3 and shown by spike in §6.
- Nothing crosses the tenant wall. What `nava_onboard` gains over the literal wording is reading four columns of the one tenant it is bound to, after it activated that tenant itself. The tenant predicate is unchanged and already pinned by CI.
- A.3's own "idempotent on the request id" needs the wider read: a retried onboarding request for an already-active tenant must compare `onboarding_request_hash` before answering.
- The human accepts one residual: `nava_onboard` can read its own tenant's `active` row (`id`, `state`, `onboarding_request_hash`, `created_at`). No other row, no other tenant.

## 2. Decisions

| # | Decision | Why / what it prevents |
|---|---|---|
| D1 | A.3 Onboarding row, `SELECT` clause, reads: *"`SELECT` on its own row (the tenant of its transaction, in either state)."* Everything else in the row is unchanged. | Brings the signed text in line with the only policy PostgreSQL can execute on this path (§3) and with the idempotency sentence in the same row. |
| D2 | `tenants_onboard_select` keeps the predicate `id = app.tenant_id` with no state test. It stays a plain tenant policy under `internal/db/catalog/policy.go`'s `testsTenant`, not an allow-list entry. | Adding a state predicate breaks activation (§6 case 2a). Removing the tenant predicate already fails CI (`TestT09PolicyAllowListCatchesLeaks`). |
| D3 | The erratum is recorded as **ADR-001-E3**, a separate file beside A1 and A2. ADR-001's signed text is not edited. | Same precedent as E1 and E2; the base record stays the one nava signed. |
| D4 | One positive acceptance test is added (CI-9 or T09, Delivery's choice): inside `InTenantTx(X)` after activation, `nava_onboard` reads X's row with `state = 'active'`, and reads zero rows for any other tenant. | Pins the behaviour the erratum describes, so the next drift is caught by a test and not by a reviewer. **Delivered** by PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState` (`internal/db/dbtest/platform_tables_test.go`). |

## 3. Library behaviour this rests on

**Claim.** When an `UPDATE` needs read access to the table, the table's `SELECT` policies are applied to the **new** row as well as the existing row, and a new row that fails them raises an error.

**Source.** PostgreSQL 16 reference, `CREATE POLICY`, section *Policies Applied by Command Type* (Table 292), row `UPDATE`, column *SELECT/ALL policy, USING expression*:

> Filter existing row [a] & check new row [a]
>
> [a] If read access is required to either the existing or new row (for example, a `WHERE` or `RETURNING` clause that refers to columns from the relation).

Same page, *Per-Command Policies*, `UPDATE`:

> Typically an `UPDATE` command also needs to read data from columns in the relation being updated (e.g., in a `WHERE` clause or a `RETURNING` clause, or in an expression on the right hand side of the `SET` clause). In this case, `SELECT` rights are also required on the relation being updated, and the appropriate `SELECT` or `ALL` policies will be applied in addition to the `UPDATE` policies.

And `WITH CHECK` semantics, same page:

> Any rows whose updated values do not pass the `WITH CHECK` expression will cause an error, and the entire command will be aborted.

**What it means here.** The activation statement in `internal/db/dbtest/platform_tables_test.go:293`, and the shape T21 is expected to use (no production activation statement exists yet), is `UPDATE tenants SET state = 'active' WHERE id = $1`. The `WHERE` reads `id`, so footnote [a] applies. The new row has `state = 'active'`. Under a `SELECT` policy with `AND state = 'provisioning'` the new row fails the check and PostgreSQL raises 42501, *new row violates row-level security policy for table "tenants"*. The spike in §6 reproduces this on the pinned image `postgres:16.10`.

## 4. Options considered

| Option | What changes | Why it loses or wins |
|---|---|---|
| **O1. Erratum (recommended).** Reword A.3 to "own row". | Words only; code unchanged. | Wins. The policy as shipped is the only shape that activates a tenant, keeps the tenant predicate, and lets the idempotency check read the row afterwards. Residual is one tenant reading its own four columns. |
| O2a. Code: add `AND state = 'provisioning'` to `tenants_onboard_select` and keep the activation statement as is. | One policy line. | **Breaks activation**: 42501 on `UPDATE ... WHERE id = $1` (§6 case 2a) and on any `RETURNING` (case 2b). `TestT09OnboardActivation` goes red. Rejected. |
| O2b. Code: add the state predicate **and** rewrite activation as `UPDATE tenants SET state = 'active'` with no `WHERE` and no `RETURNING`, relying on `tenants_activate USING` alone to pick the row. | One policy line plus an unusual SQL shape. | Works today (§6 case 2c) but is fragile: the first `WHERE`, `RETURNING` or `SET` that references a column of `tenants` reopens 42501 (`WHERE state = 'provisioning'` does; `WHERE true` does not), and reviewers rightly reject an unfiltered `UPDATE` on a platform table. After activation onboard can no longer read its own tenant (case 2c, 0 rows), which defeats the idempotency check. Rejected. |
| O2c. Code: a second permissive `SELECT` policy for `state = 'active'`, as the issue suggests. | Two policies instead of one. | Permissive policies combine with `OR`, so provisioning `OR` active on the same tenant predicate **is** "own row" (§6 case 3). It is the current policy written in two lines, and both lines would have to be pinned in `PinnedTenantPolicies` (as `tenants_activate` is today) because neither predicate is the exact tenant predicate `testsTenant` accepts. Rejected as a no-op with more surface. |

## 5. Invariants and risks

| Invariant | Effect |
|---|---|
| I1 tenant wall | **Unchanged.** `tenants_onboard_select` still tests `app.tenant_id` and is still a tenant policy, not an A.3 allow-list entry. With no tenant set, `NULLIF(...)::uuid` is `NULL` and the policy yields no rows. |
| I3 / D28 roles | Unchanged. No grant changes; `nava_onboard` already has table-level `SELECT` on `tenants` (`db/grants.yaml`). |
| I2, I4–I9 | Not touched. |

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS8 | `nava_onboard`, bound to tenant A, can read A's `active` row: `id`, `state`, `onboarding_request_hash`, `created_at`. It wrote three of these itself. Severity Info (Security round 1: not an isolation finding). | Security & Multi-Tenancy Reviewer | `tenants` gains a column `nava_onboard` must not see (then narrow the `SELECT` grant to columns, which RLS cannot do); or onboarding stops running inside `InTenantTx(tenant)`; or T21 lets the caller influence `app.tenant_id`. |

**Condition (T21, Security round 1).** "Own row" holds only while the caller cannot choose `app.tenant_id`. The onboarding tenant id is derived on the server, as D-P11 fixes it (`UUIDv5(operator principal + Idempotency-Key)`), and is never taken from the request body, path, query string or a header. T21 carries one adversarial test: operator 2 replays operator 1's Idempotency-Key, gets a different tenant id, and sees 0 rows. The Security & Multi-Tenancy Reviewer holds T21's review to this.

Listing `nava_onboard`'s `SELECT` columns in `db/grants.yaml` would make a future column closed by default. That hardening is Delivery's call and is not part of this record.

## 6. Evidence

Spike `docs/adr/ADR-001/spikes/postgres/rfc-004/spike.sql`, output in `output.txt`, run 2026-10-25 against `postgres:16.10` (the pinned image) with `tenants` and its three onboard policies copied from `db/migrations/00002_platform_tables.sql`.

| Case | Policy | Statement | Result |
|---|---|---|---|
| 1 | as shipped (no state test) | `UPDATE ... WHERE id = own` | `UPDATE 1`; own `active` row then readable (the reported drift) |
| 2a | `AND state = 'provisioning'` | `UPDATE ... WHERE id = own` | **ERROR 42501** new row violates row-level security policy |
| 2b | same | `UPDATE ... RETURNING id` | **ERROR 42501** |
| 2c | same | `UPDATE tenants SET state = 'active'` (no `WHERE`) | `UPDATE 1`; own row then **invisible** (0 rows) |
| 3 | provisioning policy `OR` active policy | `UPDATE ... WHERE id = own` | `UPDATE 1`; own row readable, same as case 1 |

**Round 1 re-run (Technical Architect).** Cases 1, 2a, 2b, 2c and 3 reproduced on a fresh `postgres:16.10`, and identically on 15.19 (the pins floor) and 17.11. Added cases: another tenant and no tenant see 0 rows (I1 holds); binding the own `active` tenant fails 42501 (CI-9 holds); the idempotency read returns 1 row under the shipped policy and 0 under a provisioning-only one; `WHERE state = 'provisioning'` raises 42501 and `WHERE true` passes. Full script and output in TICK-170 comment 01a171f7-0383.

**D4 test (Security).** PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState`: in `InTenantTx(X)` onboard sees only `X/provisioning`, then only `X/active` after activation; nothing of active A or provisioning Y by id or full scan; nothing with no tenant set; binding X after activation fails 42501. Passes on `postgres:16.10`.

Existing guards that already pin D2: `internal/db/catalog/policy.go` (`testsTenant`, `PinnedTenantPolicies`) and `TestT09PolicyAllowListCatchesLeaks`; `TestT09OnboardActivation`; PR #80's CI-9 onboard paths.

## 7. Acceptance

- **T09 (D4):** delivered by PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState`. No separate Delivery test is needed.
- **T21 review condition (§5):** server-derived onboarding tenant id plus the replayed-key adversarial test.
- **Docs:** `docs/data/platform-tables.md` already describes the shipped policy; no change. `docs/adr/ADR-001-platform-blueprint.{md,html}` are not edited (D3).
- **CI-9 wording** in ADR-001 Annex B ("`nava_onboard` cannot write to a non-provisioning tenant") is unchanged and still true.

## 8. Open questions

- **For the human:** accept RS8, the own-active-row read, with the T21 condition. There is no code change to approve.

## Critique — round 1 (summary)

- **Technical Architect:** no blocking finding. F1 to F5 and F7 confirmed; F6 confirmed with a precision; F8 non-blocking wording; F9 inferred, irrelevant to O1.
- **Security & Multi-Tenancy Reviewer:** `VERDICT: APPROVE`. RS8 acceptable at Info, not an isolation finding. One T21 review condition; one optional grant hardening for Delivery. D4 test delivered in PR #85.

## Response — round 1

| Item | Disposition |
|---|---|
| TA F8 (non-blocking) | Accepted; the proposed text is in §4 O2c and in ADR-001-E3. |
| TA F6 precision | Accepted; §4 O2b now says "references a column". |
| TA F9 | Accepted; §3 marks the T21 statement shape as expected, not existing. |
| Security T21 condition | Accepted; recorded under RS8 in §5 and in ADR-001-E3. |
| Security grant column list | Deferred to Delivery (CTO ruling); outside the record. |
