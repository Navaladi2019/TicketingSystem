---
title: "Access control and masks"
section: database
status: delivered
tables: [acl_versions, record_shares]
adrs: [ADR-001, ADR-010, ADR-010-A1, ADR-010-A2]
stories: [S10, S30, S31]
sources: [docs/data/acl-tables.md, docs/data/acl-versions-shape-check.md, db/migrations/20261119100000_acl_rules_valid_format2.sql, db/migrations/20261119100100_acl_versions_bytes_validate.sql, db/migrations/20261121000100_acl_tenant_ids_org_units.sql, db/migrations/20261121161036_acl2_under_sets_cap.sql, db/grants.yaml]
tests: [TestTICK940Format2Fixtures, TestTICK940Caps, TestTICK940ForeignIDs, TestTICK940ByteCeilings, TestTICK940WorkerCannotPublish, TestTICK940MigrationOverFormat1Rows, TestTICK1059UnderSetsCap]
---

Authorization runs inside the query (ADR-001 I5), from two insert-only
tables:

- **`acl_versions`**: one immutable row per published ACL version of an
  entity type. `rules` and `masks` are JSON documents, each checked by a
  `CHECK` that calls the `acl_*_valid` functions. **There is no mask table.**
  Field masks are the `masks` document of the ACL version, read together with
  its rules.
- **`record_shares`**: per-record shares to a principal or a group. Each of
  the two typed columns has its own foreign key, so a share cannot name a
  principal or group that does not exist.

Neither table is updated or deleted; a new ACL is a new version. Publishing
an ACL is audited under its own subject type, `acl_policy`.

## What the database checks on an `acl_versions` insert (S30, S31)

Since S30 (TICK-940, ADR-010 D4 and ADR-010-A1) `rules` holds either an S10
format-1 array or a format-2 document (`{"format": 2, "conditions": [...],
"policies": [...]}`). Every new version is stored as format 2; format-1 rows
already stored stay as they are and are converted when read. The database
repeats the route's checks, so a row that bypasses the route still cannot
store a broken or oversized document:

| Check | Object | Refuses |
|---|---|---|
| Byte ceiling on `rules` | `acl_versions_bytes_rules_check`: `NOT (rules ? 'format') OR octet_length(rules::text) <= 131072` | a format-2 `rules` over 128 KiB |
| Byte ceiling on `masks` | `acl_versions_bytes_masks_check`: `octet_length(masks::text) <= 131072` | `masks` over 128 KiB |
| Format-2 grammar and caps | `acl_versions_rules_check`, through the `acl2_*` functions | unknown node kinds, bad names, unresolved refs, cycles, depth over 8, more than 16 children, 64 policies or 128 leaves; since S31, more than 4 distinct `under` start sets in one policy, refs expanded (`under_sets_exceeded`, TICK-1059) |
| Ids belong to the tenant | trigger `acl_versions_tenant_ids` (`AFTER INSERT`, `SECURITY INVOKER`) | a group or user id that is not the row's tenant's, or (since S31, TICK-947) an `under` id that is not a **live** org unit of the row's tenant |

The two `bytes_` constraints are named so that PostgreSQL, which runs
`CHECK`s in name order, refuses an oversized row before any walk of it.
They were added `NOT VALID` and validated in a separate migration. The
128 KiB ceiling is the top of the operator setting
[`NAVA_ACL_MAX_DOCUMENT_BYTES`](../operations/acl-document-limit.md); the
route enforces the configured (lower or equal) limit first. A refusal
carries a JSON `DETAIL` with the column, code and pointer, which the route
maps to the same 422 it would have given itself.

**Only `nava_app` inserts into `acl_versions`** (ADR-010-A1 A1-3). It is the
only role in the `acl_versions_insert` policy and with `INSERT` on the table
and `EXECUTE` on the check functions. `nava_worker` reads versions but cannot
publish one. See [Roles and grants](roles-and-grants.md).

## Diagram

<!-- gen:er area=acl -->

## Tables

<!-- gen:tables area=acl -->
