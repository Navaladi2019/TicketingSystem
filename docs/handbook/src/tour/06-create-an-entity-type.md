---
title: "6. Create an entity type"
section: tour
status: delivered
sources: [docs/handbook/examples/README.md, docs/handbook/examples/loan_application.json, docs/acceptance/s8-entity-types.md, docs/acceptance/guide/bruno/S8/S8-11.bru]
adrs: [ADR-001, ADR-005]
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

## What happens behind it

The administrator's call is checked first: only a tenant administrator may define types, and the body is validated as a JSON Schema 2020-12 document and a state model before anything is stored. The schema is stored as the exact bytes sent. A new version must not break the previous one (for example, changing a field's type answers `422 schema_breaking_change`). Record numbers are `<prefix><7 digits>` per type per tenant: `LOAN0000001`, `LOAN0000002`. The `gaps` mode takes the number in its own short transaction, so a failed create can leave a gap; `gapless` is the other mode. Details: [entity types](../database/entity-definitions.md), [metadata engine](../system/core-metadata.md), [numbering](../database/numbering.md).

## Not built yet

- No form designer or screen: types are defined by API.
- No custom number formats such as `LOAN-2027-00042`. The format compiler exists, but types are not linked to a format yet.
- No file attachments, and no conditions or approvals on a state change.
- A type cannot be deleted.
