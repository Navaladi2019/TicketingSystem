---
title: "7. Set access rules and field masks"
section: tour
status: delivered
sources: [docs/handbook/examples/README.md, docs/handbook/examples/loan_application.access.json, docs/acceptance/s10-access.md, docs/acceptance/guide/bruno/S10/S10-31.bru, docs/acceptance/guide/bruno/S10/S10-18.bru]
adrs: [ADR-001]
operations: [putEntityTypeAccess, getEntityTypeAccess, explainAccess, getRecord, listRecords, patchRecord]
tables: [acl_versions]
---

## In plain words

Access rules say who may read or change each kind of record, and field masks hide single fields. Until an administrator publishes rules for a type, **nobody** can read or create its records: the platform starts closed. In the loan example, the admins group can read and change everything, auditors can read, and the person who applied sees their own application. The `income` field is visible to admins only: an auditor sees the application without it. When someone asks "why can (or can't) this person see this?", the explain call answers.

## What you do

As an administrator, publish the type's access policy. Then check it with the explain call.

## Try it

Bruno: `S10.31 The admin publishes the incident ACL with masks on description and assignee: 200` and `S10.18 Explain: why can Alice read it? allow, with the granting rule`. Needs the type from [step 6](06-create-an-entity-type.md). The example policy has the placeholders `__ADMINS__` and `__AUDITORS__` for the group ids:

```sh
A=00000000-0000-4000-8000-0000000000a1
psql_() { docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -qAt' | tr -d '\r'; }
ADMINS=$(psql_ <<<"select id from groups where tenant_id='$A' and path='/t/$A/admins'")
AUDITORS=$(psql_ <<<"select id from groups where tenant_id='$A' and path='/t/$A/auditors'")

# Before any policy: reading it is a 404, creating a record is refused
curl -i $CORE/api/v1/entity-types/loan_application/access -H "Authorization: Bearer $ADMIN"
curl -i -X POST $CORE/api/v1/records/loan_application -H "Authorization: Bearer $ADMIN" \
  -H "Idempotency-Key: tour-loan-rec0" -H 'Content-Type: application/json' \
  -d '{"data":{"applicant_name":"Jordan Example","amount":250000,"term_months":240,"income":78000}}'

# Publish
curl -i -X PUT $CORE/api/v1/entity-types/loan_application/access \
  -H "Authorization: Bearer $ADMIN" -H 'If-Match: W/"0"' -H "Idempotency-Key: tour-loan-acl" \
  -H 'Content-Type: application/json' \
  --data-binary "$(sed -e "s/__ADMINS__/$ADMINS/g" -e "s/__AUDITORS__/$AUDITORS/g" docs/handbook/examples/loan_application.access.json)"
```

`If-Match: W/"0"` means "there is no policy yet". Each later publish sends the current version.

## What you see

Before a policy: `404 not_found` for the policy, and `403 forbidden` for the create. After the publish:

```
HTTP/1.1 200 OK
Etag: W/"1"
{"version":1,
 "rules":[{"id":"admins","ops":["read","write"],"who":{"groups":["beed2119-…"]}},
          {"id":"auditors-read","ops":["read"],"who":{"groups":["66714971-…"]}},
          {"id":"applicant-read","ops":["read"],"who":{"requester":true}},
          {"id":"applicant-edit-submitted","ops":["read","write"],"who":{"requester":true},
           "where":[{"column":"state","op":"eq","values":["submitted"]}]}],
 "masks":[{"field":"income","read":{"groups":["beed2119-…"]},"write":{"groups":["beed2119-…"]}}],
 "published_at":"2026-11-13T06:13:44.047634Z"}
```

Once a record exists (step 9, `$RID` is its id), the same record as seen by different people:

| Caller | What happens |
|---|---|
| Admin | Reads the record, `income` included |
| Auditor (`GET /records/loan_application/$RID`) | `200`; `data` has no `income` |
| Auditor, `?filter[income]=78000` on the list | `400 masked_field`, `errors:[{"pointer":"/income","rule":"masked"}]` |
| Auditor, `PATCH` the record | `403 forbidden`: the auditors rule has no write |
| Alice (in neither group, not the requester) | `404 not_found`, the same answer as a record that does not exist |

The explain call, run as the administrator, for the auditor's user id and for Alice's:

```sh
curl -sS -G $CORE/api/v1/access/explain -H "Authorization: Bearer $ADMIN" \
  --data-urlencode entity=loan_application --data-urlencode record=$RID --data-urlencode user=<user id>
```

```json
{"acl_version":1,"decision":"allow","masked_fields":["income"],"rule_id":"auditors-read"}
{"acl_version":1,"decision":"deny","masked_fields":["income"],"rule_id":null}
```

## What happens behind it

Every read and write, through every route, goes through one authorization layer. It evaluates the type's newest published policy: the first rule that grants the operation wins, and a `where` clause narrows a rule to records in a given state. Masks are checked separately and per field, on read, on write, and in filters and sorts, so a hidden field cannot be learned by filtering on it. A record the caller may not read is answered as "not found", so its existence is not revealed. Policies are versioned: a publish with a stale `If-Match` answers `412`, a missing one `428`. Details: [access rules](../system/core-acl.md), [field masks](../system/core-masks.md), [access rules and masks in the database](../database/acl.md).

## Not built yet

- No screen for editing rules.
- A mask names groups and users only. It cannot say "the applicant sees their own income" or depend on the record, so in the loan example the applicant cannot see or submit `income` and intake is done by the admins group.
- Rules cannot refer to organisation units (step 5).
