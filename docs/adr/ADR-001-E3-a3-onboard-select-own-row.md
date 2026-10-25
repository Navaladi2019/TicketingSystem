---
type: adr-erratum
id: ADR-001-E3
title: Erratum E3 to ADR-001 Annex A.3, nava_onboard has SELECT on its own row, not only its own provisioning row
status: PROPOSED
date: 2026-10-25
issue: TICK-170
rfc: docs/adr/ADR-001/rfc/RFC-004-a3-onboard-select-own-row.md (rev 0; thread under TICK-170 comment 01a17126-ea84-7ce8-83a8-7e228eb5beba)
supersedes: "the A.3 Onboarding-row clause \"`SELECT` on its own provisioning row\" in [[ADR-001-platform-blueprint]]"
amends: "[[ADR-001-platform-blueprint]] Annex A.3, Onboarding row"
source: TICK-71 second review of PR #80 (Postgres & Data Modeling Specialist, comment 01a17125-65dc-7a14-986a-8a953fbe1c49)
deciders: Solution Architect (author), Technical Architect (critique pending), CTO
signs: nava (pending)
---

# ADR-001 erratum E3: `nava_onboard` reads its own row, in either state

> **Status: PROPOSED.** Awaiting the Technical Architect's critique of RFC-004 and nava's sign-off on TICK-170.

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
- Spike on `postgres:16.10` (`docs/adr/ADR-001/spikes/postgres/rfc-004/`): as shipped, activation succeeds and the own `active` row is readable; with `AND state = 'provisioning'` on the `SELECT` policy, activation fails with 42501 (with `WHERE` or with `RETURNING`); two permissive policies (provisioning `OR` active) behave exactly like the shipped one.
- A.3's own "idempotent on the request id" requires reading `onboarding_request_hash` of a tenant that may already be `active`.

## Consequences

- No migration, no grant change, no policy change. Delivery adds one positive test (RFC-004 D4): after activation inside `InTenantTx(X)`, `nava_onboard` reads X's `active` row and zero rows of any other tenant.
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
| Same, and rewrite activation as `UPDATE tenants SET state = 'active'` with no `WHERE` or `RETURNING` | Works only while the statement reads no column (spike case 2c). The first `WHERE`, `RETURNING` or `SET x = col` reopens 42501. Afterwards onboard cannot read its own tenant at all, which defeats the idempotency check. An unfiltered `UPDATE` on a platform table is also not a shape we accept in review. |
| A second permissive `SELECT` policy for `state = 'active'` | Permissive policies `OR` together, so this equals the shipped policy with one more line and two allow-list pins (spike case 3). A no-op with more surface. |

## Accepted residual

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS8 | `nava_onboard`, bound to tenant A, can read A's `active` row: `id`, `state`, `onboarding_request_hash`, `created_at`. No other row, no other tenant. | Security & Multi-Tenancy Reviewer | `tenants` gains a column `nava_onboard` must not see (narrow the `SELECT` grant to columns); or onboarding stops running inside `InTenantTx(tenant)`. |

## Recorded dissent

None at framing. The Technical Architect's critique round is pending.

## Evidence

- PostgreSQL 16 `CREATE POLICY` reference, *Policies Applied by Command Type* and *Per-Command Policies*, quoted in RFC-004 §3.
- Spike `docs/adr/ADR-001/spikes/postgres/rfc-004/spike.sql` and `output.txt`, 2026-10-25, PostgreSQL 16.10.
- RFC-004 rev 0 at `docs/adr/ADR-001/rfc/RFC-004-a3-onboard-select-own-row.md`, with `rev-0/` beside it.
