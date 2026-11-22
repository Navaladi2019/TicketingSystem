---
title: "Access API: publish, read and explain an entity type's access"
section: system
status: delivered
normative: docs/architecture/access-document-format2.md
sources: [internal/server/access.go, internal/authz/document.go, internal/authz/check.go, internal/authz/policycheck.go, internal/authz/explain.go, internal/platform/config/acl.go, api/openapi.bundled.yaml, docs/architecture/access-document-format2.md, docs/adr/ADR-010-A1-sp2-caps-size-cap-app-only-publish.md]
packages: [internal/server, internal/authz]
adrs: [ADR-010, ADR-010-A1, ADR-010-A2, ADR-001]
stories: [S30, S10, S31]
invariants: [I5, I8, I1]
operations: [getEntityTypeAccess, putEntityTypeAccess, explainAccess]
tables: [acl_versions]
tests: [TestPutAccessFormat2StoresAsSubmittedWithOneAuditEntry, TestPutAccessFormat1IsStoredAsFormat2, TestPutAccessFormat2ForeignIDsAreForeignID, TestPrepareRefusesTheFixture, TestPrepareCapsAreInclusive, TestPrepareByteLimitIsOnStoredTextAndOnMasks, TestPrepareFormat1OverTheChildCapIsChildCount, TestPrepareWarnsOnNotOverCallerLeaves, TestLoadACLMaxDocumentBytes, TestExplainPolicyTraceAndInvalid, TestS30Format2DenyTraceHasAFalseStep, TestS31UnderChecksUnitsAtTheRouteAndGrantsBySubtree]
---

## Purpose

Three tenant-admin routes manage an entity type's access. Since S30 the access
rules are a **format-2 document** of named conditions and policies
(ADR-010). The route takes format 1 (the S10 array of rules) or format 2,
always stores format 2, and always returns format 2. How the stored document
is evaluated is on [Core: record ACL](core-acl.md); the document grammar is in
the normative [access-document-format2.md](../../../architecture/access-document-format2.md).

| Route | Operation | What it does |
|---|---|---|
| `GET /api/v1/entity-types/{key}/access` | `getEntityTypeAccess` | The newest version, `rules` in format 2 (a stored format-1 row is converted on read), `masks`, and the version as `ETag` |
| `PUT /api/v1/entity-types/{key}/access` | `putEntityTypeAccess` | Publishes version N+1 with `If-Match: N`. Body `{"rules": ..., "masks": [...]}`. Accepted lints come back in `warnings` |
| `GET /api/v1/access/explain` | `explainAccess` | Why one user can or cannot read one record |

## The document

```json
{"format": 2,
 "conditions": [
   {"name": "in_assigned_group", "expr": {"cmp": {"column": "assignment_group_id", "op": "in", "caller": "my_groups"}}},
   {"name": "is_requester",      "expr": {"cmp": {"column": "requester_id", "op": "eq", "caller": "self"}}}],
 "policies": [
   {"name": "agents_see_group_not_raised", "ops": ["read"],
    "when": {"all": [{"member_of": {"groups": ["<group id>"]}},
                     {"ref": "in_assigned_group"},
                     {"not": {"ref": "is_requester"}}]}}]}
```

- Node kinds: `all`, `any`, `not`, `ref`, `member_of`, `cmp`, `shared`,
  `under`. Since S31 `under` is available: `{"under": {"of": "caller"}}`
  (the caller's own units) or `{"under": {"of": ["<unit id>", ...]}}`; a
  record matches when its org unit is at or below one of them
  ([the under leaf](core-acl.md#the-under-leaf-s31)).
- Names match `^[a-z][a-z0-9_]{0,62}$` and are unique among conditions and
  among policies.
- Allow-only: an operation is granted when any policy listing it matches.
- A format-1 body is converted (each rule becomes one policy) and stored as
  format 2. A rule id that is not a valid name is renamed in place, for
  example `r-desk` becomes `r_desk`. Only a format-1 row stored before S30
  keeps its original id, as `legacy_id`, when it is read.

## PUT check order

The order decides which answer a caller gets when more than one thing is
wrong.

| # | Check | Answer |
|---|---|---|
| 0 | Request refused by the API middleware: body not a JSON object, no `Idempotency-Key` | `400 validation_failed` |
| 1 | `rules` or `masks` larger than the byte limit (before any connection is taken) | `422 invalid_access_policy`, `document_too_large` |
| 2 | Caller is not a tenant admin | `403 forbidden` |
| 3 | Unknown or foreign entity type | `404 not_found` |
| 4 | `If-Match` missing or `*` | `428 if_match_required` |
| 5 | `If-Match` is not the newest version | `412 version_conflict`, with the current `ETag` |
| 6 | Grammar, references, caps, mask fields, ids not in this tenant | `422 invalid_access_policy` with every problem and its pointer |
| 7 | Insert the version and one `acl_policy` audit entry | `200` with the new `ETag` |

Step 1 comes first so an oversized body costs no database connection
(ADR-010-A1 A1-2c). The `acl_versions` `CHECK` constraints and the tenant-id
trigger repeat steps 1 and 6; if one of them refuses (SQLSTATE 23514), the
route maps it to the same 422, never a 500.

## Problem codes

Every 422 is `invalid_access_policy`. Each entry in `errors` has a JSON
`pointer` into the request body and a `rule` naming the problem:

- **Access-document checks** (the table below) point into the document,
  prefixed `/rules` or `/masks`, for example `/rules/policies/0/when`.
- **Request-boundary checks** run first, before the document is read
  (strict JSON, TICK-964). A body that repeats a key
  is refused there with a pointer prefixed `/body`, for
  example `/body/rules/format` with rule `duplicate`. Nothing is stored.
- **An unknown top-level key** in a format-2 body points at the body root,
  for example `/extra` with rule `unknown_key` (TICK-1017), never under
  `/rules` or `/masks`.

| Rule | Meaning |
|---|---|
| `required`, `type`, `value`, `empty`, `unknown_key`, `duplicate`, `condition` | Shape errors, shared with format 1 |
| `invalid_name` | A condition or policy name does not match the name pattern |
| `duplicate_name` | Two conditions, or two policies, have the same name |
| `unresolved_ref` | A `ref` names no condition |
| `ref_cycle` | Conditions refer to each other in a loop |
| `child_count` | An `all` or `any` has no child or more than 16 (also a format-1 rule whose converted tree has more than 16, TICK-996) |
| `depth_exceeded` | A policy is nested more than 8 deep after `ref` expands |
| `policy_leaves_exceeded` | A policy has more than 128 leaves after `ref` expands |
| `policies_exceeded` | More than 64 policies |
| `document_leaves_exceeded` | More than 128 leaves across every policy |
| `foreign_id` | A group, user or org-unit id that is not a live id of the caller's tenant. Another tenant's id, a tombstoned unit and an unknown id answer the same, and the id is not echoed |
| `under_sets_exceeded` | A policy names more than 4 distinct `under` start sets after `ref` expands (`{"of": "caller"}` is one; the same ids in any order are one), at `/rules/policies/<i>` |
| `under_duplicate_id` | One `under` `of` lists the same unit twice, at each repeat's `.../under/of/<i>` (ADR-010-A2 A2-8) |
| `under_nested` | One `under` `of` lists a unit and one of its ancestors, at that leaf's `.../under/of` |
| `not_available` | A node kind that is not available. Since S31 every kind is available, so no current document gets it |
| `document_too_large` | `rules` or `masks` is over the byte limit, measured as the stored `jsonb` text |

**The byte limit appears only here.** The `detail` of a
`document_too_large` error says "The rules exceed the platform's configured
limit of N bytes, measured on the stored form." No other response, header or
route reports it. The limit is the operator setting
[`NAVA_ACL_MAX_DOCUMENT_BYTES`](../operations/acl-document-limit.md); the
other caps are fixed. The cap table, as published limits, is on the
[S30 page](../stories/s30.md#published-limits).

A publish can be accepted with a lint in `warnings` (`pointer`, `code`).
Today the only lint is `not_over_caller_leaf`: a `not` over a test of the
caller, which is true for a caller with no id or no groups.

## Explain response

`explainAccess` runs the same planner as enforcement, for the target user,
in the caller's tenant.

| Field | Meaning |
|---|---|
| `decision` | `allow` or `deny` |
| `policy` | The first policy that grants read; absent on a deny |
| `rule_id` | Kept for one release (ADR-010 D10): the policy's original format-1 id when it was converted from one, else the policy name; `null` on a deny |
| `trace` | Each named condition reached, with `condition`, `policy` and its boolean `value` |
| `invalid` | Each voided policy, with `policy`, `code` and its `/policies/<index>` pointer. A voided policy grants nothing (D5) and is counted in `authz_policy_invalid_total` |
| `masked_fields` | Fields hidden from the target user that the asking admin may read |
| `acl_version` | The version evaluated |

## Proving tests

| Behaviour | Test |
|---|---|
| Format 2 stored as submitted, one audit entry | `TestPutAccessFormat2StoresAsSubmittedWithOneAuditEntry` |
| Format 1 stored as format 2 | `TestPutAccessFormat1IsStoredAsFormat2` |
| Every code and pointer of the shared fixture | `TestPrepareRefusesTheFixture` |
| Caps are inclusive; byte limit on stored text and on masks | `TestPrepareCapsAreInclusive`, `TestPrepareByteLimitIsOnStoredTextAndOnMasks` |
| Format 1 over the child cap is `child_count` | `TestPrepareFormat1OverTheChildCapIsChildCount` |
| Foreign ids are `foreign_id` | `TestPutAccessFormat2ForeignIDsAreForeignID` |
| `under` org-unit ids checked at the route (`foreign_id`), and grants by live subtree | `TestS31UnderChecksUnitsAtTheRouteAndGrantsBySubtree` |
| Lint on `not` over a caller test | `TestPrepareWarnsOnNotOverCallerLeaves` |
| The setting's default and range | `TestLoadACLMaxDocumentBytes` |
| Explainer `policy`, `trace`, `invalid` | `TestExplainPolicyTraceAndInvalid`, `TestS30Format2DenyTraceHasAFalseStep` |

## Links

- Evaluation, planner and the list read path: [Core: record ACL](core-acl.md).
- The table and its constraints: [Access control and masks](../database/acl.md).
- Operator setting: [Access document size limit](../operations/acl-document-limit.md).
- Stories: [S30](../stories/s30.md), [S31](../stories/s31.md); decisions: [ADR-010](../adr/adr-010.md), [ADR-010-A1](../adr/adr-010-a1.md), [ADR-010-A2](../adr/adr-010-a2.md).
