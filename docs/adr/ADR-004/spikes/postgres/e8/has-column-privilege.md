# ADR-004-E8 evidence: catalogue checks (PostgreSQL 16.10, live stack)

Source: Postgres & Data Modeling Specialist, TICK-612 comment 01a1b7fb-e87c-73a1-bd36-fb75708c5387, PR #508.

| Check | Before PR #508 | After PR #508 (as posted) | Required by E8 |
|---|---|---|---|
| `has_column_privilege('nava_audit_reader','audit_log','prev_hash','SELECT')` | f | t | t |
| `has_column_privilege('nava_audit_reader','audit_log','entry_hash','SELECT')` | f | t | t |
| `has_column_privilege('nava_audit_reader','audit_log','erasure_ref','SELECT')` | f | f | **t** |
| `has_column_privilege('nava_audit_reader','audit_log','txid','SELECT')` | f | f | f |
| `has_column_privilege('nava_app','audit_chain_head','seq','SELECT')` | t | t | t |
| `has_column_privilege('nava_app','audit_chain_head','entry_hash','SELECT')` | t | t | t |
| `has_column_privilege('nava_app','audit_chain_head','export_lease_until','SELECT')` | t | t | t |
| reader on `audit_chain_head` | no privilege (42501) | 42501 | 42501 |

`SELECT: []` in `db/grants.yaml` is a table-level grant (see the file header), which is why `nava_app` already reads the head.

Tests: `internal/db/dbtest/audit_export_reads_test.go`
- `TestP540aReaderReadsChainHashes`: red (42501) before the grant, green after; tenant B and no-tenant reads return 0 rows.
- `TestP540aExportLeaseAndHeadRead`: lease take, 409 while held, cross-tenant, expired lease take, reader 42501 on the head.
