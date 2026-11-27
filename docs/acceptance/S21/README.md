# S21 acceptance kit: filter a large ticket type on a flagged field in milliseconds (TICK-739)

Kit task TICK-749 (PLAN-007 P7-09). Story branch `TICK-739/filterable-fields` at `7284e2f2` (every task merged: TICK-741 to TICK-748, the P7-07 fixes TICK-989 to TICK-992, and the ADR-006-A1 budget row, TICK-1001). Design: ADR-006 D1 to D10, D12, D13 as amended by ADR-006-A1, D14. Budget row: `docs/budgets.md`, "Filtering on a flagged data field (S21)".

**ADR-006-A2 A2-8 (7) MET (TICK-1049, 2026-11-22 08:43Z to 09:40Z): two consecutive valid green fresh stacks, `evidence/a2-stack1/` and `evidence/a2-stack2/`, both built from `49c3ab50`** (this kit + `TICK-1046/count-budget` at `39921c66`, with TICK-1084). Each run's `host-check.txt` shows a quiet host at start, and its `p95.txt` ends `run validity: VALID` and `kit report: PASS`. Both Bruno folders: 0 failed tests. Both `verify.sql`: side tables equal the JSON scan, and tenant B has no index rows.

| Stack | ≥ 40% ACL p95 (350) | its `unknown` | `rfv_num` all-visible at the cell | page plan (`page-plan.txt`) | admin no-filter control p95 (≤ 80) | Alice no-filter control p95 |
|---|---|---|---|---|---|---|
| `a2-stack1/` | **85** | 0% (584 `capped`) | 88.5% | ordered `records_list_group` walk, 115 rows, 2.1 ms | 42 | 54 |
| `a2-stack2/` | **66** | 0% (585 `capped`) | 100% | ordered, 115 rows, 3.8 ms | 36 | 162 |

Every other D13 cell passed on both stacks (`p95.txt`). Another job's `nava-aclperf-pg-*` container came up while each run was already going (`other-containers.txt`). The rule only checks the host at start, plus the control, so this is recorded and doesn't void either run. A2-7 expects `unknown` to be high on a fresh stack, but it was 0% in this cell on both: once TICK-1084 made the page cheap, the capped count fits well inside its budget.

**After TICK-1084 (2026-11-22 04:38Z to 07:03Z, `fadf9f60` = this kit + `TICK-1046/count-budget` at `39921c66`): the ≥ 40% ACL cell held on all 4 fresh stacks (132, 247, 173, 133 ms p95, limit 350; every response `capped`, `unknown` 0%).** On every stack its page plan walks `records_list_group` in order: 115 rows read, 1.8 to 7.5 ms (`page-plan.txt`). **A2-8 (7) is still 0 of 2.** Only the first stack was green end to end. On the other three, cells that A2 doesn't change missed on a busy host: the 150 ms cells and the 0.5-2% band. Other agents' `go vet` / integration jobs were running, and the admin no-filter control was over P3-36's 80 ms gate on two of them. On the fourth stack a connection to nava-core timed out during the controls. The 08:00-10:00Z host freeze stopped further runs.

| Run (evidence) | ≥ 40% ACL p95 (350) | page plan | admin no-filter control p95 | Failed cells (limit) |
|---|---|---|---|---|
| `a2-fix-green-1/` | 132 | 1.8 ms, ordered | 62 | none: **green** |
| `a2-fix-red-1/` | 247 | 7.5 ms, ordered | **144** | Alice `priority=1` 426 (350), admin two-filter 227 (150), Alice two-filter 264 (150), admin `priority=3` 217 (150) |
| `a2-fix-red-2/` | 173 | 2.5 ms, ordered | **92** | admin `priority=1` 427 (350), admin two-filter 278 (150), Alice two-filter 161 (150) |
| `a2-fix-red-3/` | 133 | 2.2 ms, ordered | not measured (ETIMEDOUT) | admin `priority=1` 474 (350), admin `priority=3` 229 (150), admin two-filter 213 (150), Alice two-filter 180 (150); the controls never ran |

**Before TICK-1084 — ADR-006-A2 result (TICK-1049, 2026-11-22): A2-8 (7) NOT MET. 0 of the 2 consecutive green fresh stacks it needs.** Every stack was built from `TICK-1046/count-budget` plus this kit: 1 green at `000420bc`, not counted (its script exited 1 on a kit cleanup bug, fixed in `4626a2dc`), then 3 red at `4626a2dc`. The `total_state` contract held on every response of every run. In S21.19's ≥ 40% ACL cell, no response was `unknown`: on these stacks the count takes about 5 ms. **The cell now misses on the page statement, not the count.** Its plan reads all ~20k service-desk records through `records_list_group` and sorts them, because the ACL branch compiles to `assignment_group_id = ANY($3::uuid[])`. With the single group as a scalar `=`, the same statement walks the index in order: **0.99 ms instead of 61 ms** (`evidence/a2-red-3/page-plan-*.txt`). Fix routed through the Technical Lead on TICK-1049.

| Run (evidence) | SHA | Result | ≥ 40% ACL p95 (350) | its `unknown` | `rfv_num` all-visible at the cell | admin no-filter control p95 | other jobs on the Docker VM |
|---|---|---|---|---|---|---|---|
| `a2-pre-trap-fix/` | `000420bc` | every check green; exit 1 (trap bug); not counted | 115 | 0% | 100% | 33 | none |
| `a2-red-1/` | `4626a2dc` | red: ACL cell, admin two-filter 431 (150) | **429** | 0% | 84% | 52 | TICK-1071 integration tests |
| `a2-red-2/` | `4626a2dc` | red: admin `priority=1` 618 (350), admin `priority=3` 344 (150) | 277 | 0% | 100% | 50 | TICK-1071 integration tests, whole run |
| `a2-red-3/` | `4626a2dc` | red: ACL cell only | **361** | 0% | 100% | 28 | none during the cells |

On stack `a2-red-3`, after the run: Alice's `priority=3` took 72 ms single-client and 227 ms p50 with 8 clients. With `limit=1` it took 20 ms, count included (`probe-after-run.txt`). Server statement times: page 49-62 ms, count 4.5-5.3 ms (`statement-times-single-client.txt`). A2-2 assumed the page's worst case was 20.6 ms.

**Before ADR-006-A2 (re-verified after TICK-1023 merged `main` into the story branch): criteria 1, 3, 4 and 5 PASS; criterion 2's p95 FAILS on this host.** Fresh stack built from `24bdd9b4` (this kit merged with the story branch after PR #799), project `nava-s21`, `docker compose down -v`, run 2026-11-20 18:00Z to 18:30Z. `evidence/` is this run. The Bruno folder ran 19 requests and 99 tests: 94 passed, and the 5 that failed are all p95 cells of S21.19. Every row check passed, including criterion 2's range filter. `verify.sql` found both index tables equal to a JSON scan of all 400,006 tenant A incidents (0 missing, 0 surplus, 0 wrong), and tenant B has 0 index rows. The seed took 25 s. The default ACL from `seed.sh` and the kit's mask publish both pass the ACL check that changed with TICK-940.

**p95 across the three fresh stacks so far:** D13 held on 1 of 3. The two that missed both ran next to another stack on the same Docker VM, and the misses are not explained by load alone (single-client numbers below). Fix task: **TICK-1022**. The story should not be accepted on criterion 2 until it is resolved or the budget owner rules on it.

| Cell (limit) | Stack 1, 10:51Z | Stack 2, 11:31Z | Stack 3, 18:29Z (`evidence/`) |
|---|---|---|---|
| admin `priority=1`, 0.99% (350) | 330 | 144 | **426** |
| admin `priority=3`, 48% (150) | 149 | 59 | **169** |
| admin two-filter range, 0.14% (150) | **201** | 69 | **251** |
| Alice `priority=1`, 0.99% ACL (350) | 259 | 153 | 299 |
| Alice `priority=3`, 48% ACL (350) | **578** | 119 | **570** |
| Alice two-filter range, 0.14% ACL (150) | **240** | 146 | **336** |
| admin no filter (control) | 103 | 47 | 108 |
| single-client Alice `priority=3` / `priority=4`, p50 | 217 / 61 | 34 / 39 | 148 / 131 |

Stack 2's run (11:05Z to 11:32Z, images from `d108afa8`, before the merge) passed all 99 tests; its evidence is `evidence/` at commit `462f108c`. Its numbers are in "The p95 cells" below, and stack 1's in "Plan-dependent p95".

## What you can now do

- Mark a top-level field of a ticket type as filterable (`"x-nava-filterable": true` in the schema you publish). The platform copies its existing values into an index in the background.
- While it does, a list that filters on the field answers 422 `field_not_ready` with `index_progress`, the whole percentage done. When it is done, the list filters on the field: equality, a list of values, and number and date ranges.
- Filtering a 400,000-incident type takes tens to a few hundred milliseconds, not a second.
- A field you did not flag is refused (400 `field_not_filterable`), and so is a field your access policy hides from you (400 `masked_field`). Neither is ever a scan of every record.

## Try it (about 35 minutes, most of it the two backfills)

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell) and Node.js 22. If another stack holds the default ports, put other ones in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT`). The run in `evidence/` (stack 3) used project `nava-s21` with ports 28080 (nava-core) and 28081 (Keycloak) from `.env`, because another stack held the defaults. Stack 2 used the same `.env`.

   ```sh
   sh docs/acceptance/S21/tools/run-kit.sh
   ```

   It deletes the stack's volumes (`docker compose down -v`), then:
   1. starts a fresh stack from this checkout and runs `docs/acceptance/guide/tools/seed.sh` (tenants A and B and their users);
   2. `tools/kit.mjs setup`: as `admin@tenant-a`, adds an optional `due_at` (date-time) to the incident type, not flagged, and masks `description` to tenant A's admins group;
   3. `tools/seed.sql`: writes **400,000 incidents in tenant A** and 50 in tenant B, as the database superuser, in one statement per tenant (see "The seed" below; 25 s in the run in `evidence/`, 36 s on stack 2). `tools/expect.sql` then computes what a plain JSON scan of the records gives for each of the kit's filters, and the script passes those numbers to Bruno;
   4. runs the Bruno folder **S21 filterable fields** (`docs/acceptance/guide/bruno/S21/`). It flags `priority` and `due_at`, watches the backfill, then filters;
   5. `tools/verify.sql`: compares both fields' index tables with a JSON scan of every tenant A incident (none missing, none extra, none wrong) and checks that tenant B has no index rows.

   `S21_KEEP_STACK=1` skips step 1 on a stack that is up and has had nothing but `seed.sh`. `S21_P95_SECONDS` (default 15) sets the length of each p95 cell.

2. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user | What they do here |
   |---|---|---|
   | `admin@tenant-a` | `admin@tenant-a.test` | publishes the flags; reads every incident |
   | `agent@tenant-a` | `alice@tenant-a.test` (service-desk group) | lists; under the default incident ACL she reads about 5% of the incidents |
   | `agent@tenant-b` | `bob@tenant-b.test` | the other tenant |

You should see `api-test: PASS` and `s21: PASS`. The folder runs once per prepared stack: S21.8 publishes the flags, so a second pass fails at S21.4.

## The seed

`tools/seed.sql` writes records directly, as the P7-07 measurement does (`perfSeedSQLWith`): through the API, 400k creates take about 35 minutes. Every value is a hash of the record's ordinal, so reruns give the same rows:

| Field | Values in tenant A |
|---|---|
| `priority` | 1: 1% · 2: 1% · 3: 48% · 4: 50% |
| `due_at` | a whole minute in 2026, spread evenly |
| assignment group | service-desk for 5%, none for the rest (only the admins read those) |
| marked records | one with `due_at` 2026-12-31T23:59:59Z and no group (Alice cannot read it); one with 23:57:59Z in service-desk (she can). 23:58:59Z exists nowhere. |

Tenant B gets 50 incidents, all priority 1, on its unchanged incident type.

## Acceptance criteria

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| 1 | While indexing, `filter[priority]=1` answers 422 `field_not_ready` with a whole-percentage `index_progress`; once ready, 200 with exactly the rows a JSON scan gives | ✅ | S21.9, S21.11, S21.13, S21.14; verify.sql | `S21-9.json`: 422 `field_not_ready`, `index_progress` 0 for both fields right after the publish (`S21-8.json`, version 3, `compatible`), with no items and no total. `S21-11.json`: 73 polls every 15 s. The backfills ran one at a time: one field rose (0, 3, 6, 9, 12, 14 …) while the other stayed at 0, and both were ready at 1,089 s. `verify.txt`: due_at flipped to `ready` at 18:17:29Z, priority at 18:26:40Z. The order is the coordinator's, not something the kit asserts. Every answer while indexing was a 422 with a whole number from 0 to 99, never decreasing. `S21-13.json`: as the admin, 3,955 records over 20 pages, the same ids as the JSON scan (`expect.txt`, md5 of the sorted ids), `total` 1,000 with `total_capped`. `S21-14.json`: as Alice, 203 records, the same ids as the scan of what she may read, `total` 203. `verify.txt`: both side tables equal the JSON scan. |
| 2 | `filter[priority][gte]=2&filter[due_at][lt]=<date>` returns the right rows, and the list p95 for the kit's filters is within D13 as amended by ADR-006-A1. **Cut 2026-11-26 (owner-approved):** the ≥ 40% band with a narrowing ACL keeps rows, isolation and `total_state` here; its 350 ms p95 is recorded, not asserted, and moved to S21c (TICK-1159) | Rows ✅, p95 ❌ (TICK-1022) | S21.15, S21.16, S21.19 | `S21-15.json`: admin, 555 records, the scan's ids. `S21-16.json`: Alice, 25 records, the scan's ids. Every listed record has priority ≥ 2 and `due_at` before 2026-01-01T12:00:00Z. **`S21-19.json`: 5 of the 6 scored cells fail**: admin `priority=1` 426 ms (350), admin `priority=3` 169 (150), admin two-filter 251 (150), Alice `priority=3` 570 (350), Alice two-filter 336 (150). Alice `priority=1` passes at 299 (350). Across three fresh stacks D13 held on 1 (stack 2); see the table at the top and "Plan-dependent p95". |
| 3 | An unflagged field answers 400 `field_not_filterable`; a masked field 400 `masked_field` | ✅ | S21.5–S21.7 | `S21-5.json`: Alice filtering on `description` (masked to admins) gets 400 `masked_field`, with no items and no total. `S21-6.json`: the admin, whom the mask admits, gets 400 `field_not_filterable` with `errors[]` at `/filter/description`. `S21-7.json`: `due_at` before it is flagged also gets 400 `field_not_filterable`, never a scan. |
| 4 | Records created or edited during and after indexing are found (no record missed across the flip) | ✅ | S21.10, S21.12; verify.sql | `S21-10.json`: while both fields were backfilling (422 before and after), 3 creates and 3 edits onto `due_at` 2026-12-30T12:00:17Z, plus one record created on it and moved to …:18Z. `S21-12.json`: after both were ready, 2 more creates and 2 more edits. The filter lists exactly the 10, `total` 10. The moved record is not among them, and …:18Z lists exactly it. `verify.txt`: 10 records on …:17Z, first write 18:08:32Z (during), last 18:26:41Z (after both flips: due_at flipped to `ready` at 18:17:29Z, priority at 18:26:40Z). The last write followed the second flip by about 1 s. S21.12's writes start only after S21.11 has seen both fields answer 200, so they are after the flip by construction. |
| 5 | Tenant B never matches tenant A's filters; a value only in a record A's agent cannot see gives the same hits and `total` as a value that exists nowhere | ✅ | S21.17, S21.18; verify.sql | `S21-17.json`: the admin finds `due_at` 2026-12-31T23:59:59Z (`total` 1). Alice gets 200, `items: []`, `total` 0, a body identical to the one for 23:58:59Z, which no record has. Her control value 23:57:59Z gives `total` 1. `S21-18.json`: Bob lists tenant B's 50 priority-1 incidents (B flagged nothing, so this is still the legacy scan there). None of them is in the admin's 3,955; the all-tenant scan has 4,005 = 3,955 + 50. `verify.txt`: tenant B has 0 index rows. |

### The p95 cells (criterion 2)

Each cell: 8 clients list with the filter for 15 s after 2 s of warm-up. They are paced to 40 requests a second in total, because the per-tenant rate limit is 50 a second and a 429 would fail the cell. Latency is measured at the client, through nava-core and PgBouncer, with the default page of 50. Limits are ADR-006 D13 as amended by ADR-006-A1: 350 ms for the 0.5–2% band, with or without a narrowing ACL, and 150 ms for every other filter, except the ≥ 40% band with a narrowing ACL (Alice `priority=3`): since 2026-11-26 its p95 is recorded, not asserted, and its 350 ms target is scored in S21c (TICK-1159). **No 2–40% filter is asserted.** Those bands are unmeasured until TICK-1009 (S21b). The 0.25% `in` cells are not rerun here; TICK-1002 does that.

Stack 2's run (before the merge; stack 3's are in the table at the top), 8 clients, about 39 requests/s, only 200s:

| Caller | Filter | Share of the type | Limit | p50 | p95 | Result |
|---|---|---|---|---|---|---|
| admin | `priority=1` | 0.99% (0.5–2%) | 350 | 93 | **144** | ✅ |
| admin | `priority=3` | 48% (≥ 40%), no narrowing ACL | 150 | 31 | **59** | ✅ |
| admin | `priority>=2` and `due_at<2026-01-01T12:00Z` | 0.14% | 150 | 48 | **69** | ✅ |
| Alice | `priority=1` | 0.99%, narrowing ACL | 350 | 106 | **153** | ✅ |
| Alice | `priority=3` | 48%, narrowing ACL | 350 | 61 | **119** | ✅ |
| Alice | `priority>=2` and `due_at<2026-01-01T12:00Z` | 0.14%, narrowing ACL | 150 | 95 | **146** | ✅ (3% margin) |
| admin | no filter (control) | — | reported | 26 | 47 | — |
| Alice | no filter (control) | — | reported | 76 | 265 | — |

### Plan-dependent p95: the count is budgeted now (ADR-006-A2)

Since ADR-006-A2 (TICK-1046 to TICK-1049), every list with a data filter runs its count under a 200 ms `statement_timeout` (`CountBudget`, A2-2). It runs after the page, inside `SAVEPOINT list_count`. When the count runs out of budget, the list still answers 200 with the full page, `total: null` and `total_state: unknown`. So the ≥ 40% ACL cell no longer depends on which plan the planner picks for the count. Its worst case is the page plus 200 ms, whatever the statistics or the visibility map. What used to be a 0.6-1 s list after a backfill is now an `unknown` total. **`unknown` is expected after a backfill.** Until autovacuum has marked `rfv_num`'s pages all-visible, a broad ACL count reads the heap and can run past the budget. A2-7's alert on more than 5% `unknown` fires in that window, and that is the signal working, not an outage.

What the kit asserts since TICK-1049:

- Every list response it reads (S21.12 to S21.19, each page of S21.13 to S21.16, and every 200 of every p95 cell) meets the A2-1 contract. `total_state` is `exact`, `capped` or `unknown`. `total` is an integer for `exact` (0 to 1,000) and `capped` (1,000), and `null` for `unknown`, never 0. `total_capped` is true exactly when the state is `capped`.
- A result shorter than the page answers `exact` with no count (A2-4): S21.12 (10), S21.16 (25), S21.17 (0, the hidden value and the value that exists nowhere), S21.18 (Bob's 50).
- S21.13 to S21.16 accept `unknown` on a page with more than a page of matches. Otherwise `total` is still the scan's count up to 1,000.
- S21.19's ≥ 40% ACL cell (Alice, `priority=3`): only 200s, and every response is `capped` or `unknown`. Its p95 is printed, not asserted (moved to S21c, TICK-1159, 2026-11-26). Every cell prints its `total_state` counts and its share of `unknown`. The other D13 lines are unchanged.
- `tools/kit.mjs report` re-checks the same rules on the evidence files, so a run by hand and `run-kit.sh` agree. It writes `p95.txt`: every cell, the ACL cell's `unknown` share, `rfv_num`'s all-visible share just before that cell (from `visibility.txt`, sampled every 10 s while the folder runs), and the no-filter controls. `verify.sql` adds `relpages` / `relallvisible` for both side tables at the end.

The history below is from before the budget. It is kept because it is why A2 exists (TICK-1022).

#### Before ADR-006-A2 (TICK-1022's findings)

Stack 3 (after the TICK-1023 merge) missed again, and its single-client times are high for both values, not just one. That weakens the statistics explanation below: on stacks 2 and 3, `rfv_num` was last autoanalyzed 34 s and 53 s before priority flipped to `ready`, yet one was fast and one slow. TICK-1022 needs the plans, not just the analyze times.

The first fresh stack (10:22Z to 10:52Z, same images, same seed, same expectations: `evidence/stack1-p95-miss/`) passed criteria 1, 3, 4 and 5 and the row checks of 2, but **missed D13 in S21.19 three times out of three**:

| Caller, filter | Limit | Run (10:51Z) | Rerun 1 | Rerun 2 |
|---|---|---|---|---|
| Alice `priority=3` (48%, ACL) | 350 | **578** | **983** | **741** |
| Alice two-filter range (0.14%, ACL) | 150 | **240** | **267** | **264** |
| admin two-filter range (0.14%) | 150 | **201** | **351** | 130 |
| admin `priority=1` (0.99%) | 350 | 330 | **494** | **618** |

The reruns overlapped another stack (`nava`) on the same Docker VM (`contention-during-reruns.txt`), so their numbers are high across the board, controls included. But one probe rules out host load as the whole story. With **one client**, Alice's `priority=3` took **217 ms p50 on stack 1 against 34 ms on stack 2**, while `priority=4` (50%, the same shape) took 61 ms on stack 1 (`stack1-p95-miss/single-client.txt`, `single-client.txt`). Same code, same data, same query, a 6× difference for one value on one stack: the planner chose a different plan for it. The most likely cause is the side table statistics as they stood after the backfill. On stack 2, `rfv_num` was last autoanalyzed at 11:17:39Z, 34 s before priority flipped to `ready` (11:18:13Z), when its backfill was nearly done, and `rfv_ts` at 11:28:39Z, 38 s before due_at's flip. Stack 1's analyze times were not recorded; `verify.sql` records them from now on. **Severity: Breaks under a condition** (the statistics the planner has after a backfill). This kit cannot tell which plan a fresh stack will get. Fix task TICK-1022 (Postgres & Data Modeling Specialist): find the plan that takes 217 ms, and make the ≥ 40% ACL list stable, for example by analyzing the side table when the backfill flips the field to `ready`.

## Isolation check

- Tenant B's 50 priority-1 incidents never appear in tenant A's `filter[priority]=1`. Tenant A's count is the tenant A scan, not the all-tenant one (S21.18). Tenant B has no index rows (`verify.txt`).
- A value that only a record Alice cannot read has gives her the same status and body as a value that exists nowhere: 200, no items, `total` 0 (S21.17).
- The SQL-level attacks on the side tables and the registry are the isolation suite's (TICK-748, `docs/reviews/tick-748-s21-isolation-attacks.md`), in CI. This kit does not repeat them.

## Not included

- Removing a flag (S22; the publish answers 422 `filterable_field_removal_unavailable`).
- The 2–40% bands (TICK-1009, S21b) and the rerun of the provisional D13 cells (TICK-1002).
- Insert throughput with flagged fields (D10, ≥ 40% of the unflagged tps): `docs/budgets.md`, measured by P7-07, not by this kit.

## Known limits

- **Runs once per prepared stack** (S21.8 publishes the flags; `seed.sql` refuses a tenant that already has incidents).
- **The seed bypasses the API.** It writes records as the superuser with `session_replication_role = replica`, so the seeded records have no audit entries and no create history. Every record the kit itself creates or edits goes through the API.
- **p95 at the client, paced.** The D13 numbers in `docs/budgets.md` come from 8 unpaced clients on the handler inside the test process (P7-07). The kit measures through the whole stack instead, and has to stay under the rate limit, so its numbers are close to D13's but not the same measurement.
- **Docker on Windows:** run the script from Git Bash. `run-kit.sh` handles `MSYS_NO_PATHCONV` (TICK-623).
