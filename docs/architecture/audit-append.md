# The audit writer: `audit.Append` (P5-12b, TICK-430)

`audit.Append(ctx, tx, Entry) (uuid.UUID, error)` is the one writer of `audit_log` ([ADR-004](../adr/ADR-004-audit-log.md) decision 1, PLAN-005 C2 as changed by grooming TICK-449). It writes one entry inside the caller's transaction, so the entry commits or rolls back with the change it describes.

- Code: `internal/audit/append.go`, `entry.go`, `diff.go`.
- Tests: `append_test.go` (unit, a fake `pgx.Tx` that records every statement) and `append_integration_test.go` (real PostgreSQL as `nava_app`, both pgx exec modes).
- The diff comes ready-made from `internal/audit/jsondiff` (P5-12c). Secret rejection is `internal/audit/secrets` over `internal/audit/patchwalk` (P5-12a). Hashing is `internal/audit/chain` (P5-02).

## Order of work

Nothing is inserted until every check has passed. A wrong entry fails before the transaction is touched.

```mermaid
flowchart TD
    A[Entry] --> V{validate: vocabularies, one actor,<br/>R3 content hash, R4 source_ref,<br/>canonical-form encodable}
    V -- no --> EI[ErrInvalidEntry<br/>no SQL]
    V -- yes --> E{action = erase?}
    E -- yes --> ES{one add of /erased_ids,<br/>1..1,500 lowercase ids}
    ES -- no --> EI
    ES -- yes --> S
    E -- no --> S{secrets.Check on the FULL diff<br/>R5 walk, R6 reject}
    S -- secret --> ESD["*ErrSecretInDiff<br/>(audit_secret_in_diff), no SQL"]
    S -- malformed --> EI
    S -- ok --> T{diff over 65,536 bytes?}
    T -- "yes (erase)" --> EI
    T -- yes --> CUT[keep whole ops: '[' raw1 ',' … rawk ']'<br/>diff_truncated = k &lt; n]
    T -- no --> KEEP[store bytes unchanged]
    CUT --> H[16-byte salt from crypto/rand<br/>diff_sha256 = SHA-256 salt ‖ stored]
    KEEP --> H
    H --> TEN{tenancy.TenantOf tx}
    TEN -- none --> EI
    TEN -- tenant --> INS[INSERT the R10 columns only]
    INS -- "23505 on *_commit_key" --> DUP[ErrDuplicateEntry]
    INS -- ok --> ID[entry id]
```

## Rules the code holds

| Rule | Where |
|---|---|
| No `Entry.TenantID`: the tenant is the transaction's `app.tenant_id`, read with `tenancy.TenantOf`; none set → `ErrInvalidEntry`, no INSERT (I2). The canonical `WITH CHECK` policy refuses any other tenant. | `append.go` |
| Never sets `occurred_at`, `txid`, `backfilled`, the seal or the erasure columns (R10). | `insertSQL` |
| `id` is a UUIDv7 when the caller leaves it zero. | `append.go` |
| `content_sha256` is required for `domain = config`, except a `delete`, as `audit_log_config_content_check` (R3, Annex A). | `entry.go` |
| `source = ai` needs `source_ref`, a lowercase UUID (R4). | `entry.go` |
| Every envelope value must pass `chain.CanonicalEnvelope`, so nothing is stored that the sealer could not hash (empty text, U+0000, invalid UTF-8, integers past 2^53 − 1). | `entry.go` |
| Past 64 KB, whole ops only, original bytes; the flag is in the hashed envelope; `diff_sha256` is over the stored bytes (ADR-004-E1 item 2). | `diff.go` |
| An `audit/erase` entry lists 1 to 1,500 ids and is never cut (ADR-004-E1 §3). | `diff.go` |
| The diff is bound as text with `::json`, so the stored bytes are the given bytes under `QueryExecModeExec` (PgBouncer) too. | `insertSQL` |

## Errors

| Error | Meaning |
|---|---|
| `ErrInvalidEntry` | The entry is wrong in itself, or the transaction has no tenant. A caller bug. The message names the field and the rule, never a value. |
| `*ErrSecretInDiff` (alias of `secrets.ErrSecretInDiff`) | R6. Code `audit_secret_in_diff`; names the pointer, op and rule, never the value. |
| `ErrDuplicateEntry` | The transaction already has an entry for this subject and `version_after` (the R2 commit key). Not in C2's list: `docs/data/audit-log.md` asks the writer to match the `_commit_key` suffix. |
| anything else | The INSERT failed; wrapped as `audit: insert entry: …`. |

`Append` does not log or count: it returns typed errors, and the caller logs them (each one is safe to log). Secret paths are matched token by token with no array wildcard; callers pass concrete pointers.
