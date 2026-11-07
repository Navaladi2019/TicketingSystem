---
type: adr-erratum
id: ADR-004-E8
title: "Erratum E8 to ADR-004-A1 A1-1 and A1-7: the reader also reads prev_hash, entry_hash and erasure_ref; the R20 head read is nava_app's lease statement"
status: DRAFT
date: 2026-11-08
issue: TICK-612
rfc: none (erratum raised from the TICK-612 thread, comment 01a1b7fb-e87c-73a1-bd36-fb75708c5387; no design choice to debate)
supersedes: "ADR-004-A1 A1-1, the audit_log column list only. A1-7 gains one sentence."
amends: "[[ADR-004-A1-diff-read-role]] A1-1 and A1-7 only. ADR-004 R19, R20, R24 and A1-2..A1-11 are unchanged."
source: "TICK-444 (P5-40, audit export) could not run its pages on the reader pool; PR #508 evidence"
deciders: Solution Architect (author, verdict); Technical Architect (critique); CTO (tie-break if contested); Security & Multi-Tenancy Reviewer (review of PR #508)
---

# ADR-004 erratum E8: the export's three chain columns on the reader role

> **Status: DRAFT, 2026-11-08.** This erratum **corrects an omission**. It changes no rule: A1-5 already puts the export pages on the `nava_audit_reader` pool, and R20 and the export format (`docs/api/audit-export-v1.md` §2) already need these columns. A1-1 listed the reader's columns for the two read endpoints only, and left out the three the export emits.
>
> **Invariants source.** The project description is empty. As with E1–E7, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. That check is incomplete against any separate list of eight.

## Context

- A1-5: `Reader.List`, `Reader.RecordEntries` **and the export pages** run on the reader pool.
- R20: every `seq` in the range is emitted, either in full or as a stub `{seq, prev_hash, entry_hash, envelope_sha256}`, starting from the anchor at `first_seq − 1`.
- Export format §2: a full entry line carries `erased_at` **and `erasure_ref`**. R24 needs `erasure_ref` to tie an erased entry to its `audit/erase` entry.
- A1-1 lists 27 `audit_log` columns and none of `prev_hash`, `entry_hash`, `erasure_ref`. On PostgreSQL 16.10, `has_column_privilege('nava_audit_reader', 'audit_log', 'entry_hash', 'SELECT')` is `f`, so the export page fails with `42501`.
- A1-1 and A1-7 say nothing about where `head_seq` and `head_hash` (R20, "read at the start") come from. The reader has no access to `audit_chain_head`, and must not get any.

## Decision

**A1-1, the `audit_log` grant, now reads:**

> SELECT on `audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated, diff, diff_salt, seq, prev_hash, entry_hash, sealed_at, erased_at, erasure_ref)`.

The rest of A1-1 is unchanged: `entity_definition_versions (definition_id, schema)`, "nothing else", no INSERT/UPDATE/DELETE, nothing on `records` or any partition, and **`txid` stays out**. `audited.ReaderColumns()` lists the same 30 columns.

**A1-7 gains one sentence:**

> The R20 head read is the lease statement itself, on `nava_app`: `UPDATE audit_chain_head SET export_lease_until = … WHERE tenant_id = $1 AND (export_lease_until IS NULL OR export_lease_until < now()) RETURNING seq, entry_hash`. Zero rows means 409 `export_in_progress` (R19). `nava_audit_reader` gets no grant on `audit_chain_head`.

## Why these three columns are safe on the reader

| Column | What it is | New information to the reader? |
|---|---|---|
| `prev_hash`, `entry_hash` | SHA-256 chain links (R13) | No content. They are derived from columns the reader already reads, and the export publishes them by design. |
| `erasure_ref` | id of the erasure request | No. It equals `subject_id` of the matching `audit/erase` entry, which the reader already reads. `erased_at` already shows that an entry was erased. |

Row access is unchanged: `audit_log_select` under the canonical tenant predicate (A1-3). With no tenant context, or under another tenant, the reader sees zero rows (CI-A4).

## Options considered

| Option | Verdict | Why |
|---|---|---|
| **A. Add the three columns to the reader; head read in `nava_app`'s lease statement** | **Chosen** | One grant, no migration, no policy change. The reader stays off the head. |
| B. Export pages read the hashes on `nava_app` | Rejected | Breaks A1-5. The page would span two pools and two snapshots per entry. `nava_app` would need the hashes, which R14 keeps away from it. |
| C. Give the reader SELECT on `audit_chain_head` | Rejected | Widens the reader to a second table for a value the lease statement already returns. |
| D. Treat it as an implementation gap with no erratum | Rejected | A1-1 is a closed column list, and CI-A3 and the manifest comments cite it. If the list and the grant disagree, the next review would flag the grant as drift. |

## Consequences

- **Isolation (I1, I3):** row scope unchanged. `NOBYPASSRLS` unchanged. CI-A3 / A1-9 unchanged: only the reader holds `diff` / `diff_salt`, nothing holds a privilege on a partition directly, and `audit_log_select` is exactly `{nava_worker, nava_erasure, nava_audit_reader}`.
- **Snapshot:** the head read (one `nava_app` statement) and the pages (reader transactions) see different snapshots. That is safe. Pages are bounded by `head_seq`, sealed entries do not change their hashes, and erasure (R11) touches neither hash.
- **Temporal / idempotency:** not affected.
- **No new accepted risk.** Nothing is added to `docs/security/accepted-risks.md`.

## Evidence

- PR #508 (TICK-612, base `TICK-421/audit-export`): `db/grants.yaml`, `internal/db/audited/manifest.go`, `internal/db/dbtest/audit_export_reads_test.go`.
- Red to green: `TestP540aReaderReadsChainHashes` fails with `42501` before the grant and passes after it. It also checks zero rows for tenant B and with no tenant set. `TestP540aExportLeaseAndHeadRead` covers taking the lease, the 409 while it is held, the cross-tenant case, taking an expired lease, and the reader's `42501` on the head.
- Catalogue output: `docs/adr/ADR-004/spikes/postgres/e8/has-column-privilege.md`.
- **Open at DRAFT:** PR #508 as posted grants `prev_hash, entry_hash` only, and its test asserts that the reader gets `42501` on `erasure_ref`. It must add `erasure_ref` and flip that assertion before this erratum can be ACCEPTED.
