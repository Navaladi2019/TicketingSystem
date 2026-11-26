---
type: adr-erratum
id: ADR-004-E10
title: Erratum E10 to ADR-004, diff_nul_escaped joins the R1 hash envelope as a member emitted only when true
status: ACCEPTED
accepted: 2026-11-27
date: 2026-11-26
issue: TICK-1169
rfc: none (erratum raised from the TICK-1121 thread; proposal by the Postgres & Data Modeling Specialist, comment 01a217e4-cc79-7ba1-9527-47b0c0902373; the options are recorded below)
supersedes: "the R1 envelope list in [[ADR-004-audit-log]] (as extended by [[ADR-004-E1-cascade-entries-and-diff-truncated]]), and the 'every member present' rule of docs/api/audit-export-v1.md §2 for this one member"
amends: "[[ADR-004-audit-log]] R1, R10, R13 and the acceptance tests (adds CI-A19); [[ADR-004-A1-diff-read-role]] A1-1 as corrected by [[ADR-004-E8-a1-reader-chain-columns]] (adds one column); docs/api/audit-export-v1.md §2, §5, §6 by reference"
source: "TICK-1121 (publish over a stored version that holds a NUL), decision 01a216f1 (option A) and the Technical Lead routing 01a2184e-f839"
deciders: Solution Architect (author, verdict); Technical Architect (critique, no blocking findings, four non-blocking corrections folded in as rev 1, comment 01a21886-8f61-741f-af07-2bc7de6292da); Postgres & Data Modeling Specialist (proposal, migration owner, TICK-1170); CTO (tie-break if contested)
---

# ADR-004 erratum E10: `diff_nul_escaped` in the hash envelope

> **Status: ACCEPTED (rev 1), 2026-11-27.** The Solution Architect accepted it on TICK-1169 after the Technical Architect's critique (comment `01a21886`): no blocking findings, and "no objection to ACCEPTED" once corrections #1 to #3 are in the text. Rev 1 takes all four corrections (see "Response to the critique"). Nothing was contested, so no CTO tie-break was needed. No human sign-off is required under the current policy.
>
> **Invariants source.** The project description is empty. As in ADR-004 and E1–E9, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. Against any separate list of eight, these checks are incomplete.

## Plain words

When an admin repairs a ticket type whose stored version holds a NUL character, the audit entry for that publish has to describe the old version, and PostgreSQL `jsonb` cannot hold a NUL. So the old version is copied with each NUL written as the six characters `\u0000`, and the entry carries a yes/no flag, `diff_nul_escaped`, that says "this copy was escaped". The flag is sealed into the entry's tamper-evident hash, so nobody can flip it unseen. It is hashed only when it is yes, which means every entry written before this change hashes exactly as before and still verifies.

## Context

- TICK-1121 option A: publishing a clean version over a stored version that holds a NUL answers **201**. It is the admin's only repair path.
- The audit diff for that publish is `jsondiff.Diff(before, after)` (RFC 6902, only `add`, `remove`, `replace`; values come from `after` only). `before` cannot be read as `jsonb` (`22P05`), so `internal/entitytypes` (`escapeNUL`, PR #928) decodes it and writes each NUL in a string or key as the text `\u0000` before the diff.
- Without a flag, a reader cannot tell an escaped NUL from a literal `\u0000` typed by an admin. Escaping can also hide a change: a NUL value replaced by the literal text `\u0000` compares equal after escaping, so no op is emitted for it. The flag is what tells a reader "no op here means unchanged *after escaping*".
- E1 put `diff_truncated` in the envelope for the same reason: `diff_sha256` protects the bytes, not the claim about what the bytes mean. E1 needed no rule for old rows because `audit_log` was empty then. It is not empty now.

## Decision

**R1, envelope, new text:** the 22 members of E1, plus **`diff_nul_escaped`**, which is **emitted in the canonical form only when true**.

**E10-1 · the column.** `audit_log.diff_nul_escaped boolean NOT NULL DEFAULT false`, next to `diff_truncated`. The constant default makes `ADD COLUMN` catalogue-only on PostgreSQL 16 (no rewrite). It is in the R10 INSERT grant, outside both R11 transitions (erasure leaves it unchanged), and no role has UPDATE on it. Backfilled rows write `false`. **[TA #3]** The migration recreates `audit_log_guard_update` with `diff_nul_escaped` in both `same_envelope` tuples, so the trigger refuses any change to it with 42501 for every role, even if a column grant is widened later. The grant and the trigger are the two layers, as for every other envelope column.

**E10-2 · how rows sealed before E10 still verify (question 1).** The canonical envelope (`chain.CanonicalEnvelope`) omits the member when the flag is false and emits `"diff_nul_escaped":true` when it is true. Every row written before E10 has `false` (column default), so its canonical bytes, and therefore its `envelope_sha256` and `entry_hash`, are byte-identical to the pre-E10 form. There is no cut-over point, no envelope version, no re-hash.

Verifier rule (`chain.Envelope.UnmarshalJSON`, `internal/audit/verify`):

| Envelope object in the file | Parsed as | Outcome |
|---|---|---|
| member absent | `false` | normal checks |
| `"diff_nul_escaped":true` | `true` | normal checks |
| `"diff_nul_escaped":false`, `null`, or any other value | — | `malformed_line` (one encoding per value) |

Tamper cases: setting the column to true on a sealed row, or clearing it, changes the canonical bytes, so `envelope_sha256_mismatch`. Stripping the member from an export line does the same. Spike `e10/` shows both.

This is the **only** member that may be absent. Every other member keeps "NULL is emitted, never omitted". `TestEveryFieldIsHashed` and `TestCanonicalKeysAreSortedAndComplete` are updated to name it as the single optional member, so a second one cannot slip in unnoticed.

**E10-3 · what sets it.** `audit.Entry.DiffNULEscaped` is true exactly when `escapeNUL` replaced at least one NUL in the previous schema (decoded JSON, never a text replace). The escaped bytes go through `$1::jsonb::text` (TICK-451 canonical bytes), and `diff_sha256 = SHA-256(diff_salt ‖ stored escaped diff)`. The original NUL bytes are hashed nowhere in the audit log; they stay in `entity_definition_versions`, whose `content_sha256` is unchanged. Truncation (E1) runs after the diff and is independent: both flags may be true.

**E10-4 · exposure (question 2).**

| Surface | Rule |
|---|---|
| `nava_audit_reader` (A1-1 / E8) | SELECT on `diff_nul_escaped`; the list becomes 31 columns; `audited.ReaderColumns()` matches. A boolean carries no content. |
| `nava_app` read SELECT, `nava_worker` (sealer), `nava_erasure` | Every `db/grants.yaml` list that holds `diff_truncated` gets `diff_nul_escaped` beside it. No UPDATE anywhere. |
| Audit read endpoints (R8) | Each entry gains `diff_nul_escaped` (boolean, always present). |
| Export JSONL (`nava.audit.export/v1`) | The envelope object carries the member only when true (E10-2). The format id stays v1: every v1 file and vector verifies unchanged, and a pre-E10 verifier fails closed (`malformed_line`) on a line with the member, never a false pass. |
| Export CSV **[TA #2]** | New column `diff_nul_escaped` right after `diff_truncated`. An empty cell means the member is **omitted** from the rebuilt envelope: the one exception to §5's "empty is null". `true` means the member is `true`. `false` or any other value gives `malformed_line`. `envLast` in `verify/csv.go` points at the new column, and the column is **not** in `csvBools` (which accepts `false`). The verifier also accepts the pre-E10 37-column header, where the column is absent and every row is false. |
| Golden vectors | Existing vectors unchanged. New: one sealed entry with the flag true, and two tampered lines (flag cleared in the row; member stripped from the line). Technical Lead review per PLAN-005 D-5-7. |

**E10-5 · key collision (question 3): no refusal.** **[TA #1]** A collision is **two keys of one object whose escaped forms are equal**. Examples: `a<NUL>b` next to a literal `a\u0000b`, or two keys that both hold a NUL and escape to the same text (`a<NUL>\u0000` and `a\u0000<NUL>`). `escapeNUL` detects it by testing the *output* map, which holds every key's escaped form (`if _, taken := out[k2]; taken`), never the input map: the input-map test misses the two-NUL case and silently drops a member (TA spike, 5 of 5). Refusing the write (PR #928 today) would close the admin's only repair path, and the request is not at fault. Instead, when `escapeNUL` reports a collision anywhere, the diff is the single op `{"op":"replace","path":"","value":<after>}`. That is a correct RFC 6902 patch from the real `before` to `after`, it carries no `before` content, so it holds no NUL; the flag is true. The publish answers **201** like every other option-A publish. E1 truncation still applies to that one op. The diff loses detail only in a case that needs a pre-2026-11-03 stored version with two sibling keys differing only by NUL versus literal escape.

**E10-6 · ordering, enforced by the database [TA #4].** No row may be written with `diff_nul_escaped = true` until the envelope member, the sealer's column read and the verifier rule are on `main` (TICK-1170 step 2). A row sealed with the flag true but hashed without it would fail verification forever (`seal.go` `unsealedSQL` names its columns). So the rule is mechanical, not discipline:

- **Step 1** adds the column and a gate: a `BEFORE INSERT` row trigger `audit_log_nul_escaped_gate` that raises `23514` when `NEW.diff_nul_escaped` is true. UPDATE is already refused by the recreated `audit_log_guard_update` (E10-1).
- **Step 2** drops the trigger, in the same PR as the envelope member, the sealer, the verifier and the export change.

A trigger, not the `CHECK` the critique sketched: `ADD CONSTRAINT … CHECK` scans every partition under `ACCESS EXCLUSIVE` and blocks audit writes for the scan, and whether `NOT VALID` is accepted on a partitioned table is unverified. The trigger checks only new rows and needs no scan. TICK-1170 may use the `CHECK` instead if it measures the scan on the seed data and finds it negligible. Either way an early writer gets `23514`, never a sealed row that reads `tampered`. PR #928 may merge before step 2; its flag-true path then fails loudly until step 2 lands.

## Competitive Gate (ADR-008)

**Result:** N/A. This erratum adds one tamper-evident flag to an internal audit row so that a repair publish can be audited. It moves no product axis and changes no capability a customer compares.

## Response to the critique (rev 1)

| TA item | Response |
|---|---|
| #1 collision definition and the input-map check | **Accepted.** E10-5 defines a collision as two keys of one object with equal escaped forms and requires the output-map test. CI-A19(f) covers both shapes. The code fix belongs to the PR #928 owner. |
| #2 CSV empty cell read as `null` | **Accepted** with the critique's wording (E10-4). |
| #3 `audit_log_guard_update` misses the column | **Accepted.** E10-1 recreates the trigger; CI-A19(c) expects 42501 and proves the mismatch on export lines. |
| #4 ordering is discipline | **Accepted, as a trigger rather than a `CHECK`** (E10-6), for the scan and partition reasons given there. |
| #5 to #9 | Confirmed by the critique; no change. |

## Rejected options

| Option | Why rejected |
|---|---|
| Always-emitted member (the E1 way) | Every row sealed before E10 changes its `envelope_sha256`; the whole history would read `tampered`, or need a re-hash, which is what the chain exists to forbid. |
| Hash the member only after a per-tenant cut-over `seq` | The verifier needs the cut-over value from somewhere it can trust; in an offline export it can only come from the unauthenticated header. More state, same result as E10-2. |
| New `envelope_version` member, verifier accepts v1 and v2 mixed | The version member itself must be absent on old rows, so it needs the same "absent means old" rule, plus a second canonical form to maintain. |
| Flag outside the envelope, like `diff_salt` (the agreed fallback) | The flag could be flipped without detection, so an escaped diff could pass as unescaped. `diff_salt` is outside only because R11 erasure nulls it; that reason does not apply. |
| Refuse the publish on a key collision | Blocks the only repair path for a fault in stored data the admin did not just send. |
| Escape in SQL with `replace(schema::text, …)` | Corrupts a stored literal `\\u0000` (TICK-1121 criterion 2 accepts it). |
| Keep the E10-6 ordering as discipline only | A sealer older than step 2 would seal a true row that reads `tampered` forever; the database can refuse it at no cost (TA #4). |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E10-1 | With the flag true, a NUL value changed to the literal text `\u0000` emits no op. The flag says so; the exact old bytes are only in `entity_definition_versions`. | Solution Architect | An auditor needs the byte-exact old value inside the audit row. |
| RS-E10-2 | A key collision gives a whole-document `replace` diff. | Solution Architect | One is ever seen (the writer logs `audit_nul_escape_collision` with the definition id). |

No new security risk; nothing is added to `docs/security/accepted-risks.md`.

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. Same table, same policies; the column follows the row. |
| I4 audit in the same transaction | Holds and is strengthened: the repair publish is audited instead of failing with 500, and the claim that the diff was escaped is tamper-evident. |
| I7 idempotency | Not touched. |
| I8 platform-owned vocabulary | Holds: one column by platform migration, no tenant DDL. |
| I2, I3, I5, I6, I9 | Not touched. |

## Acceptance tests

| # | Proves |
|---|---|
| CI-A19 | (a) All existing golden vectors and every sealed row in the integration seed verify unchanged after the migration and the code change. (b) A publish over a stored version that holds a NUL answers 201; the entry has `diff_nul_escaped = true`, no NUL in `diff`, and the chain verifies. (c) `UPDATE … SET diff_nul_escaped` is refused with 42501 for every role (a `"nul_escaped"` case in the `updates` map of `internal/tenancy/isolation/audit_paths.go`, next to `"truncated"`). The mismatch half is proven on export lines: the member added to a sealed false row's line, and removed from a true row's line, each give `envelope_sha256_mismatch`. Before step 2, an INSERT with the flag true gives `23514`. (d) A JSONL line with the member stripped gives `envelope_sha256_mismatch`; one with `"diff_nul_escaped":false` gives `malformed_line`. (e) A previous schema with a real NUL next to a literal `\\u0000` escapes only the real one and sets the flag; one with no NUL leaves bytes unchanged and the flag false. (f) A collision gives the one root `replace` op, flag true, 201, for both shapes: a NUL key next to a literal key, and two NUL keys with the same escaped form (also in `nulescape_test.go`). (g) Pre-E10 and E10 CSV headers both verify; an empty `diff_nul_escaped` cell verifies as omitted; a `false` cell gives `malformed_line`. (h) The reader reads the column; tenant B and no-tenant see zero rows (CI-A4). |

## Evidence

- Spike `docs/adr/ADR-004/spikes/solution-architect/e10/` on `main` at `d2cc513f`: `envelope.patch` (a `trueOnlyField` in `envelopeFields`), `e10_spike_test.go.txt`, `output.txt`. With the patch, every golden-vector test (`TestGoldenVectorsChain`, `…Tampered`, `…Exports`, `…Diffs`, `…Invalid`, `TestGoldenVectorPinned`, `TestVectorsFileIsCurrent`) and the `verify`, `export` and `seal` packages pass unchanged. The flag true changes the hash, a stripped member is detected, explicit `false` and `null` are rejected. The two guard tests that assert "every member always present" fail as expected and are updated by the implementation (E10-2).
- `jsondiff` contract (`internal/audit/jsondiff/doc.go`): ops carry `after` values only, so `before` reaches the diff only through paths. This is why the root-`replace` fallback carries no NUL.
