# Acceptance kit: Go 1.27 upgrade (TICK-961)

Story branch `TICK-961/go-1-27-upgrade` at `79805116` (every task merged: #733, #736, #739, #740, #741, #750, #753; `main` merged in by #753). Story PR #751. Verified on 2026-11-19 at 12:53Z on a fresh stack (`docker compose down -v`, `up --build --wait`, guide seed) with two tenants. Another stack held the default ports, so the run used `COMPOSE_PROJECT_NAME=nava-t961`, `NAVA_CORE_PORT=48080`, `KEYCLOAK_PORT=48081` (issuer `http://localhost:48081`).

**Result: PASS.** Every acceptance criterion passes: 55 of 55 API and database checks, `nava audit verify` exit 0 on both tenants' exports, and the golden tests pass with no golden file changed (`evidence/result.txt`, `evidence/result.json`). GitHub CI on #751 is green in both the fast run (`ci:run`, run 44516144151) and the full run (`ci:full`, run 44521943742, isolation report PASS).

## What you can now do

Nothing new is visible to a user. The platform builds on Go 1.27.4. Every new id it creates is time-ordered (UUIDv7). A request whose JSON body repeats a key or holds broken text is refused with a clear `422` and changes nothing. Audit exports holding the Unicode line and paragraph separators still verify offline.

## Try it (about 15 minutes the first time, mostly image builds)

1. From the repository root, on this branch, with Docker running, in Git Bash or any POSIX shell. If another stack holds the default ports, set `COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `OPENOBSERVE_PORT` and `PROMETHEUS_PORT` in `.env`.

   ```sh
   sh docs/acceptance/go-1-27-upgrade/tools/run-kit.sh
   ```

   It deletes the stack's volumes, starts a fresh stack, seeds tenants A and B, adds `admin@tenant-b.test` (`docs/acceptance/S18/tools/kit.mjs admin-b`, then one group sync), and runs `tools/kit.mjs`. It then builds `bin/nava` and runs `nava audit verify` on both exports and the golden tests. With `KIT_SKIP_UP=1` it uses a stack you have just brought up from empty volumes.
2. Users: the dev-only seed users `admin@tenant-a.test` and `admin@tenant-b.test` (passwords in `docs/acceptance/guide/bruno/environments/local.bru`).

You should see `t961: 55/55 checks pass`, two `ok: seq 1..6 …` lines, and `t961: PASS`.

## Acceptance criteria

The evidence is in `docs/acceptance/go-1-27-upgrade/evidence/`. There is one numbered JSON file per API request, holding the request and the response with no credentials. Strict-JSON bodies are stored as `raw_hex`.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Both modules build on Go 1.27.4 | ✅ | `toolchain.txt`: `go.mod` and `temporal-server/go.mod` have `go 1.27.0` / `toolchain go1.27.4`; `go build ./...` passes in both modules on host Go 1.27.4; the `nava-core` image binary reports `go1.27.4`. Every Dockerfile uses `golang:1.27.4-*`. CI `unit`, `temporal-server`, `lint` green on #751. |
| 2 | Newly created ids are UUIDv7: entity types, workflow definitions and steps, groups, number formats | ✅ | `02`–`06` (tenant A) and `08`–`12` (tenant B): create type, ACL, workflow publish, number format, record. `kit.log` AC2: per tenant, the entity definition and version ids, the workflow definition and version ids, all three step ids (`step_ids`), the number-format id and the record id are all version 7. All 6 `groups` rows that `nava-groupsync` wrote are v7. A database sweep finds no non-v7 id in `entity_definitions` (outside the seed templates, see Known limits), `workflow_definitions`, `workflow_definition_versions`, `number_formats` or `groups` (`ids-db.txt`). |
| 3 | A body with a duplicate key or invalid UTF-8 gets `422` with the documented shape, and nothing is written | ✅ | `14`–`40`: 9 operations × 3 defects (duplicate member, raw `0xFF` byte, lone `\ud800` escape). The 9 operations are createEntityType, publishEntityTypeVersion, putEntityTypeAccess, publishWorkflowDefinitionVersion, createNumberFormat, putNumberFormat, putTenantTimeZone, createRecord and patchRecord. Every response is `422 application/problem+json` with the operation's code (`invalid_definition`, `invalid_access_policy`, `workflow_definition_invalid`, `number_format_invalid`, `invalid_time_zone`, `record_invalid`). `errors` has exactly one entry with a `/body…` pointer and `rule` `duplicate` or `invalid_utf8`, and the bad bytes are never echoed. `strict-counts.txt`: tenant A's counts in 9 tables, `audit_log` and `outbox` included, are identical before and after the 27 refusals. The time zone is unchanged (`13`, `41`). |
| 4 | An audit export containing U+2028/U+2029 passes `nava verify`; the golden hash tests are unchanged | ✅ | Tenant A's type name and record text hold U+2028 and U+2029. Once every entry was sealed, `export-tenant-a.jsonl` and `export-tenant-b.jsonl` were exported, and `verify-tenant-a.txt` / `verify-tenant-b.txt` both give `ok: seq 1..6 …`, `exit 0`. `golden-tests.txt`: `internal/audit/chain` (`TestVectorsFileIsCurrent`, the U+2028 vectors), `internal/audit/verify`, `internal/audit/jsondiff`, `cmd/nava` (`TestExitCodes`) and `tools/acceptance/s13kit` (`TestSamplesAreCurrent`) all pass. `golden-diff.txt`: the story diff against `main` touches no golden file (`internal/audit/chain/testdata`, `docs/acceptance/S13`, `internal/audit/verify/testdata`, `temporal-server/authz/testdata`). |
| 5 | Tenant B cannot see or change tenant A's new rows | ✅ | See the isolation check below. |

## Isolation check

- As `admin@tenant-b`, GET on tenant A's entity type, its ACL, its workflow definition, its number format and its record each return the same `404` as a made-up key or id. The bodies are equal apart from `instance` and name nothing of tenant A (`42`–`55`).
- As `admin@tenant-b`, PUT on tenant A's number format and PATCH on tenant A's record get the same `404` as a made-up id.
- Tenant B's number-format and entity-type lists do not contain tenant A's rows (`56`, `57`).
- `admin@tenant-b` publishing a workflow under tenant A's type key gets `422`: tenant B has no such type (`58`).
- After all of this, tenant A's workflow version, record (data and version) and number format (version 1) are unchanged (`isolation-db.txt`).
- Tenant B's audit export names none of tenant A's ids or keys.
- CI full run 44521943742: the tenant isolation suite (T11) passes, with every runtime role against every tenant table, `number_formats` and `number_format_versions` included (artifact `isolation-report`).

## GitHub CI on #751 (head `79805116`)

| Run | Mode | Jobs |
|---|---|---|
| 44516144151 | fast (`ci:run`) | runner, changes, story-branches, lint, unit, integration, stack, sdk, temporal-server, commit-check-budget, coverage: success |
| 44521943742 | full (`ci:full`) | the same jobs: success; integration ran the isolation suite (report PASS); stack ran the stack, stack-db and stack-internal phases with the CI-5, CI-6, S10, S11b and S28 reports |
| 44515974853 | govulncheck | root, temporal-server: success |

`full-suite`, `main-red` and `self-hosted-cleanup` are skipped by design on a PR into `main`. `full-suite` gates only PRs into a `TICK-*` branch. The earlier `ci` run 44515974746 was cancelled when the `ci:run` label started a new run, and its red `full-suite` / `main-red` lines are left over from that cancellation.

## Not included

- The move from `github.com/google/uuid` to the stdlib `uuid` package is deferred to TICK-966.
- PostgreSQL stays on 16. Ids are generated in Go, so the database does not need native `uuidv7()`.

## Known limits

- The seed script (`docs/acceptance/guide/tools/seed.sh`) still writes the seeded `incident` / `request` / `change` types with `gen_random_uuid()` (v4). This is deliberate and documented there: it is dev seed data, not product code, and onboarding uses UUIDv5. The id sweep leaves these out.
- No API field lets a client put U+2028 into an audit **envelope** text field: `request_id` is always server-generated. The live check therefore puts the separators in the diff, which the export carries as a JSON string, escaped as ` ` there. The verifier decodes that string before hashing, so the bytes are the same either way. The envelope-level case that broke under Go 1.27 (TICK-962) is covered by the golden vectors and the S13 samples (criterion 4).
- Spec drift, not blocking: `docs/api/strict-json-bodies.md` lists 7 operations, but the code (`internal/server/api.go`, `strictBodyRefusals`) also refuses bodies on 4 operations added from `main` in #753. These are `POST /api/v1/number-formats`, `PUT /api/v1/number-formats/{id}` and `POST …/republish-linked` (`number_format_invalid`), and `PUT /api/v1/tenant/settings/time-zone` (`invalid_time_zone`). Their behaviour matches the documented shape (criterion 3); only the table rows are missing.
