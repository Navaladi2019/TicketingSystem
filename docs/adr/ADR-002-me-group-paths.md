---
type: adr
id: ADR-002
title: GET /api/v1/me exposes the caller's group paths, tenant-scoped and with the tenant prefix stripped
status: ACCEPTED
date: 2026-10-27
accepted: 2026-10-27 (nava, TICK-266 comment 01a17de9-2f30-7a42-b1a1-957f349c7122)
issue: TICK-266
rfc: docs/adr/ADR-002/rfc/RFC-005-me-group-paths.md (rev 1, archived; thread under TICK-266 comment 01a17cba-2c9e-78fa-ba17-44c5b412abf2)
supersedes: none
source: TICK-264 (S5 acceptance, story step 1: "As Alice, GET /api/v1/me shows group service-desk")
deciders: Solution Architect (author), Technical Architect (APPROVE after R1–R4, round 2), Security & Multi-Tenancy Reviewer (APPROVE, round 1), CTO
signs: nava (signed 2026-10-27: "APPROVED")
---

# ADR-002: `/me` shows the caller's group paths

> **Status: ACCEPTED, 2026-10-27.** nava approved it on TICK-266 (comment `01a17de9`: "APPROVED"), accepting residuals R-a to R-d (§7). Delivery plans the implementation in TICK-270. This ADR comes from RFC-005 rev 1. The Security & Multi-Tenancy Reviewer approved it in round 1, and the Technical Architect approved it in round 2 once R1–R4 were applied. The CTO had those corrections folded in here without a further review round (TICK-266 comment `01a17cd9`). The plain-language explainer is `ADR-002-explained.html`.

## 1. Decision

1. **`GET /api/v1/me` gains `group_paths`.**
   - The field is an array of `GroupPath {id, path}`. It is always present and may be empty.
   - `path` is the group's path relative to the tenant root. Alice sees `/service-desk`, and a nested group reads `/service-desk/tier-2`.
   - `groups` (the uuid array) does not change.
2. **The field is tenant-scoped and membership-scoped.**
   - It lists only the groups the calling principal belongs to, in the caller's own tenant.
   - Membership is checked in SQL, under row-level security on both `groups` and `group_members` (I1).
3. **The tenant prefix `/t/<tenant_id>` is stripped.** It describes how the Keycloak adapter lays out groups (D26). It is not a product promise, and v1 is additive-only, so it stays out of the contract.
4. **Paths are for display only.** No authorization decision, ACL, cache key or metric may key on a path. Paths are not unique across tenants, and they change when a group is renamed. §5 lists how this is enforced: a test, a lint rule, and an acceptance criterion for the ACL epic.
5. **The S5 story is not amended.** Story step 1 passes as written once this ships.

This decision changes no invariant (I1–I9, ADR-001 §1). It adds one v1 contract field and one read port, so it is a decision of its own rather than an amendment to ADR-001.

## 2. Contract (`api/openapi.yaml`)

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

- **Additive.** `openapi-compat` exits 0 on this edit, and `apidiff` follows `$ref`. The evidence is in `ADR-002/spikes/technical-architect/apidiff-openapi-compat.md`.
- **Order.** Entries are sorted by `id`, the same order as `groups`. Clients match entries by `id`, never by position.

## 3. Design

| Part | Decision | Where |
|---|---|---|
| Prefix helper | `identity.TenantGroupPrefix(tid uuid.UUID) string` returns `"/t/"+tid.String()+"/"`. It replaces `groupsync.tenantPrefix`, and both callers use it. | `internal/identity`, `internal/groupsync/apply.go` |
| Port | `identity.GroupPathReader { GroupPaths(ctx, tenantID, principalID uuid.UUID) ([]GroupPath, error) }`. It returns the **raw** `groups.path`. It is a separate interface, not part of `identity.Store`, so the resolver never sees it. It carries the `TODO(TODO-013)` comment. | `internal/identity`, implemented in `internal/identity/pgstore` |
| SQL | The statement below, run inside `tenancy.InTenantTx(tid)` | `pgstore` |
| Strip and log | `GetMe` strips the prefix: `strings.CutPrefix(path, identity.TenantGroupPrefix(p.TenantID))`, then `"/"+rest`. If the prefix is missing, the entry is dropped and the handler logs `h.log.ErrorContext(ctx, "group path outside tenant prefix", slog.String("tenant_id", …), slog.String("group_id", …))`. The path never goes into a log or an error. A nil result becomes `[]`. A store error is a 500, as for any other store failure. | `internal/server/api.go` |
| Lint rule | Type-qualified `forbidigo` rules, plus a per-file exclusion for `api.go` (YAML below) | `.golangci.yml` |
| What stays the same | `IdentityResolver`, `Principal` and `principalSQL`. Paths stay off the per-request authentication path. The cost is one short extra transaction, on `/me` only. | n/a |

The SQL:

```sql
SELECT g.id, g.path
FROM group_members gm
JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
WHERE gm.tenant_id = $1 AND gm.principal_id = $2
ORDER BY g.id
```

The lint rules go under the existing `forbidigo.forbid` list, which already sets `analyze-types: true`:

```yaml
        # ADR-002: group paths are display-only; only GetMe reads them.
        - pattern: ^identity\.GroupPathReader\.GroupPaths$
          pkg: ^github\.com/Navaladi2019/TicketingSystem/internal/identity$
          msg: group paths are display-only (ADR-002); only GetMe may call GroupPaths
        - pattern: ^pgstore\.Store\.GroupPaths$
          pkg: ^github\.com/Navaladi2019/TicketingSystem/internal/identity/pgstore$
          msg: group paths are display-only (ADR-002); only GetMe may call GroupPaths
```

and under `linters.exclusions.rules`:

```yaml
      - path: ^internal/server/api\.go$
        linters: [forbidigo]
```

The exclusion is per file, so any handler in `api.go` could call `GroupPaths`, not only `GetMe`. That is accepted as residual **R-b** (§7). The spike in `ADR-002/spikes/technical-architect/forbidigo-scope.md` shows that the rev-1 pattern `\.GroupPaths\(` matches nothing, and that `\.GroupPaths$` also wrongly flags the generated response field `api.Me.GroupPaths`.

## 4. Acceptance criteria (one implementation task for Delivery)

1. **Story step.** Alice is in tenant A and a member of `/t/A/service-desk`. `GET /me` returns `group_paths: [{id, path: "/service-desk"}]`, and that `id` is in `groups`.
2. **HTTP isolation suite.**
   - **Seed** (`internal/tenancy/isolation/testdata/seed.sql`):
     - a B-only marker group `/t/<B>/b-only-marker`, with a B member;
     - a second tenant-A group that `a1` is **not** a member of.
   - **Runner** (`apiRunner.run`, `internal/tenancy/isolation/http_isolation_test.go`). It fails when:
     - the body contains `b-only-marker`; or
     - the set of `group_paths[].id` is not exactly the caller's membership set (`{…4a1}` for `a1`). That catches a leak from tenant B and a regression in membership scoping inside tenant A.
   - The existing forged-tenant `/me` paths in `registry.go` must pass under these checks.
3. **No groups.** A caller with no groups gets `group_paths: []`, never `null`. Handler unit test.
4. **A path without the prefix is dropped and logged.** The handler unit test uses a fake `GroupPathReader` that returns a path without the caller's prefix. Expected:
   - the entry is absent from `group_paths`, and `groups` is unchanged;
   - exactly one ERROR line is logged, with `tenant_id` and `group_id`;
   - the captured `slog` output and the response body contain no path text.
5. **Row-level security holds under a wrong predicate (CI-2).** A registered SQL `Path` runs the statement as `nava_app` inside `InTenantTx(TenantA)`, with `$1 = TenantB` and `$2` set to B's principal. Expected: `Denied` (0 rows).
6. **Contract.**
   - Both replaced literals in `internal/platform/apidiff/apidiff_test.go` become `required: [user_id, tenant_id, groups, idp_alias, group_paths]`.
   - `go test ./internal/platform/apidiff/` passes, and `openapi-compat` passes against `main`.
7. **`Principal` stays narrow.**
   - A `reflect` test in `internal/identity` pins the field set to `TenantID`, `PrincipalID`, `Groups` and `IdPAlias`.
   - Adding a field fails the test, and the failure message says Security review is required.
8. **The lint rule fires.** The PR shows it red then green: a `GroupPaths` call in a disallowed file makes `task lint` fail, and removing the call makes it pass.
9. **ACL epic criterion.**
   - Add tracker row **TODO-013** to `docs/delivery/todo.md`, with a matching `TODO(TODO-013)` comment on `GroupPathReader`. The row reads: "The I5 ACL compiler takes group uuids only; a test asserts it never reads `groups.path` or `identity.GroupPath`. Owner: the I5 ACL epic, as an acceptance criterion. Trigger: the first ACL compiler PR."
   - Before merging, re-check the highest TODO number across open PRs. TODO-010 to TODO-012 are already in use on other branches.
10. **No path in logs.** No log line anywhere contains a group path.

## 5. How "display only" is enforced

| When | Control | Catches |
|---|---|---|
| Now | AC7: the `Principal` field-set test | A path or name added to the authorization context |
| Now | AC8: the `forbidigo` rules | A second caller of `GroupPaths` outside `api.go` |
| Now | The narrow `GroupPathReader` interface, kept off `identity.Store` | The resolver or the middleware reaching paths |
| Deferred (TODO-013) | ACL compiler test: group uuids only | An ACL keyed on a path. No ACL compiler exists yet, so a test today would prove nothing. |

## 6. CTO sign-off on protected paths (`docs/delivery/review-paths.yaml`)

The CTO approved the changes to `internal/tenancy/isolation/http_isolation_test.go` and `internal/tenancy/isolation/testdata/seed.sql` in two comments on 2026-10-27:
- TICK-266 comment `01a17cc5`: the S1 changes. These are the B-only marker group with a B member, a runner failure on the marker, and a runner failure on any foreign group id.
- TICK-266 comment `01a17cd9`: R4. That adds a second A group that `a1` is not in, and the exact-membership-set assertion.

The implementation PR cites both comments.

## 7. What is being accepted (residual risks)

| # | Residual | Why it is accepted |
|---|---|---|
| R-a | A tenant admin can name a subgroup `t/<other-uuid>/…`, so its stripped path *looks like* another tenant's. | It affects only what that tenant's own members see. Nothing crosses tenants. Security agreed: no action. |
| R-b | Any handler in `internal/server/api.go` may call `GroupPaths`, not only `GetMe`. | `forbidigo` exclusions are per file. Limiting the rule to one function needs a custom analyzer, which isn't worth it for one read. Review of `api.go` is the control. |
| R-c | `pattern: "^/"` and "never `null`" are not enforced at runtime, because responses are not validated. | AC3 and AC4 cover the cases that matter as unit tests. |
| R-d | Membership is read in a second transaction after authentication, so a sync can land in between and make `group_paths` differ from `groups`. | Both lists come from the database, and the next request is consistent. Display data is allowed to lag by one request. |

No new accepted-risk row. Group paths are the tenant's own text, shown only to that tenant's own members.

## 8. Options considered and rejected

| Option | Why rejected |
|---|---|
| Amend the S5 story instead of the API | A human still could not read their memberships without SQL, which is the gap TICK-264 found. |
| `group_paths: [string]` parallel to `groups` (Delivery's proposal) | The two arrays line up only by position, and the contract guarantees no order. They also drift if a group is deleted between the reads. |
| `group_names` (last path segment only) | Ambiguous for nested groups: `/a/leads` and `/b/leads` both read `leads`. |
| Keep the `/t/<tenant>/` prefix | Locks the adapter's layout into an additive-only v1 contract, and repeats `tenant_id`. |
| Put paths in `Principal` or `principalSQL` | Every request would pay for a field only `/me` uses, and group names would sit next to authorization inputs. |
| `GroupPaths(ctx, tid, ids)` fed from `p.Groups` (RFC rev 0) | The port would return the path of any group id in the tenant. The first caller passing ids from a request would let a member list every group. Isolation finding S2, ruled over O7. |
| A 500 when the prefix is missing (RFC rev 0) | One bad row would take down `/me` for every member of that group, and the operator learns no more than from an ERROR line. |
| Strip and log in `pgstore` (RFC rev 1) | `pgstore.Store` has no logger, and a fake-store handler test would never run that code (R1). |
| `forbidigo` pattern `\.GroupPaths\(` (RFC rev 1) | It matches nothing. forbidigo matches the selector text, which never contains `(` (R3, spike). |
| `GET /api/v1/groups/{id}` | Needs its own ACL design. Larger than S5 needs; a future epic can add it. |

## 9. Evidence

- `ADR-002/spikes/technical-architect/apidiff-openapi-compat.md` (round 1):
  - `openapi-compat` exits 0 on the edit;
  - `apidiff_test.go` goes red until the fixture literals are updated (AC6);
  - with oapi-codegen v2.8.0, the inline item schema generates an anonymous struct, which is why `GroupPath` is a component.
- `ADR-002/spikes/technical-architect/forbidigo-scope.md` (round 2), run with golangci-lint 2.14.0:
  - the rev-1 pattern reports `0 issues`;
  - a name-only pattern also flags the response field;
  - the type-qualified rules catch all 3 disallowed calls and flag nothing else.
- The archived RFC: `ADR-002/rfc/rev-0/` (Draft), `ADR-002/rfc/rev-1/` (Revised), and `ADR-002/rfc/RFC-005-me-group-paths.md` (as approved).

## 10. Invariant check

| Invariant | Effect |
|---|---|
| I1 (tenant wall) | `InTenantTx` applies row-level security on both joined tables, and every predicate and join key includes `tenant_id`. AC2 and AC5 test it. |
| I2 (tenant source) | The tenant and the principal come from the resolved `Principal`, never from the request. |
| I5 (one authorization layer) | Membership is checked in SQL, and paths are kept out of authorization (§5). |
| I3, I4, I6–I9 | Not touched. The change is a read; it adds no role, grant, write or workflow. |
