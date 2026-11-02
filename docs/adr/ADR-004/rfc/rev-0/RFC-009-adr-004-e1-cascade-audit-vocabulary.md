---
type: rfc
id: RFC-009
title: ADR-004 erratum E1, R2 vocabulary and key shape for the two cascade audit entries
status: Draft (rev 0)
date: 2026-11-02
issue: TICK-422
target: ADR-004-E1 (erratum to [[ADR-004-audit-log]] R2; touches [[ADR-001-A5-i9-cycle-suppress-start]] A5-4 and PLAN-003 D-3-7a by reference only)
author: Solution Architect
---

# RFC-009: how the two cascade audit entries fit ADR-004's log

> **Invariants source.** The project description defines no invariants. As in ADR-004 and ADR-001 A5, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

> **Numbering.** The highest RFC in the repo is RFC-008 (ADR-004). This is RFC-009. ADR-004 has no erratum yet, so this one is **E1**.

## Recommendation

Add two actions to R2, `cascade_cycle` and `cascade_too_deep`, both in `domain = record`. Add one subject type, `cascade_key`.

| Entry | When and where it is written | `subject_type` / `subject_id` | `version_before` / `version_after` | `causation_id` / `causation_depth` | `diff` (ids only) |
|---|---|---|---|---|---|
| `cascade_cycle` (A5-4) | In the write's transaction, once per suppressed start | `cascade_key` / **UUIDv5(ns, tenant ‖ O ‖ R ‖ D)** | NULL / R's version after the write that fired D again | O / the write's depth | `{origin_id: O, record_id: R, definition_id: D, reason: "ErrCascadeCycle"}` |
| `cascade_too_deep` (D-3-7a) | On the pipeline's `TxRunner`, after the caller's transaction rolled back | `record` / R | the `ExpectedVersion` the rejected write held / **NULL** | O / the rejected depth (≥ 9) | `{origin_id: O, record_id: R, depth: n, limit: 8, reason: "ErrCascadeTooDeep"}` |

`entity_definition_id` is R's entity type in both. Actor and `source` are the rejected or suppressing writer's, as A5-4 says. Neither entry carries the attempted record data.

**The idempotency key:** I agree with PLAN-005 D-5-3. Moving the key to `record_write_keys` leaves R1 unchanged, and neither cascade entry carries a key. It also retires D-3-13's "primary audit row only" rule, which needed a column R1 never had.

## Why this shape

**1. `cascade_cycle` cannot use record R as its subject.** Shown by the spike:
- With `subject = (record, R)` and the write's `version_after`, it collides with the record's own `update` entry on the R2 unique key (case A).
- With `version_after = NULL`, two definitions suppressed on one write collide with each other (case B). A5-4 requires one row each.
- It also **weakens the commit check**: a cycle entry for R alone satisfies R's trigger, so a pipeline that forgot the `update` entry would commit (case C). With a `cascade_key` subject, the `update` entry is still required (case F).

**2. `cascade_cycle` cannot use definition D as its subject either.** One transaction can call `Pipeline.Write` more than once (the method takes the caller's `tx`). If R1 and R2 both reach version v in that transaction and D is suppressed on each, `(workflow_definition, D, v)` repeats. This follows from the key's columns; it was not spiked.

**3. The cascade key is unique where it needs to be.**
- Different (O, R, D) give different subject ids (case D: three suppressions in one transaction across two records and two definitions).
- The same (O, R, D) twice in one transaction means R was written twice, and D-3-6 bumps `records.version` exactly once per write, so `version_after` differs (case E).
- The tenant is in the name, so the same O, R, D in two tenants never share a subject id (case H). `tenant_id` also leads the unique key, so this is belt and braces.
- UUIDv5 is a pure function. The `EventSink` computes it without reading `cascade_start`, and an auditor can recompute it from the diff. The namespace is a fixed constant in `internal/audit`, pinned by golden vectors next to R13's.

**4. `cascade_too_deep` can use record R.** It is written in its own runner transaction, which writes no audited table, so the commit check does not apply and nothing else shares its `txid`. Each rejected attempt is one transaction with one entry (cases G and G2). Subject R puts the failure in R's history, where a user looking at "why did my workflow stop" will find it. `version_after = NULL` is honest: no version was produced.
- For a create (no row exists yet), the subject is the record id the pipeline would have used. M1 interpreter writes are `update_record` only (D-3-15), so this case is future-only.

**5. O goes in `causation_id`, D goes in the diff.** `causation_id` and `causation_depth` already hold the origin and the depth on every entry (R1), so filtering by origin works for both entries with no new column. D has no envelope column. Adding one would change R1 and the hash input (R13), which is a bigger change than this erratum. The diff is chain-covered through `diff_sha256`, and it holds ids only, so erasure (R23) has nothing personal to remove. For `cascade_cycle`, the subject id commits to D anyway.

**6. Domain `record`, not `audit`.** `audit` is for actions on the log itself (export, erase, reseed). Both entries are about record writes, and auditors filter by domain. The per-record history (R8) filters by subject. It shows `cascade_too_deep` on R and does not show `cascade_cycle`. That is deliberate. A5-4 calls cycle rows routine, since every "on update, set X" workflow writes one, and they name definition ids that a record reader does not need.

## Options considered

| Option | Verdict |
|---|---|
| **A. `cascade_key` subject for cycles, record subject for too-deep (recommended)** | Unique in every case, keeps the commit check strict, and needs no read and no new column. |
| B. Record R subject for both, `version_after = NULL` on cycles | Collides when two definitions are suppressed on one write (case B), and lets a cycle row stand in for the `update` entry (case C). |
| C. Definition D subject for cycles, R's version as `version_after` | Collides when one transaction writes two records to the same version. |
| D. `cascade_start.outbox_id` of the winning claim as the subject | Unique, but needs a `SELECT` after `ON CONFLICT DO NOTHING` returns nothing, and it ties the audit subject to a table's internal id. |
| E. Fold the cycle fact into the record's `update` entry | Breaks A5-4's "one row per suppressed attempt", and an `update` entry would then mean two things. |
| F. New envelope columns `definition_id` and `origin_id` | Changes R1 and the R13 hash input. `causation_id` already holds O. |

## Proposed erratum text (becomes ADR-004-E1 after critique)

**R2, `action` set, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, **cascade_cycle, cascade_too_deep**}.

**R2, `subject_type`, added:** `cascade_key`, registered in `internal/audit/subjects.go` with the CHECK.

**New rule R2a · cascade entries.**
- `cascade_cycle` (ADR-001-A5 A5-4): `domain = record`. `subject_type = cascade_key`, `subject_id = UUIDv5(audit.CascadeKeyNamespace, tenant_id ‖ origin_id ‖ record_id ‖ definition_id)`, over the 16 raw bytes of each id. `version_after` is R's version after the write that fired D again. `version_before` is NULL. `causation_id = O`, `causation_depth` is the write's depth. Written in the write's transaction through `audit.Append`.
- `cascade_too_deep` (PLAN-003 D-3-7a, which renames its `cascade_rejected`): `domain = record`. `subject_type = record`, `subject_id = R`. `version_before` is the rejected command's `ExpectedVersion`, and `version_after` is NULL. `causation_id = O`, `causation_depth` is the rejected depth. Written through `audit.Append` on the pipeline's `TxRunner`, in a transaction that writes no audited table.
- In both: `entity_definition_id` is R's entity type. Actor and `source` are the writer's. The diff is a single `add` of `/cascade`, holding ids, the depth and limit (too-deep only) and the reason, and **never the attempted record data**. No idempotency key (PLAN-005 D-5-3).
- `cascade_start` goes in the CI-A3 exemption list, reason: "claim bookkeeping; a won claim is covered by the record's entry in the same transaction, a lost claim by `cascade_cycle`".

**CI-A17 (new):** (a) one write suppressing D1 and D2 on R writes two `cascade_cycle` entries and one `update` entry, and commits. (b) One transaction writing R1 and R2 to the same version, with D suppressed on each, commits. (c) A transaction that writes R with only a `cascade_cycle` entry fails at COMMIT. (d) A depth-9 write leaves no record change and exactly one `cascade_too_deep` entry. (e) Golden vectors for the cascade key.

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. Both entries go through `audit.Append` under the canonical policy. The tenant is in the cascade key, and `tenant_id` leads the unique key. |
| I4 audit in the same transaction | Holds, and stays strict: a cycle entry cannot stand in for the record's own entry (case F). The too-deep entry is the one documented exception: it survives a rollback by design (D-3-7a), in a transaction that writes no audited row. |
| I8 platform-owned vocabulary | Holds. Two actions and one subject type, added by platform migration. |
| I9 cascades bounded | Holds. Both audit entries that I9 requires now have a defined shape. |
| I2, I3, I5–I7 | Not touched. |

## Not verified

- The pipeline's real `TxRunner` and `EventSink` do not exist yet (P5-13 builds them). The spike uses a simplified table and trigger, with no RLS and no partition management.
- No measurement of cycle-entry volume. A5-4 says it is one per "on update" run. Estimated at most one extra row per workflow-caused write. Not measured.

## Evidence

Spike `spike.sql` and `output.txt`, postgres:16.10, 2026-11-02. These go to `docs/adr/ADR-004/spikes/solution-architect/rfc-009/` with the erratum.

| Case | Result |
|---|---|
| A | Record subject with the same `version_after`: unique violation. |
| B | Record subject with NULL version, two definitions: unique violation. |
| C | A record-subject cycle entry alone passes the commit check (the weakness). |
| D | Three cascade-key entries in one transaction, across two records at the same version: commits. |
| E | Same key, R written twice in one transaction (v3, v4): commits. |
| F | A cascade-key entry alone fails the commit check with `audit_missing`. |
| G, G2 | A too-deep entry on the runner after rollback, twice: both commit, and R is unchanged. |
| H | The same O, R, D in another tenant gives a different key. |
