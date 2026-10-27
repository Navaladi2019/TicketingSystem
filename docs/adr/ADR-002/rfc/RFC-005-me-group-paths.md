---
type: rfc
id: RFC-005
title: Expose group paths in GET /api/v1/me
status: Archived (rev 1, approved; corrections R1-R4 folded into ADR-002)
thread: TICK-266 comment 01a17cba-2c9e-78fa-ba17-44c5b412abf2
adr: docs/adr/ADR-002-me-group-paths.md
date: 2026-10-27
issue: TICK-266
raised-by: TICK-264 (S5 acceptance, step 1)
---

> **Archived.** This is RFC-005 rev 1 as reviewed. The Technical Architect's round-2 corrections R1–R4 were folded into ADR-002 without another round (CTO, comment `01a17cd9`); where they differ, **ADR-002 wins**: strip-and-log moved to `GetMe` (R1), the tracker id is TODO-013 not TODO-010 (R2), the `forbidigo` rule is type-qualified (R3), and the isolation runner asserts the exact membership set (R4).

> **Invariants note.** The project description is empty, so I checked against I1–I9 in `docs/adr/ADR-001-platform-blueprint.md` §1. That section lists nine invariants, not eight. **Numbering:** this is RFC-005, because RFC-004 is the highest on `main`.

## Changelog (rev 0 → rev 1)

| Item | Change |
|---|---|
| S2 (CTO ruling: S2 over O7) | The port takes `principalID` and checks membership in SQL with a join to `group_members`. It no longer takes an id list. |
| O3 + S3 (CTO ruling) | A path without the tenant prefix is now left out and logged at ERROR, with no path in the log line. This replaces the old 500. AC4 is rewritten, and its test captures `slog` output and checks the response body. |
| S1 (CTO sign-off) | The HTTP isolation runner fails on a seeded B-only marker group, and on any `group_paths[].id` that is not one of tenant A's seeded ids. |
| Open question (Security) | Answered: a test now pins the fields of `Principal`; a lint rule limits who may call `GroupPaths`; and a new row, TODO-010, covers the I5 ACL epic. |
| O2 | `apidiff_test.go` fixture literals are updated. |
| O4 | New `GroupPath` component schema; the array items use `$ref`. |
| O8 | One exported tenant-prefix helper, shared by `groupsync` and `pgstore`. |
| O5 | `pattern` is documentation only, because responses are not validated at runtime. The nil→`[]` case is covered by a handler unit test (AC3). |

## Decision

1. **`/me` gains a new field, `group_paths`.**
   - It is an array of `GroupPath {id, path}`, always present and possibly empty.
   - `path` is `groups.path` with `/t/<tenant_id>` stripped. Alice sees `/service-desk`.
   - `groups` (the uuid array) does not change.
2. **The field is tenant-scoped and membership-scoped.**
   - It lists only groups that the calling principal belongs to, in the caller's tenant.
   - Membership is checked in SQL, under row-level security (RLS) on both `groups` and `group_members` (I1).
3. **The tenant prefix is stripped.**
   - Reason: `/t/<tenant_id>/` is how the Keycloak adapter lays out groups (D26), not a product contract. The v1 API is additive-only, so publishing the prefix would make that layout permanent.
   - Reason: the response already carries `tenant_id`.
4. **No amendment to the S5 story.**
5. **Paths are for display only.** They are never an input to authorization. A test and a lint rule enforce this now, and the I5 ACL epic gets an acceptance criterion (TODO-010).

## Contract (`api/openapi.yaml`)

```yaml
Me:
  required: [user_id, tenant_id, groups, idp_alias, group_paths]   # group_paths added
  properties:
    group_paths:
      type: array
      description: >
        The caller's groups in this tenant, with each path relative to the
        tenant root. For display only: authorization keys on `groups` ids,
        never on a path. Paths are not unique across tenants and change when
        a group is renamed.
      items: { $ref: "#/components/schemas/GroupPath" }
GroupPath:
  type: object
  required: [id, path]
  properties:
    id:   { type: string, format: uuid }
    path: { type: string, pattern: "^/" }   # documentation; responses are not validated at runtime
```

- **The change is additive.** The Technical Architect (TA) ran `openapi-compat` against the edit and it exited 0. `apidiff` follows `$ref`.
- **Order.** Items are sorted by `id`, the same order as `groups`. Clients still match entries by `id`.

## Implementation shape (for Delivery)

| Item | Where |
|---|---|
| **Shared helper** `identity.TenantGroupPrefix(tid uuid.UUID) string` returns `"/t/"+tid.String()+"/"`. `groupsync.tenantPrefix` is deleted and its callers use the helper. (O8) | `internal/identity`, `internal/groupsync/apply.go` |
| **Port.** A separate narrow interface, `identity.GroupPathReader { GroupPaths(ctx, tenantID, principalID uuid.UUID) ([]GroupPath, error) }`. It is **not** added to `identity.Store`, so the resolver never sees it. (S2) | `internal/identity` |
| **SQL.** Runs inside `tenancy.InTenantTx(tid)` (see below). | `internal/identity/pgstore` |
| **Strip.** `strings.CutPrefix(path, identity.TenantGroupPrefix(tid))`, then `"/"+rest`. If the prefix is missing, the entry is left out and the code logs ERROR `group path outside tenant prefix` with `tenant_id` and `group_id` only. It never puts the path in an error or a log. (O3, S3) | `pgstore` |
| **Handler.** `GetMe` calls `GroupPathReader.GroupPaths(ctx, p.TenantID, p.PrincipalID)` and returns `[]` for a nil result. A store error is a 500, the same as any other store failure. | `internal/server/api.go` |
| **Lint rule.** A `forbidigo` rule bans `\.GroupPaths\(`, with path exclusions for `internal/server/api.go`, `internal/identity/pgstore/` and `_test.go`. The PR proves the rule fires, as a red/green change in the PR description. | `.golangci.yml` |
| **Contract and regeneration** | `api/openapi.yaml`, `api/api.gen.go` |

The SQL:

```sql
SELECT g.id, g.path
FROM group_members gm
JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
WHERE gm.tenant_id = $1 AND gm.principal_id = $2
ORDER BY g.id
```

**What does not change:** `IdentityResolver`, `Principal` and `principalSQL`. Paths stay off the per-request authentication path. The new query costs one extra short transaction, and only on `/me`.

## Acceptance criteria

1. **Alice sees her group.** Alice is in tenant A and a member of `/t/A/service-desk`. `GET /me` returns `group_paths: [{id, path: "/service-desk"}]`, and `id` is in `groups`.
2. **The HTTP isolation suite catches a leak (S1).**
   - Seed a B-only marker group `/t/<B>/b-only-marker`, with a B member.
   - `apiRunner.run` fails when the body contains `b-only-marker`.
   - It also fails when any `group_paths[].id` is not one of tenant A's seeded group ids.
   - The existing forged-tenant `/me` paths in `registry.go` must pass under the new checks.
   - This changes `internal/tenancy/isolation/http_isolation_test.go` and `internal/tenancy/isolation/testdata/seed.sql`. The CTO approved both changes; see the sign-off section.
3. **No groups means an empty array.** A caller with no groups gets `group_paths: []`, never `null`. This is a handler unit test.
4. **A bad path is left out and logged (replaces rev 0 AC4, includes S3).**
   - Setup: a `groups` row whose path lacks the caller's tenant prefix.
   - Expected: the entry is absent from `group_paths`, `groups` is unchanged, and the server logs one ERROR line with `tenant_id` and `group_id`.
   - Unit test with a fake store: it captures `slog` output and asserts the path text appears in neither the log output nor the response body.
5. **RLS holds even with a wrong predicate (S2, CI-2).**
   - A registered SQL `Path` runs the new statement as `nava_app` inside `InTenantTx(TenantA)`, with `$1 = TenantB` and `$2` set to B's principal.
   - Expected: `Denied` (no rows).
6. **Contract checks pass (O2).**
   - In `internal/platform/apidiff/apidiff_test.go`, both replaced literals become `required: [user_id, tenant_id, groups, idp_alias, group_paths]`.
   - `go test ./internal/platform/apidiff/` passes, and `openapi-compat` passes against `main`.
7. **`Principal` stays narrow (Security answer, item 1).**
   - A test in `internal/identity` pins `Principal`'s field set using `reflect`: `TenantID`, `PrincipalID`, `Groups`, `IdPAlias`.
   - Adding a field fails the test. Its failure message says Security review is required.
8. **The lint rule fires (Security answer, item 2).** A `GroupPaths` call outside the allowed files fails `task lint`.
9. **The ACL epic gets its criterion (Security answer, item 3).**
   - Row TODO-010 is added to `docs/delivery/todo.md`, with a matching `TODO(TODO-010)` comment on `GroupPathReader`.
   - The row: "I5 ACL compiler takes group uuids only. A test asserts it never reads `groups.path` or `identity.GroupPath`. Owner: the I5 ACL epic, as an acceptance criterion. Trigger: the first ACL compiler PR."
10. **Logging.** No log line anywhere contains a group path.

## CTO sign-off (`docs/delivery/review-paths.yaml`)

On 2026-10-27, in TICK-266 comment `01a17cc5`, the CTO approved the S1 changes to `internal/tenancy/isolation/http_isolation_test.go` and `internal/tenancy/isolation/testdata/seed.sql`:
- add a B-only marker group with a B member;
- the runner fails on that marker;
- the runner fails on any `group_paths[].id` that is not one of A's seeded ids.

The implementation PR cites this sign-off.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 (tenant wall) | `InTenantTx` applies RLS on both joined tables. Every predicate and join key includes `tenant_id`. AC2 and AC5 test it. |
| I2 (tenant source) | The tenant and the principal both come from the resolved `Principal`, never from the request. |
| I5 (authorization) | Membership is checked in SQL (S2). Paths are for display only: AC7, AC8 and TODO-010 keep them out of authorization. |
| I3, I4, I6–I9 | Not touched. The change is a read; it adds no role and no grant. |

## Rejected options

| Option | Why rejected |
|---|---|
| `GroupPaths(ctx, tid, ids)` with ids from `p.Groups` (rev 0, TA's O7) | Sound for `GetMe` today. But the port would return the path of any group id in the tenant, so the first caller that passes ids from a request could list every group. The CTO ruled that S2 outranks O7. |
| 500 when the prefix is missing (rev 0) | One bad row would take down `/me` for every member of that group, and the operator would learn no more than from an ERROR log line. |
| A parallel `group_paths: [string]` | The two arrays line up only by position, which the contract does not guarantee. |
| `group_names` (last path segment only) | Ambiguous for nested groups: `/a/leads` and `/b/leads` both become `leads`. |
| Keep the `/t/<tenant>/` prefix | Makes the adapter layout permanent in v1, and repeats `tenant_id`. |
| Paths in `Principal` or `principalSQL` | Every request would pay for the lookup, and group names would end up next to authorization inputs. |
| `GET /api/v1/groups/{id}` | Needs its own ACL design. Too large for S5. |
| Amend the S5 story instead | A human still could not read their memberships without SQL. |

## Residual risk

- A tenant admin could name a subgroup `t/<other-uuid>/…`, so that its stripped path *looks like* another tenant's path. This affects only what that tenant's own members see, and nothing crosses tenants. No action is needed, and Security agreed.
- I add no new accepted-risk row: the paths are the tenant's own text, shown only to that tenant's own members.
