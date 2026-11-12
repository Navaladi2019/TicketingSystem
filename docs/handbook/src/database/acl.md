---
title: "Access control and masks"
section: database
status: delivered
tables: [acl_versions, record_shares]
adrs: [ADR-001]
sources: [docs/data/acl-tables.md, docs/data/acl-versions-shape-check.md]
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

## Diagram

<!-- gen:er area=acl -->

## Tables

<!-- gen:tables area=acl -->
