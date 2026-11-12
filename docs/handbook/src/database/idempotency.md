---
title: "Idempotency"
section: database
status: delivered
tables: [idempotency_keys]
adrs: [ADR-001]
sources: [docs/data/idempotency-lease.md, docs/data/idempotency-response-headers.md, docs/data/purge-indexes.md]
---

`idempotency_keys` is the replay cache for the HTTP `Idempotency-Key`
header (ADR-001 D6), keyed on `(tenant_id, principal_id, key)`. The first
request inserts an `in_flight` row with its request hash. When the request
finishes, the row keeps the status, the body and the allow-listed headers, so
a retry gets the same response. `lease_until` lets a retry take over an
abandoned `in_flight` row with one conditional `UPDATE`, without holding a
transaction while the handler runs. `nava_purge` deletes rows past
`expires_at`.

## Diagram

<!-- gen:er area=idempotency -->

## Tables

<!-- gen:tables area=idempotency -->
