---
type: adr-erratum
id: ADR-001-E3
title: Erratum E3 to ADR-001 Annex A.3, nava_onboard has SELECT on its own row, not only its own provisioning row
status: PROPOSED
date: 2026-10-25
issue: TICK-170
rfc: docs/adr/ADR-001/rfc/RFC-004-a3-onboard-select-own-row.md (rev 1, archived; thread under TICK-170 comment 01a17126-ea84-7ce8-83a8-7e228eb5beba)
supersedes: "the A.3 Onboarding-row clause \"`SELECT` on its own provisioning row\" in [[ADR-001-platform-blueprint]]"
amends: "[[ADR-001-platform-blueprint]] Annex A.3, Onboarding row"
source: TICK-71 second review of PR #80 (Postgres & Data Modeling Specialist, comment 01a17125-65dc-7a14-986a-8a953fbe1c49); RFC-004 round 1
deciders: Solution Architect (author), Technical Architect (no blocking finding, round 1), Security & Multi-Tenancy Reviewer (APPROVE, round 1), CTO
signs: nava (pending)
---

# ADR-001 erratum E3: `nava_onboard` reads its own row, in either state

> **Status: PROPOSED.** RFC-004 rev 1 was approved in round 1 by the Technical Architect (no blocking finding) and the Security & Multi-Tenancy Reviewer (APPROVE). Awaiting nava's sign-off on TICK-170.

> **Invariants source.** The project description defines no invariants. As in A1 and A2, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

## Decision

**The words change; the code does not.** In Annex A.3, row *Onboarding* (`nava_onboard`), the clause

> `SELECT` on its own provisioning row.

reads from now on

> `SELECT` on its own row (the tenant of its transaction, in either state).

The rest of the row is unchanged: `INSERT` on `tenants` and `tenant_identity_binding`, seeding through the pipeline grants with `WITH CHECK (tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'))`, idempotent on the request id.

`tenants_onboard_select` in `db/migrations/00002_platform_tables.sql` stays `USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)`, with no state test, and stays a tenant policy under `internal/db/catalog/policy.go`, not an allow-list entry.

## Context

- The Postgres & Data Modeling Specialist found while reviewing PR #80 (TICK-71, CI-9) that `tenants_onboard_select` has no state predicate, so inside tenant A's transaction `nava_onboard` can read A's `active` row. A.3 said "own provisioning row".
- The literal wording is not implementable on the activation path. PostgreSQL 16, `CREATE POLICY`, *Policies Applied by Command Type*, row `UPDATE`: the `SELECT` policy's `USING` expression is applied as *"Filter existing row [a] & check new row [a]"*, where [a] is *"if read access is required to either the existing or new row (for example, a `WHERE` or `RETURNING` clause that refers to columns from the relation)"*. Activation is `UPDATE tenants SET state = 'active' WHERE id = $1`. A provisioning-only `SELECT` policy rejects the new `active` row with SQLSTATE 42501.
- Spike on `postgres:16.10` (`docs/adr/ADR-001/spikes/postgres/rfc-004/`), re-run in round 1 by the Technical Architect on 16.10, 15.19 and 17.11 with identical results: as shipped, activation succeeds and the own `active` row is readable; with `AND state = 'provisioning'` on the `SELECT` policy, activation fails with 42501 (with `WHERE` or with `RETURNING`); two permissive policies (provisioning `OR` active) behave exactly like the shipped one.
- A.3's own "idempotent on the request id" requires reading `onboarding_request_hash` of a tenant that may already be `active`.

## Consequences

- No migration, no grant change, no policy change. The positive test RFC-004 D4 asks for is PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState`: inside `InTenantTx(X)` `nava_onboard` sees only X, `provisioning` then `active`, and nothing of any other tenant or with no tenant set.
- Reviewers checking "every role reaches only what A.3 gives it" (PR #80's claim) now have text that matches the code.
- `docs/data/platform-tables.md` already describes the shipped policy. ADR-001's signed text is not edited; this file sits beside A1 and A2, as E1 and E2 did.

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Unchanged. The policy still tests `app.tenant_id`; with no tenant set it yields no rows. `TestT09PolicyAllowListCatchesLeaks` already fails CI if that predicate is removed. |
| I3 / D28 | Unchanged. `nava_onboard` already holds table-level `SELECT` on `tenants` (`db/grants.yaml`). |
| I2, I4–I9 | Not touched. |

## Rejected options

| Option | Why rejected |
|---|---|
| Add `AND state = 'provisioning'` to `tenants_onboard_select` | Breaks activation: 42501 on `UPDATE ... WHERE id = $1` and on any `RETURNING` (spike cases 2a, 2b). `TestT09OnboardActivation` goes red. |
| Same, and rewrite activation as `UPDATE tenants SET state = 'active'` with no `WHERE` or `RETURNING` | Works only while the statement references no column (spike case 2c). The first `WHERE`, `RETURNING` or `SET` that references a column reopens 42501. Afterwards onboard cannot read its own tenant at all, which defeats the idempotency check. An unfiltered `UPDATE` on a platform table is also not a shape we accept in review. |
| A second permissive `SELECT` policy for `state = 'active'` | Permissive policies `OR` together, so this equals the shipped policy (spike case 3). It is the current policy written in two lines, and both lines would have to be pinned in `PinnedTenantPolicies` (as `tenants_activate` is today) because neither predicate is the exact tenant predicate `testsTenant` accepts. Rejected as a no-op with more surface. |

## Accepted residual

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS8 | `nava_onboard`, bound to tenant A, can read A's `active` row: `id`, `state`, `onboarding_request_hash`, `created_at`. No other row, no other tenant. Severity Info: not an isolation finding (Security, round 1). | Security & Multi-Tenancy Reviewer | `tenants` gains a column `nava_onboard` must not see (narrow the `SELECT` grant to columns); or onboarding stops running inside `InTenantTx(tenant)`; or T21 lets the caller influence `app.tenant_id`. |

**Condition (T21, Security round 1).** "Own row" holds only while the caller cannot choose `app.tenant_id`. The onboarding tenant id is derived on the server, as D-P11 fixes it (`UUIDv5(operator principal + Idempotency-Key)`), and is never taken from the request body, path, query string or a header. T21 carries one adversarial test: operator 2 replays operator 1's Idempotency-Key, gets a different tenant id, and sees 0 rows. The Security & Multi-Tenancy Reviewer holds T21's review to this.

## Recorded dissent

None. Both reviewers approved in round 1.

## Evidence

- PostgreSQL 16 `CREATE POLICY` reference, *Policies Applied by Command Type* and *Per-Command Policies*, quoted in RFC-004 §3.
- Spike `docs/adr/ADR-001/spikes/postgres/rfc-004/spike.sql` and `output.txt`, 2026-10-25, PostgreSQL 16.10.
- Technical Architect round-1 re-run on PostgreSQL 16.10, 15.19 and 17.11, with added cases for the tenant wall, CI-9 and the idempotency read (TICK-170 comment 01a171f7-0383).
- PR #85, `TestT09OnboardReadsOnlyOwnRowInEitherState` in `internal/db/dbtest/platform_tables_test.go` (D4).
- RFC-004 rev 1 (final, archived) at `docs/adr/ADR-001/rfc/RFC-004-a3-onboard-select-own-row.md`, with `rev-0/` and `rev-1/` beside it.
