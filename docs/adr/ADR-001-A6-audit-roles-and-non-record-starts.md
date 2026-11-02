---
type: adr-amendment
id: ADR-001-A6
title: Amendment A6 to ADR-001 Annex A.2 and A.3 — the nava_erasure role, onboarding seeds the audit sealer, and start rows not tied to a record
status: ACCEPTED
accepted: 2026-11-02
date: 2026-11-02
issue: TICK-409
rfc: docs/adr/ADR-004/rfc/RFC-008-audit-log.md (rev 1, §A6; archived under ADR-004 because it is ADR-004's companion)
supersedes: "in [[ADR-001-platform-blueprint]] Annex A.2 'Writing outbox rows': 'There is no other way to get a target_seq' and 'In the same transaction the producer inserts workflow_ref', for start rows that are not tied to a record only"
amends: "[[ADR-001-platform-blueprint]] Annex A.2 (producer rules), Annex A.3 (role table: new nava_erasure row; nava_onboard row)"
companion: "[[ADR-004-audit-log]]"
deciders: Solution Architect (author); Technical Architect (A6.3 conforms to D22, TICK-409 comment 01a19883-7e0c); Security & Multi-Tenancy Reviewer (A6.1 and A6.2 narrow enough, with tightenings, comment 01a19882-ca7b); Postgres & Data Modeling Specialist (comment 01a19882-e822); CTO
---

# ADR-001 amendment A6: audit roles and start rows not tied to a record

> **Status: ACCEPTED** on 2026-11-02, with [[ADR-004-audit-log]], by the Solution Architect (TICK-409), after the Technical Architect's fidelity check of PR #359 (comment 01a1988b-0779, F1–F3 applied) and the CTO's rulings (six locks: comment 01a19889-7338; F1: comment 01a1988b-3e8c). No human sign-off under the current policy. This amendment ships with [[ADR-004-audit-log]] and only records what changes ADR-001's text. ADR-001's signed text is not edited, the same as A1, A2, A5 and E3. D22, the workflow-id scheme, the relay and the allow-list of 10 non-tenant policies are **unchanged**.
>
> **Invariants source.** [[ADR-001-platform-blueprint]] §1 I1–I9. The project description defines no invariants of its own.
>
> **Numbering.** A5 is the highest amendment so far, and A3 and A4 are skipped (see A5), so this is **A6**.

## A6.1 · New role `nava_erasure` (Annex A.3, new row)

| Actor | Role | Owns | Member of | BYPASSRLS | Access |
|---|---|---|---|---|---|
| Erasure (one-shot job per erasure request, Epic 30) | `nava_erasure` | — | none | No | Runs with `app.tenant_id` set, under **canonical** policies only, so it adds no allow-list entry. On `audit_log`: SELECT by column only and **never on `diff` or `diff_salt`**; the M1 grant is `(tenant_id, occurred_at, id, subject_type, subject_id, erased_at, erasure_ref)`, plus the ADR-004 R12 commit-check columns if Epic 30 makes it a writer of an audited table. UPDATE on `(diff, diff_salt, erased_at, erasure_ref)`, limited by the ADR-004 R11 erase transition. INSERT on the ADR-004 R10 columns, for its `audit/erase` entries. Epic 30 adds its record-table grants in its own design. |

Rules:
- `LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE`. No membership in any role. Granted to no one.
- **Its credential is held by the job runner, the same way as the migrator's, and never on an app or worker host.** Whoever holds it can erase and record a valid erase entry. That shows up as an erasure, but it is still loss of evidence.

## A6.2 · `nava_onboard` seeds the sealer (Annex A.3, `nava_onboard` row, added)

Onboarding gets two INSERT grants:
- INSERT on `outbox` (columns needed for a start row);
- INSERT on `audit_chain_head`.

Both use pinned policies with the existing provisioning predicate (the E3 pattern). **The outbox policy also requires `kind = 'audit.seal'`**, so onboarding cannot queue any other start for a tenant that is still provisioning. There is **no SELECT on `outbox`**: a plain INSERT, no `RETURNING`, no `ON CONFLICT DO UPDATE`. Seeding is idempotent through the `audit_chain_head` primary key.

## A6.3 · Start rows not tied to a record (Annex A.2, "Writing outbox rows", exception)

A start row whose target is not a record, of which `audit.seal` is the only one today:
- takes `target_seq` from the current `version` of its own head row (`audit_chain_head.version`), read under that row's `FOR UPDATE` lock in the same transaction. The sealer's A1 bumps `version` (ADR-004 R16); `chain:reseed` (ADR-004 R17) reuses the current value without bumping it, because each seal target has exactly one row and `target_seq` orders nothing across targets;
- inserts **no** `workflow_ref`, because `workflow_ref.record_id` references `records` and a seal run is never signalled.

Everything else in A.2 still applies:
- `target_key = <kind>:<own outbox id>`;
- the workflow id is `t:<outbox.tenant_id>:<target_key>`;
- `REJECT_DUPLICATE`, with error-on-started;
- references-only payloads;
- causation fields (I9).

The per-target FIFO claim rule is trivially met, because each seal target has exactly one row.

**Not part of this amendment:** adding `audit.seal` to the Temporal workload allow-list. Under [[ADR-001-A1-temporal-server-1-29-7-workload-allow-list]] that is a policy PR with Security review and an image rebuild. ADR-004 R18 makes it a prerequisite of the sealer.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 | Holds. Only canonical or pinned provisioning policies are added; there are no new allow-list entries. |
| I2 | Holds. The tenant comes from `app.tenant_id`, which the platform sets, or from `outbox.tenant_id`. |
| I3 | Holds. One new least-privilege role: no owner, no membership, no `BYPASSRLS`, no SECURITY DEFINER. Grants come from `db/grants.yaml`. |
| I6 | Holds. Seal runs start only from committed outbox rows, through the relay. A6.3 changes only producer bookkeeping. |
| I7 | Holds. Onboarding seeding is idempotent through the head primary key. Sealer scheduling is idempotent through `next_seal_outbox_id` (ADR-004 R16). |

## Acceptance tests (added to Annex B)

| # | Proves |
|---|---|
| CI-1 / CI-3 (extended) | `nava_erasure` is listed in the manifest with no membership, no ownership and no `PUBLIC` reach. The allow-list is still 10 entries. |
| CI-A7 (ADR-004) | `nava_erasure` can make the erase transition and nothing else, and cannot read `diff` or `diff_salt`. |
| CI-A16 (ADR-004) | `nava_onboard` can insert an `audit.seal` outbox row for its provisioning tenant only. A row of any other kind, or for another tenant, is rejected. It cannot SELECT `outbox`. |
| A6.3 relay test | An `audit.seal` start row with no `workflow_ref` is claimed and started as `t:<tenant>:audit.seal:<id>`. A later row for the same tenant starts a new run. |
