---
title: "9. Create and work with records"
section: tour
status: delivered
sources: [docs/acceptance/s9-records.md, docs/acceptance/guide/bruno/S9/S9-4.bru, docs/acceptance/guide/bruno/S9/S9-9.bru, docs/acceptance/guide/bruno/S9/S9-13.bru]
adrs: [ADR-001, ADR-005]
operations: [createRecord, listRecords, getRecord, patchRecord, getRecordHistory]
tables: [records]
---

## In plain words

A record is one case: one loan application, one incident. People create it, find it in a list, move it through its states and see who changed what. Each record gets a short number to quote in conversations, such as `LOAN0000001`. Two people editing the same record at once cannot overwrite each other: the second is told the record changed.

## What you do

Create a record of a type from [step 6](06-create-an-entity-type.md), with the access policy from [step 7](07-access-rules-and-masks.md) in place.

## Try it

Bruno: `S9.4 AC1: Alice creates an incident: 201, INC number, ETag`, `S9.9 AC2: update with the current ETag: 200`, `S9.13 AC3: the history`. Here, as the administrator:

```sh
H='Content-Type: application/json'
curl -i -X POST $CORE/api/v1/records/loan_application -H "Authorization: Bearer $ADMIN" \
  -H "Idempotency-Key: tour-loan-rec1" -H "$H" \
  -d '{"data":{"applicant_name":"Jordan Example","amount":250000,"term_months":240,"income":78000}}'
# list, filtered and sorted
curl -sS -G $CORE/api/v1/records/loan_application -H "Authorization: Bearer $ADMIN" \
  --data-urlencode 'filter[state]=submitted' --data-urlencode sort=-created_at --data-urlencode limit=10
# read one: RID is the "id" of the create response
curl -i $CORE/api/v1/records/loan_application/$RID -H "Authorization: Bearer $ADMIN"
# change its state; the ETag of the read goes in If-Match
curl -i -X PATCH $CORE/api/v1/records/loan_application/$RID -H "Authorization: Bearer $ADMIN" \
  -H 'If-Match: W/"1"' -H "Idempotency-Key: tour-p1" -H 'Content-Type: application/merge-patch+json' \
  -d '{"state":"under_review"}'
curl -sS $CORE/api/v1/records/loan_application/$RID/history -H "Authorization: Bearer $ADMIN"
```

## What you see

The create:

```
HTTP/1.1 201 Created
Etag: W/"1"
Location: /api/v1/records/loan_application/01a1d38a-629a-70ff-a578-e438265b5edf
{"id":"01a1d38a-629a-70ff-a578-e438265b5edf","number":"LOAN0000001","state":"submitted",
 "schema_version":2,"version":1,
 "data":{"amount":250000,"applicant_name":"Jordan Example","income":78000,
         "requester":"e4a636ce-…","state":"submitted","term_months":240}, …}
```

A second create returns `LOAN0000002`. The filtered list returns both with `"total":2,"next_cursor":null`. The state change returns `200`, `Etag: W/"2"`, `"state":"under_review"`. The history lists both changes, newest first:

```json
{"items":[
 {"action":"update","source":"user","changed":["/data/state"],
  "before":{…,"state":"submitted"},"after":{…,"state":"under_review"}},
 {"action":"create","source":"user","changed":["/data/amount","/data/applicant_name","/data/income","/data/requester","/data/state","/data/term_months"],"after":{…}}]}
```

Failure paths, each a problem body with a stable `code`:

| Call | Answer |
|---|---|
| Empty `applicant_name` and negative `amount` | `422 record_invalid`, `errors` pointing at `/data/amount` (`minimum`) and `/data/applicant_name` (`minLength`) |
| `PATCH` with no `If-Match` | `428 if_match_required` |
| `PATCH` with an `If-Match` that is out of date | `412 version_conflict`; the current ETag is in the `ETag` header |
| `PATCH` `{"state":"submitted"}` from `under_review` | `422 illegal_transition` |
| Same `Idempotency-Key` again | the first answer, nothing created |

## What happens behind it

Every create and update goes through one write pipeline: it validates the data against the type's current schema version, applies defaults, checks the state model, checks the access rules and masks, takes the record number, saves the record and its history entry together, and queues the audit entry. A `PATCH` is a JSON Merge Patch (RFC 7396), and the `ETag` is the record's version, so a stale write is refused instead of overwriting. Lists return only records the caller may read, with a `total` counted the same way. Details: [write pipeline](../system/core-write-pipeline.md), [records](../database/records.md), [numbering](../database/numbering.md).

## Not built yet

- No record screen or search box: lists filter on fixed columns and top-level scalar fields, and sort on fixed columns, `created_at` and `updated_at`. No range filter and no text search.
- No attachments, comments or notifications.
- A state change is a plain state change: no required reason, no approval and no condition (workflows, step 8).
- No delete.
