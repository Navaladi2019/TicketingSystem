---
type: adr-erratum
id: ADR-004-E1
title: Erratum E1 to ADR-004, the R2 vocabulary and key shape for the two cascade audit entries, and diff_truncated in the R1 envelope
status: PROPOSED
date: 2026-11-02
issue: TICK-422
rfc: docs/adr/ADR-004/rfc/RFC-009-adr-004-e1-cascade-audit-vocabulary.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-422 comment 01a198bc-f986-78f8-a49c-dd0fef86ae3f)
supersedes: "the R2 `action` list, and the R1 envelope list, in [[ADR-004-audit-log]]"
amends: "[[ADR-004-audit-log]] R1, R2 and the acceptance tests (adds CI-A17); PLAN-003 D-3-7a by reference (`cascade_rejected` becomes `cascade_too_deep`)"
source: "PLAN-005 (TICK-417) found the gap. ADR-001-A5 A5-4 and PLAN-003 D-3-7a require two audit rows that R2 cannot hold. Item 2 is the Solution Architect's ruling at grooming (TICK-449, comment 01a198c1-53f0)."
deciders: Solution Architect (author); Technical Architect (round 1, no blocking, comment 01a198c2-c512); Postgres & Data Modeling Specialist (round 1, one blocking P5, comment 01a198c6-7470); CTO (routing, comment 01a198bd-ce63)
---

# ADR-004 erratum E1: the cascade audit entries, and `diff_truncated` in the envelope

> **Status: PROPOSED (rev 1), 2026-11-02.** Round 1:
> - The Technical Architect raised no blocking findings and four non-blocking fixes (comment `01a198c2`).
> - The Postgres & Data Modeling Specialist raised one blocking finding, P5 (comment `01a198c6`).
>
> Rev 1 takes every item as written. The SA gives the verdict after the re-check and any CTO tie-break. Item 2 joined E1 at grooming (TICK-449), after round 1 was framed, so the re-check covers it.

> **Invariants source.** The project description defines no invariants. As in ADR-004, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

## Decision

### Item 1: the cascade entries (R2, new R2a)

**R2, `action`, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, **cascade_cycle, cascade_too_deep**}.

**R2, `subject_type`, added:** `cascade_key`, registered in `internal/audit/subjects.go` and mirrored by the CHECK.

**R2a · cascade entries (new rule).**

| | `cascade_cycle` (ADR-001-A5 A5-4) | `cascade_too_deep` (PLAN-003 D-3-7a) |
|---|---|---|
| When and where | Written through `audit.Append` in the write's transaction, once per suppressed start. | Written through `audit.Append` on the pipeline's `TxRunner`. It commits before `Write` returns, independently of the caller's transaction, which writes nothing for this command. **One entry per runner transaction.** |
| `domain` | `record` | `record` |
| `subject_type` / `subject_id` | `cascade_key` / **K(T, O, R, D)** | `record` / R, when the runner transaction reads R under RLS. Otherwise (a create, or R not found): `cascade_key` / **K(T, O, nil, nil)**, and the diff has no `record_id`. |
| `version_before` | NULL | the command's `ExpectedVersion` on an update; NULL on a create (never 0) |
| `version_after` | R's version after the write that fired D again | NULL |
| `causation_id` / `causation_depth` | O / the write's depth | O / the rejected depth (≥ 9) |
| `entity_definition_id` | R's entity type | R's entity type |
| actor, `source` | the writer's | the rejected command's |
| `diff` | one `add` of `/cascade`: `{origin_id, record_id, definition_id, reason: "ErrCascadeCycle"}` | one `add` of `/cascade`: `{origin_id, record_id?, depth, limit, reason: "ErrCascadeTooDeep"}` |

- **The cascade key.** K(T, O, R, D) = `uuid.NewSHA1(audit.CascadeKeyNamespace, T[:] ‖ O[:] ‖ R[:] ‖ D[:])`, a UUIDv5 over the 64 raw bytes in that order.
  - `audit.CascadeKeyNamespace` is pinned to **`e3045a06-15ad-46d2-b1a9-0dd05b182547`**. It is a fresh random UUID used for this purpose only: not an RFC 4122 namespace, not `onboard.Namespace`, and not a tenant id.
  - `internal/audit` computes it in Go, and the verifier recomputes it in Go. A SQL version (`pgcrypto`) exists only as a spike and test fixture; no migration installs `pgcrypto`.
- **At most one cycle entry per (O, R, D) per write [PG P5].** The `EventSink` claims each `definition_id` at most once per write and deduplicates before `audit.Append`. Otherwise a second identical entry would hit the unique key and roll back the user's write.
- **Neither entry ever carries the attempted record data,** and neither carries an idempotency key. PLAN-005 D-5-3 moves the key to `record_write_keys`, so R1 is unchanged on that point.
- **Why the unique key holds.** It depends on R10: `occurred_at = now()` and `txid = pg_current_xact_id()` are asserted for every row, so all entries in one transaction share both columns. Within one transaction the key therefore reduces to (subject, `version_after`):
  - different (O, R, D) give different keys;
  - the same (O, R, D) twice in one transaction means R was written twice, and D-3-6 bumps `records.version` once per write;
  - a cycle entry never has subject R, so it cannot collide with R's `update` entry. It also cannot stand in for that entry at COMMIT: R's own entry is still required.
- **`cascade_start` is exempt from CI-A3.** Reason: "claim bookkeeping: rows are inserted only by a won claim, in a write transaction whose record entry covers it".

### Item 2: `diff_truncated` in the envelope (R1)

**R1, envelope, new text:** the existing list, plus **`diff_truncated`**. The envelope has 22 members.
- `diff_truncated` is a boolean, `NOT NULL DEFAULT false`, and it is always emitted.
- It is in the R10 INSERT grant and outside both R11 transitions. Erasure leaves it unchanged.
- **Past 64 KB, `diff` keeps whole RFC 6902 ops only,** and the flag is set.
- `diff_sha256 = SHA-256(diff_salt ‖ stored diff bytes)`. This is R13 restated over the stored bytes.
- The JCS mapping in `docs/api/audit-export-v1.md` gains the field.
- No migration and no backfill: `audit_log` is empty until P5-13b. Rows with `backfilled = true` are written with `false`.

**Why inside the envelope.** Only `diff` and `diff_salt` sit outside the envelope, and for one reason: R11 erasure sets them to NULL and the chain must still verify. That reason does not apply to `diff_truncated`. The flag also changes what the evidence means:
- true → false makes a cut diff look complete;
- false → true suggests that ops were removed when none were.

`diff_sha256` protects the bytes, not the claim that they are complete.

## Consequences

**P5-02 (TICK-424, PR #365)** pins the 22-member envelope in the spec and the golden vectors. It adds:
- a vector with `diff_truncated = true`;
- cascade-key vectors, including one permuted tuple.

**P5-03:** the verifier reports a flipped `diff_truncated` as `tampered`.

**P5-12b** sets the flag before it hashes the envelope.

**P5-13** is mechanical:
- `ClaimStart`'s `SELECT … FOR UPDATE` also reads R's `version` and `entity_definition_id`, and carries them on `Claim`. This adds no statement. A5-1 already puts the claim after the write's `UPDATE`, so `version_after` comes out right.
- The rename `cascade_rejected` → `cascade_too_deep` lands at the sites the Technical Architect listed (TICK-422 comment `01a198c2`, §3c):
  - `internal/records/cascade/cascade.go`, `internal/records/pipeline/pipeline.go` and their tests;
  - `internal/db/dbtest/records_pipeline_test.go`;
  - the S9 migration comment;
  - `internal/tenancy/isolation/testdata/seed.sql`.
- The cycle diff changes from `{reason, definition_id}` to the `/cascade` object.

**PLAN-003 D-3-7a**, text change by reference: `cascade_rejected` becomes `cascade_too_deep`. The Technical Lead commits it.

**No new index.** Neither entry needs one: no planned caller finds a cycle row by record id (Technical Architect, §1). Filtering by `causation_id` is a scan, not a supported access path: R28 has no index on it.

## Rejected options

| Option | Why rejected |
|---|---|
| Cycle subject = record R, with the write's `version_after` | It collides with R's `update` entry (spike case A). |
| Cycle subject = record R, with `version_after` NULL | Two definitions suppressed on one write collide (case B). A cycle row alone also passes R's commit check, so a forgotten `update` entry would commit (case C). |
| Cycle subject = definition D, with R's version | It collides when one transaction writes two records to the same version and D is suppressed on each. This follows from the key's columns; it was not spiked. |
| Subject = `cascade_start.outbox_id` of the winning claim | It needs a `SELECT` after `ON CONFLICT DO NOTHING`, and it ties the subject to a table's internal id. |
| Fold the cycle fact into R's `update` entry | It breaks A5-4's "one row per suppressed attempt". |
| New `origin_id` and `definition_id` envelope columns | They change R1 and the R13 hash input. `causation_id` already holds O. |
| Too-deep subject = the caller's unverified `RecordID`, for creates and not-found cases | It puts an id nobody checked into the log as a subject (Technical Architect, 3b). |
| `diff_truncated` outside the envelope | The flag would be editable without breaking the chain, so a cut diff could pass as complete. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E1-1 | Cycle rows are not in the per-record history, and nothing finds them by record id without O and D. Deliberate: they are routine, and they name definition ids. | Solution Architect | A caller needs cycle rows by R. Then add the partial index from the Postgres Specialist's spike: `(tenant_id, ((diff->0->'value'->>'record_id')::uuid), occurred_at DESC) WHERE action = 'cascade_cycle'`, built `ON ONLY` plus `CONCURRENTLY` per partition. It measured 0.039 ms against 298 ms, 23 MB, and about 7 µs per cycle insert. An erased diff drops out of it. |
| RS-E1-2 | A crash-retried too-deep attempt is audited twice. Each entry is a real rejected attempt, not a doubled effect (case G2). | Technical Lead | Audit noise from retries shows in production. |

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. Both entries go through `audit.Append` under the canonical policy, with the tenant from `tenancy.TenantOf(tx)`. T is in the cascade key, and `tenant_id` leads the unique key. |
| I4 audit in the same transaction | Holds and stays strict. A cycle entry cannot stand in for R's entry (case F). The too-deep entry belongs to a write that changed nothing, in a transaction with no audited table, so the same-transaction rule has nothing to cover. Item 2 strengthens tamper evidence. |
| I7 idempotency | Holds. Each rejected attempt is one entry. A crash-retried attempt is a new attempt and is audited again (RS-E1-2). |
| I8 platform-owned vocabulary | Holds. Two actions, one subject type and one envelope field, all by platform migration. |
| I9 cascades bounded | Holds. Both entries that A5's I9 text requires now have a defined shape. |
| I2, I3, I5, I6 | Not touched. The runner uses the pipeline's own role. Both entries carry ids only. Cycle rows are readable only with `audit.read`. |

## Acceptance tests

| # | Proves |
|---|---|
| CI-A17 | (a) One write that suppresses D1 and D2 on R writes two `cascade_cycle` entries and one `update` entry, and commits. (b) One transaction that writes R1 and R2 to the same version, with D suppressed on each, commits. (c) A transaction that writes R with only a `cascade_cycle` entry fails at COMMIT. (d) A depth-9 update leaves R unchanged and writes exactly one `cascade_too_deep` entry with subject R. A depth-9 create writes one with subject K(T, O, nil, nil) and no `record_id`. (e) Cascade-key golden vectors, including a permuted tuple, written through `audit.Append` with `pgx` and read back byte-equal. (f) D matching one write through two triggers writes exactly one cycle entry, and the write commits. (g) The verifier reports a flipped `diff_truncated` as `tampered`, and a diff over 64 KB is stored as whole ops with the flag set. |

## Evidence

- `docs/adr/ADR-004/spikes/solution-architect/rfc-009/`: `spike.sql` and `output.txt`, postgres:16.10, cases A–H. The Technical Architect re-ran it, and the output was identical after masking timestamps and txids.
- `docs/adr/ADR-004/spikes/postgres/rfc-009/`: `spike2.sql`, `output2.txt` and `main.go`, from the Postgres & Data Modeling Specialist on postgres:16.10.
  - X0: byte order.
  - X1: the same (O, R, D) twice on one write is a unique violation. This is P5, now closed by R2a.
  - X2: two too-deep entries in one runner transaction collide, hence "one entry per runner transaction".
  - X3 and X4: the fallback index and erasure.
  - Go `uuid.NewSHA1` matched SQL on 1000 of 1000 tuples.
- RFC-009 rev 1, with the round-1 responses, is archived at `docs/adr/ADR-004/rfc/RFC-009-adr-004-e1-cascade-audit-vocabulary.md`, with `rev-0/` and `rev-1/` beside it.
