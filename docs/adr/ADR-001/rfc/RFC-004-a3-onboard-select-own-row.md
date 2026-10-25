---
type: rfc
id: RFC-004
title: ADR-001 Annex A.3 erratum, nava_onboard SELECT is "own row", not "own provisioning row" (erratum E3)
status: DRAFT (rev 0, framed 2026-10-25, awaiting critique)
date: 2026-10-25
issue: TICK-170
source: TICK-71 second review of PR #80, comment 01a17125-65dc-7a14-986a-8a953fbe1c49 (Postgres & Data Modeling Specialist); CTO framing request, TICK-170 comment 01a17126-ea84-7ce8-83a8-7e228eb5beba
amends: "[[ADR-001-platform-blueprint]] Annex A.3, Onboarding row"
author: Solution Architect
---

# RFC-004: A.3 says "own provisioning row"; the code and PostgreSQL say "own row"

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
| D4 | One positive acceptance test is added (CI-9 or T09, Delivery's choice): inside `InTenantTx(X)` after activation, `nava_onboard` reads X's row with `state = 'active'`, and reads zero rows for any other tenant. | Pins the behaviour the erratum describes, so the next drift is caught by a test and not by a reviewer. |

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

**What it means here.** The activation statement in `internal/db/dbtest/platform_tables_test.go:293`, and the one T21 will run, is `UPDATE tenants SET state = 'active' WHERE id = $1`. The `WHERE` reads `id`, so footnote [a] applies. The new row has `state = 'active'`. Under a `SELECT` policy with `AND state = 'provisioning'` the new row fails the check and PostgreSQL raises 42501, *new row violates row-level security policy for table "tenants"*. The spike in §6 reproduces this on the pinned image `postgres:16.10`.

## 4. Options considered

| Option | What changes | Why it loses or wins |
|---|---|---|
| **O1. Erratum (recommended).** Reword A.3 to "own row". | Words only; code unchanged. | Wins. The policy as shipped is the only shape that activates a tenant, keeps the tenant predicate, and lets the idempotency check read the row afterwards. Residual is one tenant reading its own four columns. |
| O2a. Code: add `AND state = 'provisioning'` to `tenants_onboard_select` and keep the activation statement as is. | One policy line. | **Breaks activation**: 42501 on `UPDATE ... WHERE id = $1` (§6 case 2a) and on any `RETURNING` (case 2b). `TestT09OnboardActivation` goes red. Rejected. |
| O2b. Code: add the state predicate **and** rewrite activation as `UPDATE tenants SET state = 'active'` with no `WHERE` and no `RETURNING`, relying on `tenants_activate USING` alone to pick the row. | One policy line plus an unusual SQL shape. | Works today (§6 case 2c) but is fragile: the first `WHERE`, `RETURNING` or `SET x = col` added to that statement reopens 42501, and reviewers rightly reject an unfiltered `UPDATE` on a platform table. After activation onboard can no longer read its own tenant (case 2c, 0 rows), which defeats the idempotency check. Rejected. |
| O2c. Code: a second permissive `SELECT` policy for `state = 'active'`, as the issue suggests. | Two policies instead of one. | Permissive policies combine with `OR`, so provisioning `OR` active on the same tenant predicate **is** "own row" (§6 case 3). It is the current policy written in two lines, with two allow-list entries to pin instead of none. Rejected as a no-op with more surface. |

## 5. Invariants and risks

| Invariant | Effect |
|---|---|
| I1 tenant wall | **Unchanged.** `tenants_onboard_select` still tests `app.tenant_id` and is still a tenant policy, not an A.3 allow-list entry. With no tenant set, `NULLIF(...)::uuid` is `NULL` and the policy yields no rows. |
| I3 / D28 roles | Unchanged. No grant changes; `nava_onboard` already has table-level `SELECT` on `tenants` (`db/grants.yaml`). |
| I2, I4–I9 | Not touched. |

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS8 | `nava_onboard`, bound to tenant A, can read A's `active` row: `id`, `state`, `onboarding_request_hash`, `created_at`. It wrote three of these itself. | Security & Multi-Tenancy Reviewer | `tenants` gains a column `nava_onboard` must not see (then narrow the `SELECT` grant to columns, which RLS cannot do); or onboarding stops running inside `InTenantTx(tenant)`. |

## 6. Evidence

Spike `docs/adr/ADR-001/spikes/postgres/rfc-004/spike.sql`, output in `output.txt`, run 2026-10-25 against `postgres:16.10` (the pinned image) with `tenants` and its three onboard policies copied from `db/migrations/00002_platform_tables.sql`.

| Case | Policy | Statement | Result |
|---|---|---|---|
| 1 | as shipped (no state test) | `UPDATE ... WHERE id = own` | `UPDATE 1`; own `active` row then readable (the reported drift) |
| 2a | `AND state = 'provisioning'` | `UPDATE ... WHERE id = own` | **ERROR 42501** new row violates row-level security policy |
| 2b | same | `UPDATE ... RETURNING id` | **ERROR 42501** |
| 2c | same | `UPDATE tenants SET state = 'active'` (no `WHERE`) | `UPDATE 1`; own row then **invisible** (0 rows) |
| 3 | provisioning policy `OR` active policy | `UPDATE ... WHERE id = own` | `UPDATE 1`; own row readable, same as case 1 |

Existing guards that already pin D2: `internal/db/catalog/policy.go` (`testsTenant`, `PinnedTenantPolicies`) and `TestT09PolicyAllowListCatchesLeaks`; `TestT09OnboardActivation`; PR #80's CI-9 onboard paths.

## 7. Acceptance

- **CI-9 / T09 (D4):** one new positive read assertion as described in D4. Delivery decides whether it extends `TestT09OnboardActivation` or the CI-9 `nava_onboard` path in `internal/tenancy/isolation/role_paths.go`.
- **Docs:** `docs/data/platform-tables.md` already describes the shipped policy; no change. `docs/adr/ADR-001-platform-blueprint.{md,html}` are not edited (D3).
- **CI-9 wording** in ADR-001 Annex B ("`nava_onboard` cannot write to a non-provisioning tenant") is unchanged and still true.

## 8. Open questions

- **For the human:** accept RS8, the own-active-row read. There is no code change to approve.
- **For the Technical Architect (critique):** the §3 citation and the §6 spike are the facts the recommendation rests on. If either is wrong, O1 falls and O2 has to be re-opened.
