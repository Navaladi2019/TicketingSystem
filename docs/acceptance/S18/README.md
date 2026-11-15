# S18 acceptance kit: a tenant admin defines a number format and links it to a ticket type (TICK-669)

Kit task TICK-677 (PLAN-006 P6-12). Story branch `TICK-669/number-formats` at `26651ace` (every S18 task merged, TICK-868 / #705 and the `main` merge #708 included). Design: ADR-005 (§3, §4.2), ADR-005-E2, ADR-004-E9 (PLAN-005-E9 D-E9-5). Runbook: [`docs/data/record-counters-scope-key.md`](../../data/record-counters-scope-key.md). Verified on 2026-11-15 at 13:12Z on a fresh stack that started on the S17 build and was upgraded to S18.

**Result: PASS.** All five acceptance criteria and the E9 check pass. The run had 38 requests and 100 tests, all passed (`evidence/run-summary.json`). The forced-failure run (below) exits 1, as it must.

## What you can now do

As a tenant admin you write a number format such as `{prefix}-{yyyy}{MM}{seq:4}`, publish it, and link it to a ticket type. New tickets of that type get numbers like `VISA-2026110001`, counted per month in your tenant's time zone. A format that could clash with numbers the type already issued is refused, with the reason.

## Release R2 (M3)

S18 is release **R2**. Its first migration, M3, makes `(tenant_id, entity_definition_id, scope_key)` the primary key of `record_counters`. That is the point of no return in the S17 runbook: after M3 the pre-S17 build can never be deployed again. The kit starts on the S17 build (`ac253bad`, `main` with S17 merged, the commit the story branch was cut from) and upgrades the same stack, so M3 runs on a database that already has counters, as it will in an environment.

## Try it (about 25 minutes the first time, most of it two image builds; about 5 minutes once the images are built)

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell). Optional: put other ports in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `OPENOBSERVE_PORT`, `PROMETHEUS_PORT`) when another stack holds the defaults. The verified run used project `nava-s18` and ports 38080 / 38081.

   ```sh
   sh docs/acceptance/S18/tools/run-kit.sh
   ```

   It deletes the stack's volumes (`docker compose down -v`), then:
   1. starts a fresh stack on the **S17 build** from the sibling worktree `../s18-pre` and seeds it; adds `admin@tenant-b.test` to tenant B's admins group (`tools/kit.mjs admin-b`, one group sync);
   2. as `admin@tenant-a`, creates the ticket types `visa` (prefix `VISA`) and `permit` (prefix `PMT`), each with an access policy for tenant A's admins and service desk, and on the default numbering: `VISA0000001` (admin), `VISA0000002` (Alice, an agent), `INC0000001..2` in tenant A and in tenant B (`tools/kit.mjs pre`);
   3. upgrades the same stack to this checkout: the `migrate` service applies S18's 14 migrations, M3 first;
   4. runs the Bruno folder **S18 number formats** (`docs/acceptance/guide/bruno/S18/`), with tenant A's time zone set to `Pacific/Kiritimati` (UTC+14);
   5. reads the counters, the primary key and the audit rows from the database.

2. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | `admin@tenant-a` | `admin@tenant-a.test` |
   | `agent@tenant-a` | `alice@tenant-a.test` (service desk, not an admin) |
   | `admin@tenant-b` | `admin@tenant-b.test` (added by the kit: tenant B has no admin user in the realm) |

You should see `api-test: PASS` and `s18: PASS`. In Bruno desktop you can open folder S18 with the `local` environment, but its exact numbers only hold once, on a stack the script has just prepared.

## Acceptance criteria

The evidence is in `evidence/`: one JSON file per API request (request and response, no credentials; sign-ins are left out), `run-summary.json` with every test, and the `*.txt` database reads.

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| 1 | After `admin@tenant-a` publishes `{prefix}-{yyyy}{MM}{seq:4}` and links it to the Visa type, the next Visa ticket is `VISA-<yyyymm>0001` (month in the tenant's time zone), then `…0002`. Other types keep their numbers. | ✅ | S18N.5–S18N.15 | `S18N-5.json`: tenant A's zone set to `Pacific/Kiritimati`. `S18N-6.json`: format created (201, version 1, `refuse`). `S18N-7.json` / `S18N-8.json`: Alice (agent) can neither write nor link a format (403). `S18N-9.json`: Visa version 2 pins the format. **`S18N-10.json`: `VISA-2026110001` (admin). `S18N-11.json`: `VISA-2026110002` (Alice).** The month counter row is `scope_key = e1\|d:202611` with `period_end = 2026-11-30 10:00 UTC`, which is 1 December 00:00 in Kiritimati: the period follows the tenant's zone, not UTC (`counters-after-run.txt`). Other types: `S18N-12.json` tenant A incident `INC0000003`, `S18N-13.json` tenant B incident `INC0000003`; `S18N-14.json` / `S18N-15.json`: `VISA0000001` and `VISA0000002` from the S17 build are unchanged. |
| 2 | `{yyyy}{MM}{seq:4}` and `{prefix}X{seq:4}` are refused with `422 number_format_prefix_required`, naming the position. | ✅ | S18N.16, S18N.17 | `S18N-16.json`: 422, `errors[0]` = `/document/template`, `position:0`. `S18N-17.json`: 422, `/document/template`, `position:8` (the `X` after `{prefix}`). |
| 3 | Linking `{prefix}{yy}{seq:3}` to a type that already has default numbers is refused with `422 number_format_overlap`. | ✅ | S18N.18–S18N.21 | `S18N-18.json`: the format itself is valid (201). `S18N-19.json`: linking it to tenant A's incident type (which has `INC0000001..3`) is 422 `number_format_overlap`, rule `overlap:{prefix}{seq:7}` (the default numbering it clashes with). `S18N-20.json`: the type's version and default numbering are unchanged; `S18N-21.json`: the next incident is `INC0000004`. |
| 4 | Publishing an unrelated change to the type (adding a field) keeps its format and its numbering; existing tickets keep their numbers. | ✅ | S18N.22–S18N.26 | `S18N-22.json`: Visa version 3, `compat: compatible`, new property `passport_no`, and `number_format` still `{id, version: 1}` (the publish sent no `number_format`). `S18N-23.json`: the next Visa ticket is `VISA-2026110003`. `S18N-24.json`–`S18N-26.json`: `VISA-2026110001`, `VISA0000001` and `INC0000001` read back with their numbers. |
| 5 | `admin@tenant-b` gets the same `404` for tenant A's format as for a made-up id, and linking it gives `422 number_format_not_found`. | ✅ | S18N.4, S18N.27–S18N.31 | `S18N-4.json`: `admin@tenant-b` is in tenant B, group `/admins`. `S18N-27.json` / `S18N-28.json`: 404 `not_found`, no `instance`, the two bodies byte for byte equal, nothing of tenant A in them. `S18N-29.json` / `S18N-30.json`: linking tenant A's format, or a made-up id, to tenant B's incident type: 422 `number_format_not_found`, the same `detail` and `errors` (`/number_format`, `not_found`). `S18N-31.json`: tenant B's next incident is `INC0000004`. |

### E9 check (PLAN-005-E9 D-E9-5)

| Check | Result | Steps | Evidence |
|---|---|---|---|
| A create against a full `refuse` format returns 409 `number_space_exhausted` | ✅ | S18N.32–S18N.35 | `S18N-32.json` / `S18N-33.json`: `{prefix}-{seq:1}` (`refuse`) linked to the permit type. `S18N-34.json`: nine creates, `PMT-1`..`PMT-9`, fill it. **`S18N-35.json`: the tenth, as `admin@tenant-a`: 409 `number_space_exhausted`, `errors` `/entity`, `scope:e1`.** |
| The permit type's audit timeline then has exactly one `number_space_exhausted` entry with `epoch`, `scope_key`, `width`, `counter` | ✅ | S18N.36 | `S18N-36.json`: `GET /api/v1/audit/entries?subject_type=entity_definition&subject_id=<permit type>`: one entry, domain `record`, diff `[{"op":"add","path":"/number_space","value":{"counter":10,"epoch":1,"scope_key":"e1","width":1}}]`. `audit-exhausted-db.txt`: one row in the database, tenant A. |
| `admin@tenant-b`'s timeline shows none | ✅ | S18N.37, S18N.38 | `S18N-37.json`: tenant B's record timeline (4 entries, its own incident creates) has no `number_space_exhausted` entry. `S18N-38.json`: the same filter as S18N.36, as `admin@tenant-b`: an empty page. The kit also checks 0 rows for tenant B in the database. |

### M3 on an upgraded database

| Check | Result | Evidence |
|---|---|---|
| Before the upgrade, the S17 schema: 2-column primary key and the 3-column unique index beside it | ✅ | `schema-before-upgrade.txt`, `counters-before-upgrade.txt` (visa `next = 3`, incidents `next = 3`, `scope_key = ''`) |
| After the upgrade: 14 migrations applied, `record_counters_pkey` is `PRIMARY KEY (tenant_id, entity_definition_id, scope_key)`, the old second index is gone | ✅ | `migrate-log.txt` (`migrations_applied: 14`), `schema-after-upgrade.txt`; the kit fails if the key is not the 3-column one |
| The counters survive M3: the default-numbered Visa and incident rows carry on (`INC0000003`, `VISA0000001` unchanged) | ✅ | `counters-after-run.txt`, `numbers-after-run.txt` |

### Forced failure

`S18_FORCE_FAIL=1` makes one extra Visa create right after the format is linked and before the criterion-1 create (S18N.10's pre-request, acting as a second client), so the criterion-1 create gets `VISA-<yyyymm>0002`. The kit must then exit 1.

```sh
S18_FORCE_FAIL=1 S18_EVIDENCE=docs/acceptance/S18/evidence/forced-failure sh docs/acceptance/S18/tools/run-kit.sh
```

**Forced-failure run:** 2026-11-15 at 05:15Z on a fresh stack, same commit: `EXIT 1` (`s18: FAILED: S18`). 3 of 100 tests fail, and all three come from the one extra create. The extra create took `VISA-2026110001` (`run.log`: "S18 FORCED FAILURE: an extra Visa create took VISA-2026110001"). S18N.10 then got `VISA-2026110002` where it expects `…0001`, S18N.11 got `…0003` where it expects `…0002`, and S18N.23 got `…0004` where it expects `…0003`. Every other criterion still passes. Evidence: `evidence/forced-failure/` (`run-summary.json`, `run.log`, `result.txt`).

## Isolation check

- `admin@tenant-b` reading tenant A's format gets the same 404 as for a made-up id, byte for byte, naming nothing of tenant A (S18N.27, S18N.28).
- Linking tenant A's format to a tenant B type is the same 422 as a made-up id (S18N.29, S18N.30).
- Tenant A's format, its monthly counter and its exhausted format do not change tenant B's numbering (`INC0000003`, `INC0000004`) or its audit timeline (S18N.13, S18N.31, S18N.37, S18N.38).
- The SQL-level isolation of `number_formats`, `number_format_versions`, the epochs and the period counter rows is the isolation suite's (`internal/tenancy/isolation/registry.go`), run in CI. This kit does not repeat it.

## Not included

- Plugin writes of formats and `{field:…}` tokens: TICK-684 and S19.
- `republish-linked` with a linked type: the S18-API folder (TICK-672) covers its statuses with no linked type; the fan-out is covered by TICK-673's integration tests.
- Purge of expired period counters (TICK-675): its own integration tests; a period here ends on 1 December.
- A mixed R1/R2 fleet. The kit upgrades every service at once, as the runbook asks.
- A CI stack case for this kit. It needs two image builds of two commits.

## Known limits

- **Runs once per prepared stack.** The exact numbers hold only right after `run-kit.sh` has prepared the stack. A second Bruno pass on the same stack fails by design.
- **The month in the number equals the UTC month on most days.** On 2026-11-15 both are `202611`, so the number alone cannot show the time zone. The counter row's `period_end` does (1 December 00:00 in Kiritimati, `2026-11-30 10:00 UTC`). The test accepts the month in the tenant's zone at send time or at response time, so a run across midnight at the end of a month does not fail falsely.
- **A refused create still moves the counter.** After the 409 the permit counter is `next = 11` (`counters-after-run.txt`): the tenth number was taken and refused, as `refuse` does (the audit `counter: 10`). Every later create is refused the same way until a wider format is linked.
- **`admin@tenant-b` is kit-local.** The realm has no tenant B admin; `kit.mjs admin-b` creates `admin@tenant-b.test` (dev-only password) in organization `tenant-b` and in `/t/<B>/admins`. It is not in the shared seed, so other folders do not depend on it.
- **Docker on Windows:** run the script from Git Bash. The seed script needs `MSYS_NO_PATHCONV` unset (TICK-623); `run-kit.sh` handles this.
