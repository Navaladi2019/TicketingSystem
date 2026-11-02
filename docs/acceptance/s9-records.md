# Acceptance kit: S9, an agent logs an incident, updates it, sees its history; a stale update is refused

Story TICK-335, kit TICK-359 (P3-17). Story branch `TICK-335/records-pipeline`.
Guide page: `docs/acceptance/guide/s9.html`. Bruno folder: `docs/acceptance/guide/bruno/S9/` ("S9 records").
Stack case: `deploy/compose/test/cases/s9_records.sh`.

## What you can now do

- Create incidents over the API. Each gets the tenant's next number (`INC0000001`, `INC0000002`, …) and an `ETag`.
- Send the same create again with the same `Idempotency-Key` and you get the same incident back, not a second one.
- Update an incident with the `ETag` of the copy you read. An update from an out-of-date copy is refused (`412`) instead of overwriting someone else's change; one with no `ETag` is refused too (`428`).
- Read an incident's history: one row per change, naming who made it.

## Try it (about 10 minutes on a running stack)

Prerequisites as for the other stories (`docs/acceptance/guide/prerequisites.html`): stack up, Node.js for the headless run, Bruno to click through.

```sh
sh docs/acceptance/guide/tools/seed.sh          # again, if you seeded before S8
sh docs/acceptance/guide/tools/api-test.sh S9   # or click S9.1-S9.20 in Bruno
sh deploy/compose/test/stack-test.sh s9_records # all of it, plus the database checks
```

| Story user | Seeded user (dev-only credentials in `bruno/environments/local.bru`) | Tenant |
|---|---|---|
| `agent@tenant-a` | `alice@tenant-a.test` (Alice) | A |
| `agent@tenant-b` | `bob@tenant-b.test` (Bob) | B |

What you should see: every S9 check green. On a fresh stack S9.4 is `INC0000001`, S9.8 `INC0000002`, and Bob's S9.20 is tenant B's own `INC0000001`. The folder runs again on the same stack; later runs get later numbers.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| AC1 | Create returns `201`, `number: INC0000001` and an `ETag`. A retry with the same `Idempotency-Key` returns the same record, and only one exists. | ✅ | S9.4: 201, `INC0000001 (fresh stack)`, `ETag: W/"1"`, `Location`. S9.5: same id, number and `ETag`. Stack case: `ok: AC1: the retried create stored exactly one record`, `ok: AC1: fresh stack: INC0000001`. |
| AC2 | Update with the current `ETag` succeeds; with the old one `412 version_conflict`; without `If-Match` `428`. | ✅ | S9.9: 200, version 2, `ETag: W/"2"`. S9.10: 412 `version_conflict`, current `ETag` `W/"2"` in header and `detail`. S9.11: 428 `if_match_required`. S9.12: still version 2, unchanged by the refused updates. |
| AC3 | History lists one audit row per change, naming the agent and `source=user`. | ✅ | S9.13: exactly `update`, `create`; each `actor_id` = Alice's `user_id` from `GET /me`, `source: user`; `changed` = `/data/priority`, `/data/state`. S9.14: one `create` row. Stack case: `ok: AC3: INC0000001 has two audit rows (create, update), source user, both by Alice`. |
| AC4 | A create missing a required field returns field errors and creates no record and no history. | ✅ | S9.6: 422 `record_invalid`, `errors[].pointer` `/data/short_description` and `/data/priority`. Checked as **no gap in the next number**: S9.8 is `INC0000002` right after S9.4. Stack case: `ok: AC1, AC4: numbers INC0000001 and INC0000002, … no gap`, `ok: AC4: the refused create stored no record and no history`. |
| AC5 | `agent@tenant-b` gets the same `404` body for tenant A's incident as for an id that does not exist. | ✅ | S9.15: reference 404. S9.16 (GET), S9.17 (PATCH with A's current `ETag`) and S9.18 (history): byte-identical to S9.15, never 412 or 428, naming nothing of A's incident. S9.19: A's incident unchanged. |

How "the number" is checked: the stack case checks the shape `^INC\d{7}$` and that the second number follows the first with no gap. The exact `INC0000001` is asserted only when the tenant had no incident before the run (`s9_fresh_a` / `s9_fresh_b` = 1, set by the stack case), which is the fresh-stack walk below.

**Status: ready.** All five criteria pass on a fresh stack in GitHub Actions, run [41430373623](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41430373623), every job green.

## Isolation check

As Bob (tenant B), after Alice created and updated `INC0000001` in tenant A:

- `GET`, `PATCH` (with A's current `ETag` `W/"2"`) and `GET …/history` of A's incident each return a 404 byte-identical to the 404 for `00000000-0000-4000-8000-00000000dead`, never 412 or 428;
- none of these responses contains A's id, number or short description;
- A's incident is still version 2 afterwards (S9.19);
- Bob's own incident is tenant B's `INC0000001`: numbers are per tenant.

Isolation report of the same run (`isolation-report` artifact, extracts in `s9-evidence/`):

| Suite | Rows | Result |
|---|---|---|
| CI-2 (P3-18) | forged `tenant_id` in the body, forged `data.tenant_id`, forged `X-Tenant-Id` on `POST /api/v1/records/incident` | 3 named PASS rows: refused, nothing written, tenant B unchanged |
| CI-8 | records CI-8 (a), (b), (c), (e), (f), (g) | 6 named PASS rows. (d) masked-field filter is PENDING → P3-37 (S10, D-3-19), as planned |
| CI-10 | (f) start key per tenant | 7 named PASS rows (`cascade_start: …`) |
| CI-10 | (a)–(e): depth audit and the cycle case per the TICK-361 ruling | Not rows in the report; the report's coverage line points to `TestCI10_*`, and all 11 passed in the same run's `integration` job, including `TestCI10_WriteBackToTriggeringRecordCommits` (cascade_cycle audit row), `TestCI10_SiblingAndSelfRetriggeringWritesAreNotCycles` and `TestCI10_DepthBeyondLimitRejectedAndAuditedOnTheRunner` (`s9-evidence/ci10-integration-tests.txt`). (g) signal ping-pong is P3-42 (S11b). |

The I9 ruling (TICK-361) was made, so the fallback (cycle case moved to S11b) does not apply.

## Not included

- A UI, and listing or searching incidents.
- Record-level permissions and field masks: S10.
- Concurrency on one record, the cascade depth and cycle checks, and the database guarantees: CI (`TestRecords*`, `TestCI10_*`, the isolation report). Not repeated in this kit.

## Known limitations

- **AR-S9-01, on purpose: until S10 lands, any member of a tenant may read and write every record of that tenant.** Alice can update any tenant A incident. This is not a bug; do not report it. Other tenants stay out (AC5). S9 is never released without S10.
- The exact `INC0000001` is only what a fresh stack shows. On a used stack the folder checks the shape and the no-gap rule.
- The CI-10 cycle and depth cases are Go tests, not named rows in `isolation-report.md`; see the isolation check above.
- The story branch is behind `main` (50 commits at `0667682`). This run is on the story branch, not with `main` merged in; the story → `main` checklist re-runs CI after the merge.

## Last verified run

Fresh stack in GitHub Actions, `ci` workflow (`workflow_dispatch` on `TICK-359/s9-acceptance-kit`, the story branch at `0667682` plus this kit), job `stack`, case `s9_records`.

| Run | Bruno S9 | Database checks | Result |
|---|---|---|---|
| [41430373623](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41430373623) | **20 of 20 requests, 67 of 67 tests**; S9.4 `INC0000001 (fresh stack)`; S9.8 `INC0000002: no gap`; S9.20 `INC0000001 (fresh stack, tenant B)` | 5 `ok:` lines (AC1, AC1/AC4 numbers, fresh INC0000001, AC4 nothing stored, AC3 audit rows) | **PASS**, every job green |

The bru output of that run is `s9-evidence/s9-fresh-stack-run.txt` (no tokens or passwords). The full Bruno reports stay on the runner: they hold short-lived tokens.
