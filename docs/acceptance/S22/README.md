# S22 acceptance kit: unflag a filterable field, raise the quota with operator approval, retire the JSONB scan (TICK-740)

Kit task TICK-754 (PLAN-007 P7-14), which also carries the S29 kit (`docs/acceptance/S29/`, formerly TICK-928). Design: ADR-006 D6 (scan removal), D8 (busy), D11 (removal), D12 (quota). Story branch: `TICK-740/search-finish`, shared with S29.

**Status (2026-11-28): S22 PASSES on a fresh stack: all 4 criteria. fresh stack `nava-s22s29`, 2026-11-27 22:46Z to 22:51Z, from `a48a82e7` plus this kit's fixes (the commit after it on PR #918), `run-s22-s29.sh`. Bruno S22: 24 requests, 75/75 tests. `verify.sql`: every check `ok`. Evidence: `evidence/`.**

## What you can now do

- Remove `"x-nava-filterable": true` from a field and publish. Filters on it answer 400 `field_not_filterable` at once, and the platform deletes its index in the background.
- While that cleanup runs, flagging the field again answers 409 `field_index_busy`. Once it ends, the flag publishes and the field is indexed again.
- A sixth flagged field on one ticket type answers 422 `index_quota_exceeded`. The operator can raise the tenant's limit to 10 per type, with a reason, and the change is audited.
- The seeded ticket types flag the fields the earlier kits filter on (`priority` on incidents), so those filters read the index. The old JSON scan is gone.

## Try it (about 10 minutes, most of it the image build)

From the repository root, on the story branch, Docker running, Git Bash or any POSIX shell, Node.js 22:

**In GitHub Actions:** the stack case `deploy/compose/test/cases/s22_s29_acceptance.sh` (job `stack`, order 70, profile `operator`) runs both kits in the `ci` workflow's stack job on every pull request into the story branch. Until every S22 and S29 feature is on the branch it prints `SKIP` with what is missing. On a laptop:

```sh
sh docs/acceptance/S22/tools/run-s22-s29.sh    # S22 and S29 on one fresh stack
sh docs/acceptance/S22/tools/run-kit.sh        # S22 alone on a fresh stack
```

`run-kit.sh` deletes the stack's volumes (`docker compose down -v`), then:

1. starts a fresh stack with the `operator` profile, runs `docs/acceptance/guide/tools/seed.sh` and `tools/tunnels.sh up` (the operator API at `{{onboard_url}}`);
2. `tools/kit.mjs setup`: as `admin@tenant-a`, creates the type `s22_asset` (prefix `S22A`) with an optional `s22_code` (string, not flagged) and an ACL for tenant A's admins group;
3. `tools/seed.sql`: 20,000 `s22_asset` records, `s22_code` K00 to K19, exactly 1,000 each, ids `md5('s22/' || n)::uuid`;
4. AC3, static half: `git grep` for `DataFilterTimeout`, `LegacyScanFields` and `boundDataFilter` in non-test Go code. Each must have 0 hits (`evidence/ac3-static.txt`);
5. Bruno folder **S22 unflag and quota** (`docs/acceptance/guide/bruno/S22/`);
6. `tools/verify.sql`: the database behind criteria 1, 2 and 4. Every `check|` line must end `ok`.

You should see `s22: PASS`. The folder runs once per prepared stack: it creates `s22_asset` and raises tenant A's limit.

| Story user | Seeded user | What they do here |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test` | publishes the flags, creates and edits records |
| `agent@tenant-a` | `alice@tenant-a.test` | filters the seeded incident type (AC3) |
| `operator` | `operator@nava.test` (client `nava-operator-cli`) | raises tenant A's per-type limit |

Passwords: dev-only seed users, `docs/acceptance/guide/bruno/environments/local.bru`.

## Acceptance criteria

| # | Criterion | Steps | What is asserted | Result |
|---|---|---|---|---|
| 1 | Publishing without the flag makes filters answer 400 `field_not_filterable` at once; flagging again while the removal runs answers 409 `field_index_busy`; after removal it can be flagged again | S22.8 to S22.14; verify.sql | S22.10: the unflag publishes 201, `compatible`. S22.11: right after it, 400 `field_not_filterable` with `errors[]` at `/filter/s22_code`. S22.12: the flag again, 409 `field_index_busy`. S22.13: the flag is re-sent every 3 s with a new `Idempotency-Key`: every answer is 409 until one is 201, and the type is exactly one version past the unflag (no 409 wrote a version). S22.14: after the new backfill, K07 lists its records again. verify.sql: `s22_code` has one registry row, `ready`. | ✅ S22.9: indexed after 24 s. S22.11: 400 at once. S22.12: 409 212 ms after the unflag. S22.13: 11 attempts answered 409 `field_index_busy`, then the 12th answered 201 at 33 s (version 4 = unflag + 1). `verify.txt`: one `ready` row. |
| 2 | A sixth filterable field on one type answers 422 `index_quota_exceeded`; after the operator raises the limit to 10 (audited) it publishes | S22.15 to S22.24; verify.sql | S22.15: a new type with `q1` to `q5` flagged, 201. S22.16: flagging `q6` too, 422 `index_quota_exceeded`, `errors[]` at `/schema/properties/q6`. S22.17: the tenant admin on the operator route, 403 `forbidden`. S22.18 to S22.20: the operator asks for 4, for 11, and with no `reason`, each 400 `validation_failed`. S22.21: an unknown tenant, 404 `not_found`. S22.22: the operator sets 10 with a reason, 200. S22.23: a replay, 200. S22.24: the sixth flag publishes, 201. verify.sql: tenant A's `tenant_limits` row is 10; it has exactly one audit entry, which carries S22.22's reason and an actor (none from the refusals or the replay); tenant B has no row. | ✅ S22.16: 422, `errors[]` at `/definition/schema/properties/q6`. S22.17: 403. S22.18–S22.20: 400. S22.21: 404. S22.22: 200, limit 10. S22.24: 201. `verify.txt`: limit 10, exactly 1 audit entry (`config`/`update`/`tenant_limits`) carrying the reason, no tenant B row. |
| 3 | The seeded types' filters used by earlier kits still work through the index; `DataFilterTimeout` and the JSONB filter path are gone | S22.4 to S22.6; run-kit step 4 | S22.4: the incident type's `priority` is flagged from the seed template. S22.5: Alice's `filter[priority]=1`, `filter[priority][in]=1,2` and `filter[priority][gte]=3` each list exactly the incidents a full unfiltered read gives when filtered in the test. S22.6: an unflagged seed field (`short_description`), 400 `field_not_filterable`. Step 4: no hits. | ✅ S22.4: `priority` flagged. S22.5: the three filters equal a full read of the 2,050 incidents Alice sees: 25 at `=1`, 50 at `in 1,2` and at `lte 2`. S22.6: 400. `ac3-static.txt`: 0 hits for all three patterns. |
| 4 | Records written during removal leave no orphan side rows | S22.11, S22.14; verify.sql | S22.11, during the removal: 3 creates with K07 and 3 seeded K08 records moved to K07. S22.14: K07 lists 1,006 (1,000 + 3 + 3), K08 lists 997. verify.sql: 0 side rows in `rfv_text`, `rfv_num`, `rfv_ts` and `rfv_bool` whose field has no registry row, and `s22_code`'s rebuilt index equals the JSON of all 20,003 records (0 missing, 0 surplus). The concurrency and race suite is TICK-750 (CI-5) and TICK-753, in CI. | ✅ S22.11: 3 creates and 3 edits during the removal, done 142 ms after the unflag. S22.14: K07 1,006, K08 997. `verify.txt`: 0 orphan side rows in any table; 20,003 records, 0 missing, 0 surplus. |

## What the fresh-stack run fixed in the kit (2026-11-27)

- **The stack seed now registers template-flagged fields.** `docs/acceptance/guide/tools/seed.sh` inserts the seed types with SQL, not through onboarding, so `priority` (and `risk` on change) was flagged in the schema with no registry row. Every filter on it answered 400 `field_not_filterable` for both seeded tenants. Onboarding itself registers them (`RegisterSeedFilterable`); `seed.sh` now does the same (straight to `ready`: the seeded types have no records). Test tooling only; product code is right.
- Publish errors point at `/definition/schema/...` (S22.16); a tenant admin on the operator route is 403; the access policy is ACL format 2; a type's number prefix is letters only (`SAST`).

## Contracts confirmed since day 1

| Where | Shape | Source |
|---|---|---|
| S22.17 to S22.23 | `PUT /api/v1/operator/tenants/{tenant}/limits` on the operator API, body `{filterable_fields_per_type (5-10), reason}`, both required; a tenant token is 403; an unknown tenant 404; the value already in force writes nothing; the audit entry has subject `tenant_limits`, the operator as actor and the reason | TICK-751 scope 3, built in TICK-916 (PR #921, `api/paths/operator.yaml`); Technical Lead, TICK-754, 2026-11-26 |
| S22.12 | the removal still runs about 1 s after the unflag: it goes through the relay and the coordinator, then deletes 20,000 side rows in batches of 1,000 | ADR-006 D11, TICK-750 scope 2 (PR #933) |
| S22.5 | the `in` syntax is `filter[f][in]=a,b`, as S10's kit uses for `assignment_group` | S10 kit |

## Isolation check

The kit stays in tenant A, except that verify.sql checks tenant B has no `tenant_limits` row. The cross-tenant attacks (unflag B's field, a crafted remove workflow, writing `tenant_limits` as a tenant role) are TICK-753's suite, in CI. The kit doesn't repeat them.

## Not included

- p95 and the removal's duration as a budget. No performance gate (owner, 2026-11-26).
- **The S21 kit re-run** is not on this stack: its seed refuses a tenant with incidents and writes 400,000. It runs on its own fresh stack (`docs/acceptance/S21/tools/run-kit.sh`). Since TICK-752 flags `priority` in the seed template, S21's `kit.mjs setup` removes that flag in its first publish and `run-kit.sh` waits for the removal before the bulk seed (which skips the index trigger), so S21.4 and S21.8 hold unchanged.

## Known limits

- **Runs once per prepared stack.** `s22_asset` must not exist yet, and the limit raise stays.
- **The seed bypasses the API**, as S21's does. Its records have no audit entries or create history. Every record the kit creates or edits goes through the API.
- **AC3 compares against what exists.** Run alone, a fresh stack has no tenant A incidents, so S22.5 compares empty lists. `run-s22-s29.sh` runs S29's prep first, so S22.5 compares about 2,050 incidents, 25 of them priority 1.
- **Docker on Windows:** run from Git Bash. The scripts set `MSYS_NO_PATHCONV` (TICK-623).
