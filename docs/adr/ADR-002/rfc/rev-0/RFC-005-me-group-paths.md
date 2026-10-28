---
type: rfc
id: RFC-005
title: Expose group paths in GET /api/v1/me
status: Draft
date: 2026-10-27
issue: TICK-266
raised-by: TICK-264 (S5 acceptance, step 1)
---

> **Invariants note.** The project description is empty. I checked against I1–I9 in `docs/adr/ADR-001-platform-blueprint.md` §1 (accepted 2026-10-15). There are nine of them, not eight.
>
> **Numbering.** The highest RFC on `main` is RFC-004, so this one is RFC-005.

## Decision

1. **Yes, `/me` gets a new field. It carries the path, with the tenant prefix stripped.**
   - The new field is `group_paths`: an array of `{id, path}` objects, always present and possibly empty.
   - `path` is `groups.path` with `/t/<tenant_id>` removed, so Alice sees `/service-desk`. A nested group reads `/service-desk/tier-2`.
   - `groups` (the uuid array) does not change.
2. **The field is tenant-scoped.** It lists only groups the caller belongs to in the caller's own tenant. It is read under `tenancy.InTenantTx`, so RLS applies (I1).
3. **No story amendment is needed.** S5 step 1 passes as written once this ships.

## Contract (`api/openapi.yaml`, `Me`)

```yaml
Me:
  required: [user_id, tenant_id, groups, idp_alias, group_paths]   # group_paths added
  properties:
    group_paths:
      type: array
      description: >
        The caller's groups in this tenant, with each group's path relative to
        the tenant root. For display only: authorization keys on `groups` ids,
        never on a path.
      items:
        type: object
        required: [id, path]
        properties:
          id:   { type: string, format: uuid }   # equals an entry of `groups`
          path: { type: string, pattern: "^/" }  # e.g. "/service-desk"
```

- **It is additive.** `apidiff` fails on response properties that are removed or no longer always present. A new required response property passes.
- **Order:** `group_paths` is sorted by `id`, the same order as `groups`. Clients still match entries by `id`, never by position.
- **No other text in `api/openapi.yaml` changes.** "Platform group ids, never IdP group names" on `groups` stays true.

## Why strip the prefix

- `/t/<tenant_id>/` is how the Keycloak adapter lays out groups (D26). It is not something the product promises. v1 is additive-only, so publishing the prefix would make the layout permanent. A future IdP adapter (the Zitadel revisit trigger in ADR-001) would then have to fake it.
- The response already carries `tenant_id`. The prefix repeats it and adds nothing a person reading the response needs.
- Stripping is safe. `groups_path_check` guarantees every row starts with `/t/<tenant_id>/`. Org units (`/ou/`) never reach `groups`.

## Implementation shape (for Delivery)

| Item | Where |
|---|---|
| A new read port method, `GroupPaths(ctx, tenantID, ids []uuid.UUID) ([]GroupPath, error)`, which `pgstore` implements | `internal/identity` (port) and `internal/identity/pgstore` |
| The SQL: `SELECT id, path FROM groups WHERE tenant_id = $1 AND id = ANY($2) ORDER BY id`, inside `tenancy.InTenantTx` | `pgstore` |
| Strip the prefix in Go with `strings.CutPrefix(path, "/t/"+tid+"/")` and return `"/"+rest`. A missing prefix is a 500 and the path is not sent. | `pgstore` |
| `GetMe` calls the port with `p.Groups`. An id with no row (sync deleted the group mid-request) is left out. | `internal/server/api.go` |
| Regenerate the API code and add `group_paths` to the contract | `api/openapi.yaml`, `api/api.gen.go` |

**What stays the same:**
- The `IdentityResolver`, `Principal` and `principalSQL` do not change. Paths stay off the per-request authentication path.
- It costs one indexed read, and only on `/me`. The read is on `groups_pkey (tenant_id, id)`.

## Acceptance criteria

1. Alice (tenant A, member of `/t/A/service-desk`) calls `GET /me`. The response contains `group_paths: [{id: <that id>, path: "/service-desk"}]`.
2. **Cross-tenant check:** Bob in tenant B belongs to a group with the same name. His group never appears in Alice's response, and hers never appears in his. Integration test against real Postgres.
3. A caller with no groups gets `group_paths: []`, never `null`.
4. A `groups` row whose path lacks the tenant prefix makes `/me` fail with 500, and the path is not returned. Unit test using a fake store.
5. `apidiff` passes against `main`.
6. No log line contains a group path. Paths are tenant-authored text.

## Invariant check

| Invariant | Effect |
|---|---|
| I1 tenant wall | The read is inside `InTenantTx` under RLS, and the query uses the `tenant_id` key. Acceptance criterion 2 tests it. |
| I2 tenant source | The tenant comes from the principal (through the binding), never from the request. |
| I5 authorization | Paths are display-only. No ACL, script or GraphQL field may use a path as an authorization input. The new description says so. |
| I3, I4, I6–I9 | Not touched. The change is a read: no writes, no new roles, no grants. |

## Rejected options

| Option | Why it was rejected |
|---|---|
| `group_paths: [string]`, parallel to `groups` (the Delivery proposal) | The two arrays line up only by position, and nothing in the contract guarantees that. If a group disappears mid-request, the arrays no longer line up. `{id, path}` costs nothing more. |
| `group_names` (last path segment only) | Nested groups become ambiguous: `/a/leads` and `/b/leads` both read `leads`. |
| Keeping the `/t/<tenant>/` prefix | Makes the adapter layout a permanent part of v1, and repeats `tenant_id`. |
| Putting paths in `Principal` or `principalSQL` | Every authenticated request would pay for a field only `/me` uses. Group names would sit next to authorization inputs, and someone would start using them as one. |
| A new `GET /api/v1/groups/{id}` endpoint | Needs its own ACL design (who may resolve which group). That is larger than S5 needs, and a future epic can add it. |
| Amending the S5 story text instead | Leaves people with no way to read their memberships without SQL. That is the gap TICK-264 found. |

## Residual risk

- A group path is visible only to that group's own members. Group names are tenant-authored text, and a tenant could put sensitive words in one. That is the tenant's own data shown to its own members. I see no new accepted-risk row.

## Open question for critique

- **For the Security & Multi-Tenancy Reviewer:** is "paths are display-only" enough as stated in the contract description? Or do you want a lint or test that keeps `groups.path` out of ACL compilation?
