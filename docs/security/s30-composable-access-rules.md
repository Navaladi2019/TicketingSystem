# S30 composable access rules: isolation review and attack tests (P11-11)

TICK-943, story TICK-935 (S30), ADR-010 I1, D4, D5b, D10. The attacker holds a valid token for one tenant, usually its admin token (the access routes are admin only), and tries to make a format-2 access document reach another tenant, or keep access after a mistake voids a policy. Every attack below is a runnable test; the passing ones stay as regression guards.

**Verdict: no exploitable isolation gap.** Three findings: F1 was recorded as AR-S30-01 and is fixed by TICK-947 (S31), before `under` is switched on; F2 is fixed in this change; F3 is a 500 on a publish body, which fails closed and is left to the engineers.

## Paths to data

| Path | Tenant comes from | Lowest-layer guard |
| --- | --- | --- |
| `PUT /entity-types/{key}/access` | the token (`identity.FromContext`), `InTenantTx` | `authz.Prepare` and `NotInTenant` (route); `acl_versions_rules_check` CHECK, `acl_versions_tenant_ids` trigger, RLS, the composite FK (database) |
| `GET /entity-types/{key}/access` | the token | `definitionSQL` filters `tenant_id`; RLS |
| `GET /access/explain` | the token | `pipeline.Get` and `PrincipalOf` filter `tenant_id`; RLS |
| Record list, read, PATCH | the token | the planner voids invalid policies; `CompileRead` adds the predicate; RLS |
| `PublishIfNone` (onboarding, backfill) | the transaction's tenant | `Prepare`, `NotInTenant`, then the same CHECK and trigger |
| Direct `nava_app` INSERT | `app.tenant_id` | CHECK, trigger, RLS `WITH CHECK`, composite FK |

```mermaid
sequenceDiagram
    participant A as Tenant A admin (valid token)
    participant R as PUT /access route
    participant DB as acl_versions (CHECK, trigger, RLS)
    participant P as Planner (read path)
    A->>R: document naming tenant B's group, or ref to B's condition name
    R-->>A: 422 foreign_id / unresolved_ref (same bytes as an unknown id or name)
    A->>DB: direct INSERT (any route bypass)
    DB-->>A: 23514 foreign_id (trigger), 23514 CHECK, or 42501 RLS
    Note over DB: under {of: [B's org unit]} is refused by the tenant-id trigger since TICK-947 (F1)
    DB->>P: a stored voided policy (legacy row, or under)
    P-->>A: policy voided: no row on list, read, PATCH (404); explainer lists it in invalid
```

## Attack table

| # | Attack | Result | Test |
| --- | --- | --- | --- |
| 1 | Tenant A's document names tenant B's group, user or (in a `cmp`) principal | Route: 422 `foreign_id`, same bytes as an id that does not exist. `PublishIfNone`: `InvalidPolicyError` `foreign_id`. Direct insert: 23514 from the trigger | `TestPutAccessFormat2ForeignIDsAreForeignID`, `TestPublishIfNoneRefusesWhatTheRouteRefuses` (TICK-939); `TestTICK940ForeignIDs` (TICK-940) |
| 2 | A `ref` to a name defined only in tenant B's document, or only in an older version of A's, directly or through a condition | 422 `unresolved_ref`, same bytes as a name nobody defined | `internal/server/access_s30_isolation_integration_test.go` `TestS30RefNeverResolvesAcrossTenantsOrVersions` |
| 3 | A voided policy (stored before TICK-939; the route and the CHECK refuse every voidable shape, asserted; see F3 for the status code) | Nothing on list, single read, PATCH with current, stale or no `If-Match` (the unknown-record 404, never 412/428); the explainer lists it in `invalid` with code and `/policies/<i>`, and decides from the valid policy only; a voided-only document denies everyone | `TestS30VoidedPolicyGrantsNothingAnywhere` |
| 3b | Format-2 deny trace (TICK-941 carry-over) | `trace` has the false step of the named condition | `TestS30Format2DenyTraceHasAFalseStep` |
| 4 | Tenant A's explainer pointed at tenant B's record, user, admin, group, or ids of the wrong kind, with format-2 documents in both tenants | The unknown-record 404, byte for byte; A's explanation never carries B's policy or condition names | `TestS30ExplainerStaysInItsTenantUnderFormat2` |
| 5 | Tenant B's admin `GET`/`PUT` on a key only tenant A has, with no, `*`, A's current or a stale `If-Match` | The unknown-key 404, byte for byte, no `ETag` (a 412 would leak A's version); nothing written in either tenant; a non-admin gets the same 403 for both keys. Timing: one `(tenant_id, key)` lookup that finds nothing on both paths | `TestS30AnotherTenantsKeyIsTheUnknownKeyForGetAndPut` |
| 6 | A format-2 document changes how masks behave | The same masks give the same record bodies and explainer `masked_fields` under format 1 and format 2; masks inside a format-2 `rules` value are 422 `unknown_key` | `TestS30MasksBehaveTheSameUnderEitherFormat` |
| 7 | `under` (not available in S30) at the route, in both forms, with tenant B's org unit | 422 `not_available`, the id not echoed. Past the route: 23514 `foreign_id` (TICK-947); an own-tenant unit stored past the route is voided and grants nothing (F1) | `TestS30UnderIsRefusedAtTheRouteAndGrantsNothingWhenStored` |
| 8 | `legacyACLInsert` (TICK-940 test helper that turns the tenant-id trigger off) reachable outside tests | No package of the module imports `internal/tenancy/isolation`, under any build tag | `internal/tenancy/isolation/reach_isolation_test.go` |
| 9 | The trigger stays off after the helper | On again after fn succeeds, fails, panics, calls `runtime.Goexit` (`t.Fatal`), or cancels the context; a failed ENABLE is reported (F2) | `internal/tenancy/isolation/legacy_acl_insert_isolation_test.go` |
| 10 | A runtime role turns the trigger off or skips it | `ALTER TABLE ... DISABLE TRIGGER` and `SET session_replication_role = replica` are 42501 for every runtime role | `TestRuntimeRolesCannotSkipTheTenantIDTrigger` |

The `not` lint (TICK-937 finding 2) warns on a `not` over a caller-dependent leaf directly, through refs and through chains of refs (`TestPrepareWarnsOnNotOverCallerLeaves`, `TestPutAccessWarnsOnNotOverCallerLeaf`). It is intra-tenant only: RLS still holds the rows to the caller's tenant.

## Findings

### F1: `under` is refused at the route only

- **Attack:** a holder of `nava_app` database access (not an API caller: no route writes it) inserts tenant A's document with `{"under": {"of": ["<tenant B's org unit>"]}}`. The CHECK accepts it (the shared fixture marks `not_available` as `route_only`) and the trigger checks no org-unit id.
- **Status:** fixed by TICK-947 (S31 P11-14, `20261121000100_acl_tenant_ids_org_units.sql`): the tenant-id trigger refuses every `under.of` id that is not a live org unit of `NEW.tenant_id` with `foreign_id`, and the migration refuses to apply while any stored row names one. AR-S30-01 is deleted. Was: will bite us at scale. The planner voided `under`, so the row granted nothing on any read path, but when S31 marked `under` available every such stored row would have become live, with an org-unit id nothing had checked.
- **Severity:** Medium.
- **Fix (lowest layer):** before `under` becomes available, the trigger checks every `under.of` id against `org_units` of `NEW.tenant_id`, and a one-off query proves no stored row holds a foreign or missing org-unit id. Until then the CHECK can refuse `under` outright. Owner: S31 (TICK-947 / TICK-950). Recorded as AR-S30-01.
- **Test:** `internal/server/access_s30_isolation_integration_test.go` `TestS30UnderIsRefusedAtTheRouteAndGrantsNothingWhenStored` (its middle step now asserts the refusal) and `internal/db/dbtest/org_units_test.go` `TestTICK947UnderForeignIDs` (superuser and `nava_app`; foreign, tombstoned and missing units).

The Technical Lead's hand-off says `under` is refused "through a direct insert and in the CHECK". It is not; this finding corrects that.

### F2: `legacyACLInsert` dropped a failed re-enable

- **Attack:** none from outside; test integrity. When fn returned an error and the ENABLE also failed (for example the superuser connection was lost), the helper returned fn's error only. The suite ran on with the trigger off, so later paths that expect the trigger's 23514 could pass for the wrong reason.
- **Status:** fixed in this change (`errors.Join` of both errors).
- **Severity:** Low (test only, throwaway cluster).
- **Test:** `TestLegacyACLInsertReportsAFailedReenable`.

### F3: a converted format-1 rule over the child cap is a 500, not a 422

- **Attack:** tenant A's admin sends `PUT /api/v1/entity-types/job/access` with a format-1 rule that has 16 `where` conditions. `ConvertFormat1` makes it `all[any[subject], 16 cmps]`, 17 children. `prepareFormat1` checks the caps through `finish` but not the per-node child cap, so `Prepare` accepts it; `publishRaw`'s `Decode` then refuses it inside the transaction and the route answers 500 `internal_error`. By the same code path, `PublishIfNone` returns a plain error instead of an `InvalidPolicyError` (read from the code, not tested).
- **Status:** fixed by TICK-996 (`prepareFormat1` runs the format-2 grammar over the converted document; 422 `child_count` at `/rules/<i>`). Was: not an isolation gap. It failed closed: the transaction rolls back, no version and no audit entry are written, and the CHECK would refuse the row anyway.
- **Severity:** Low (a wrong status on admin input, and an ERROR log line per attempt).
- **Fix:** in `prepareFormat1`, refuse a rule whose converted `all` has more than `MaxChildren` children (`len(where) + 1`), at `/rules/<i>/where` with `child_count`. Better still, run the format-2 grammar check over the converted document, so no other format-1 shape can slip through the same way.
- **Test:** `TestS30VoidedPolicyGrantsNothingAnywhere` asserts the fail-closed half (not 2xx, nothing written). Tighten it to 422 `child_count` together with the fix.

### Recorded, not blocking

- The helper's `DISABLE` is committed, because fn writes on another connection that an open `ALTER TABLE` would block. While fn runs, the trigger is off for the whole cluster. The isolation suite runs one test at a time on its own throwaway cluster, so nothing else writes then. A killed test binary (the `go test -timeout` panic runs no defers) leaves it off in that throwaway cluster only. `i5InWorld` shows the safer shape: disable inside one transaction that always rolls back.
- When S31 enables `under {of: "caller"}`, the `not` lint does not count `under` as caller-dependent (`callerLeaf`). Add it with TICK-947.
