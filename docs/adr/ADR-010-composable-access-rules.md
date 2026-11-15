---
type: adr
id: ADR-010
title: Composable access rules - named conditions and allow-only AND/OR/NOT policies
status: ACCEPTED
date: 2026-11-15
accepted: 2026-11-15
issue: TICK-901
rfc: RFC-022 (docs/adr/ADR-010/rfc/RFC-022-composable-access-rules.md)
supersedes: none
---

# ADR-010: Composable access rules

**Status: ACCEPTED** on 2026-11-15 by the Solution Architect (merge policy set by vknavaladi), PR #716. Technical Architect: RFC critique round 1 was CHANGES (3 blocking, 6 non-blocking). All 9 items were accepted in RFC-022 rev 1, and the re-check was APPROVE with no new items. No tie-break was needed. The Technical Lead decided Q3 and asked for one wording fix on B3; both are folded in (D6, D8). RFC thread: TICK-901, root comment `01a1de87-1025`, rev 1 in thread `01a1dec3-ede0`. The ADR critique on PR #716 was CHANGES with one blocking item (A1, the missing `IS NOT NULL` guard in D7) and one non-blocking item (N1, the planning check in D6). Both are fixed. The comprehension Goldfish check ran on this text, and its wording findings are folded in.

> **Invariants source.** ADR-001 I1 to I9, the platform's nine architectural rules; the project description is empty. This ADR touches I1 (tenant wall), I3 (database roles), I4 (one write pipeline), I5, I6 (workflow determinism) and I8. I2, I7 and I9 are unchanged.

## In plain words

- **What changes:** admins build access rules from named, reusable yes/no tests called conditions, for example "I raised it", "it is assigned to my group" or "it is in my region". They combine the tests with "and", "or" and "not" into named policies, and each policy says who may read or change which tickets. A step in an automated workflow can point at a policy by its name.
- **Why:** nava asked for ticket and workflow security to be configuration, not code. Today's rules cannot be reused and have no OR or NOT.
- **What you'll see when it's done:**
  - A broken rule is refused when the admin saves it, with a pointer to the bad part.
  - Editing one condition updates every policy of that ticket type that uses it, in one saved change.
  - The admin's "why can this user see this ticket" screen names the policy and the conditions that decided.
  - Existing rules keep giving the same answers.
- **Codes used below:**
  - I5: access rules become a database filter, and the single-record check in Go gives the same answers.
  - I8: configuration, not code.
  - SP-1, SP-2: two planned measurement spikes. SP-1 measures region-access speed; SP-2 measures the save-time check cost and sets the size limits.
  - nava: the product owner.
  - S10: the story that shipped today's flat rules.
  - TICK-900: the open fix that proves the database filter and the Go check agree on tenant-defined JSON fields.

## Decision

| # | Decision |
|---|---|
| **D1** | **Allow-only.** Access to an op is the OR of the policies that grant that op. There are no deny rules, no precedence and no ordering. NOT exists only inside an expression (a condition or a policy's `when`, e.g. `in_assigned_group AND NOT is_requester`), never as a separate deny rule. |
| **D2** | **One document per entity type, one version.** `acl_versions.rules` format 2 is `{"format": 2, "conditions": [{name, expr}], "policies": [{name, ops, when}]}`. Arrays, so a duplicate name is a 422 and the declared order is kept. Names match `^[a-z][a-z0-9_]{0,62}$`. Publishing keeps the existing optimistic version and the audit in the same transaction. `masks` is unchanged. |
| **D3** | **Closed grammar.** An expression is one of `all`, `any` (1 to 16 children), `not`, `ref`, `member_of` (the caller is one of the listed groups or users), `cmp` (today's `Cond`: a fixed column compared with values or with the caller), `shared` (the ticket has a share naming the caller, or a group share naming one of the caller's groups) or `under` (org-unit subtree). There is no node kind for a function, a script or an external call, so those are refused as `unknown_key`. |
| **D4** | **Save-time checks**, in the publish route (422 with a JSON pointer) and in the PL/pgSQL CHECK as backstop: every `ref` resolves; no reference cycle; depth at most 8 after expansion; at most 256 leaves per policy; at most 64 policies and 1,024 leaves per document; every group, user and org-unit id belongs to the tenant. SP-2 reports before the feature ships and may lower the caps, so no stored document is ever above them. |
| **D5** | **NULL and invalid are different.** (a) Every valid leaf is two-valued in SQL and in Go. A NULL column, a missing caller id or an empty resolved set makes the leaf false, so its NOT is true. `shared` is false when the caller has no id, even if it has groups. (b) `plan` returns `(tree, error)` per policy. An invalid node anywhere voids the **whole policy**, which then grants nothing, and is counted in `authz_policy_invalid_total`. Invalid is never a boolean, so it never passes through `not`. |
| **D6** | **One planner, two renderings.** `plan` expands references and folds caller-only leaves, then `Allows` (Go) and `CompileRead` (SQL) render the same tree. `ReadBranches` splits on the top-level OR across policies and on a policy's top-level `any`. A top-level `all` with exactly one `any` child is distributed into one branch per child (today's `disjuncts`, at most 16 per policy). Above a **total-branch threshold set by SP-2** (not a fixed 16), it falls back to `CompileRead`. The fallback is the slower OR form (38-166 ms per page, see Context), so at planning the Technical Lead confirms, by a query over each tenant's newest `acl_versions` row, that no published document is above the threshold. |
| **D7** | **`in_my_region`.**<br/>• `org_units` gains `deleted_at`. Groupsync tombstones a unit instead of deleting it and loses its DELETE grant.<br/>• A new `org_unit_members` table, with FORCE RLS and FKs that carry `tenant_id`, is written by groupsync.<br/>• A fixed column `records.org_unit_id` with FK to `org_units` and index `(tenant_id, org_unit_id)`.<br/>• Only when an `under` leaf survives folding, one recursive query over **live** units expands the caller's units to set S. The leaf is `org_unit_id IS NOT NULL AND org_unit_id = ANY(S)` in SQL and membership in S in Go, so a NULL `org_unit_id` is false on both sides (D5a). A tombstoned unit drops out of S with its subtree, so every grant that needs `under` to be true is removed (fails closed). A policy written with `not under` treats those tickets as outside the region, which is what it says.<br/>• More than 10,000 ids in S gives a typed 500 and an alert. Groupsync reports `groupsync_org_subtree_max` so an oversized tree shows up first. |
| **D8** | **Who sets `records.org_unit_id`** (Q3, decided by the Technical Lead):<br/>• The write pipeline sets it on create, from the requester's primary org unit. It stays NULL if the requester has none, and NULL never matches `under`.<br/>• After that it is a stored field: field-masked on write, audited, never recomputed on read.<br/>• When a user moves to another unit, their existing tickets stay put. Moving tickets is a workflow step. |
| **D9** | **Workflow references** are `(entity type key, policy name)`. They are resolved in an activity against the **current** version, so revocation reaches running workflows. A missing name denies, with `authz_policy_ref_missing_total` and an audit entry. |
| **D10** | **Explainer** adds `Policy` (the first granting policy in declared order) and `Trace` (each named condition reached, with its value), both from the tree `Allows` used. A voided policy shows as `invalid`. `RuleID` stays for one release. |
| **D11** | **One condition language.** This tree is the base expression format. A node registry marks kinds `sql_safe`, and access rules accept only those. A build test fails any `sql_safe` kind that lacks a SQL renderer, a Go renderer or generator coverage. Whether Epic 10's expression language extends this tree is **open** (Q1). |
| **D12** | **Migration.** Stored rows are not rewritten. Format 1 is converted on read: each `Rule` becomes a policy named `Rule.ID` with `when = all[any[subjects], ...Where]`. Publish always stores format 2. Stored versions are immutable, so the format-1 converter stays for as long as any format-1 row exists, including history read by the explainer and audit. |
| **D13** | **Not in this ADR:** conditions on JSON `data` fields (after TICK-900 is green, as one new `field` leaf with the schema-version default); row-dependent masks; deny rules; relationship-based authorization. |

## Context it rests on

- S10's planner already resolves everything about the caller into fixed sets, and keeps every SQL term two-valued (`internal/authz/acl.go:25-34`, `:152`). D5 and D6 keep both properties.
- Index-per-branch lists cost 2.0-4.0 ms per page against 38-166 ms for the OR form (`docs/data/records-list-indexes.md:14-17`), and the support-agent list sorted by last update has 3.8 ms of headroom under its 150 ms p95 budget (`docs/quality/acl-perf-800k.md:16`).
- Groupsync applies a tenant in one transaction and hard-deletes vanished units today (`internal/groupsync/apply.go:507`).

## Consequences

- An admin change to access rules is a row insert (I8). The planner, the two renderings and the agreement suite grow by the new node kinds only (I5).
- Epic 03 gains `org_unit_members`, `org_units.deleted_at` and the subtree metric. The records table gains `org_unit_id`. These are platform migrations, not tenant DDL.
- Tests: the agreement suite (random policies, users and tickets, comparing the SQL filter and the Go check) runs random trees up to depth 4 with NULLs, plus fixed cases at the caps (depth 8, 256 leaves), plus these mutants: `not_unguarded`, `invalid_leaf_false`, `under_direct_only`, `under_ignores_tombstone`, `ref_previous_version`, `fold_wrong`, `shared_nil_caller`. Other tests:
  - order independence;
  - the save-time rejection table at both layers;
  - propagation of a condition change;
  - format-1 equivalence, with the same branch count and SQL as today up to the SP-2 threshold;
  - the 800k benchmark;
  - sync resilience when a unit is deleted;
  - the registry check.
  The full list is RFC-022 §5, CI-1 to CI-11.

## API test scenarios

| Operation | Scenarios |
|---|---|
| `PUT /api/v1/entity-types/{key}/access` | Valid format 2 gives 200, a new version and one audit entry. Valid format 1 gives 200 and is stored as format 2. Each D4 violation, an unknown node kind and a duplicate name give 422 with a pointer and write nothing. A stale expected version gives 409 with the current version and writes nothing. Two PUTs with the same expected version write once. Another tenant's key gets the unknown-key response and writes nothing. |
| `GET /api/v1/entity-types/{key}/access` | 200 with format 2 for a stored format-1 or format-2 version. Another tenant's key gets the unknown-key response. |
| Explainer | Admin gets 200 with `Policy` and `Trace`. A non-admin is refused. An admin of another tenant gets the missing-record response. |

## Risks and residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R1 | `ANY(S)` is over the list budget for large subtrees | Technical Architect (SP-1) | SP-1 above 150 ms p95: add index `(tenant_id, org_unit_id, updated_at DESC, id DESC)`, then a closure table |
| R2 | The caps (D4) or the SP-2 branch threshold block a real policy | Solution Architect | An admin reports one |
| R3 | Group and org-unit changes reach access only at the next sync (10 minutes), as groups do today | Security & Multi-Tenancy Reviewer | A faster-revocation requirement |
| R4 | Conditions are duplicated across entity types | Solution Architect | More than 3 entity types in one tenant carry an identical condition |
| R5 | A workflow references a removed policy and denies at run time | Technical Lead | Workflow definitions store policy references, so publish can refuse to remove one |
| R6 | The CHECK insert cost grows (0.30 ms today) | Postgres & Data Modeling Specialist (SP-2) | Above 5 ms at the caps |
| R7 | A voided policy silently removes access | Technical Lead | `authz_policy_invalid_total` above zero |
| R8 | One user's subtree exceeds 10,000 units, so that user's lists fail | Technical Lead | `groupsync_org_subtree_max` near the cap |
| Q1 | Whether Epic 10's expression language extends this tree is not decided | CTO | Epic 10's expression-language design starts, or any second condition grammar is proposed |
| SP-2 | The `org_unit_members` keys and grants, and the cap and threshold measurements | Postgres & Data Modeling Specialist | Scheduled in the plan, ahead of the stories that depend on it |

## Rejected options

| Option | Why |
|---|---|
| Tenant-wide condition library, versioned on its own | One edit fans out across many documents and versions. It needs a second table, cache and publish path. |
| Deny rules with precedence | Precedence in two renderings is a new way for SQL and Go to disagree. Every deny in the brief can be written as `allow AND NOT x`. |
| Text language (CEL or a custom DSL) | It puts a parser in the trust path and still needs a SQL translation and a "safe subset" checker. It can be admin-UI sugar later. |
| Conditions and policies as JSON object keys | `encoding/json` and `jsonb` keep the last duplicate key without any error. |
| Closure-table join inside the predicate | It makes the row test read a second table and breaks the planner's "caller resolved first" rule. It is the fallback if SP-1 fails. |
| Materialized org path on records, or `ON DELETE SET NULL` | Both rewrite records outside the write pipeline (I4) and without audit. |
| Hard-deleting org units with the new FK | One ticket on a deleted unit would fail every sync for that tenant, and revocations would stop. |

**Dissent:** none.

## Competitive Gate (ADR-008)

**Result:** PASS with accepted risk CP-SF-02
1. **Axis moved:** X2 (change cost): composing or changing an access rule becomes one publish, with zero code and zero DDL. Today OR and NOT need a code change.
2. **Pain points:**
   - Avoids CP-SN-03 and CP-SF-03: stored S10 rules are converted on read, so no rule needs review after the change.
   - Avoids CP-SF-06: a change to a rule cannot widen access outside the tenant (I1, D4 tenant ids), and a broken rule grants nothing (D5b).
   - Risks repeating CP-SF-02, because D4 adds caps. Under the D11 test, items 1 to 4 hold: I5 requires the caps, they are published here, they are the same at every tier, and they are refused at save with a typed 422. Item 5 (sized from a measurement) waits for SP-2. Owner: Postgres & Data Modeling Specialist. Revisit when SP-2 reports.
3. **Measured by:** M-X2 (the upgrade-compatibility suite) and spikes SP-1 and SP-2 (planned).
4. **Parity:** I1 to I9 unchanged. I5 is strengthened with NOT, and I8 is strengthened with composition.
