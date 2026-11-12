# S17 acceptance kit: every ticket type keeps counting after the upgrade (TICK-664)

Kit task TICK-668 (PLAN-006 P6-04). Story branch `TICK-664/counter-scope-key` with this kit merged in, at `3626155` (M1 and M2, TICK-665, PR #581, re-stamped by TICK-773, PR #616; the 3-column upsert, TICK-666, PR #585; the compiler, TICK-667, PR #615; runbook, TICK-770, PR #612). Design: ADR-005 §4 and ADR-005-E1. Runbook: [`docs/data/record-counters-scope-key.md`](../../data/record-counters-scope-key.md). Verified on 2026-11-12 at 20:54Z on a fresh stack.

**Result: PASS.** All four acceptance criteria pass. The run had 15 requests and 42 tests, all passed (`evidence/run-summary.json`). The forced-failure run (below) exits 1, as it must.

## What you can now do

Nothing changes for you, and that is the point. After the upgrade, every ticket type keeps counting from where it was, whether it is a `gaps` or a `gapless` type. Creating tickets from many places at once still never gives two tickets the same number.

## Releases R1 and R2

S17 is release **R1**: migrations M1 and M2 add `scope_key` and `period_end` to `record_counters` and a second unique index, and both allocation paths upsert on `(tenant_id, entity_definition_id, scope_key)`. The old primary key stays, so the pre-S17 build (R0) still runs on R1's schema, and R1 can be rolled back to R0.

**M3 is not in this story.** It ships in S18 (release **R2**), only after R1 runs on every API and worker replica. M3 makes the 3-column key the primary key. After M3 the pre-S17 build can never be deployed again (it fails every create), so there is no rollback past it. The runbook says both (AC4).

## Try it (about 15 minutes, most of it two image builds)

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell). Optional: put other ports in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT`) when another stack holds the defaults. The verified run used project `nava-s17` and ports 28080 / 28081.

   ```sh
   sh docs/acceptance/S17/tools/run-kit.sh
   ```

   It deletes the stack's volumes (`docker compose down -v`), then:
   1. starts a fresh stack on the **pre-S17 build** (`ec863ece`: the `main` merged into the story branch, S10 included, so the upgrade applies S17's four migrations and nothing else) from the sibling worktree `../s17-pre`, and seeds it;
   2. as `admin@tenant-a`, sets tenant A's incident type to `gapless`; tenant B's stays `gaps` (the default). It then creates incidents in both tenants until each one's last number is `INC0000042` (`tools/kit.mjs pre`);
   3. upgrades the same stack to this checkout: the `migrate` service applies M1 and M2;
   4. runs the Bruno folder **S17 numbering** (`docs/acceptance/guide/bruno/S17/`);
   5. reads the counters and schema from the database and checks the runbook.

2. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | admin, tenant A (`gapless` incident type) | `admin@tenant-a.test` |
   | agent, tenant B (`gaps` incident type) | `bob@tenant-b.test` |

You should see `api-test: PASS` and `s17: PASS`. In Bruno desktop you can open folder S17 with the `local` environment, but its exact numbers only hold once, on a stack the script has just prepared.

**Why two tenants:** the prefix is unique per tenant, so one tenant cannot have both a `gaps` and a `gapless` type with prefix `INC`. Tenant A's incident type is the `gapless` one and tenant B's the `gaps` one. That way both give the literal `INC0000043` the story asks for, and the two counters, side by side, also serve as the isolation check.

## Acceptance criteria

The evidence is in `evidence/`. There is one JSON file per API request, with the request and the response and no credentials (sign-ins are left out). `run-summary.json` lists every test. The `*.txt` files hold the database reads.

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| 1 | The first incident after the upgrade is `INC0000043`, on a `gaps` type and on a `gapless` type | ✅ | kit.mjs pre, upgrade, S17.4–S17.8 | `pre-upgrade.json`: on the pre-S17 build, A set to `gapless` (201), then INC0000001..INC0000042 in each tenant. `counters-before-upgrade.txt`: both counters at `next = 43`, no `scope_key` column. `migrate-log.txt` and `schema-after-upgrade.txt`: 4 migrations applied (M1: `20261112193000`, `…193100`; M2: `…193200`, `…193300`; none of them before the upgrade, `schema-before-upgrade.txt`), `scope_key text NOT NULL DEFAULT ''`, `period_end`, `record_counters_scoped_key` beside the old `record_counters_pkey`. `counters-after-upgrade.txt`: same rows, `scope_key = ''`, `next = 43`. `S17-4.json` / `S17-5.json`: the modes survived the upgrade. **`S17-6.json`: tenant A (gapless) `INC0000043`. `S17-7.json`: tenant B (gaps) `INC0000043`.** |
| 2 | 8 clients creating incidents of one type at once still get unique numbers | ✅ | S17.9, S17.10 | `S17-9.json`: 8 concurrent creates on the `gapless` type, all 201, exactly `INC0000044`..`INC0000051`. `S17-10.json`: 8 concurrent creates on the `gaps` type, all 201, 8 different numbers (`INC0000044`..`INC0000051`). `unique-numbers.txt`: 52 records and 52 distinct numbers in each tenant. |
| 3 | Tenant A cannot take a number from tenant B's counter (the isolation suite is green on the new key) | ✅ | S17.11–S17.15; CI | Through the API: Bob gets the same `404 not_found` for tenant A's `INC0000043` on read (`S17-11.json`) and on change (`S17-12.json`), naming nothing of tenant A; the record is unchanged (`S17-15.json`). Each tenant's next number follows its own series (`S17-13.json`: A `INC0000052`; `S17-14.json`: B `INC0000052`); neither tenant's 9 or 10 creates moved the other's counter (`counters-after-run.txt`: both `next = 53`, 52 records each). In SQL: the isolation suite's `record_counters` rows run on the 3-column key since TICK-666 (`internal/tenancy/isolation/registry.go`): read B's counter by type and scope gives own rows only, while upserting into B's counter (`ON CONFLICT (tenant_id, entity_definition_id, scope_key)`), starting a counter in A for B's type, and moving B's counters are all Denied. They were green in the CI run of PR #585 ([43267146517](https://github.com/Navaladi2019/TicketingSystem/actions/runs/43267146517)). |
| 4 | The runbook says M3 comes in a later release, and after M3 the pre-S17 build cannot be deployed again | ✅ | run-kit.sh | `ac4-runbook.txt`: `docs/data/record-counters-scope-key.md` lines 32 ("then and only then run M3 with R2"), 146 ("R2 / M3 → anything earlier: **Not possible.**"), 148 ("**M3 is the point of no return.**", followed by "R0 can never be deployed again after M3"), 156 ("M3 (S18, for reference)"). The kit fails if these lines go. |

### Forced failure

`S17_FORCE_FAIL=1` makes one extra tenant A create after the upgrade and before Bruno, so tenant A's next number is `INC0000044`. The kit must then exit 1. Verified on 2026-11-12 at 20:56Z on a fresh stack, same commit: `EXIT 1`, and 3 of 42 tests fail, all of them the forced create's own consequences. S17.6 gets `expected 'INC0000044' to deeply equal 'INC0000043'`, S17.8 reads the same, and S17.9's gapless range shifts to 45–52. Tenant B and the isolation steps still pass. Evidence: `evidence/forced-failure/` (`forced-extra-create.json`, `run-summary.json`, `run.log`, `result.txt`: `s17: FAILED: S17`).

```sh
S17_FORCE_FAIL=1 S17_EVIDENCE=docs/acceptance/S17/evidence/forced-failure sh docs/acceptance/S17/tools/run-kit.sh
```

## Isolation check

- Bob (tenant B) reading or changing tenant A's `INC0000043` by its id gets the same 404 as for a record that does not exist; the body names nothing of tenant A (S17.11, S17.12), and the record is unchanged (S17.15).
- The two tenants' counters share the type key `incident`, the prefix `INC` and `scope_key = ''`, and still number independently: 9 creates in B and 10 in A after the upgrade, and each tenant's next number is the one after its own last (S17.13, S17.14).
- The SQL-level checks (a tenant taking from another tenant's counter on the 3-column key) are the isolation suite's, in CI (criterion 3).

## Not included

- M3, the switch of the primary key, and any non-default number format: S18 (R2), PLAN-006 D-6-2.
- The template compiler (TICK-667): pure Go with no runtime path in R1; its tests are its own.
- A mixed R0/R1 fleet. The kit upgrades every service at once. `TestP601CounterScopeExpandWindow` and the ADR-005-E1 first-create race tests cover both binaries writing to the same counter.
- A CI stack case for this kit. It needs two image builds of two commits.

## Known limits

- **Runs once per prepared stack.** The exact numbers (43, 44–51, 52) hold only right after `run-kit.sh` has prepared the stack. A second Bruno pass on the same stack fails by design.
- **The 200 ms allocation budget can fail a `gaps` create on a cold or busy host (seen once in the kit, once in CI).** In the first forced-failure run (2026-11-12 18:00Z), Bob's first create right after the upgrade got `503 retry_later`: "the number allocation took longer than 200ms: … take record number: timeout: context deadline exceeded" (nava-core log, request `6d0b941a…`). That was the process's first allocation on the dedicated allocator pool, after the upgrade had also recreated PgBouncer. As `gaps` mode allows (D-3-9), the allocation committed, so `INC0000043` was used up in tenant B. The clean run, and restarting nava-core alone three times, did not reproduce it (first create 47–80 ms). Restarting PgBouncer and nava-core together brought the first create to 392 ms (the 200 ms budget covers only the allocation, not the whole request). The 200 ms budget (`DefaultAllocTimeout`, including pool acquire) and the allocator pool with no warm connections both come from S10 (P3-36e); S17 does not change them. The same error failed `TestUniqueConcurrentCreatesOneWins` (`internal/records/pipeline/numbering_integration_test.go:449`) in this PR's first CI run ([43299468911](https://github.com/Navaladi2019/TicketingSystem/actions/runs/43299468911)). Fix task: **TICK-778** (warm the allocator pool before ready, and give the pipeline integration tests an explicit allocation budget).
- **Docker on Windows:** run the script from Git Bash. The seed script needs `MSYS_NO_PATHCONV` unset (TICK-623); `run-kit.sh` handles this.
