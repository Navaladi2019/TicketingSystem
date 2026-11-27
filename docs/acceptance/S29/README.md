# S29 acceptance kit: old tickets pick up a new default value, and filters find them before the job ends (TICK-915)

Kit task TICK-754 (it absorbed TICK-928, PLAN-010 P10-13). Design: ADR-009 D1 to D15. Acceptance tests CI-n: `docs/adr/ADR-009/rfc/RFC-021-schema-version-backfill.md` §10. Story branch: `TICK-740/search-finish`, shared with S22.

**Status (2026-11-28): criteria 1, 2, 3, 5 and 6 PASS on a fresh stack; criterion 4 FAILS on the error code only** (`invalid_definition` where ADR-009 D13 and the story say `invalid_schema`; the Technical Lead rules). fresh stack `nava-s22s29`, 2026-11-27 22:57Z to 23:03Z, from `c7db3d80` (PR #918 with `TICK-740/search-finish` merged, including PR #988 and PR #931), `run-s22-s29.sh`. Bruno S29: 24 requests, 72/73 tests. `verify.sql`: every check `ok`. Evidence: `evidence/`.

**Criterion 1 runs on `s29_level`, not `priority`.** The incident template already defaults `priority` to 3, and removing a default is breaking (D14), so a fresh stack cannot reach "priority has no default". `kit.mjs setup` adds `s29_level` (integer 1–4, flagged, no default) and the kit publishes `default: 3` on it: the same shape the story describes.

## What you can now do

- Publish a version that gives an existing field a default. Every old ticket that has no value for it gets the default written in, in the background, with no follow-up step.
- From the moment of the publish, a filter on the default value also finds the old tickets.
- The incident type's page shows the job: `state`, and the counts `rewritten`, `skipped`, `failed` and `remaining`.
- A ticket that can't be rewritten is listed, by id and error code only, for the type's admins.
- A unique field can't have a default. An optional field can become required in the same version that gives it a default.

## Try it (about 10 minutes, most of it the image build)

From the repository root, on the story branch, Docker running, Git Bash or any POSIX shell, Node.js 22:

**In GitHub Actions:** the stack case `deploy/compose/test/cases/s22_s29_acceptance.sh` runs both kits in the `ci` workflow's stack job (see `docs/acceptance/S22/README.md`). On a laptop:

```sh
sh docs/acceptance/S22/tools/run-s22-s29.sh    # S22 and S29 on one fresh stack
sh docs/acceptance/S29/tools/run-kit.sh        # S29 alone on a fresh stack
```

`run-kit.sh` deletes the stack's volumes, then:

1. starts a fresh stack with the `operator` profile (the Temporal UI, for listing workflow runs), runs `docs/acceptance/guide/tools/seed.sh`, `tools/tunnels.sh up`, and adds `admin@tenant-b` (S18's `kit.mjs admin-b` and one group sync);
2. **prep:**
   - `tools/kit.mjs setup`: as `admin@tenant-a`, publishes an incident version that drops the template's `default: 3` from `priority` and adds an optional `s29_tier` (string);
   - `tools/seed.sql`: 2,000 tenant A incidents with neither key, all in the service-desk group so Alice reads them. Ordinal 2000 is the **poison record**: its `short_description` is 200 characters, over the schema's `maxLength` of 160;
   - `tools/kit.mjs held`: through the API, 50 incidents that already hold a priority (25 at 3, 25 at 1), and 5 tenant B incidents at priority 1;
   - `tools/expect.sql`: what the filter must give, passed to Bruno;
3. **check:**
   - Bruno folder **S29 schema backfill** (`docs/acceptance/guide/bruno/S29/`);
   - `tools/verify.sql`: every `check|` line must end `ok`.

The two jobs over about 2,050 records take about a minute. ADR-009 measured 7.3 ms a record, plus the D10 pacer. `S29_PHASE=prep` or `S29_PHASE=check` runs one half; `S29_KEEP_STACK=1` skips step 1.

| Story user | Seeded user | What they do here |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test` | publishes; reads the type, the failures and the sampled records |
| `agent@tenant-a` | `alice@tenant-a.test` (service-desk) | filters; gets 404 on the failures |
| `admin@tenant-b` | `admin@tenant-b.test` | cross-tenant checks |

## Acceptance criteria

| # | Criterion | Steps | What is asserted | Result |
|---|---|---|---|---|
| 1 | Publishing `default: 3` on `priority` answers 201; the type shows `backfill` with its state and counts; at every point until the job ends, `filter[priority]=3` returns the tickets with no priority plus those holding 3, with the right `total` | S29.10 to S29.12 | S29.10: 201, the next version. S29.11, at once: every page of the filter as Alice gives exactly expect.sql's set (count and md5 of the ids), and every record reads `priority: 3`. S29.12 samples the type and the filter every 2 s until the job ends: each sample's `total` is right, and the states only move forward, ending `failed_records` (the poison record). Final counts: `failed` 1, `remaining` 0, `rewritten + skipped + failed` = the records that lacked priority. | ✅ S29.11: 2,025 listed over 11 pages, the scan's md5, all `capped` 1,000 (0 `unknown`). S29.12: 21 samples over 41 s, every one `capped` 1,000, `running` then `failed_records`. Final: rewritten 1,999, skipped 50 (the held records), failed 1, remaining 1 (= the failure, D10). |
| 2 | When the job ends, each ticket that had no priority holds 3, its `version` moved by one, its `updated_at` did not move, and it has exactly one history entry from `schema-backfill`; the 50 are untouched; the on-update test workflow shows no new run | S29.7 to S29.9, S29.13, S29.17; verify.sql | S29.7 publishes a `record.updated` workflow and S29.8 starts one run of it with a real edit. S29.9 and S29.13 read 10 seeded records, 10 of the 50 and the poison record before and after: priority, version, `updated_at` and history. S29.17 lists tenant A's workflow runs: the same count as S29.8. verify.sql: no seeded record's `updated_at` moved, and each non-poison seeded record has exactly 2 `actor_system = 'schema-backfill'` audit entries (this job and the criterion 5 job). | ✅ S29.13: the 10 rewritten samples hold 3, version + 1, same `updated_at`, one new history entry (`source: system`, `update`); the 10 held unchanged. S29.17: 1 run before and after. `verify.txt`: 0 `updated_at` moved; 1,999 records with exactly 2 `schema-backfill` audit entries. |
| 3 | A ticket that can't be rewritten is listed by `GET .../backfill/failures` (id and a code, never a value) for the admin, answers 404 for an agent, and still matches the filter; the job ends `failed_records` | S29.12, S29.14 to S29.16; verify.sql | S29.14: the admin sees exactly the poison record's id with a code, and no field value or field name in the body. S29.15: Alice, 404 `not_found`. S29.16: the poison record is still in `filter[priority]=3`, which is the same set as at once. verify.sql: the failure rows are the poison record. | ✅ S29.14: exactly the poison id, code `validation_failed`, no value. S29.15: 404. S29.16: still listed. |
| 4 | `default` + `x-nava-unique` answers 422 `invalid_schema` and stores nothing | S29.5, S29.8 | 422 `invalid_schema`. S29.8's pre-request reads the type: still at the version S29.4 read. | ❌ 422, nothing stored (S29.8's version check passes), but code `invalid_definition`, not `invalid_schema`. Spec drift: Technical Lead to rule. |
| 5 | An optional field made required with a default is accepted (201) and backfilled like criterion 2; without a default it is refused as breaking | S29.6, S29.18 to S29.20; verify.sql | S29.6: required with no default, 422 `schema_breaking_change`. S29.18: required with `default: "standard"`, 201. S29.19 waits for that job. S29.20: every sampled record, the 50 included, holds `s29_tier: "standard"`, with version + 1, the same `updated_at` and one more history entry. verify.sql: only the poison record lacks `s29_tier`. | ✅ S29.6: 422 `schema_breaking_change`. S29.18: 201. S29.19: the job (target 4) ended. S29.20: every sample holds `standard`, version + 1, one more entry. |
| 6 | Tenant B sees none of tenant A's jobs, failures or tickets (404, the same as a missing type) | S29.21 to S29.24; verify.sql | S29.21: tenant B's own `incident` type has no `backfill`. S29.22: its failures route answers 404 `not_found`, the same as a type nobody has, and never names the poison id. S29.23: tenant A's poison record by id, 404. S29.24: B's `filter[priority]=3`, no items, `total` 0. verify.sql: tenant B has no `schema_backfills`, pending or failure rows, and no `schema-backfill` audit entry. | ✅ S29.21: no `backfill` on B's type. S29.22: B's own type 200 `items: []`, a missing type 404. S29.23: 404. S29.24: 0. `verify.txt`: tenant B 0 jobs, 0 pending, 0 failures, 0 `schema-backfill` entries. |

## Criterion 1's total (Technical Lead ruling, 2026-11-26)

A list can't report a `total` over 1,000 (ADR-006-A2), and TICK-915's wording now says so. The kit asserts, at once (S29.11), on every sample until the job ends (S29.12) and after it (S29.16):

- paging `filter[priority]=3` to the end lists exactly the expected records (expect.sql: 2,025 on a fresh stack), with no duplicates and none missing;
- every page answers `capped` with `total` 1,000 and `total_capped: true`, or `unknown` with `total` a JSON `null` (present, not missing, not 0);
- `exact`, or a `total` under 1,000, fails, in every job state;
- the number of `unknown` pages is logged (A2-8(7)); allowed, not expected on a 2k seed.

## Assumptions the day-2 run confirms

| Where | Assumed | From |
|---|---|---|
| S29.13, S29.20 | a history entry written by the backfill carries the string `schema-backfill` (as `actor_system` or another field) | ADR-009 CI-2 (`actor_system='schema-backfill'`) |
| S29.14 | each failure item has `record_id` (or `id`) and `code` | ADR-009 D10 |
| S29.8, S29.17 | the backfill starts no `nava.interpreter` workflow in tenant A; Temporal visibility is reached at `{{temporal_ui_url}}` | ADR-009 D4, CI-2 |
| S29.5 | `invalid_schema`, as ADR-009 D13 and criterion 4 say. TICK-919's PR #932 answers `invalid_definition` instead ("`invalid_schema` does not exist"). Spec drift, raised with the Technical Lead on TICK-754; the kit follows the ADR until they rule | ADR-009 D13 |
| S29.11 to S29.16, S29.21, S29.22 | the `backfill` object on `GET /entity-types/{key}`, `GET .../backfill/failures`, effective values on reads and the D12 filter window. PR #932 lists these (TICK-922, 923, 924) as "still open follow-ups", but those issues were cancelled into TICK-919 | ADR-009 D10 to D12 |
| S29.21, S29.24 | admin B reads its own `incident` type; its `priority` has no default | the seed template has `default: 3`, but tenant B never published a default-adding version, so it has no job. If the template default itself opens a window in B, S29.24 needs B records at priority 1 only (they are) and still lists 0 |

## Not in the kit yet

The four additions the Technical Lead carried over from TICK-920 (2026-11-26 23:01Z): CI-4's wait bound with 4 live writers, CI-11's real `nava-worker` kill mid-page, two tenants' timing under `NAVA_SCHEMA_BACKFILL_CONCURRENCY=1`, and the R9 history-length record.

## Not included

- CI-4 (live writers, no lost update), CI-5 (supersede), CI-11 (redelivery, a killed worker): integration tests in TICK-920 and the TICK-753 suite, in CI.
- CI-12 (100k cost) is `Performance pending` (TICK-927). No timing is asserted.

## Known limits

- **Runs once per stack.** The seed refuses a stack where it has already run. Other incidents may exist, and expect.sql counts whatever is there.
- **The 2,000 seeded records bypass the API**, so they have no create history. Criterion 2's "exactly one history entry" is checked as the entries added since S29.9.
- **Docker on Windows:** run from Git Bash.
