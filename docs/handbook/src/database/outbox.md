---
title: "Outbox"
section: database
status: delivered
tables: [outbox]
adrs: [ADR-001]
sources: [docs/data/outbox-payload-check.md, docs/data/purge-indexes.md]
---

A write that must start or signal a workflow inserts an `outbox` row in its
own transaction (ADR-001 D22). The relay (`nava_relay`) is the only process
that starts workflows. It reads pending rows of every tenant through
`outbox_relay_select`, an allow-listed policy with no tenant test, and takes
the tenant only from the row. Payloads carry references, never record
content; a `CHECK` enforces the payload keys. `dedupe_key` stops a duplicate
start, and `target_seq` orders signals per target.

`nava_purge` deletes dispatched rows after 7 days and dead rows after 30,
through paired `SELECT` and `DELETE` policies with the same expiry
predicate.

## Diagram

<!-- gen:er area=outbox -->

## Tables

<!-- gen:tables area=outbox -->
