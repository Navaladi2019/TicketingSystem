---
title: "9. ACL and field masks on the read side"
section: security
status: delivered
sources: [docs/security/s10-acl-attacks.md, docs/security/s30-composable-access-rules.md, internal/authz, internal/tenancy/isolation/acl_attack_paths.go, docs/security/accepted-risks.md]
evidence: [docs/security/s10-acl-attacks.md, docs/security/s30-composable-access-rules.md]
adrs: [ADR-001, ADR-010, ADR-010-A1]
invariants: [I5, I1]
stories: [S10, S30]
tables: [records, record_shares, acl_versions]
packages: [internal/authz, internal/tenancy/isolation]
tests: [TestRegisteredPaths, TestShapeDropsMaskedFieldsAndNeverMutates, TestCheckMaskFields, TestEveryMaskOnAFieldMustAdmit, TestRowDependentMaskSubjectsNeverMatch, TestExplainNeverListsAFieldTheAdminCannotRead, TestExplainMatchesTheEnforcementOnState, TestExplainAccessNamesTheRuleAndHidesWhatTheAdminCannotRead, TestRequireTenantAdminTxNeverReadsGroupsOrPaths, TestMaskedFieldInFilterIsRefusedBeforeSQL, TestMaskedFieldInSortIsRefusedBeforeSQL, TestMaskedFieldInDefaultSortIsRefusedBeforeSQL, TestListRecordsMaskedSortKeyInCursor, TestListRecordsItemsAreShaped, TestHistoryOfAMaskedOnlyChange, TestPatchIsNoOracleForAMaskedField, TestUnreadableRecordIsTheMissingRecord, TestListRecordsHidesUnreadableFromPageAndTotal, TestUniqueViolationDoesNotIdentifyAnUnreadableRecord, TestNotFoundBodyIsIdenticalAndHasNoInstance, TestListRecordsCrossTenant, TestListRecordsFilterOnAnotherTenantsGroup, TestAuditEntriesMaskFieldsHiddenFromTheReader, TestPutAccessFormat2ForeignIDsAreForeignID, TestTICK940ForeignIDs, TestS30RefNeverResolvesAcrossTenantsOrVersions, TestS30VoidedPolicyGrantsNothingAnywhere, TestS30Format2DenyTraceHasAFalseStep, TestS30ExplainerStaysInItsTenantUnderFormat2, TestS30AnotherTenantsKeyIsTheUnknownKeyForGetAndPut, TestS30MasksBehaveTheSameUnderEitherFormat, TestS30UnderIsRefusedAtTheRouteAndGrantsNothingWhenStored, TestRuntimeRolesCannotSkipTheTenantIDTrigger, TestTICK940WorkerCannotPublish]
---

## Boundary

A user of the right tenant whom the record ACL does not admit, or who may not
read a masked field, learns nothing about that record or field through any
read surface: get, list (filter, sort, total, cursor), history, the access
explainer, or an error response. A record they cannot see answers exactly like
a record that does not exist.

## Enforcement point

`internal/authz` is the one authorization layer (ADR-001 I5), evaluated inside
the query and in the caller's tenant transaction:

- **Rows.** `CompileRead` turns the tenant's published policy into the
  `WHERE` of the record query, with the share `EXISTS` matched on `tenant_id`
  and the `(kind, id)` pair. `canRead` maps a hidden record to the
  missing-record 404, before `If-Match`, so there is no 403 or 412 oracle.
- **Fields.** `authz.Shape` drops masked fields from every response body,
  including history `before` and `after` (`changed` is computed after
  shaping).
- **Queries.** `checkPlan` (`authz.CheckQueryFields`) refuses a filter or sort
  on a masked field with a 400 **before any SQL runs**, so neither the result,
  the total nor the timing depends on the hidden value.
- **Explainer.** Requires the tenant admin first, runs in the caller's tenant,
  and lists only masked fields the admin can read.
- **Subjects.** `authz.NotInTenant` refuses another tenant's group or user in a
  policy or mask, with the same 422 as an unknown id (`foreign_id` in a
  format-2 document; the id is not echoed). Since S30 the database repeats
  it: the `acl_versions_tenant_ids` trigger (`SECURITY INVOKER`) refuses a
  row naming another tenant's id, and no runtime role can turn it off.
- **Composed rules (S30).** A `ref` resolves only inside the same document
  and version, never across tenants. A policy the planner cannot use is
  **voided**: it grants nothing on any read surface and the explainer lists
  it under `invalid`. Only `nava_app` may insert an `acl_versions` row
  (ADR-010-A1 A1-3).

How the ACL is compiled and how masks apply are on
[Core: ACL](../system/core-acl.md) and [Core: field masks](../system/core-masks.md).

```mermaid
sequenceDiagram
    participant D as Desk agent (tenant A, not admitted to cost)
    participant API as nava-core
    participant Z as authz
    participant PG as PostgreSQL (nava_app, RLS)
    D->>API: GET /api/v1/records/incident?filter[cost]=600
    API->>Z: checkPlan(filter, sort)
    Z-->>API: cost is masked for this caller
    API-->>D: 400, no page, no total, no SQL run
    D->>API: GET /api/v1/records/incident/{admin-only record}
    API->>PG: CompileRead WHERE (policy, tenant A)
    PG-->>API: no row
    API-->>D: 404, byte-identical to a random id
```

## Proving tests

The S10 attack rows (P3-37) are registered isolation paths under CI-8, run by
`TestRegisteredPaths` through the real handler as `nava_app` via PgBouncer;
each row and its red mutation is in
[s10-acl-attacks.md](../../../security/s10-acl-attacks.md#rows).

| Surface | Test |
|---|---|
| Filter, sort, default sort on a masked field: 400 before SQL | `TestMaskedFieldInFilterIsRefusedBeforeSQL`, `TestMaskedFieldInSortIsRefusedBeforeSQL`, `TestMaskedFieldInDefaultSortIsRefusedBeforeSQL`, `TestCheckMaskFields` |
| A masked sort key never enters a cursor | `TestListRecordsMaskedSortKeyInCursor` |
| List and get bodies are shaped | `TestListRecordsItemsAreShaped`, `TestShapeDropsMaskedFieldsAndNeverMutates` |
| History of a change to a masked field only | `TestHistoryOfAMaskedOnlyChange` |
| PATCH is no oracle for a masked value | `TestPatchIsNoOracleForAMaskedField` |
| GET, history and PATCH (no, `*`, stale and current `If-Match`) on a record the ACL denies answer exactly like a missing record, and the refused PATCH writes nothing | `TestUnreadableRecordIsTheMissingRecord` (the admin's GET of the same record is the control) |
| An unreadable match is in neither the page nor `total` | `TestListRecordsHidesUnreadableFromPageAndTotal` |
| A unique-violation 409 caused by an unreadable record names no id or number of it and carries no `Location` or `ETag` | `TestUniqueViolationDoesNotIdentifyAnUnreadableRecord` |
| The not-found body itself is fixed and has no `instance` (supporting, not the proof above) | `TestNotFoundBodyIsIdenticalAndHasNoInstance` |
| Another tenant's records and groups in a list | `TestListRecordsCrossTenant`, `TestListRecordsFilterOnAnotherTenantsGroup` |
| Explainer lists only what the admin may read, and matches enforcement | `TestExplainNeverListsAFieldTheAdminCannotRead`, `TestExplainMatchesTheEnforcementOnState`, `TestExplainAccessNamesTheRuleAndHidesWhatTheAdminCannotRead`, `TestRequireTenantAdminTxNeverReadsGroupsOrPaths` |
| Every mask on a field must admit; row-dependent mask subjects never match | `TestEveryMaskOnAFieldMustAdmit`, `TestRowDependentMaskSubjectsNeverMatch` |
| Masks on audit entries | `TestAuditEntriesMaskFieldsHiddenFromTheReader` |

### S30: composed rules (TICK-943)

The S30 rows are in
[s30-composable-access-rules.md](../../../security/s30-composable-access-rules.md#attack-table).

| Attack | Result | Test |
|---|---|---|
| Tenant A's document names tenant B's group, user or principal | 422 `foreign_id`, same bytes as an unknown id; a direct insert gets 23514 from the trigger | `TestPutAccessFormat2ForeignIDsAreForeignID`, `TestTICK940ForeignIDs` |
| A `ref` to a name defined only in tenant B's document or an older version | 422 `unresolved_ref`, same bytes as a name nobody defined | `TestS30RefNeverResolvesAcrossTenantsOrVersions` |
| A voided policy stored past the checks | Grants nothing on list, read or PATCH (the missing-record 404, never 412 or 428); listed in the explainer's `invalid` | `TestS30VoidedPolicyGrantsNothingAnywhere`, `TestS30Format2DenyTraceHasAFalseStep` |
| Tenant A's explainer pointed at tenant B's record, user or group | The missing-record 404; no tenant B policy or condition name | `TestS30ExplainerStaysInItsTenantUnderFormat2` |
| Tenant B's admin `GET`/`PUT` on a key only tenant A has | The unknown-key 404, no `ETag` | `TestS30AnotherTenantsKeyIsTheUnknownKeyForGetAndPut` |
| Masks under format 2 | Same bodies and `masked_fields` as format 1 | `TestS30MasksBehaveTheSameUnderEitherFormat` |
| `under` with tenant B's org unit | 422 `not_available` at the route; voided if stored (accepted risk `AR-S30-01` until S31) | `TestS30UnderIsRefusedAtTheRouteAndGrantsNothingWhenStored` |
| A runtime role skips the tenant-id trigger, or `nava_worker` publishes | 42501 for every runtime role; the worker's insert fails | `TestRuntimeRolesCannotSkipTheTenantIDTrigger`, `TestTICK940WorkerCannotPublish` |

## Fixture that turns it red

Each S10 row was shown red by a named mutation: `checkPlan` returning nil
gives 200 on a masked filter; `canRead` answering 403 fails the oracle check;
no shaping puts `cost` into history `after`; the explainer's `Readable` filter
dropped names a field the admin cannot read. Row 7 (share forgery) needs both
the tenant predicate dropped **and** RLS off on `record_shares` to go red,
which shows the two layers are independent.

## Accepted risks

Errors and logs are part of the read side: AR-006 (a recovered panic value is
logged as is) and AR-010 (`error.type` export shape) apply here.

S30 adds `AR-S30-01` (`under` is refused at the route only, until S31), below,
and `AR-TICK-971-01` (a crafted document costs up to about 90 ms per publish
at the default byte limit, about 340 ms at the 128 KiB ceiling), on the
[ADR-010-A1 page](../adr/adr-010-a1.md#residuals) and in
[accepted-risks.md](../../../security/accepted-risks.md).

<!-- gen:accepted-risks ids=AR-006,AR-010,AR-TICK-653-01,AR-TICK-653-02,AR-S30-01 -->

## Related

- [Opaque tokens](06-opaque-tokens.md), for cursors.
- [ACL tables](../database/acl.md).
