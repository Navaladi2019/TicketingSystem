# Access policy size limit

`nava-core` limits the size of an entity type's access policy. The limit is
set by `NAVA_ACL_MAX_DOCUMENT_BYTES` and applies to `rules` and to `masks`
separately, each measured as the bytes of its PostgreSQL `jsonb` text form
(`octet_length(rules::text)`), the same number the database measures.

| Setting | Value |
|---|---|
| Variable | `NAVA_ACL_MAX_DOCUMENT_BYTES` |
| Default (unset or empty) | 32768 (32 KiB) |
| Range | 16384 to 131072, an integer |
| Outside the range, or not an integer | `nava-core` refuses to start and names the variable |

The upper end is the database's own ceiling: the `acl_versions_bytes_rules_check`
and `acl_versions_bytes_masks_check` constraints refuse more than 131,072 bytes
whatever this setting says. Lowering the setting does not touch stored versions.

A publish above the limit is answered `422 invalid_access_policy` with the
problem code `document_too_large`, the pointer `/rules` or `/masks`, and the
limit in the detail. Nothing is written. No other response carries the value.

`nava-onboard` publishes its default access policy with the default limit; that
document is far under 16,384 bytes, so it passes at every allowed setting.

The setting is per deployment, not per tenant. The caps on depth, children,
policies and leaves are fixed (ADR-010-A1 A1-1) and are not settings.
