## CI-2

Coverage: full: generic checks on every tenant table; forged-tenant HTTP paths (T17); the forged write on POST /api/v1/records/incident, StoredUnderA (P3-18).
| forged tenant in the query string | - | PASS | 200, tenant A only |
| forged tenant in a header | - | PASS | 200, tenant A only |
| forged tenant_id in the body of POST /api/v1/records/incident | - | PASS | refused 404, the unknown key's byte for byte; nothing written, tenant B unchanged at {records:4 audit:2} |
| forged data.tenant_id in POST /api/v1/records/incident | - | PASS | refused 404, the unknown key's byte for byte; nothing written, tenant B unchanged at {records:4 audit:2} |
| forged X-Tenant-Id header on POST /api/v1/records/incident | - | PASS | refused 404, the unknown key's byte for byte; nothing written, tenant B unchanged at {records:4 audit:2} |

## CI-8

Coverage: partial → P3-37 (S10): the masked-field filter 400 needs S10's masks; here the record routes' byte-identical 404 on GET, PATCH and history, stale If-Match 412, and (e)/(f) (TICK-87); cursor cases from T23; idempotency cases from T22.

| records: tenant A naming tenant B's only key gets the unknown key's 404, byte for byte, on POST and GET | nava_app | PASS |  |
| records: tenant A naming tenant B's group or principal gets a random UUID's 422, with no foreign key detail | nava_app | PASS |  |
| records CI-8: (a) tenant B's GET of tenant A's record is a random UUID's 404: status, headers and body byte-identical | nava_app | PASS |  |
| records CI-8: (b) tenant B's PATCH and GET history of tenant A's record are a random UUID's 404, byte-identical, and change nothing | nava_app | PASS |  |
| records CI-8: (c) tenant A's PATCH with its record's previous ETag is 412 version_conflict, and changes nothing and audits nothing | nava_app | PASS |  |
| records CI-8: (e) tenant B's PATCH of tenant A's record with A's current or a stale ETag is a random UUID's 404, never 412 | nava_app | PASS |  |
| records CI-8: (f) tenant B's PATCH of tenant A's record with no If-Match is a random UUID's 404, not 428 | nava_app | PASS |  |
| records CI-8: (g) the mutation control of (e) and (f) rejects a control failure: a leaky handler that also intercepts the random UUID does not pass it | nava_app | PASS |  |
| records CI-8: (d) a filter on a masked field is 400 | - | PENDING | pending P3-37 (S10, PLAN-003 D-3-19) |
