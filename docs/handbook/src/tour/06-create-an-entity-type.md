---
title: "6. Create an entity type"
section: tour
status: delivered
sources: [docs/handbook/examples/README.md, docs/handbook/examples/loan_application.json, docs/handbook/examples/incident.json, docs/quality/x1a-first-routed-record.md, docs/acceptance/s8-entity-types.md, docs/acceptance/guide/bruno/S8/S8-11.bru, internal/configsurface/testdata/inventory.json, internal/configsurface/doc.go]
adrs: [ADR-001, ADR-005, ADR-008]
stories: [S27, S28]
operations: [createEntityType, listEntityTypes, getEntityType, publishEntityTypeVersion]
tables: [entity_definitions, entity_definition_versions]
---

## In plain words

An entity type is a kind of record the customer wants to track: an incident, a loan application, a visa application. The administrator defines what each record holds (applicant name, amount), which states it moves through (submitted, under review, approved) and the prefix of its numbers (`LOAN`). The platform does not change when a customer adds a new kind of record. Changing a type later publishes a new version; records created earlier keep the version they were created under.

## What you do

As a tenant administrator, send the type's definition once. Later changes are published as new versions.

## Try it

Bruno: `S8.11 AC1: admin@tenant-a publishes a new incident schema: 201` is the closest request (it publishes a version of `incident`). The body used here is `docs/handbook/examples/loan_application.json`:

```sh
curl -i -X POST $CORE/api/v1/entity-types \
  -H "Authorization: Bearer $ADMIN" -H "Idempotency-Key: tour-loan-create" \
  -H 'Content-Type: application/json' --data-binary @docs/handbook/examples/loan_application.json
```

The file holds `key`, `name`, `number_prefix` and a `definition` with a JSON Schema (`schema`), the state model (`states`) and the mapping of the schema's `state`, `requester`, `assignee` and `assignment_group` to record columns (`fixed_columns`).

## What you see

```
HTTP/1.1 201 Created
{"key":"loan_application","name":"Loan application","number_prefix":"LOAN",
 "numbering_mode":"gaps","version":1,"current":true,
 "schema":{ … the schema as sent … },"schema_sha256":"d72758cd…",
 "states":{"initial":"submitted","states":["submitted","under_review","approved","rejected"],
   "transitions":[{"from":"submitted","to":"under_review"},
                  {"from":"under_review","to":"approved"},{"from":"under_review","to":"rejected"}]},
 "compat":"initial","published_at":"2026-11-13T06:13:18.907881Z"}
```

Sending it again with the same key answers `201` with the header `Idempotent-Replayed: true` and creates nothing. The list now shows the three starter types and `loan_application`.

Refusals:

| Call | Answer |
|---|---|
| A non-admin (the auditor) sends it | `403 forbidden` |
| `"number_prefix":"a"` and an empty definition | `422 invalid_definition`, with `errors` naming `/number_prefix` (`pattern`), `/definition/schema`, `/definition/states/initial` and `/definition/states/states` |

Publishing version 2, with an optional `purpose` field added (`POST /api/v1/entity-types/loan_application/versions`, body `{"definition": {…}}`), answers `201` with `"version":2,"current":true`. Records created afterwards report `"schema_version":2`; version 1 is still readable with `GET /api/v1/entity-types/loan_application/versions/1`.

### An example whose type the tenant already has

Every new tenant already has `incident`, `request` and `change` at version 1 (step 1). Sending `docs/handbook/examples/incident.json` to `POST /api/v1/entity-types` therefore answers `409 entity_type_exists`. Publish the example's `definition` as the next version of `incident` instead:

```sh
jq '{definition: .definition}' docs/handbook/examples/incident.json > incident-v2.json
curl -i -X POST $CORE/api/v1/entity-types/incident/versions \
  -H "Authorization: Bearer $ADMIN" -H "Idempotency-Key: tour-incident-v2" \
  -H 'Content-Type: application/json' --data-binary @incident-v2.json
```

The first line needs `jq`. Without it, copy the file's `definition` object into `{"definition": …}` by hand. If `incident-v2.json` ends up empty, the call answers `400 validation_failed` with only `/body` in `errors`: that means the body was empty.

It answers `201` with `"key":"incident","version":2,"current":true`, and `incident` keeps its default access rule. This is what the S28 kit does (Bruno `S28.9`). `loan_application` and `visa_application` are new keys, so `POST /api/v1/entity-types` works for them, but they have no default access rule: publish their `.access.json` with your tenant's group ids in place of `__ADMINS__` and `__AUDITORS__` (step 7) before anyone can read their records.

## What happens behind it

The administrator's call is checked first: only a tenant administrator may define types, and the body is validated as a JSON Schema 2020-12 document and a state model before anything is stored. The schema is stored as the exact bytes sent. A new version must not break the previous one (for example, changing a field's type answers `422 schema_breaking_change`). Record numbers are `<prefix><7 digits>` per type per tenant: `LOAN0000001`, `LOAN0000002`. The `gaps` mode takes the number in its own short transaction, so a failed create can leave a gap; `gapless` is the other mode. Details: [entity types](../database/entity-definitions.md), [metadata engine](../system/core-metadata.md), [numbering](../database/numbering.md).

## Adding or removing a type-definition keyword

The keywords a type's schema can use are listed in `internal/configsurface/testdata/inventory.json` under `type_definition`: the JSON Schema keywords the seed templates use (`type`, `properties`, `required`, `enum`, `maxLength` and the rest), the extensions `x-nava-secret` and `x-nava-unique`, and the fields of the type-definition document (`key`, `schema`, `states`, `numbering_mode` and the rest). Within a major version this list only grows (ADR-008 D12), so a type a tenant published keeps compiling after an upgrade. The upgrade-compatibility suite (M-X2, S27) checks it.

- **Adding** a keyword or extension: list it under `type_definition` and add a fixture `internal/configsurface/testdata/fixtures/typedef/<name>.json` that uses it. A new `x-nava-*` extension or document field that is not listed fails `task test` with `unlisted: …`. The standard keyword list is hand-curated, because `internal/metadata` accepts any JSON Schema 2020-12 keyword; its fixtures catch a removal.
- **Deprecating** one: keep it accepted and listed, and add it to the inventory's `deprecated` list with a `since` version.
- **Removing** one: only with a new major, a raised `"major"` and a `docs/upgrade/v<major>.md` note naming it (for example `type_definition.extensions/x-nava-unique`). Otherwise `task test` fails with `removed: …` or `not accepted: typedef/<name>.json: …`, and `task lint` with `compat: removed: …`.

What each failure line means: [Troubleshooting](../operations/troubleshooting.md#a-lint-or-test-check-fails).

## Not built yet

- No form designer or screen: types are defined by API.
- No custom number formats such as `LOAN-2027-00042`. The format compiler exists, but types are not linked to a format yet.
- No file attachments, and no conditions or approvals on a state change.
- A type cannot be deleted.
