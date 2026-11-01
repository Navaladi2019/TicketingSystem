# Acceptance kit: S8, a tenant admin publishes an incident type and reads every version

Story TICK-333, kit TICK-346. Story branch `TICK-333/entity-types-versioned`.
Guide page: `docs/acceptance/guide/s8.html`. Bruno folder: `docs/acceptance/guide/bruno/S8/` ("S8 entity types").

## What you can now do

- Every tenant starts with three ticket types, `incident`, `request` and `change`, at version 1.
- A tenant admin changes the `incident` type and publishes it as version 2. Every version reads back exactly as it was published.
- A change that would break existing tickets (a field changing type) is refused, and nothing is stored.
- Other users of the tenant can read the types but not publish them. Another tenant sees only its own types.

## Try it (about 10 minutes on a running stack)

The prerequisites are as for the other stories (`docs/acceptance/guide/prerequisites.html`): stack up, Node.js for the headless run, Bruno to click through.

```sh
sh docs/acceptance/guide/tools/seed.sh        # again, if you seeded before this story
sh docs/acceptance/guide/tools/tunnels.sh up
sh docs/acceptance/guide/tools/api-test.sh S8 # or click S8.1-S8.19 in Bruno
sh deploy/compose/test/stack-test.sh s8_entity_types   # all of it, plus the AC5 database check
```

How the story's users map to the seeded ones (dev-only credentials, in `bruno/environments/local.bru`):

| Story user | Seeded user | Tenant |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test`, in tenant A's `admins` group (`tools/seed.sh` creates it) | A |
| `agent@tenant-a` | `alice@tenant-a.test` (Alice), not an admin | A |
| `agent@tenant-b` | `bob@tenant-b.test` (Bob) | B |

What you should see: every S8 check green. On a fresh stack, S8.11 publishes `version: 2`. The folder runs again on the same stack, and each later run publishes the next version.

## Acceptance criteria

"Byte-identical" is checked in two ways. The `schema` bytes in the response are compared with the bytes published, read raw from the response text and not re-serialised. `schema_sha256` is compared with the SHA-256 of those bytes. Version 1's "published" bytes are the `schema` member of `internal/metadata/templates/incident.json` as it appears in the file. The publish request writes its schema with a 3-space indent, so a store that re-formats JSON would fail the check.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| AC1 | `admin@tenant-a` publishes a new `incident` schema and gets `201` with `version: 2`. Version 1 and version 2 each read back byte for byte as published. | see the run below | S8.11 (201, `version: 2 (fresh stack)`, response schema equals the published bytes, `schema_sha256` = SHA-256 of them). S8.12: version 2 read back. S8.6 and S8.13: version 1 equals the template bytes, before and after the publish. |
| AC2 | A field type change is refused with `422 schema_breaking_change`, naming the field. Nothing new is stored. | see the run below | S8.8: 422 `schema_breaking_change`, `errors[].pointer` `/definition/schema/properties/priority/type`. S8.9: the next version is 404. S8.10: current version and bytes unchanged. |
| AC3 | `agent@tenant-a` can read entity types but gets `403` on publish. | see the run below | S8.4, S8.5, S8.12 (reads, 200). S8.7: 403 `forbidden` on publish. S8.10: nothing stored. |
| AC4 | `agent@tenant-b` sees only tenant B's `incident`, `request` and `change`, never tenant A's version 2. | see the run below | S8.14: exactly `change`, `incident`, `request`, each at version 1. S8.15: B's `incident` is version 1 with the template hash, not A's new hash. S8.17: A's version 2 read as B is a 404 byte-identical to S8.16's 404 for a version that exists nowhere. S8.14, S8.15 and S8.17 never name A's new field or hash. |
| AC5 | A newly onboarded tenant starts with `incident`, `request` and `change` at version 1. | see the run below | S8.19 onboards a tenant through nava-onboard (201, `active`). The stack case `s8_entity_types` then checks in the database that the tenant has exactly those three types. Each is at `current_version` 1 with one version: `initial`, no publisher, and the same `schema_sha256` as the template. |

**Verified on a fresh stack in GitHub Actions:** see the "Last verified run" section below.

## Isolation check

As Bob (tenant B), after tenant A published version 2:

- the list shows only B's three types at version 1;
- B's `incident` is B's own version 1;
- `GET /entity-types/incident/versions/2` returns a 404 whose body is byte-identical to the 404 for a version that exists nowhere.

None of these responses contains A's new field name or A's new hash. Deeper isolation (every entity-type route in the CI-2 suite, publish issuing no DDL) is P3-07 (TICK-345), covered in CI.

## Not included

- Creating tickets of these types: lane A's next stories (PLAN-003).
- A UI for publishing types. This story is API only.
- Editing or deleting a published version. This is never allowed: no role has UPDATE or DELETE on versions (D-3-4).
- Concurrency (50 concurrent publishers get versions 2 to 51) and the database guarantees: `TestPublish*` in `internal/server` and the CI-9 report, in CI. They are not repeated in this kit.

## Known limits

- **The seeded dev tenants are back-filled by the kit, not by onboarding.** Tenants A and B were never onboarded, so before this kit they had no `admins` group, no `tenant_admin_group` row and no entity types. Every publish was 403 and every list was empty. `tools/seed.sh` now fills these in: it creates the Keycloak group `/t/<A>/admins` with `admin@tenant-a.test` in it, runs group sync, inserts the row, and seeds the three types from the templates. This is the dev-stack half of TODO-S8-01. The realm import file itself still has no admins group, so a stack seeded by any other route still denies publishing.
- `tools/seed.sh` seeds the types with SQL, using random row ids. Onboarding uses UUIDv5 ids. The schema bytes are the same (AC5 checks the hashes match), but the ids differ.
- The folder publishes one more `incident` version on every run. Only a fresh stack shows `version: 2`, and S8.11 asserts it only when the previous version was 1.

## Last verified run

Filled in from the GitHub Actions run below.
